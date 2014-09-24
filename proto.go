package filia

import (
	"io"
	"mime"
	"net/http"
	"net/url"
	"sync"

	"github.com/jlaffaye/ftp"
)

type Proto interface {
	Get(url *url.URL) (doc Document, body io.ReadCloser, err error)
}

type FTPProto struct {
	conns      map[string]*ftp.ServerConn
	connsMutex sync.Mutex
}

func NewFTPProto() *FTPProto {
	return &FTPProto{
		conns: make(map[string]*ftp.ServerConn),
	}
}

func (p *FTPProto) Get(url_ *url.URL) (doc Document, body io.ReadCloser, err error) {
	doc.Init()
	doc.URL = url_
	if p.conns[url_.Host] == nil {
		conn, err := ftp.Connect(url_.Host + ":21")
		if err != nil {
			return doc, body, err
		}
		conn.Login("anonymous", "anonymous")
		p.conns[url_.Host] = conn
	}
	conn := p.conns[url_.Host]

	if url_.Path[len(url_.Path)-1] == '/' {
		doc.Type = DocumentDirectory
		entries, err := conn.List(url_.Path)
		if err != nil {
			return doc, body, err
		}
		for _, entry := range entries {
			link := entry.Name
			if entry.Type == ftp.EntryTypeFolder {
				link = entry.Name + "/"
			}
			doc.Links = append(doc.Links, link)
		}

		return doc, body, err
	}

	body, err = conn.Retr(url_.Path)
	doc.ContentType = "application/octet-stream"

	return
}

type HTTPProto struct {
	Client http.Client
}

func (p HTTPProto) Get(url_ *url.URL) (doc Document, body io.ReadCloser, err error) {
	doc.Init()
	doc.URL = url_

	resp, err := p.Client.Get(url_.String())
	if err != nil {
		return
	}

	doc.Type = DocumentFile
	doc.Size = resp.ContentLength
	doc.ContentType, _, err = mime.ParseMediaType(resp.Header.Get("Content-Type"))
	body = resp.Body

	return
}

type SFTPProto struct {
	conns map[string]*ftp.ServerConn
	Creds map[string][2]string
}

func NewSFTPProto() *SFTPProto {
	return &SFTPProto{
		conns: make(map[string]*ftp.ServerConn),
		Creds: make(map[string][2]string),
	}
}

func (s *SFTPProto) Get(url_ *url.URL) (doc Document, body io.ReadCloser, err error) {
	return
}
