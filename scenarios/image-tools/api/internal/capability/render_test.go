package capability

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGeneratedUICatalogIsUpToDate fails when the committed generated TypeScript
// catalog does not match what the manifest seed renders. This is the drift gate
// for the one surface that used to degrade silently: an operation present in Go
// but missing from the UI catalog.
//
// Fix by running `make capability-gen` from the scenario root.
func TestGeneratedUICatalogIsUpToDate(t *testing.T) {
	want, err := RenderTS()
	if err != nil {
		t.Fatalf("RenderTS: %v", err)
	}
	path := filepath.Join("..", "..", "..", "ui", "src", "features", "workspace", "capabilityCatalog.generated.ts")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated catalog %s: %v (run `make capability-gen`)", path, err)
	}
	if string(got) != want {
		t.Fatalf("generated UI catalog is stale; run `make capability-gen` from the scenario root")
	}
}

// TestGeneratedCLICommandsUpToDate fails when the committed generated CLI command
// table does not match the manifest seed. This is the drift gate for the AI
// subcommand surface: a manifest with a cli spec that never reached the CLI, or a
// hand-edited generated file, fails the build.
//
// Fix by running `make capability-gen` from the scenario root.
func TestGeneratedCLICommandsUpToDate(t *testing.T) {
	want, err := RenderCLI()
	if err != nil {
		t.Fatalf("RenderCLI: %v", err)
	}
	path := filepath.Join("..", "..", "..", "cli", "domains", "ai", "commands.generated.go")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated CLI table %s: %v (run `make capability-gen`)", path, err)
	}
	if string(got) != want {
		t.Fatalf("generated CLI command table is stale; run `make capability-gen` from the scenario root")
	}
}
