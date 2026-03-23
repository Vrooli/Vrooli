package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdInitiativesList(args []string) error {
	fs := flag.NewFlagSet("initiatives list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.getV1("/initiatives", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ListInitiativesResponse](body)
	if err != nil {
		return err
	}

	if len(response.Items) == 0 {
		printSection("Summary")
		fmt.Println("  No initiatives found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("initiatives", "create", "--data", "'{\"name\":\"my-init\",\"title\":\"My Initiative\"}'"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d initiative(s)\n", len(response.Items))

	printSection("Initiatives")
	for _, item := range response.Items {
		init := item.Initiative
		rollup := item.Rollup
		fmt.Printf("  %s (%s)\n", init.Name, init.Status)
		fmt.Printf("    Title: %s\n", init.Title)
		if init.Description != "" {
			fmt.Printf("    Description: %s\n", init.Description)
		}
		fmt.Printf("    Items: %d total, %d completed, %d in-progress, %d failed, %d pending\n",
			rollup.Total, rollup.Completed, rollup.InProgress, rollup.Failed, rollup.Pending)
		if len(init.Items) > 0 {
			fmt.Printf("    References: %s\n", strings.Join(init.Items, ", "))
		}
		fmt.Println()
	}

	first := response.Items[0].Initiative
	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "get", "--name", first.Name),
	})
	return nil
}

func (a *App) cmdInitiativesGet(args []string) error {
	fs := flag.NewFlagSet("initiatives get", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives get --name NAME [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	body, err := a.getV1("/initiatives/"+name, nil)
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
	init := response.Initiative
	rollup := response.Rollup

	printSection("Initiative")
	fmt.Printf("  Name: %s\n", init.Name)
	fmt.Printf("  Title: %s\n", init.Title)
	if init.Description != "" {
		fmt.Printf("  Description: %s\n", init.Description)
	}
	fmt.Printf("  Status: %s\n", init.Status)
	fmt.Printf("  Created: %s\n", init.Created)
	fmt.Printf("  Updated: %s\n", init.Updated)

	printSection("Rollup")
	fmt.Printf("  Total: %d\n", rollup.Total)
	fmt.Printf("  Completed: %d\n", rollup.Completed)
	fmt.Printf("  In Progress: %d\n", rollup.InProgress)
	fmt.Printf("  Failed: %d\n", rollup.Failed)
	fmt.Printf("  Pending: %d\n", rollup.Pending)

	if len(init.Items) > 0 {
		printSection("Items")
		for _, item := range init.Items {
			fmt.Printf("  - %s\n", item)
		}
	}

	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "update", "--name", init.Name, "--data", "'{\"title\":\"...\",\"status\":\"completed\"}'"),
		cliCommand("initiatives", "delete", "--name", init.Name),
	})
	return nil
}

func (a *App) cmdInitiativesCreate(args []string) error {
	fs := flag.NewFlagSet("initiatives create", flag.ContinueOnError)
	data := fs.String("data", "", "JSON payload (inline or @file)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("data", *data); err != nil {
		return fmt.Errorf("usage: initiatives create --data JSON [--json]\n\nExample:\n  initiatives create --data '{\"name\":\"my-init\",\"title\":\"My Initiative\"}'\n\n%s", err)
	}

	payload, err := parseJSONString(*data)
	if err != nil {
		return err
	}

	var req struct {
		Name  string `json:"name"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if req.Name == "" || req.Title == "" {
		return fmt.Errorf("name and title are required fields")
	}

	body, err := a.requestV1("POST", "/initiatives", nil, payload)
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

	printSection("Created")
	fmt.Printf("  Name: %s\n", response.Initiative.Name)
	fmt.Printf("  Title: %s\n", response.Initiative.Title)
	fmt.Printf("  Status: %s\n", response.Initiative.Status)

	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "get", "--name", response.Initiative.Name),
	})
	return nil
}

func (a *App) cmdInitiativesUpdate(args []string) error {
	fs := flag.NewFlagSet("initiatives update", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	data := fs.String("data", "", "JSON payload (inline or @file)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "data", *data); err != nil {
		return fmt.Errorf("usage: initiatives update --name NAME --data JSON [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	payload, err := parseJSONString(*data)
	if err != nil {
		return err
	}

	body, err := a.requestV1("PUT", "/initiatives/"+name, nil, payload)
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

	printSection("Updated")
	fmt.Printf("  Name: %s\n", response.Initiative.Name)
	fmt.Printf("  Title: %s\n", response.Initiative.Title)
	fmt.Printf("  Status: %s\n", response.Initiative.Status)
	return nil
}

func (a *App) cmdInitiativesDelete(args []string) error {
	fs := flag.NewFlagSet("initiatives delete", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives delete --name NAME\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	_, err := a.requestV1("DELETE", "/initiatives/"+name, nil, nil)
	if err != nil {
		return err
	}

	printSection("Deleted")
	fmt.Printf("  Initiative %q deleted.\n", name)
	return nil
}
