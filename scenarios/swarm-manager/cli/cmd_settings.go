package main

import (
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdSettingsGet(args []string) error {
	fs := flag.NewFlagSet("settings get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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
		cliCommand("settings", "update", "--data", "<json-or-@file>"),
		cliCommand("status"),
	})
	return nil
}

func (a *App) cmdSettingsUpdate(args []string) error {
	fs := flag.NewFlagSet("settings update", flag.ContinueOnError)
	data := fs.String("data", "", "JSON payload (inline or @file)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("data", *data); err != nil {
		return fmt.Errorf("usage: settings update --data JSON [--json]\n\n%s", err)
	}

	payload, err := parseJSONString(*data)
	if err != nil {
		return fmt.Errorf("usage: settings update --data JSON [--json]\n\n%s", err)
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
