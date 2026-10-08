package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/vrooli/internal/deployability"
)

func main() {
	root := flag.String("root", ".", "repository root")
	check := flag.Bool("check", false, "check generated schemas without modifying them")
	flag.Parse()

	var err error
	if *check {
		err = deployability.CheckCapabilitySchemaEnums(*root)
	} else {
		err = deployability.GenerateCapabilitySchemaEnums(*root)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
