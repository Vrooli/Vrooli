// Package distribution provides CLI commands for distribution management.
package distribution

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"scenario-to-desktop/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

const appName = "scenario-to-desktop"

// Commands provides distribution CLI commands.
type Commands struct {
	api *cliutil.APIClient
}

// New creates a new distribution Commands instance.
func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

func (c *Commands) apiPath(path string) string {
	return cmdutil.APIPath(path)
}

func (c *Commands) apiGet(path string, query map[string]string) ([]byte, error) {
	return c.api.Get(c.apiPath(path), cmdutil.MapToValues(query))
}

func (c *Commands) apiPost(path string, body interface{}) ([]byte, error) {
	return c.api.Request("POST", c.apiPath(path), nil, body)
}

func (c *Commands) apiPut(path string, body interface{}) ([]byte, error) {
	return c.api.Request("PUT", c.apiPath(path), nil, body)
}

func (c *Commands) apiDelete(path string) ([]byte, error) {
	return c.api.Request("DELETE", c.apiPath(path), nil, nil)
}

// TargetsList lists distribution targets.
func (c *Commands) TargetsList(args []string) error {
	fs := flag.NewFlagSet("dist-targets", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	body, err := c.apiGet("/distribution/targets", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		Targets []struct {
			Name    string `json:"name"`
			Type    string `json:"type"`
			Enabled bool   `json:"enabled"`
		} `json:"targets"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Count == 0 {
		fmt.Println("No distribution targets configured")
		return nil
	}

	fmt.Println("Distribution Targets:")
	for _, t := range resp.Targets {
		enabled := "disabled"
		if t.Enabled {
			enabled = "enabled"
		}
		fmt.Printf("  %-20s %-10s %s\n", t.Name, t.Type, enabled)
	}
	return nil
}

// TargetGet gets a distribution target.
func (c *Commands) TargetGet(args []string) error {
	fs := flag.NewFlagSet("dist-target-get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: dist-target-get <name>")
	}

	name := fs.Args()[0]
	body, err := c.apiGet("/distribution/targets/"+name, nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// TargetCreate creates a distribution target.
func (c *Commands) TargetCreate(args []string) error {
	fs := flag.NewFlagSet("dist-target-create", flag.ContinueOnError)
	configJSON := fs.String("config", "", "Target configuration JSON or @file.json")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *configJSON == "" {
		return fmt.Errorf("usage: dist-target-create --config <json|@file.json>")
	}

	var configData []byte
	if strings.HasPrefix(*configJSON, "@") {
		var err error
		configData, err = os.ReadFile(strings.TrimPrefix(*configJSON, "@"))
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		configData = []byte(*configJSON)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	body, err := c.apiPost("/distribution/targets", config)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Distribution target created")
	return nil
}

// TargetUpdate updates a distribution target.
func (c *Commands) TargetUpdate(args []string) error {
	fs := flag.NewFlagSet("dist-target-update", flag.ContinueOnError)
	configJSON := fs.String("config", "", "Target configuration JSON or @file.json")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 || *configJSON == "" {
		return fmt.Errorf("usage: dist-target-update <name> --config <json|@file.json>")
	}

	name := fs.Args()[0]

	var configData []byte
	if strings.HasPrefix(*configJSON, "@") {
		var err error
		configData, err = os.ReadFile(strings.TrimPrefix(*configJSON, "@"))
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		configData = []byte(*configJSON)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	body, err := c.apiPut("/distribution/targets/"+name, config)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Distribution target updated")
	return nil
}

// TargetDelete deletes a distribution target.
func (c *Commands) TargetDelete(args []string) error {
	fs := flag.NewFlagSet("dist-target-delete", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: dist-target-delete <name>")
	}

	name := fs.Args()[0]
	body, err := c.apiDelete("/distribution/targets/" + name)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Distribution target '%s' deleted\n", name)
	return nil
}

// TargetTest tests a distribution target.
func (c *Commands) TargetTest(args []string) error {
	fs := flag.NewFlagSet("dist-target-test", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: dist-target-test <name>")
	}

	name := fs.Args()[0]
	body, err := c.apiPost("/distribution/targets/"+name+"/test", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// Validate validates all distribution targets.
func (c *Commands) Validate(args []string) error {
	fs := flag.NewFlagSet("dist-validate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	body, err := c.apiPost("/distribution/validate", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// CheckCredentials checks distribution credentials.
func (c *Commands) CheckCredentials(args []string) error {
	fs := flag.NewFlagSet("dist-check-credentials", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	body, err := c.apiPost("/distribution/check-credentials", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// Distribute starts distribution.
func (c *Commands) Distribute(args []string) error {
	fs := flag.NewFlagSet("distribute", flag.ContinueOnError)
	artifacts := fs.String("artifacts", "", "Comma-separated artifact paths")
	targets := fs.String("targets", "", "Comma-separated target names (default: all enabled)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 || *artifacts == "" {
		return fmt.Errorf("usage: distribute <scenario> --artifacts <paths>")
	}

	scenario := fs.Args()[0]
	req := map[string]interface{}{
		"scenario_name": scenario,
		"artifacts":     strings.Split(*artifacts, ","),
	}
	if *targets != "" {
		req["targets"] = strings.Split(*targets, ",")
	}

	body, err := c.apiPost("/distribution/distribute", req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		DistributionID string `json:"distribution_id"`
		StatusURL      string `json:"status_url"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Distribution started: %s\n", resp.DistributionID)
	fmt.Printf("Check status: %s dist-status %s\n", appName, resp.DistributionID)
	return nil
}

// Status gets distribution status.
func (c *Commands) Status(args []string) error {
	fs := flag.NewFlagSet("dist-status", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: dist-status <id>")
	}

	distID := fs.Args()[0]
	body, err := c.apiGet("/distribution/status/"+distID, nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// Cancel cancels distribution.
func (c *Commands) Cancel(args []string) error {
	fs := flag.NewFlagSet("dist-cancel", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: dist-cancel <id>")
	}

	distID := fs.Args()[0]
	body, err := c.apiPost("/distribution/cancel/"+distID, nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Distribution cancelled")
	return nil
}

// List lists all distributions.
func (c *Commands) List(args []string) error {
	fs := flag.NewFlagSet("dist-list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	body, err := c.apiGet("/distribution/list", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}
