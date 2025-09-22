package image

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/m8rmclaren/website/internal/api"
	"github.com/m8rmclaren/website/internal/cache"
	"github.com/m8rmclaren/website/internal/util"
	"github.com/m8rmclaren/website/vips"
)

const (
	imageSourceQueryParam = "src"
	imageWidthQueryParam  = "w"
)

var (
	errFileNotFound     = errors.New("file not found")
	errInvalidWidth     = errors.New("invalid width")
	libvipsError        = errors.New("libvips error")
	libvipsLoadError    = fmt.Errorf("%w: load error", libvipsError)
	libvipsResizeError  = fmt.Errorf("%w: resize error", libvipsError)
	libvipsSharpenError = fmt.Errorf("%w: sharpen error", libvipsError)
	libvpsSaveError     = fmt.Errorf("%w: save error", libvipsError)
)

type bufferWriteCloser struct{ buf *bytes.Buffer }

func (b bufferWriteCloser) Write(p []byte) (int, error) { return b.buf.Write(p) }
func (b bufferWriteCloser) Close() error                { return nil } // no-op

type optimizer struct {
	staticDirectoryName string
	logger              echo.Logger
	cache               cache.Cache
}

func NewImageOptimizer(staticDirectoryName string, logger echo.Logger, cache cache.Cache) *optimizer {
	return &optimizer{
		staticDirectoryName: staticDirectoryName,
		logger:              logger,
		cache:               cache,
	}
}

// Handler returns an Echo HandlerFunc that invokes the image optimizer to serve the requested image size back to the
// browser.
// Syntax for query params:
// - src : The path to the image served out of the static content directory. E.g., "/image/image.jpeg"
// - w   : The width of the image that should be returned
func (o *optimizer) Handler() echo.HandlerFunc {
	return func(c echo.Context) error {
		imageSource := c.QueryParam(imageSourceQueryParam)

		w := c.QueryParam(imageWidthQueryParam)
		width, err := strconv.Atoi(w)
		if err != nil {
			return api.RespondError(c, http.StatusBadRequest, fmt.Errorf("%w: failed to convert %s to int: %w", errInvalidWidth, w, err))
		}

		startTime := time.Now()
		res := c.Response()

		// Prepare a multi-write closer to stream output to a buffer for cache and to the response
		var buf bytes.Buffer
		mw := util.NewMultiWriteCloser([]io.WriteCloser{
			util.WrapResponseWriter(res.Writer),
			bufferWriteCloser{buf: &buf},
		})

		// Select the appropriate image saver depending on Accept headers
		var saver saver
		saver = newWebpSaver(mw)

		// Prepare response headers up front - response is streamed to requestor
		res.Header().Set(echo.HeaderContentType, saver.contentType())
		res.WriteHeader(http.StatusOK)

		cacheKey := imageCacheKey(imageSource, width)
		cachedImage, hit, err := o.cache.Get(context.Background(), cacheKey)
		if err != nil {
			return err
		}

		if hit {
			_, err = res.Write(cachedImage)
			if err != nil {
				return err
			}
			c.Logger().Printf("Returned cached image for %s [w=%d] in %s", imageSource, width, time.Since(startTime))
			return nil
		}

		err = o.resizeToWriter(imageSource, width, saver)
		switch {
		case errors.Is(err, errFileNotFound):
			api.RespondError(c, http.StatusNotFound, err)
			return nil
		case err != nil:
			api.RespondError(c, http.StatusInternalServerError, fmt.Errorf("couldn't generate properly sized image"))
			return nil
		}
		c.Logger().Printf("Resized %s [w=%d] in %s", imageSource, width, time.Since(startTime))

		err = o.cache.Set(context.Background(), cacheKey, buf.Bytes())
		if err != nil {
			return err
		}

		return nil
	}
}

func (o *optimizer) resizeToWriter(path string, width int, saver saver) error {

	filePath := filepath.Join(o.staticDirectoryName, strings.TrimLeft(path, "/"))

	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("%w: couldn't find file at '%s'", errFileNotFound, path)
	}

	loadOpts := &vips.LoadOptions{} // empty: no 'n' passed

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".gif", ".webp", ".tif", ".tiff", ".pdf", ".heif", ".heic", ".svg":
		// For multi-page/animated formats you actually want all frames/pages
		loadOpts.N = -1 // load all
	}

	img, err := vips.NewImageFromFile(filePath, loadOpts)
	if err != nil {
		return fmt.Errorf("%w: %w", libvipsLoadError, err)
	}
	defer img.Close()

	// Maintain aspect ratio based on original dimensions
	inW := img.Width()
	if inW == 0 {
		return fmt.Errorf("%w: image loaded from %s had an invalid width", errInvalidWidth, path)
	}
	scale := float64(width) / float64(inW)
	if scale > 1 {
		o.logger.Printf("Not scaling %s [w=%d] since scale factor is %.3f", filePath, width, scale)
		scale = 1 // don’t upscale images
	}

	resizeOpts := vips.DefaultResizeOptions()
	resizeOpts.Kernel = vips.KernelLanczos3
	// resizeOpts.Kernel = vips.KernelMks2021

	err = img.Resize(scale, resizeOpts)
	if err != nil {
		return fmt.Errorf("%w: %w", libvipsResizeError, err)
	}

	err = img.Sharpen(&vips.SharpenOptions{
		Sigma: 0.9, // overall strength (~0.6–1.2 is common)
		X1:    2,   // low-frequency control
		Y2:    10,  // amplitude for high-pass
		Y3:    20,  // cutoff
		M1:    0,   // thresholding params; keep small
		M2:    0,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", libvipsSharpenError, err)
	}

	// Use the save interface to write the image to the correct format
	return saver.save(img)
}

func (o *optimizer) thumbnailToWriter(path string, width int, saver saver) error {
	filePath := filepath.Join(o.staticDirectoryName, strings.TrimLeft(path, "/"))

	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("%w: couldn't find file at '%s'", errFileNotFound, path)
	}

	thumbnailOptions := &vips.ThumbnailOptions{}

	img, err := vips.NewThumbnail(filePath, width, thumbnailOptions)
	if err != nil {
		return fmt.Errorf("%w: %w", libvipsLoadError, err)
	}
	defer img.Close()

	// // Maintain aspect ratio based on original dimensions
	// inW := img.Width()
	// if inW == 0 {
	// 	return fmt.Errorf("%w: image loaded from %s had an invalid width", errInvalidWidth, path)
	// }
	// scale := float64(width) / float64(inW)
	// if scale > 1 {
	// 	scale = 1 // don’t upscale images
	// }

	// resizeOpts := vips.DefaultResizeOptions()
	// resizeOpts.Kernel = vips.KernelLanczos3

	// err = img.Resize(scale, resizeOpts)
	// if err != nil {
	// 	return fmt.Errorf("%w: %w", libvipsResizeError, err)
	// }

	// err = img.Sharpen(&vips.SharpenOptions{
	// 	Sigma: 0.9, // overall strength (~0.6–1.2 is common)
	// 	X1:    2,   // low-frequency control
	// 	Y2:    10,  // amplitude for high-pass
	// 	Y3:    20,  // cutoff
	// 	M1:    0,   // thresholding params; keep small
	// 	M2:    0,
	// })
	// if err != nil {
	// 	return fmt.Errorf("%w: %w", libvipsSharpenError, err)
	// }

	return saver.save(img)
}

func imageCacheKey(href string, w int) string {
	return fmt.Sprintf("cache:%s:w", href, w)
}
