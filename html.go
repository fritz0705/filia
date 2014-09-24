package filia

import (
	"bufio"
	"log"
	"io"
	"strings"

	"code.google.com/p/go.net/html"
)

type HTMLDecoder struct{}

func (h HTMLDecoder) headAndBody(node *html.Node) (head *html.Node, body *html.Node) {
	var f func(node *html.Node)
	f = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch node.Data {
			case "body":
				body = node
				return
			case "head":
				head = node
				return
			}
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(node)
	return
}

func (h HTMLDecoder) scanMetaTag(meta *html.Node, doc *Document) {
	var (
		name    string
		content string
	)

	for _, attr := range meta.Attr {
		switch attr.Key {
		case "name":
			name = attr.Val
		case "content":
			content = attr.Val
		}
	}

	if name == "" || content == "" {
		return
	}

	// TODO Recognize more meta tags?
	switch name {
	case "author":
		doc.Author = content
	case "description":
		doc.Description = content
	}
}

func (h HTMLDecoder) extractText(node *html.Node) (str string) {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			str += c.Data
		} else if c.Type == html.ElementNode {
			str += h.extractText(c)
		}
	}

	return
}

func (h HTMLDecoder) scanLinkTag(link *html.Node, doc *Document) {
	var (
		linkType string
		href     string
	)

	for _, attr := range link.Attr {
		switch attr.Key {
		case "rel":
			linkType = attr.Val
		case "href":
			href = attr.Val
		}
	}

	// FIXME Use linkType to crawl only interesting files
	_ = linkType

	if href != "" {
		doc.Links = append(doc.Links, href)
	}
}

func (h HTMLDecoder) scanHead(head *html.Node, doc *Document) {
	for node := head.FirstChild; node != nil; node = node.NextSibling {
		if node.Type != html.ElementNode {
			continue
		}

		switch node.Data {
		case "title":
			doc.Title = h.extractText(node)
		case "meta":
			h.scanMetaTag(node, doc)
		case "link":
			h.scanLinkTag(node, doc)
		}
	}
}

type htmlSection struct {
	Heading string
	Content string
	Node    *html.Node
}

func (h htmlSection) Empty() bool {
	return h.Heading == "" && h.Content == ""
}

func (h htmlSection) WordCount() int {
	scanner := bufio.NewScanner(strings.NewReader(h.Content))
	scanner.Split(bufio.ScanWords)
	count := 0
	for scanner.Scan() {
		count += 1
	}
	return count
}

func (h HTMLDecoder) scanBody(body *html.Node, doc *Document) {
	// Fetch sections from section scanner
	sections := h.scanSection(body, doc)

	log.Print(sections)

	var (
		headerSections []htmlSection
		mainSections   []htmlSection
		mainWordCount  int
	)

	for _, section := range sections {
		if section.Node.Data == "header" {
			headerSections = append(headerSections, section)
		}
		if section.Node.Data == "header" || section.Node.Data == "footer" {
			continue
		}

		mainSections = append(mainSections, section)
		mainWordCount += section.WordCount()
	}

	var bestSections []htmlSection
	for _, section := range mainSections {
		if section.WordCount() >= (mainWordCount / len(mainSections)) {
			bestSections = append(bestSections, section)
			if section.Heading != "" {
				headerSections = append(headerSections, section)
			}
		}
	}

	if len(bestSections) != 0 {
		content := ""
		var bestSection *htmlSection
		for _, section := range bestSections {
			if bestSection == nil || section.WordCount() > bestSection.WordCount() {
				bestSection = &section
			}
			content += section.Content + " "
		}

		doc.Content = strings.TrimSpace(content)
	}

	if len(headerSections) != 0 && doc.Title == "" {
		var (
			headers   []string
			headerLen int
		)

		for _, section := range headerSections {
			headers = append(headers, section.Heading)
			headerLen += len(section.Heading)
		}

		var header string
		for _, hdr := range headers {
			if len(hdr) >= headerLen/len(headers) {
				header = hdr
			}
		}

		if header != "" {
			doc.Title = header
		}
	}

	// TODO Do more things with sections, e.g. build a tree of sections
}

func (h HTMLDecoder) scanHyperlink(node *html.Node, doc *Document) {
	for _, attr := range node.Attr {
		if attr.Key == "href" {
			doc.Links = append(doc.Links, attr.Val)
		}
	}
}

func (h HTMLDecoder) scanSection(body *html.Node, doc *Document) []htmlSection {
	var (
		curSection = htmlSection{
			Node: body,
		}
		sections []htmlSection
	)

	var f func(node *html.Node)
	f = func(node *html.Node) {
		if node.Type == html.ElementNode {
			descend := true
			switch node.Data {
			case "header", "footer", "main", "section":
				if node != body {
					if !curSection.Empty() {
						sections = append(sections, curSection)
						curSection = htmlSection{Node: body}
					}
					sections = append(sections, h.scanSection(node, doc)...)
					descend = false
				}
			case "h1":
				curSection.Heading = h.extractText(node)
			case "a":
				h.scanHyperlink(node, doc)
			case "script", "style":
				descend = false
			}
			if descend {
				for c := node.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
			}
		} else if node.Type == html.TextNode {
			curSection.Content += node.Data
		}
	}
	f(body)

	if !curSection.Empty() {
		sections = append(sections, curSection)
	}

	return sections
}

func (h HTMLDecoder) Decode(doc *Document, rc io.ReadCloser) error {
	root, err := html.Parse(rc)
	if err != nil {
		return err
	}

	head, body := h.headAndBody(root)
	h.scanHead(head, doc)
	h.scanBody(body, doc)

	return nil
}
