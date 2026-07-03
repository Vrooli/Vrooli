// Audio-tools API binary. Boots through bootstrap.Build so every
// composition concern (env, sqlite, stores, chains, modules) stays
// out of main and behind a single tested entrypoint.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"audio-tools/internal/bootstrap"

	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"
)

func main() {
	if preflight.Run(preflight.Config{ScenarioName: "audio-tools"}) {
		return
	}

	srv, cleanup, err := bootstrap.Build(context.Background())
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	handler := srv.Handler()
	var httpServer *http.Server
	if err := apiserver.Run(apiserver.Config{
		StartServer: func(addr string) error {
			protocols := new(http.Protocols)
			protocols.SetHTTP1(true)
			protocols.SetUnencryptedHTTP2(true)
			httpServer = &http.Server{
				Addr:         addr,
				Handler:      handler,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 180 * time.Second,
				IdleTimeout:  120 * time.Second,
				Protocols:    protocols,
			}
			return httpServer.ListenAndServe()
		},
		ShutdownServer: func(ctx context.Context) error {
			if httpServer == nil {
				return nil
			}
			return httpServer.Shutdown(ctx)
		},
		Cleanup: func(ctx context.Context) error { return cleanup() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
