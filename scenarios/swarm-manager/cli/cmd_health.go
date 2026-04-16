package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.Get("/health", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	var parsed healthResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil && parsed.Status != "" {
		fmt.Printf("Status: %s (ready=%v)\n", parsed.Status, parsed.Readiness)
		if parsed.Service != "" {
			fmt.Printf("Service: %s\n", parsed.Service)
		}
		if parsed.Version != "" {
			fmt.Printf("Version: %s\n", parsed.Version)
		}
		fmt.Println("\nTriage:")
		if len(parsed.Deps) == 0 {
			fmt.Println("  No dependency issues reported.")
		}
		for key, value := range parsed.Deps {
			fmt.Printf("  %s: %s\n", key, formatDependencyValue(value))
		}
		fmt.Println("\nNext Steps:")
		if parsed.Readiness {
			fmt.Printf("  %s\n", cliCommand("scenarios", "list"))
			fmt.Printf("  %s\n", cliCommand("backlog", "list"))
		} else {
			fmt.Printf("  %s\n", cliCommand("--auto-start", "status"))
			fmt.Println("  vrooli scenario start swarm-manager")
		}
		return nil
	}

	return fmt.Errorf("failed to parse health response")
}

func formatDependencyValue(v any) string {
	switch typed := v.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return "unknown"
		}
		return typed
	case map[string]any:
		if connected, ok := typed["connected"].(bool); ok {
			if connected {
				return "connected"
			}
			return "disconnected"
		}
	case bool:
		if typed {
			return "connected"
		}
		return "disconnected"
	}

	encoded, err := json.Marshal(v)
	if err != nil {
		return "unknown"
	}
	return string(encoded)
}
