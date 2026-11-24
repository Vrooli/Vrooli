// Audio-tools API binary. Boots through bootstrap.Build so every
// composition concern (env, sqlite, stores, chains, modules) stays
// out of main and behind a single tested entrypoint.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"audio-tools/internal/bootstrap"
	audioserver "audio-tools/internal/server"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/devrouting"
	apilifecycle "github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/provenance"
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
	provider, err := apilifecycle.NewProviderFromEnv(nil, apilifecycle.Hooks{})
	if err != nil {
		log.Fatalf("lifecycle provider setup failed: %v", err)
	}
	authConfig, err := audioserver.AuthenticationConfig(os.Getenv)
	if err != nil {
		log.Fatalf("authentication configuration failed: %v", err)
	}
	provenance.InstallDefaultForwardingTransport()
	handler := audioserver.IdentityMiddleware(authConfig)(provider.Mount(apihttp.TestModeMiddleware(rootMux)))
	var httpServer *http.Server
	if err := apiserver.Run(apiserver.Config{
		StartServer: func(addr string) error {
			protocols := new(http.Protocols)
			protocols.SetHTTP1(true)
			protocols.SetUnencryptedHTTP2(true)
			httpServer = &http.Server{
				Addr:              addr,
				Handler:           handler,
				ReadHeaderTimeout: 10 * time.Second,
				// The absolute request-body read deadline must cover long-lived bidi
				// STT streams and the 15/60-minute browser qualification run. Per-message
				// size bounds and the stream inactivity timeout remain the tighter
				// controls for untrusted input.
				ReadTimeout: 2 * time.Hour,
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
		LifecycleConfiguredExternally: true,
		Cleanup:                       func(ctx context.Context) error { return cleanup() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
