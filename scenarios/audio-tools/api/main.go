// Audio-tools API binary. Boots through bootstrap.Build so every
// composition concern (env, sqlite, stores, chains, modules) stays
// out of main and behind a single tested entrypoint.
package main

import (
	"context"
	"log"

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

	if err := apiserver.Run(apiserver.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error { return cleanup() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
