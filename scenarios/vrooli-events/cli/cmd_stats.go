package main

import (
	"encoding/json"
	"flag"
	"fmt"
)

func (a *App) cmdStats(args []string) error {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	body, err := a.core.APIClient.Get(a.apiPath("/health"), nil)
	if err != nil {
		return err
	}

	if *asJSON {
		fmt.Println(string(body))
		return nil
	}

	var health map[string]any
	if err := json.Unmarshal(body, &health); err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	fmt.Printf("Status:      %s\n", str(health, "status"))
	fmt.Printf("Subscribers: %v\n", health["subscribers"])
	if st, ok := health["store"].(map[string]any); ok {
		fmt.Printf("Events:      %v\n", st["totalEvents"])
		if bytes, ok := st["totalPayloadBytes"].(float64); ok {
			fmt.Printf("Store Size:  %.1f MB\n", bytes/1024/1024)
		}
	}
	return nil
}
