package filia

import (
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/jlaffaye/ftp"
)

type (
	Proto interface {
		Get(url *url.URL) (doc Document, body io.ReadCloser, err error)
	}

	FTPProto struct {
		conns      map[string]*ftp.ServerConn
		connsMutex sync.Mutex
	}

	HTTPProto struct {
		Client http.Client
	}

	SFTPProto struct {
		conns map[string]*ftp.ServerConn
		Creds map[string][2]string
	}
)

func NewFTPProto() *FTPProto {
	return &FTPProto{
		conns: make(map[string]*ftp.ServerConn),
	}
}

func (p *FTPProto) acquireConn(url_ *url.URL) (conn *ftp.ServerConn, err error) {
	host := url_.Host
	if !strings.ContainsRune(host, ':') {
		host += ":21"
	}
	user, password := "anonymous", "anonymous"
	if url_.User != nil {
		user = url_.User.Username()
		newPassword, has := url_.User.Password()
		if has {
			password = newPassword
		}
	}

	p.connsMutex.Lock()
	defer p.connsMutex.Unlock()

	if p.conns[host] == nil {
		conn, err = ftp.Connect(host)
		if err == nil {
			conn.Login(user, password)
		}
		p.conns[host] = conn
		return
	}

	conn = p.conns[host]
	return
}

func (p *FTPProto) Get(url_ *url.URL) (doc Document, body io.ReadCloser, err error) {
	doc.Init()
	doc.URL = url_

	conn, err := p.acquireConn(url_)
	if err != nil {
		return
	}

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

func NewSFTPProto() *SFTPProto {
	return &SFTPProto{
		conns: make(map[string]*ftp.ServerConn),
		Creds: make(map[string][2]string),
	}
}

func (s *SFTPProto) Get(url_ *url.URL) (doc Document, body io.ReadCloser, err error) {
	return
}
