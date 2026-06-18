package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/vrooli/packages/proto/internal/genprune"
)

func main() {
	protoRoot := flag.String("proto-root", ".", "packages/proto root")
	flag.Parse()

	if err := genprune.PruneBeforeGenerate(*protoRoot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
