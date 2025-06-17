package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a-h/templ"
	"github.com/m8rmclaren/website/internal/controller"
	"github.com/m8rmclaren/website/view"
)

func main() {
	var addr string
	flag.StringVar(&addr, "addr", ":3000", "The address the service will bind to")

	var staticDirectory string
	flag.StringVar(&staticDirectory, "static-root", "static", "The address the service will bind to")

	flag.Parse()

	ctx := context.Background()

	index := view.Index("Alyssa")

	service := controller.NewController(ctx,
		controller.WithStaticRoot(staticDirectory),
		controller.WithStaticFile(staticDirectory, "favicon.ico"),
		controller.WithStaticFile(staticDirectory, "robots.txt"),
		controller.WithRoute("", templ.Handler(index)),
	)

	srv := &http.Server{
		Addr:    addr,
		Handler: service,
	}

	// Start the server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()
	log.Printf("server started [%s]", addr)

	// Wait for a signal to shut down
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	s := <-sigc
	log.Printf("signal received [%v], shutting down", s)

	// Create a deadline to wait for the shutdown
	ctxShutDown, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()

	if err := srv.Shutdown(ctxShutDown); err != nil {
		log.Fatalf("server shutdown failed:%+v", err)
	}
	log.Println("server exited properly")

}
