package image

import (
	"fmt"
	"io"
	"net/http"

	"github.com/m8rmclaren/website/internal/util"
	"github.com/m8rmclaren/website/vips"
)

type saver interface {
	save(img *vips.Image) error
	contentType() string
}

var _ saver = &webpSaver{}

type webpSaver struct {
	writer io.WriteCloser
}

func newWebpSaver(writer http.ResponseWriter) *webpSaver {
	return &webpSaver{
		writer: util.WrapResponseWriter(writer),
	}
}

// save implements saver.
func (w *webpSaver) save(img *vips.Image) error {
	target := vips.NewTarget(w.writer)
	defer target.Close()

	// Save the result as WebP target with options
	err := img.WebpsaveTarget(target, &vips.WebpsaveTargetOptions{
		Q:              100,  // Quality factor (0-100)
		Effort:         0,    // Compression effort (0-6)
		SmartSubsample: true, // Better chroma subsampling
	})
	if err != nil {
		return fmt.Errorf("%w: %w: failed to save image as webp", libvipsSharpenError, err)
	}

	return nil
}

// contentType implements saver.
func (w *webpSaver) contentType() string {
	return "image/webp"
}
