package util

import (
	"io"
	"net/http"
)

type writeCloser struct {
	http.ResponseWriter
}

func (wc writeCloser) Close() error {
	if f, ok := wc.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}

func WrapResponseWriter(w http.ResponseWriter) io.WriteCloser {
	return writeCloser{w}
}
