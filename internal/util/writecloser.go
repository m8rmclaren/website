package util

import (
	"io"
	"net/http"
)

type writeCloser struct {
	io.Writer
}

func (wc writeCloser) Close() error {
	if f, ok := wc.Writer.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}

func WrapResponseWriter(w http.ResponseWriter) io.WriteCloser {
	return writeCloser{w}
}
