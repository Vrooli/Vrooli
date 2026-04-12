package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
)

type fingerprintFunc func(root string, options buildinfo.FingerprintOptions, relPaths ...string) (buildinfo.FingerprintReport, error)

type command struct {
	computeFingerprint fingerprintFunc
}

func main() {
	cmd := command{
		computeFingerprint: buildinfo.ComputeSourceFingerprintReport,
	}
	os.Exit(cmd.run(os.Args[1:], os.Stdout, os.Stderr))
}

func (c command) run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("vrooli-buildmeta", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		_, _ = fmt.Fprintf(stderr, "Usage: vrooli-buildmeta --root <repo-root> <relative-path> [<relative-path>...]\n")
		_, _ = fmt.Fprintln(stderr, "Computes deterministic Go-source build metadata for Vrooli binaries.")
		_, _ = fmt.Fprintln(stderr, "All requested paths must exist and must include at least one Go source file.")
		flags.PrintDefaults()
	}
	root := flags.String("root", ".", "repository root used for fingerprinting")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if len(flags.Args()) == 0 {
		_, _ = fmt.Fprintln(stderr, "vrooli-buildmeta: at least one relative path is required")
		flags.Usage()
		return 2
	}

	report, err := c.computeFingerprint(*root, buildinfo.FingerprintOptions{
		RequireExistingTargets: true,
		RequireGoFiles:         true,
	}, flags.Args()...)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "vrooli-buildmeta: %v (root=%q targets=%s)\n", err, *root, strings.Join(flags.Args(), ","))
		return 1
	}

	_, _ = fmt.Fprintln(stdout, report.Fingerprint)
	return 0
}
