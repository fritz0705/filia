package filia

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/jlaffaye/ftp"
	"github.com/pkg/sftp"

	"code.google.com/p/go.crypto/ssh"
)

type (
	Proto interface {
		Get(url *url.URL) (doc Document, body io.ReadCloser, err error)
	}

	FTPProto struct {
		conns       map[string]*ftp.ServerConn
		connsMutex  sync.Mutex
		connMutexes map[string]*sync.Mutex
	}

	HTTPProto struct {
		Client http.Client
	}

	SFTPProto struct {
		conns       map[string]*sftp.Client
		connsMutex  sync.Mutex
		connMutexes map[string]*sync.Mutex
	}

	FileProto struct {
	}
)

func NewFTPProto() *FTPProto {
	return &FTPProto{
		conns:       make(map[string]*ftp.ServerConn),
		connMutexes: make(map[string]*sync.Mutex),
	}
}

func (p *FTPProto) acquireConn(url_ *url.URL) (conn *ftp.ServerConn, mutex *sync.Mutex, err error) {
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
		mutex = new(sync.Mutex)
		p.conns[host] = conn
		p.connMutexes[host] = mutex
		return
	}

	conn = p.conns[host]
	mutex = p.connMutexes[host]
	return
}

func (p *FTPProto) Get(url_ *url.URL) (doc Document, body io.ReadCloser, err error) {
	doc.Init()
	doc.URL = url_

	conn, mutex, err := p.acquireConn(url_)
	if err != nil {
		return
	}
	mutex.Lock()
	defer mutex.Unlock()

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
		conns:       make(map[string]*sftp.Client),
		connMutexes: make(map[string]*sync.Mutex),
	}
}

func (s *SFTPProto) acquireConn(url_ *url.URL) (conn *sftp.Client, mutex *sync.Mutex, err error) {
	s.connsMutex.Lock()
	defer s.connsMutex.Unlock()

	host := url_.Host
	if !strings.ContainsRune(host, ':') {
		host += ":22"
	}

	if s.conns[host] == nil {
		if url_.User == nil {
			err = fmt.Errorf("sftp: Missing credentials in URL")
			return
		}

		config := &ssh.ClientConfig{
			User: url_.User.Username(),
		}

		if pw, ok := url_.User.Password(); ok {
			config.Auth = append(config.Auth, ssh.Password(pw))
		}

		ssh, err := ssh.Dial("tcp", host, config)
		if err != nil {
			return conn, mutex, err
		}

		conn, err = sftp.NewClient(ssh)
		if err != nil {
			return conn, mutex, err
		}
		mutex = new(sync.Mutex)

		s.conns[host] = conn
		s.connMutexes[host] = mutex
		return conn, mutex, err
	}

	conn = s.conns[host]
	mutex = s.connMutexes[host]
	return
}

func (s *SFTPProto) Get(url_ *url.URL) (doc Document, body io.ReadCloser, err error) {
	conn, mutex, err := s.acquireConn(url_)
	if err != nil {
		return
	}
	mutex.Lock()
	defer mutex.Unlock()

	fileInfo, err := conn.Lstat(url_.Path)
	if err != nil {
		return
	}

	doc.Init()
	doc.URL = url_

	if fileInfo.IsDir() {
		doc.Type = DocumentDirectory
		files, err := conn.ReadDir(url_.Path)
		if err != nil {
			return doc, body, err
		}
		for _, file := range files {
			doc.Links = append(doc.Links, file.Name())
		}
		return doc, body, err
	} else if !fileInfo.Mode().IsRegular() {
		doc.Type = DocumentSpecial
		return
	}

	body, err = conn.Open(url_.Path)
	doc.ContentType = "application/octet-stream"

	return
}

func (p *FileProto) Get(url_ *url.URL) (doc Document, body io.ReadCloser, err error) {
	fileInfo, err := os.Lstat(url_.Path)
	if err != nil {
		return
	}

	doc.Init()
	doc.URL = url_

	if fileInfo.IsDir() {
		doc.Type = DocumentDirectory
		files, err := ioutil.ReadDir(url_.Path)
		if err != nil {
			return doc, body, err
		}
		for _, file := range files {
			doc.Links = append(doc.Links, file.Name())
		}
		return doc, body, err
	} else if !fileInfo.Mode().IsRegular() {
		doc.Type = DocumentSpecial
		return
	}

	body, err = os.Open(url_.Path)
	doc.ContentType = "application/octet-stream"

	return
}
