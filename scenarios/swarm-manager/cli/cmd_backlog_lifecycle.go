package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdBacklogRecreate(args []string) error {
	fs := flag.NewFlagSet("backlog recreate", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog recreate --kind KIND --name NAME [--json]\n\n%s", err)
	}
	kind, name := strings.TrimSpace(*kindFlag), strings.TrimSpace(*nameFlag)
	body, err := a.core.Request("POST", "/backlog/"+kind+"/"+name+"/recreate", nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	item, err := decodeResponse[BacklogItem](body)
	if err != nil {
		return err
	}
	printSection("Result")
	fmt.Printf("  Recreated %s/%s as %s/%s\n", kind, name, item.Kind, item.Name)
	fmt.Printf("  Lineage: %s\n", item.SpawnedFrom)
	return nil
}

func (a *App) cmdBacklogResetArtifacts(args []string) error {
	fs := flag.NewFlagSet("backlog reset-artifacts", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	scopeFlag := fs.String("scope", "", "Comma-separated scopes: workshop,clarifications,review,handoff_executions,plan_unbind")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag, "scope", *scopeFlag); err != nil {
		return fmt.Errorf("usage: backlog reset-artifacts --kind KIND --name NAME --scope SCOPE,... [--json]\n\n%s", err)
	}
	scopes := splitCSV(*scopeFlag)
	body, err := a.core.Request("POST", "/backlog/"+strings.TrimSpace(*kindFlag)+"/"+strings.TrimSpace(*nameFlag)+"/reset-artifacts", nil, map[string]any{"scope": scopes})
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	printSection("Result")
	fmt.Printf("  Reset artifacts for %s/%s: %s\n", strings.TrimSpace(*kindFlag), strings.TrimSpace(*nameFlag), strings.Join(scopes, ", "))
	return nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			values = append(values, value)
		}
	}
	return values
}
