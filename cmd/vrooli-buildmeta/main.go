package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/vrooli/internal/buildinfo"
)

func main() {
	root := flag.String("root", ".", "repository root used for fingerprinting")
	flag.Parse()

	fingerprint, err := buildinfo.ComputeSourceFingerprintForPaths(*root, flag.Args()...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vrooli-buildmeta: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(fingerprint)
}
