package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("vrooli-dist", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	goos := flags.String("goos", "", "target operating system")
	goarch := flags.String("goarch", "", "target architecture")
	output := flags.String("output", "", "output binary path")
	outDir := flags.String("out-dir", "dist", "output directory used with --all")
	root := flags.String("root", "", "repository root")
	version := flags.String("version", "", "release version (normally the git tag)")
	all := flags.Bool("all", false, "build every supported target")
	matrixJSON := flags.Bool("matrix-json", false, "print the supported target matrix as JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *matrixJSON {
		payload, err := json.Marshal(buildinfo.DistributionTargets())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(string(payload))
		return 0
	}
	if *all {
		for _, target := range buildinfo.DistributionTargets() {
			path := filepath.Join(*outDir, buildinfo.DistributionAssetName(target))
			if code := buildOne(*root, path, *version, target); code != 0 {
				return code
			}
		}
		return 0
	}
	if strings.TrimSpace(*goos) == "" || strings.TrimSpace(*goarch) == "" || strings.TrimSpace(*output) == "" {
		fmt.Fprintln(os.Stderr, "--goos, --goarch, and --output are required (or use --all/--matrix-json)")
		return 2
	}
	return buildOne(*root, *output, *version, buildinfo.DistributionTarget{OS: *goos, Arch: *goarch})
}

func buildOne(root, output, version string, target buildinfo.DistributionTarget) int {
	artifact, err := buildinfo.BuildDistribution(context.Background(), buildinfo.DistributionBuildOptions{
		Root: root, Output: output, Version: version, Target: target,
		Stdout: os.Stdout, Stderr: os.Stderr,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("%s/%s: %s (fingerprint %s)\n", target.OS, target.Arch, artifact.BinaryPath, artifact.Fingerprint)
	return 0
}
