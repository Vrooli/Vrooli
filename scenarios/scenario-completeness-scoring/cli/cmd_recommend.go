package main

import (
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdRecommend(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: recommend <scenario>")
	}
	scenarioName := args[0]
	body, err := a.services.Scoring.Recommend(scenarioName)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}
