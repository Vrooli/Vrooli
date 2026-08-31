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
	mode := flag.String("mode", "shape", "census mode: shape or duplication")
	flag.Parse()
	var value any
	var err error
	if *mode == "shape" {
		value, err = catalogcoverage.ShapeCensus(filepath.Join(filepath.Clean(*root), "scenarios", "react-component-library", "library"))
	} else if *mode == "duplication" {
		value, err = catalogcoverage.DuplicationCensus(filepath.Clean(*root))
	} else {
		err = fmt.Errorf("unknown census mode %q", *mode)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	out, _ := json.MarshalIndent(value, "", "  ")
	fmt.Println(string(out))
}
