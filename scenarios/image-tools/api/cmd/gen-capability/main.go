// Command gen-capability renders the generated capability surfaces from the
// manifest seed: the UI catalog and the AI CLI command table. It is the governed
// owner of ui/src/features/workspace/capabilityCatalog.generated.ts and
// cli/domains/*/commands.generated.go; do not hand-edit those files.
//
// Usage (from the api module directory):
//
//	go run ./cmd/gen-capability -out ../ui/src/features/workspace/capabilityCatalog.generated.ts
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"image-tools/internal/capability"
	"image-tools/internal/wireops"
)

func main() {
	out := flag.String("out", "../ui/src/features/workspace/capabilityCatalog.generated.ts", "output path for the generated TypeScript catalog")
	cliOut := flag.String("cli-out", "../cli/domains/ai/commands.generated.go", "output path for the generated Go AI CLI command table")
	opsOut := flag.String("ops-out", "../cli/domains/ops/commands.generated.go", "output path for the generated Go deterministic CLI command table")
	opspecsOut := flag.String("opspecs-out", "../ui/src/features/workspace/opSpecs.generated.ts", "output path for the generated deterministic parameter forms")
	wireOut := flag.String("wire-out", "internal/wireops/bindings.generated.go", "output path for the generated deterministic wire-parameter bindings")
	flag.Parse()

	write := func(path, content, label string) {
		if dir := filepath.Dir(path); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "gen-capability: create %s: %v\n", dir, err)
				os.Exit(1)
			}
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "gen-capability: write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s (%s)\n", path, label)
	}

	rendered, err := capability.RenderTS()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-capability: %v\n", err)
		os.Exit(1)
	}
	write(*out, rendered, "ui catalog")

	cliRendered, err := capability.RenderCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-capability: %v\n", err)
		os.Exit(1)
	}
	write(*cliOut, cliRendered, "ai cli commands")

	opsRendered, err := capability.RenderOpsCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-capability: %v\n", err)
		os.Exit(1)
	}
	write(*opsOut, opsRendered, "ops cli commands")

	opspecsRendered, err := capability.RenderOpSpecs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-capability: %v\n", err)
		os.Exit(1)
	}
	write(*opspecsOut, opspecsRendered, "ops parameter forms")

	wireRendered, err := wireops.RenderBindings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-capability: %v\n", err)
		os.Exit(1)
	}
	write(*wireOut, wireRendered, "ops wire bindings")
}
