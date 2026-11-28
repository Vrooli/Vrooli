// gen-endpoints emits .vrooli/endpoints.json from the shared modules
// registry's AllEndpoints() — the same registry main.go consumes for
// AllSchemas. The center never accumulates endpoint metadata; the registry
// collects what each handler package exports.
//
// The generator body (transport validation + the API↔CLI mapping cross-check
// against cli/manifest.json, which is the single source of truth for the CLI
// surface) lives in the shared github.com/vrooli/api-core/endpoints/gen
// package, so this file is a thin wrapper. CI runs
// `make endpoints && git diff --exit-code .vrooli/endpoints.json`; the fix on
// drift is always: run `make endpoints` locally and commit.
//
// Adding a new domain: register it once in
// api/internal/modules/registry.go (one line in AllEndpoints, one in
// AllSchemas) and add the matching command to cli/manifest.json (a binding,
// or an omitted[] entry with a reason). The cross-check at codegen time
// catches drift before the diff gate ever sees it.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"decentralized-deploy/internal/modules"

	gen "github.com/vrooli/api-core/endpoints/gen"
)

var exitProcess = os.Exit

func main() {
	exitProcess(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("gen-endpoints", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("output", "../.vrooli/endpoints.json", "path to write the generated endpoints.json")
	manifest := flags.String("manifest", "../cli/manifest.json", "path to the scenario cli manifest")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if err := gen.Generate(modules.AllEndpoints(), *manifest, *output); err != nil {
		fmt.Fprintf(stderr, "gen-endpoints: %v\n", err)
		return 1
	}
	return 0
}
