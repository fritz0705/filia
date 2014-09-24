package filia

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"path"

	"code.google.com/p/go.net/html"
)

type Decoder interface {
	Decode(doc *Document, rc io.ReadCloser) error
}

var (
	DefaultHTMLDecoder  = HTMLDecoder{}
	DefaultPDFDecoder   = PDFDecoder{}
	DefaultImageDecoder = ImageDecoder{}
	DefaultMediaDecoder = MediaDecoder{}
	DefaultZIPDecoder   = ZIPDecoder{}
	DefaultGzipDecoder  = GzipDecoder{}
)

type HTMLDecoder struct {
}

func (h HTMLDecoder) Decode(doc *Document, rc io.ReadCloser) error {
	root, err := html.Parse(rc)
	if err != nil {
		return err
	}

	c := make(chan string)
	var f func(*html.Node)
	f = func(node *html.Node) {
		if node.Type == html.ElementNode && (node.Data == "a" || node.Data == "img") {
			// Link
			var link string
			for _, attr := range node.Attr {
				if attr.Key == "href" || attr.Key == "src" {
					link = attr.Val
				}
			}

			if link != "" {
				c <- link
			}
		} else if node.Type == html.TextNode && node.Parent != nil && node.Parent.Type == html.ElementNode && node.Parent.Data == "title" {
			doc.Title = node.Data
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	go func() {
		f(root)
		close(c)
	}()

	for link := range c {
		doc.Links = append(doc.Links, link)
	}

	return nil
}

type PDFDecoder struct {
}

func (p PDFDecoder) Decode(doc *Document, rc io.ReadCloser) error {
	return nil
}

type ImageDecoder struct {
}

func (i ImageDecoder) Decode(doc *Document, rc io.ReadCloser) error {
	return nil
}

type MediaDecoder struct {
}

func (m MediaDecoder) Decode(doc *Document, rc io.ReadCloser) error {
	return nil
}

type ZIPDecoder struct {
}

func (z ZIPDecoder) Decode(doc *Document, rc io.ReadCloser) error {
	return nil
}

type TarDecoder struct {
}

func (t TarDecoder) Decode(doc *Document, rc io.ReadCloser) error {
	r := tar.NewReader(rc)

	for {
		_, err := r.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		// TODO Parse subdocuments
	}

	return nil
}

type GzipDecoder struct {
	Tar TarDecoder
}

func (g GzipDecoder) Decode(doc *Document, rc io.ReadCloser) error {
	r, err := gzip.NewReader(rc)
	if err != nil {
		return err
	}
	defer r.Close()

	if path.Ext(r.Name) == ".tar" {
		return g.Tar.Decode(doc, r)
	}

	doc.Title = r.Name

	return nil
}
