package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdStatus() error {
	body, err := a.api.Get("/health", nil)
	if err != nil {
		return err
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	status := getString(parsed, "status")
	fmt.Printf("Status: %s\n", status)
	if readiness, ok := parsed["readiness"].(bool); ok {
		fmt.Printf("Ready: %v\n", readiness)
	}
	if ops, ok := parsed["operations"].(map[string]interface{}); ok && len(ops) > 0 {
		fmt.Println("Operations:")
		cliutil.PrintJSONMap(ops, 2)
	} else {
		cliutil.PrintJSON(body)
	}
	return nil
}

func (a *App) cmdCollectors() error {
	body, err := a.api.Get("/api/v1/health/collectors", nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdCircuitBreaker(args []string) error {
	if len(args) > 0 && args[0] == "reset" {
		body, err := a.api.Request(http.MethodPost, "/api/v1/health/circuit-breaker/reset", nil, map[string]interface{}{})
		if err != nil {
			return err
		}
		cliutil.PrintJSON(body)
		return nil
	}
	body, err := a.api.Get("/api/v1/health/circuit-breaker", nil)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}
