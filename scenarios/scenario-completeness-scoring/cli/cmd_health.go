package main

import (
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdStatus() error {
	body, parsed, err := a.services.Health.Status()
	if err != nil {
		return err
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
	body, err := a.services.Health.Collectors()
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdCircuitBreaker(args []string) error {
	if len(args) > 0 && args[0] == "reset" {
		body, err := a.services.Health.ResetCircuitBreaker()
		if err != nil {
			return err
		}
		cliutil.PrintJSON(body)
		return nil
	}
	body, err := a.services.Health.CircuitBreakerStatus()
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}
