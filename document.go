package filia

import (
	"net/url"
	"time"
)

type DocumentType int

const (
	DocumentFile = DocumentType(iota)
	DocumentDirectory
	DocumentLink
)

type Document struct {
	URL         *url.URL
	Type        DocumentType
	ContentType string
	Time        time.Time
	Size        int64

	Links []string

	Title        string
	Version      string
	Album        string
	Artist       string
	Performer    string
	Copyright    string
	License      string
	Organisation string
	Genre        string
	Date         string
	ISRC         string
	Author       string
	Description  string

	Content string

	NoIndex  bool
	NoFollow bool
}

func (d *Document) Init() {
	d.Time = time.Now()
}

func (d Document) AbsLinks() (r []string) {
	for _, link := range d.Links {
		l, _ := d.URL.Parse(link)
		r = append(r, l.String())
	}
	return
}
