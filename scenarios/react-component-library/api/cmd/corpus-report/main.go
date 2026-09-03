package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"react-component-library/internal/catalogcoverage"
)

func main() {
	root := flag.String("root", "../../..", "repository root")
	flag.Parse()
	report, err := catalogcoverage.BuildCorpusReport(filepath.Clean(*root))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	out, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(out))
	for _, invariant := range report.Invariants {
		if invariant.Status == "failed_measurement" {
			os.Exit(2)
		}
	}
}
