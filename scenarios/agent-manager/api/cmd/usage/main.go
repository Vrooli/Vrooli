// The optional desktop usage component serves Agent Manager's quota API.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"agent-manager/internal/pricing"
	"agent-manager/internal/usageslice"
)

func main() {
	port, err := strconv.Atoi(os.Getenv("API_PORT"))
	if err != nil || port < 1 || port > 65535 {
		log.Fatal("API_PORT must be assigned by the lifecycle or desktop runtime")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: usageslice.New(pricing.DefaultClaudeStatuslineSamplesPath()), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
