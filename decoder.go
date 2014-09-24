package filia

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"path"
)

var (
	DefaultHTMLDecoder  = HTMLDecoder{}
	DefaultPDFDecoder   = PDFDecoder{}
	DefaultImageDecoder = ImageDecoder{}
	DefaultMediaDecoder = MediaDecoder{}
	DefaultZIPDecoder   = ZIPDecoder{}
	DefaultGzipDecoder  = GzipDecoder{}
)

type (
	Decoder interface {
		Decode(doc *Document, rc io.ReadCloser) error
	}

	PDFDecoder   struct{}
	ImageDecoder struct{}
	MediaDecoder struct{}
	ZIPDecoder   struct{}
	TarDecoder   struct{}
	GzipDecoder  struct {
		Tar TarDecoder
	}
)

func (p PDFDecoder) Decode(doc *Document, rc io.ReadCloser) error {
	return nil
}

func (i ImageDecoder) Decode(doc *Document, rc io.ReadCloser) error {
	return nil
}

func (m MediaDecoder) Decode(doc *Document, rc io.ReadCloser) error {
	return nil
}

func (z ZIPDecoder) Decode(doc *Document, rc io.ReadCloser) error {
	return nil
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
	}

	return nil
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
