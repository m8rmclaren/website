package util

import (
	"io"
)

type multiWriteCloser struct {
	writers []io.WriteCloser
}

func (m *multiWriteCloser) Write(p []byte) (int, error) {
	for _, w := range m.writers {
		if _, err := w.Write(p); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}
func (m *multiWriteCloser) Close() error {
	var first error
	for _, w := range m.writers {
		if err := w.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func NewMultiWriteCloser(writers []io.WriteCloser) *multiWriteCloser {
	return &multiWriteCloser{
		writers: writers,
	}
}
