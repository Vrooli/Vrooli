package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/packages/monetization-go"
)

func main() {
	root := flag.String("root", ".", "repository root")
	out := flag.String("out", "packages/monetization-go/ai-features.generated.json", "generated registry output path")
	check := flag.Bool("check", false, "fail if generated output is stale")
	flag.Parse()
	artifact := *out
	if !filepath.IsAbs(artifact) {
		artifact = filepath.Join(*root, artifact)
	}
	if *check {
		if err := monetization.CheckAIFeatureRegistry(*root, artifact); err != nil {
			fail(err)
		}
		return
	}
	registry, err := monetization.BuildAIFeatureRegistry(*root)
	if err != nil {
		fail(err)
	}
	data, err := monetization.RenderAIFeatureRegistry(registry)
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(artifact, data, 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("generated %d AI feature(s) into %s\n", len(registry.Features), artifact)
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
