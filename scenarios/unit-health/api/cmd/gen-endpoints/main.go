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

	"unit-health/internal/modules"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
		exitProcess(1)
	}
}

var exitProcess = os.Exit

func run(args []string) error {
	flags := flag.NewFlagSet("gen-endpoints", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	output := flags.String("output", "../.vrooli/endpoints.json", "path to write the generated endpoints.json")
	manifest := flags.String("manifest", "../cli/manifest.json", "path to the scenario cli manifest")
	if err := flags.Parse(args); err != nil {
		return err
	}

	return gen.Generate(modules.AllEndpoints(), *manifest, *output)
}
