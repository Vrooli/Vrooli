// gen-endpoints emits .vrooli/endpoints.json from the shared modules
// registry's AllEndpoints(). The generator body (transport validation + the
// API↔CLI mapping cross-check against cli/manifest.json, the single source of
// truth for the CLI surface) lives in github.com/vrooli/api-core/endpoints/gen,
// so this file is a thin wrapper. CI runs `make endpoints && git diff
// --exit-code .vrooli/endpoints.json`; the fix on drift is always: run
// `make endpoints` locally and commit.
package main

import (
	"flag"
	"fmt"
	"os"

	gen "github.com/vrooli/api-core/endpoints/gen"

	"proto-health/internal/modules"
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
