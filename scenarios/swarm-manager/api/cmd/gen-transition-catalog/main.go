// Command gen-transition-catalog renders the operator-facing transition
// catalog from the declared registry.
//
// docs/reference/transition-catalog.md is generated output with a DO NOT EDIT
// header, and a test fails when the committed file drifts from the registry.
// Without this command that test could only tell you the file was stale, not
// give you a way to fix it.
//
// Usage, from scenarios/swarm-manager/api:
//
//	go run ./cmd/gen-transition-catalog
//	go run ./cmd/gen-transition-catalog --check
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"swarm-manager/internal/transitions"
)

func main() {
	check := flag.Bool("check", false, "exit non-zero when the committed catalog differs instead of rewriting it")
	registryDir := flag.String("registry", filepath.Join("..", ".vrooli", "swarm-transitions"), "transition registry directory")
	out := flag.String("out", filepath.Join("..", "docs", "reference", "transition-catalog.md"), "catalog markdown path")
	flag.Parse()

	registry, err := transitions.LoadDir(*registryDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load transition registry: %v\n", err)
		os.Exit(1)
	}
	rendered := transitions.RenderCatalogMarkdown(registry)

	if *check {
		current, readErr := os.ReadFile(*out)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", *out, readErr)
			os.Exit(1)
		}
		if string(current) != rendered {
			fmt.Fprintf(os.Stderr, "%s is stale; regenerate it with: go run ./cmd/gen-transition-catalog\n", *out)
			os.Exit(1)
		}
		fmt.Printf("%s matches the declared registry\n", *out)
		return
	}

	if err := os.WriteFile(*out, []byte(rendered), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *out, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s from %d declared transitions\n", *out, len(registry.Definitions()))
}
