package main

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdSettingsGet(_ []string) error {
	body, err := a.getV1("/settings", nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdSettingsUpdate(args []string) error {
	payload, err := parseJSONArg(args)
	if err != nil {
		return fmt.Errorf("usage: settings update <json-or-@file>\n\n%s", err)
	}

	var patch map[string]any
	if err := json.Unmarshal(payload, &patch); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	body, err := a.requestV1("PUT", "/settings", nil, payload)
	if err != nil {
		return err
	}

	cliutil.PrintJSON(body)
	return nil
}
