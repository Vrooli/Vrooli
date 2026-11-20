// Command gen-catalog renders the generated provider-catalogue surfaces from
// api/internal/providercatalog/catalog.seed.json. It is the governed owner of
// api/internal/capabilities/catalog.generated.go and
// ui/src/features/providers/providerCatalog.generated.ts; do not hand-edit
// those files.
//
// Usage (from the api module directory):
//
//	go run ./cmd/gen-catalog
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"audio-tools/internal/providercatalog"
)

func main() {
	goOut := flag.String("go-out", "internal/capabilities/catalog.generated.go", "output path for the generated Go catalogue")
	tsOut := flag.String("ts-out", "../ui/src/features/providers/providerCatalog.generated.ts", "output path for the generated TypeScript catalogue")
	flag.Parse()

	goRendered, err := providercatalog.RenderCapabilitiesGo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-catalog: render Go catalogue: %v\n", err)
		os.Exit(1)
	}
	write(*goOut, goRendered, "go catalogue")

	tsRendered, err := providercatalog.RenderTS()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-catalog: render TypeScript catalogue: %v\n", err)
		os.Exit(1)
	}
	write(*tsOut, tsRendered, "typescript catalogue")
}

func write(path, content, label string) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "gen-catalog: create %s: %v\n", dir, err)
			os.Exit(1)
		}
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "gen-catalog: write %s: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%s)\n", path, label)
}
