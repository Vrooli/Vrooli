package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/buildinfo"
)

func main() {
	root := flag.String("root", ".", "directory to fingerprint")
	flag.Parse()

	fingerprint, err := buildinfo.ComputeFingerprint(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to compute fingerprint: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(fingerprint)
}
