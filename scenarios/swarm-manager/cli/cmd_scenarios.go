package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdScenariosList(args []string) error {
	fs := flag.NewFlagSet("scenarios list", flag.ContinueOnError)
	search := fs.String("search", "", "Filter by name or description")
	status := fs.String("status", "", "Filter by status (running|stopped|error|unknown)")
	tags := fs.String("tags", "", "Filter by tags (comma-separated)")
	sortField := fs.String("sort", "", "Sort by field (priority|name|displayName)")
	order := fs.String("order", "", "Sort order (asc|desc)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*search) != "" {
		query.Set("search", strings.TrimSpace(*search))
	}
	if strings.TrimSpace(*status) != "" {
		query.Set("status", strings.TrimSpace(*status))
	}
	if strings.TrimSpace(*tags) != "" {
		query.Set("tags", strings.TrimSpace(*tags))
	}
	if strings.TrimSpace(*sortField) != "" {
		query.Set("sort", strings.TrimSpace(*sortField))
	}
	if strings.TrimSpace(*order) != "" {
		query.Set("order", strings.TrimSpace(*order))
	}

	body, err := a.getV1("/scenarios", query)
	if err != nil {
		return err
	}

	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}

	var response ListScenariosResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Scenarios) == 0 {
		fmt.Println("No scenarios found.")
		return nil
	}

	fmt.Printf("Found %d scenario(s):\n\n", len(response.Scenarios))
	for _, scenario := range response.Scenarios {
		display := scenario.DisplayName
		if display == "" {
			display = scenario.Name
		}
		fmt.Printf("  %s (status: %s, priority: %d)\n", scenario.Name, scenario.Status, scenario.Priority)
		fmt.Printf("    Display: %s\n", display)
		if scenario.Description != "" {
			fmt.Printf("    Description: %s\n", scenario.Description)
		}
		if len(scenario.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(scenario.Tags, ", "))
		}
		fmt.Println()
	}
	return nil
}

func (a *App) cmdScenariosGet(args []string) error {
	if err := requireArgs(args, 1, "scenarios get <name>"); err != nil {
		return err
	}
	name := strings.TrimSpace(args[0])

	body, err := a.getV1("/scenarios/"+name, nil)
	if err != nil {
		return err
	}

	var response ScenarioResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	scenario := response.Scenario

	fmt.Printf("Name: %s\n", scenario.Name)
	fmt.Printf("Display Name: %s\n", scenario.DisplayName)
	fmt.Printf("Description: %s\n", scenario.Description)
	fmt.Printf("Status: %s\n", scenario.Status)
	fmt.Printf("Priority: %d\n", scenario.Priority)
	if scenario.CompletenessScore != nil {
		fmt.Printf("Completeness: %d\n", *scenario.CompletenessScore)
	}
	fmt.Printf("Greenfield: %v\n", scenario.IsGreenfield)
	if len(scenario.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(scenario.Tags, ", "))
	}
	return nil
}

func (a *App) cmdScenariosUpdate(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: scenarios update <name> <json-or-@file>\n\nExample:\n  scenarios update my-scenario '{\"is_greenfield\":true}'")
	}
	name := strings.TrimSpace(args[0])
	payload, err := parseJSONArg(args[1:])
	if err != nil {
		return err
	}

	var patch map[string]any
	if err := json.Unmarshal(payload, &patch); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	body, err := a.requestV1("PATCH", "/scenarios/"+name, nil, payload)
	if err != nil {
		return err
	}

	var response ScenarioResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	scenario := response.Scenario

	fmt.Printf("Updated scenario: %s\n", scenario.Name)
	fmt.Printf("  Greenfield: %v\n", scenario.IsGreenfield)
	return nil
}

func (a *App) cmdScenariosDelete(args []string) error {
	fs := flag.NewFlagSet("scenarios delete", flag.ContinueOnError)
	archive := fs.Bool("archive", false, "Archive scenario to backlog (idea) before deletion")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: scenarios delete <name> [--archive]")
	}
	name := strings.TrimSpace(fs.Arg(0))

	query := url.Values{}
	if *archive {
		query.Set("archive", "true")
	}

	body, err := a.requestV1("DELETE", "/scenarios/"+name, query, nil)
	if err != nil {
		return err
	}

	var response DeleteScenarioResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println(response.Message)
	return nil
}

func (a *App) cmdScenariosFiles(args []string) error {
	if err := requireArgs(args, 1, "scenarios files <name>"); err != nil {
		return err
	}
	name := strings.TrimSpace(args[0])

	body, err := a.getV1("/scenarios/"+name+"/files", nil)
	if err != nil {
		return err
	}

	var response ScenarioFilesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if len(response.Files) == 0 {
		fmt.Println("No files found.")
		return nil
	}

	fmt.Printf("Files for scenario %s:\n", name)
	printTree(response.Files,
		func(item ScenarioFile) []ScenarioFile { return item.Children },
		func(item ScenarioFile) string {
			if item.Type == "directory" {
				return item.Name + "/"
			}
			return fmt.Sprintf("%s (%d bytes)", item.Name, item.Size)
		},
		0,
	)
	return nil
}

func (a *App) cmdScenariosStart(args []string) error {
	return a.runScenarioLifecycle(args, "start")
}

func (a *App) cmdScenariosStop(args []string) error {
	return a.runScenarioLifecycle(args, "stop")
}

func (a *App) cmdScenariosRestart(args []string) error {
	return a.runScenarioLifecycle(args, "restart")
}

func (a *App) runScenarioLifecycle(args []string, action string) error {
	if err := requireArgs(args, 1, "scenarios "+action+" <name>"); err != nil {
		return err
	}
	name := strings.TrimSpace(args[0])
	body, err := a.requestV1("POST", "/scenarios/"+name+"/"+action, nil, nil)
	if err != nil {
		return err
	}

	var response ScenarioResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	fmt.Printf("Scenario %s: %s\n", action, response.Scenario.Name)
	fmt.Printf("  Status: %s\n", response.Scenario.Status)
	return nil
}
