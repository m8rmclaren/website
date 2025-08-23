package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/m8rmclaren/website/internal/cache"
	"github.com/m8rmclaren/website/internal/image"
	"github.com/m8rmclaren/website/internal/render"
	"github.com/m8rmclaren/website/template/view"
	"github.com/m8rmclaren/website/template/view/blog"
	"github.com/m8rmclaren/website/vips"
)

func main() {
	var addr string
	flag.StringVar(&addr, "addr", ":3000", "The address the service will bind to")

	var staticDirectory string
	flag.StringVar(&staticDirectory, "static-root", "static", "The address the service will bind to")

	flag.Parse()

	// Setup libvips
	vips.Startup(nil)

	// Setup Echo
	e := echo.New()
	render.NewTemplateRenderer(e)

	// Add health route
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Setup fileserver
	// (serve any file from static directory for path /*)
	e.Static("/", staticDirectory)

	// Add index route
	indexConfig := view.NewPageConfig("Hayden Roszell", "Full-stack engineer specializing in Kubernetes, Go, and cloud-native infrastructure.")
	e.GET("/", func(c echo.Context) error {
		c.Logger().Printf("Serving GET / [ip %s]", c.RealIP())
		return render.Html(c, view.Page(indexConfig, view.Index()))
	})

	// Add SES blog post route
	e.GET("/blog/simple-environment-service", func(c echo.Context) error {
		description := "Click to Deploy: Scalable, On-Demand Application Provisioning using Kubernetes"
		c.Logger().Printf("Serving GET /blog/simple-environment-service [ip %s]", c.RealIP())
		config := blog.NewBlogPostConfig(
			description,
			"How I built a scalable, declarative platform that deploys and configures apps in Kubernetes for dev, demo, and testing using Go and ArgoCD.",
			"Hayden Roszell",
			"Jun 18, 2025",
			"/images/headshot.jpeg",
			12,
		)
		return render.Html(c, view.Page(view.NewPageConfig("Click to Deploy", description), blog.BlogPost(config, blog.SESBlog())))
	})

	// Add image optimizer route
	e.GET("_image", image.NewImageOptimizer(staticDirectory, e.Logger, cache.NewSimpleCache(0)).Handler())

	// Start the server in a goroutine
	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	// Wait for a signal to shut down
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	s := <-sigc
	e.Logger.Printf("signal received [%v], shutting down", s)

	// Create a deadline to wait for the shutdown
	ctxShutDown, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()

	vips.Shutdown()

	if err := e.Shutdown(ctxShutDown); err != nil {
		e.Logger.Fatalf("server shutdown failed:%+v", err)
	}
	e.Logger.Print("server exited properly")

}
