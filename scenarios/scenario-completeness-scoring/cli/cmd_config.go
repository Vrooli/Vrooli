package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdConfigure(args []string) error {
	if len(args) == 0 {
		payload, _ := json.MarshalIndent(a.config, "", "  ")
		fmt.Println(string(payload))
		return nil
	}
	if len(args) != 2 {
		return fmt.Errorf("usage: configure <key> <value>")
	}
	key := args[0]
	value := args[1]
	switch key {
	case "api_base":
		a.config.APIBase = value
	case "token":
		a.config.Token = value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}
	if err := a.saveConfig(); err != nil {
		return err
	}
	fmt.Printf("Updated %s\n", key)
	return nil
}

func (a *App) cmdConfig(args []string) error {
	if len(args) > 0 && args[0] == "set" {
		return a.cmdConfigSet(args[1:])
	}
	body, err := a.api.Get("/api/v1/config", nil)
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
	body, err := a.api.Request(http.MethodPut, "/api/v1/config", nil, payload)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdPresets() error {
	body, err := a.api.Get("/api/v1/config/presets", nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdPreset(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: preset apply <name>")
	}
	if args[0] != "apply" {
		return fmt.Errorf("unknown preset subcommand: %s", args[0])
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: preset apply <name>")
	}
	name := args[1]
	path := fmt.Sprintf("/api/v1/config/presets/%s/apply", name)
	body, err := a.api.Request(http.MethodPost, path, nil, map[string]interface{}{})
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}
