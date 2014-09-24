package filia

import (
	"fmt"
	"io"
	"log"
	"net/url"

	"gopkg.in/fatih/set.v0"
)

type (
	Settings struct {
		Protos   map[string]Proto
		Decoders map[string]Decoder
	}

	Crawler struct {
		Settings
		Queue CrawlerQueue
		Set   set.Set

		Output chan Document
		ErrC   chan error
	}
)

var (
	DefaultSettings = Settings{
		Protos: map[string]Proto{
			"http":  HTTPProto{},
			"https": HTTPProto{},
			"ftp":   NewFTPProto(),
			"sftp":  NewSFTPProto(),
		},
		Decoders: map[string]Decoder{
			"text/html":             DefaultHTMLDecoder,
			"application/xhtml+xml": DefaultHTMLDecoder,
			"application/pdf":       DefaultPDFDecoder,
			"image/png":             DefaultImageDecoder,
			"image/jpeg":            DefaultImageDecoder,
			"image/gif":             DefaultImageDecoder,
			"video/webm":            DefaultMediaDecoder,
			"audio/mpeg":            DefaultMediaDecoder,
			"application/ogg":       DefaultMediaDecoder,
			"application/zip":       DefaultZIPDecoder,
			"application/x-gzip":    DefaultGzipDecoder,
		},
	}
	DefaultCrawler = Crawler{
		Settings: DefaultSettings,
		Queue:    make(StdCrawlerQueue),
		Set:      *set.New(),
		Output:   make(chan Document),
	}
)

func NewCrawler() *Crawler {
	return &Crawler{
		Queue:  make(StdCrawlerQueue),
		Set:    *set.New(),
		Output: make(chan Document),
	}
}

func (c *Crawler) Emit(urls ...string) {
	for _, url := range urls {
		if !c.Set.Has(url) {
			c.Queue.Send(url)
			c.Set.Add(url)
		}
	}
}

func (c *Crawler) CrawlURL(url string) (doc Document, err error) {
	doc, body, err := c.Fetch(url)
	if body != nil {
		defer body.Close()
	}
	if err != nil {
		return
	}

	decoder := c.Decoders[doc.ContentType]
	if decoder != nil {
		err = decoder.Decode(&doc, body)
		if err != nil {
			log.Print(err)
			return
		}
	}

	return
}

func (c *Crawler) Crawl() {
	select {
	case _, ok := <-c.Output:
		if !ok {
			// Exit crawler when executed as goroutine and output channel was closed
			return
		}
	default:
		url := c.Queue.Recv()
		doc, err := c.CrawlURL(url)
		if c.ErrC != nil {
			c.ErrC <- err
		}
		c.Output <- doc

		go c.Crawl()
		c.Emit(doc.AbsLinks()...)
	}
}

func (c *Crawler) Fetch(urls string) (Document, io.ReadCloser, error) {
	url, err := url.Parse(urls)
	if err != nil {
		return Document{}, nil, err
	}

	proto := c.Protos[url.Scheme]
	if proto == nil {
		return Document{}, nil, fmt.Errorf("filia: Invalid protocol " + url.Scheme)
	}

	return proto.Get(url)
}

func (c *Crawler) Errors() <-chan error {
	if c.ErrC == nil {
		c.ErrC = make(chan error)
	}
	return c.ErrC
}
