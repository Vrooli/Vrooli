package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	envresolve "github.com/vrooli/envresolve-go"
)

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	findings, err := envresolve.ConformanceScan(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(findings); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, finding := range findings {
		if finding.Severity == "ERROR" {
			os.Exit(2)
		}
	}
}
