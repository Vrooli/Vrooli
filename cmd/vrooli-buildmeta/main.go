package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/vrooli/vrooli/internal/buildinfo"
)

var computeFingerprintForPathsFn = buildinfo.ComputeSourceFingerprintForPaths

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("vrooli-buildmeta", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "repository root used for fingerprinting")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	fingerprint, err := computeFingerprintForPathsFn(*root, flags.Args()...)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "vrooli-buildmeta: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintln(stdout, fingerprint)
	return 0
}
