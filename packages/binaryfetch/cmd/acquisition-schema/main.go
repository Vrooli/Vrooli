// Command acquisition-schema owns regeneration/checking of the acquisition schema fragment.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/binaryfetch"
)

func main() {
	root := flag.String("root", "", "Vrooli repository root (required)")
	check := flag.Bool("check", false, "Check committed schema without writing")
	flag.Parse()
	if *root == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: acquisition-schema --root REPO [--check]")
		os.Exit(2)
	}
	var err error
	if *check {
		err = binaryfetch.ValidateAcquisitionSchema(*root)
	} else {
		err = binaryfetch.SyncAcquisitionSchema(*root)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
