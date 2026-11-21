// gen-endpoints emits .vrooli/endpoints.json from the shared modules
// registry's AllEndpoints(), which collects what each handler package
// exports.
//
// The generator body (transport validation + the API↔CLI mapping cross-check
// against cli/manifest.json, which is the single source of truth for the CLI
// surface) lives in the shared github.com/vrooli/api-core/endpoints/gen
// package, so this file is a thin wrapper. The fix on drift is `make
// endpoints`.
//
// Adding a domain: register it in api/internal/app/registry.go and add
// the matching commands to cli/manifest.json (a binding, or an omitted[]
// entry with a reason).
package main

import (
	"flag"
	"fmt"
	"os"

	"research-desk/internal/app"

	gen "github.com/vrooli/api-core/endpoints/gen"
)

func main() {
	output := flag.String("output", "../.vrooli/endpoints.json", "path to write the generated endpoints.json")
	manifest := flag.String("manifest", "../cli/manifest.json", "path to the scenario cli manifest")
	flag.Parse()

	if err := gen.Generate(app.AllEndpoints(), *manifest, *output); err != nil {
		fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
		os.Exit(1)
	}
}
