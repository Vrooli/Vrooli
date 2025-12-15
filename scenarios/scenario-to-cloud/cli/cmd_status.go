package main

import (
	"net/url"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdStatus(args []string) error {
	payload, err := a.api.Get("/health", url.Values{})
	if err != nil {
		return err
	}
	cliutil.PrintJSON(payload)
	return nil
}
