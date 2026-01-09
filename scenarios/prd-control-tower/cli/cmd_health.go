package main

import (
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	body, parsed, err := a.services.Health.Status()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Status: %s\n", parsed.Status)
	fmt.Printf("Ready: %v\n", parsed.Readiness)
	if len(parsed.Operations) > 0 {
		fmt.Println("Operations:")
		cliutil.PrintJSONMap(parsed.Operations, 2)
	}
	return nil
}
