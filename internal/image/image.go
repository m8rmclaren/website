package image

import (
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/m8rmclaren/website/vips"
)

const (
	imageSourceQueryParam = "src"
	imageWidthQueryParam  = "w"
)

type optimizer struct {
}

type image struct {
	content     string
	contentType string
}

func NewImageOptimizer() *optimizer {
	return &optimizer{}
}

// Handler returns an Echo HandlerFunc that invokes the image optimizer to serve the requested image size back to the
// browser.
// Syntax for query params:
// - src : The path to the image served out of the static content directory. E.g., "/image/image.jpeg"
// - w   : The width of the image that should be returned
func (o *optimizer) Handler() echo.HandlerFunc {
	return func(c echo.Context) error {
		src := c.QueryParam(imageSourceQueryParam)
		w := c.QueryParam(imageWidthQueryParam)

		c.Logger().Printf("Serving GET %s src=%s w=%s, [ip %s]", c.Path(), src, w, c.RealIP())
		startTime := time.Now()
		test()
		log.Printf("Test took %s", time.Since(startTime))

		return nil
	}
}

func test() {
	// Fetch an image from http.Get
	resp, err := http.Get("https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png")
	if err != nil {
		log.Fatalf("Failed to fetch image: %v", err)
	}
	defer resp.Body.Close()

	// Create source from io.ReadCloser
	source := vips.NewSource(resp.Body)
	defer source.Close() // source needs to remain available during image lifetime

	// Shrink-on-load via creating image from thumbnail source with options
	image, err := vips.NewThumbnailSource(source, 800, &vips.ThumbnailSourceOptions{
		Height: 1000,
		FailOn: vips.FailOnError, // Fail on first error
	})
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}
	defer image.Close() // always close images to free memory

	// Add a yellow border using vips_embed
	border := 10
	if err := image.Embed(
		border, border,
		image.Width()+border*2,
		image.Height()+border*2,
		&vips.EmbedOptions{
			Extend:     vips.ExtendBackground,       // extend with colour from the background property
			Background: []float64{255, 255, 0, 255}, // Yellow border
		},
	); err != nil {
		log.Fatalf("Failed to add border: %v", err)
	}

	log.Printf("Processed image: %dx%d\n", image.Width(), image.Height())

	log.Println("Successfully saved processed images")
}
