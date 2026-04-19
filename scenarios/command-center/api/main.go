package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

const scenarioName = "command-center"

func main() {
	if preflight.Run(preflight.Config{
		ScenarioName: scenarioName,
	}) {
		return
	}

	registryPath := resolveRegistryPath()
	reg, err := LoadRegistry(registryPath)
	if err != nil {
		log.Fatalf("failed to load gap registry at %s: %v", registryPath, err)
	}
	slog.Info("loaded gap registry", "path", registryPath, "dashboards", len(reg.Dashboards))

	srv := NewServer(reg)

	if err := server.Run(server.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// resolveRegistryPath returns the path to the gap registry JSON file.
// Honors REGISTRY_PATH for tests; otherwise resolves ../config/gap-registry.json
// relative to the API binary's working directory.
func resolveRegistryPath() string {
	if p := os.Getenv("COMMAND_CENTER_REGISTRY_PATH"); p != "" {
		return p
	}
	return filepath.Join("..", "config", "gap-registry.json")
}
