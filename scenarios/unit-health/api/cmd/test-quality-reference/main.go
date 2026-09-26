// test-quality-reference generates the catalog's public enforcement reference.
package main

import (
	"flag"
	"fmt"
	"os"

	"unit-health/internal/testquality"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fail(err)
	}
}

var exitProcess = os.Exit

func run(args []string) error {
	flags := flag.NewFlagSet("test-quality-reference", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	output := flags.String("output", "../docs/reference/test-quality-rules.md", "generated reference path")
	check := flags.Bool("check", false, "check committed output without writing")
	if err := flags.Parse(args); err != nil {
		return err
	}
	catalog, err := testquality.LoadCatalog()
	if err != nil {
		return err
	}
	want := catalog.EnforcementReference()
	if *check {
		data, err := os.ReadFile(*output)
		if err != nil {
			return err
		}
		if string(data) != want {
			return fmt.Errorf("enforcement reference is stale: %s", *output)
		}
		return nil
	}
	if err := os.WriteFile(*output, []byte(want), 0o644); err != nil {
		return err
	}
	return nil
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); exitProcess(1) }
