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
	fmt.Printf("Status: %s\n", parsed.Status)
	fmt.Printf("Ready: %v\n", parsed.Readiness)
	if len(parsed.Operations) > 0 {
		fmt.Println("Operations:")
		cliutil.PrintJSONMap(parsed.Operations, 2)
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
