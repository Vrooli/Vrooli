package main

import (
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdRecommend(args []string) error {
	fs := flag.NewFlagSet("recommend", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: recommend <scenario> [--json]")
	}
	scenarioName := fs.Arg(0)
	path := fmt.Sprintf("/api/v1/recommendations/%s", scenarioName)
	body, err := a.api.Get(path, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	cliutil.PrintJSON(body)
	return nil
}
