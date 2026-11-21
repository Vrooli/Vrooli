// Command gen-operations renders the operation catalog's transform-family UI
// specs. The seed is the source of truth; this command exists so regenerating
// the checked-in TypeScript module is one deterministic command.
//
// Usage (from scenarios/audio-tools/api):
//
//	go run ./cmd/gen-operations
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"audio-tools/internal/operationcatalog"
)

func main() {
	out := flag.String("out", "", "output path (default: <scenario>/ui/src/features/diagnostics/audioOps.generated.ts)")
	flag.Parse()

	path := *out
	if path == "" {
		root, err := scenarioRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gen-operations: %v\n", err)
			os.Exit(1)
		}
		path = filepath.Join(root, "ui", "src", "features", "diagnostics", "audioOps.generated.ts")
	}

	catalog, err := operationcatalog.Default()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-operations: load catalog: %v\n", err)
		os.Exit(1)
	}
	rendered, err := operationcatalog.RenderTypeScript(catalog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-operations: render: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, rendered, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "gen-operations: write %s: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d bytes)\n", path, len(rendered))
}

// scenarioRoot walks up from the working directory until it finds the
// scenario's ui/src tree. That makes the command work from api/, from the
// repository root, or from the catalog package directory.
func scenarioRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "ui", "src", "features", "diagnostics")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate scenario root from working directory")
		}
		dir = parent
	}
}
