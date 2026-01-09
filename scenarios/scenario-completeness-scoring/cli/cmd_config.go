package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdConfig(args []string) error {
	if len(args) > 0 && args[0] == "set" {
		return a.cmdConfigSet(args[1:])
	}
	body, err := a.services.Config.Get()
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdConfigSet(args []string) error {
	fs := flag.NewFlagSet("config set", flag.ContinueOnError)
	filePath := fs.String("file", "", "Path to JSON config file")
	inline := fs.String("json", "", "Inline JSON payload")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var payload map[string]interface{}
	switch {
	case *filePath != "":
		data, err := os.ReadFile(*filePath)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("parse json: %w", err)
		}
	case *inline != "":
		if err := json.Unmarshal([]byte(*inline), &payload); err != nil {
			return fmt.Errorf("parse json: %w", err)
		}
	default:
		return fmt.Errorf("config set requires --file or --json")
	}
	body, err := a.services.Config.Set(payload)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}
