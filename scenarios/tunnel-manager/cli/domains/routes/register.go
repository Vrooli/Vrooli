package routes

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"tunnel-manager/cli/internal/flags"
	"tunnel-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type response struct {
	ID           int    `json:"id"`
	Subdomain    string `json:"subdomain"`
	ScenarioName string `json:"scenario_name"`
	LocalPort    int    `json:"local_port"`
	HealthPath   string `json:"health_path"`
	PublicURL    string `json:"public_url"`
	Enabled      bool   `json:"enabled"`
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "route",
		Description: "List, inspect, and mutate tunnel routes",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "Display route manifest with live per-route status", Run: func(args []string) error { return runList(deps, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get a single route by ID", Run: func(args []string) error { return runGet(deps, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create a new route", Run: func(args []string) error { return runCreate(deps, args) }},
			{Name: "update", NeedsAPI: true, Description: "Update an existing route by ID", Run: func(args []string) error { return runUpdate(deps, args) }},
			{Name: "delete", NeedsAPI: true, Description: "Delete a route by ID", Run: func(args []string) error { return runDelete(deps, args) }},
		},
	}
}

func runList(deps support.Dependencies, args []string) error {
	body, err := deps.ScenarioApp().Get("/routes", nil)
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var routes []response
	if err := json.Unmarshal(body, &routes); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	report := cliapp.ListReport{
		Summary: []string{fmt.Sprintf("Routes configured: %d", len(routes))},
		Results: formatRoutes(routes),
		RetrievalHints: []string{
			"tunnel-manager route get <id>",
			"tunnel-manager probe run",
		},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(deps support.Dependencies, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: route get <id>")
	}
	id := args[0]

	body, err := deps.ScenarioApp().Get("/routes/"+id, nil)
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var route response
	if err := json.Unmarshal(body, &route); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Route %d", route.ID),
			fmt.Sprintf("Scenario: %s", route.ScenarioName),
		},
		Results: []string{
			"Subdomain: " + route.Subdomain,
			fmt.Sprintf("Port: %d", route.LocalPort),
			"Health path: " + route.HealthPath,
			"Public URL: " + route.PublicURL,
			fmt.Sprintf("Enabled: %t", route.Enabled),
		},
		RetrievalHints: []string{
			fmt.Sprintf("tunnel-manager route update %d --port <port>", route.ID),
			fmt.Sprintf("tunnel-manager route delete %d", route.ID),
		},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(deps support.Dependencies, args []string) error {
	portStr, hasPort := flags.StringValue(args, "port")
	if !hasPort {
		return fmt.Errorf("--port is required")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid --port value: %s", portStr)
	}

	payload := map[string]any{
		"local_port": port,
		"enabled":    true,
	}
	if v, ok := flags.StringValue(args, "subdomain"); ok {
		payload["subdomain"] = v
	}
	if v, ok := flags.StringValue(args, "scenario"); ok {
		payload["scenario_name"] = v
	}
	if v, ok := flags.StringValue(args, "health-path"); ok {
		payload["health_path"] = v
	}
	if v, ok := flags.StringValue(args, "public-url"); ok {
		payload["public_url"] = v
	}
	if v, ok := flags.StringValue(args, "enabled"); ok {
		payload["enabled"] = v == "true"
	}

	body, err := deps.ScenarioApp().Request("POST", "/routes", nil, payload)
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var route response
	if err := json.Unmarshal(body, &route); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Created route %d", route.ID),
		},
		Changes: []string{
			"Subdomain: " + route.Subdomain,
			fmt.Sprintf("Port: %d", route.LocalPort),
			fmt.Sprintf("Enabled: %t", route.Enabled),
		},
		NextCommand: []string{
			fmt.Sprintf("tunnel-manager route get %d", route.ID),
			"tunnel-manager probe run",
		},
	})
}

func runUpdate(deps support.Dependencies, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: route update <id> [--subdomain ...] [--scenario ...] [--port ...] [--health-path ...] [--public-url ...] [--enabled ...]")
	}
	id := args[0]

	payload := map[string]any{}
	if v, ok := flags.StringValue(args, "subdomain"); ok {
		payload["subdomain"] = v
	}
	if v, ok := flags.StringValue(args, "scenario"); ok {
		payload["scenario_name"] = v
	}
	if v, ok := flags.StringValue(args, "port"); ok {
		port, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid --port value: %s", v)
		}
		payload["local_port"] = port
	}
	if v, ok := flags.StringValue(args, "health-path"); ok {
		payload["health_path"] = v
	}
	if v, ok := flags.StringValue(args, "public-url"); ok {
		payload["public_url"] = v
	}
	if v, ok := flags.StringValue(args, "enabled"); ok {
		payload["enabled"] = v == "true"
	}

	body, err := deps.ScenarioApp().Request("PUT", "/routes/"+id, nil, payload)
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var route response
	if err := json.Unmarshal(body, &route); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Updated route %d", route.ID),
		},
		Changes: []string{
			"Subdomain: " + route.Subdomain,
			fmt.Sprintf("Port: %d", route.LocalPort),
			fmt.Sprintf("Enabled: %t", route.Enabled),
		},
		NextCommand: []string{
			fmt.Sprintf("tunnel-manager route get %d", route.ID),
			"tunnel-manager probe run",
		},
	})
}

func runDelete(deps support.Dependencies, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: route delete <id> [--yes]")
	}
	id := args[0]

	if !flags.BoolValue(args, "yes") {
		fmt.Printf("Delete route %s? [y/N] ", id)
		scanner := bufio.NewScanner(deps.Input())
		answer := ""
		if scanner.Scan() {
			answer = strings.TrimSpace(scanner.Text())
		}
		if answer != "y" && answer != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	_, err := deps.ScenarioApp().Request("DELETE", "/routes/"+id, nil, nil)
	if err != nil {
		return err
	}

	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result: []string{
			"Deleted route " + id,
		},
		NextCommand: []string{
			"tunnel-manager route list",
		},
	})
}

func formatRoutes(routes []response) []string {
	if len(routes) == 0 {
		return []string{"No routes configured."}
	}
	lines := make([]string, 0, len(routes))
	for _, route := range routes {
		lines = append(lines, fmt.Sprintf(
			"%d | %s | %s | port %d | enabled=%t | %s",
			route.ID,
			route.Subdomain,
			route.ScenarioName,
			route.LocalPort,
			route.Enabled,
			route.PublicURL,
		))
	}
	return lines
}
