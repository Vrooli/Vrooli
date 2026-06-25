package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdInitiativesAddItems(args []string) error {
	fs := flag.NewFlagSet("initiatives add-items", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	itemsFlag := fs.String("items", "", "Comma-separated item references (kind/name)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "items", *itemsFlag); err != nil {
		return fmt.Errorf("usage: initiatives add-items --name NAME --items kind/name,kind/name [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)
	items := parseCommaSeparated(*itemsFlag)
	if len(items) == 0 {
		return fmt.Errorf("at least one item reference is required")
	}

	payload, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		return err
	}

	body, err := a.core.Request("POST", "/initiatives/"+name+"/items", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[InitiativeResponse](body)
	if err != nil {
		return err
	}

	printSection("Items Added")
	fmt.Printf("  Initiative: %s\n", response.Initiative.Name)
	fmt.Printf("  Total items: %d\n", len(response.Initiative.Items))
	for _, item := range response.Initiative.Items {
		fmt.Printf("  - %s\n", item)
	}
	return nil
}

func (a *App) cmdInitiativesRemoveItems(args []string) error {
	fs := flag.NewFlagSet("initiatives remove-items", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	itemsFlag := fs.String("items", "", "Comma-separated item references (kind/name)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "items", *itemsFlag); err != nil {
		return fmt.Errorf("usage: initiatives remove-items --name NAME --items kind/name,kind/name [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)
	items := parseCommaSeparated(*itemsFlag)
	if len(items) == 0 {
		return fmt.Errorf("at least one item reference is required")
	}

	payload, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		return err
	}

	body, err := a.core.Request("DELETE", "/initiatives/"+name+"/items", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[InitiativeResponse](body)
	if err != nil {
		return err
	}

	printSection("Items Removed")
	fmt.Printf("  Initiative: %s\n", response.Initiative.Name)
	fmt.Printf("  Remaining items: %d\n", len(response.Initiative.Items))
	for _, item := range response.Initiative.Items {
		fmt.Printf("  - %s\n", item)
	}
	return nil
}

// parseCommaSeparated splits a comma-separated string and trims whitespace.
func parseCommaSeparated(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
