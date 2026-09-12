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
	"os"

	gen "github.com/vrooli/api-core/endpoints/gen"

	"workflow-health/internal/modules"
)

func main() {
	output := flag.String("output", "../.vrooli/endpoints.json", "path to write the generated endpoints.json")
	manifest := flag.String("manifest", "../cli/manifest.json", "path to the scenario cli manifest")
	flag.Parse()

	if err := gen.Generate(modules.AllEndpoints(), *manifest, *output); err != nil {
		fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
		os.Exit(1)
	}
}
