package main

import (
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdSettingsGet(args []string) error {
	fs := flag.NewFlagSet("settings get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	body, err := a.getV1("/settings", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	printSection("Summary")
	fmt.Println("  Current settings")
	cliutil.PrintJSON(body)
	printCommandListSection("Next Steps", []string{
		cliCommand("settings", "update", "<json-or-@file>"),
		cliCommand("status"),
	})
	return nil
}

func (a *App) cmdSettingsUpdate(args []string) error {
	fs := flag.NewFlagSet("settings update", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	payload, err := parseJSONArg(fs.Args())
	if err != nil {
		return fmt.Errorf("usage: settings update <json-or-@file>\n\n%s", err)
	}

	body, err := a.requestV1("PUT", "/settings", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	printSection("Result")
	fmt.Println("  Updated settings")
	cliutil.PrintJSON(body)
	printCommandListSection("Next Steps", []string{
		cliCommand("settings", "get"),
		cliCommand("status"),
	})
	return nil
}
