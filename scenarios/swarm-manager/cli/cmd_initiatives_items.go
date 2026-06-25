package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdInitiativesAddItems(args []string) error {
	return a.runInitiativesItemsCommand(args, "add-items", "POST", "Items Added", "Total items")
}

func (a *App) cmdInitiativesRemoveItems(args []string) error {
	return a.runInitiativesItemsCommand(args, "remove-items", "DELETE", "Items Removed", "Remaining items")
}

// runInitiativesItemsCommand is shared logic for add-items and remove-items.
// verb is the CLI sub-command name (e.g. "add-items"), method is the HTTP verb,
// sectionTitle is the human-readable header, and countLabel is the item-count label.
func (a *App) runInitiativesItemsCommand(args []string, verb, method, sectionTitle, countLabel string) error {
	fs := flag.NewFlagSet("initiatives "+verb, flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	itemsFlag := fs.String("items", "", "Comma-separated item references (kind/name)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "items", *itemsFlag); err != nil {
		return fmt.Errorf("usage: initiatives %s --name NAME --items kind/name,kind/name [--json]\n\n%s", verb, err)
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

	body, err := a.core.Request(method, "/initiatives/"+name+"/items", nil, payload)
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

	printSection(sectionTitle)
	fmt.Printf("  Initiative: %s\n", response.Initiative.Name)
	fmt.Printf("  %s: %d\n", countLabel, len(response.Initiative.Items))
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
