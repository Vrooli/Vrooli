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

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"
)

func main() {
	if preflight.Run(preflight.Config{ScenarioName: "audio-tools"}) {
		return
	}

	srv, deps, cleanup, err := bootstrap.BuildWithDeps(context.Background())
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	// Register the dev-only database routing service before the scenario API.
	// Test-genie installs a leased test pool here; the middleware below routes
	// explicitly marked test requests to it and leaves ordinary traffic alone.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, deps.DB, deps.FileRoots)
	rootMux.Handle("/", srv.Handler())
	handler := apihttp.TestModeMiddleware(rootMux)
	var httpServer *http.Server
	if err := apiserver.Run(apiserver.Config{
		StartServer: func(addr string) error {
			protocols := new(http.Protocols)
			protocols.SetHTTP1(true)
			protocols.SetUnencryptedHTTP2(true)
			httpServer = &http.Server{
				Addr:        addr,
				Handler:     handler,
				ReadTimeout: 30 * time.Second,
				// The out-of-band browser qualification endpoint owns a
				// wall-clock 15/60-minute run and returns one conformance
				// document only after the browser session closes. A three-minute
				// write deadline truncates a valid turn into an EOF.
				WriteTimeout: 2 * time.Hour,
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
