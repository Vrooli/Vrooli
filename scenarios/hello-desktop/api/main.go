package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hello-desktop-api/desktop"
)

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "23100"
	}
	handler, lifecycle, err := desktop.New()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := serve(ctx, "127.0.0.1:"+port, handler, lifecycle); err != nil {
		log.Fatal(err)
	}
}

func serve(ctx context.Context, address string, handler http.Handler, lifecycle desktop.Lifecycle) error {
	if err := lifecycle.Start(ctx); err != nil {
		return err
	}
	server := &http.Server{Addr: address, Handler: handler}
	serveResult := make(chan error, 1)
	go func() { serveResult <- server.ListenAndServe() }()
	select {
	case err := <-serveResult:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			_ = lifecycle.Stop(context.Background())
			return err
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		shutdownErr := server.Shutdown(shutdownCtx)
		cancel()
		serveErr := <-serveResult
		lifecycleErr := lifecycle.Stop(context.Background())
		if shutdownErr != nil {
			return shutdownErr
		}
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return serveErr
		}
		return lifecycleErr
	}
	return lifecycle.Stop(context.Background())
}
