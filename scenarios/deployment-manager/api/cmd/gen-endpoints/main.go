package main

import (
	"flag"
	"fmt"
	"os"

	"deployment-manager/internal/modules"

	gen "github.com/vrooli/api-core/endpoints/gen"
)

func main() {
	output := flag.String("output", "../.vrooli/endpoints.json", "path to write endpoints.json")
	manifest := flag.String("manifest", "../cli/manifest.json", "path to cli manifest")
	flag.Parse()
	if err := gen.Generate(modules.AllEndpoints(), *manifest, *output); err != nil {
		fmt.Fprintf(os.Stderr, "gen-endpoints: %v\n", err)
		os.Exit(1)
	}
}
