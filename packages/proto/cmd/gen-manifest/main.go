package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/packages/proto/genmanifest"
)

func main() {
	repoRoot := flag.String("repo-root", filepath.Clean(filepath.Join("..", "..")), "repository root")
	protoRoot := flag.String("proto-root", ".", "packages/proto root")
	flag.Parse()

	scenarios, err := genmanifest.ScenarioNames(*protoRoot)
	if err != nil {
		fatal(err)
	}
	for _, scenario := range scenarios {
		manifest, err := genmanifest.BuildManifest(genmanifest.Options{
			RepoRoot:  *repoRoot,
			ProtoRoot: *protoRoot,
		}, scenario)
		if err != nil {
			fatal(fmt.Errorf("%s: %w", scenario, err))
		}
		if err := genmanifest.WriteManifest(genmanifest.ManifestPath(*protoRoot, scenario), manifest); err != nil {
			fatal(fmt.Errorf("%s: %w", scenario, err))
		}
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
