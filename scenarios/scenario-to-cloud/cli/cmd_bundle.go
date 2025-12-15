package main

import (
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdBundleBuild(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: scenario-to-cloud bundle-build <manifest.json>")
	}
	manifest, err := readJSONFile(args[0])
	if err != nil {
		return err
	}
	payload, err := a.api.Request("POST", "/api/v1/bundle/build", nil, manifest)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
}

