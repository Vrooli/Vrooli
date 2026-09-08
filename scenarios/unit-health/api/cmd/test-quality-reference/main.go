// test-quality-reference generates the catalog's public enforcement reference.
package main

import (
	"flag"
	"fmt"
	"os"
	"unit-health/internal/testquality"
)

func main() {
	output := flag.String("output", "../docs/reference/test-quality-rules.md", "generated reference path")
	check := flag.Bool("check", false, "check committed output without writing")
	flag.Parse()
	catalog, err := testquality.LoadCatalog()
	if err != nil {
		fail(err)
	}
	want := catalog.EnforcementReference()
	if *check {
		data, err := os.ReadFile(*output)
		if err != nil {
			fail(err)
		}
		if string(data) != want {
			fail(fmt.Errorf("enforcement reference is stale: %s", *output))
		}
		return
	}
	if err := os.WriteFile(*output, []byte(want), 0644); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
