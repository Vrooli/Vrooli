// Package signing provides CLI commands for code signing configuration.
package signing

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"scenario-to-desktop/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Commands provides signing CLI commands.
type Commands struct {
	deps support.Dependencies
}

// New creates a new signing Commands instance.
func New(deps support.Dependencies) *Commands {
	return &Commands{deps: deps}
}

func (c *Commands) apiGet(path string, query map[string]string) ([]byte, error) {
	return c.deps.Get(path, query)
}

func (c *Commands) apiPost(path string, body interface{}) ([]byte, error) {
	return c.deps.Request("POST", path, nil, body)
}

func (c *Commands) apiPut(path string, body interface{}) ([]byte, error) {
	return c.deps.Request("PUT", path, nil, body)
}

func (c *Commands) apiDelete(path string) ([]byte, error) {
	return c.deps.Request("DELETE", path, nil, nil)
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	cmds := New(deps)
	return cliapp.SubcommandGroup{
		Name:        "signing",
		Description: "Code signing configuration (run 'signing help' for details)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "get", Description: "Get signing config: get <scenario>", Run: cmds.Get},
			{Name: "set", Description: "Set signing config: set <scenario> --config <json>", Run: cmds.Set},
			{Name: "delete", Description: "Delete signing config: delete <scenario>", Run: cmds.Delete},
			{Name: "validate", Description: "Validate signing config: validate <scenario>", Run: cmds.Validate},
			{Name: "ready", Description: "Check signing readiness: ready <scenario>", Run: cmds.Ready},
			{Name: "prerequisites", Description: "List available signing tools", Run: cmds.Prerequisites},
			{Name: "discover", Description: "Discover certificates: discover <platform>", Run: cmds.Discover},
			{Name: "generate-key", Description: "Generate Linux GPG key: generate-key <scenario> --name <name> --email <email>", Run: cmds.GenerateKey},
		},
	}
}

// Get gets signing config.
func (c *Commands) Get(args []string) error {
	fs := flag.NewFlagSet("signing-get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: signing-get <scenario>")
	}

	scenario := fs.Args()[0]
	body, err := c.apiGet("/signing/"+scenario, nil)
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

// Set sets signing config.
func (c *Commands) Set(args []string) error {
	fs := flag.NewFlagSet("signing-set", flag.ContinueOnError)
	configJSON := fs.String("config", "", "Signing configuration JSON or @file.json")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 || *configJSON == "" {
		return fmt.Errorf("usage: signing-set <scenario> --config <json|@file.json>")
	}

	scenario := fs.Args()[0]

	// Read config from file or string
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

	body, err := c.apiPut("/signing/"+scenario, config)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Signing configuration updated")
	return nil
}

// Delete deletes signing config.
func (c *Commands) Delete(args []string) error {
	fs := flag.NewFlagSet("signing-delete", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: signing-delete <scenario>")
	}

	scenario := fs.Args()[0]
	body, err := c.apiDelete("/signing/" + scenario)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Signing configuration deleted for %s\n", scenario)
	return nil
}

// Validate validates signing config.
func (c *Commands) Validate(args []string) error {
	fs := flag.NewFlagSet("signing-validate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: signing-validate <scenario>")
	}

	scenario := fs.Args()[0]
	body, err := c.apiPost("/signing/"+scenario+"/validate", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		Valid  bool `json:"valid"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.Valid {
		return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
			Status:    []string{"Signing configuration is valid."},
			NextSteps: []string{fmt.Sprintf("scenario-to-desktop signing ready %s", scenario)},
		})
	} else {
		report := cliapp.OperationalReport{
			Status:    []string{"Signing configuration has issues."},
			NextSteps: []string{fmt.Sprintf("scenario-to-desktop signing set %s --config <json|@file>", scenario)},
		}
		for _, e := range resp.Errors {
			report.Triage = append(report.Triage, cliapp.TriageGroup{
				Heading: "Validation",
				Items:   []string{e.Message},
			})
		}
		return cliapp.RenderOperationalReport(os.Stdout, report)
	}
}

// Ready checks signing readiness.
func (c *Commands) Ready(args []string) error {
	fs := flag.NewFlagSet("signing-ready", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: signing-ready <scenario>")
	}

	scenario := fs.Args()[0]
	body, err := c.apiGet("/signing/"+scenario+"/ready", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		Ready     bool `json:"ready"`
		Platforms map[string]struct {
			Ready  bool   `json:"ready"`
			Reason string `json:"reason"`
		} `json:"platforms"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	report := cliapp.OperationalReport{
		NextSteps: []string{
			fmt.Sprintf("scenario-to-desktop signing validate %s", scenario),
			"scenario-to-desktop signing prerequisites",
		},
	}
	if resp.Ready {
		report.Status = append(report.Status, "Signing is ready.")
	} else {
		report.Status = append(report.Status, "Signing is not ready.")
	}
	for platform, status := range resp.Platforms {
		icon := "x"
		if status.Ready {
			icon = "+"
		}
		reason := status.Reason
		if reason == "" {
			reason = "Ready"
		}
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: platform,
			Items:   []string{fmt.Sprintf("%s %s", icon, reason)},
		})
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

// Prerequisites lists available signing tools.
func (c *Commands) Prerequisites(args []string) error {
	fs := flag.NewFlagSet("signing-prerequisites", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := c.apiGet("/signing/prerequisites", nil)
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

// Discover discovers certificates.
func (c *Commands) Discover(args []string) error {
	fs := flag.NewFlagSet("signing-discover", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: signing-discover <platform>\nPlatforms: windows, macos, linux")
	}

	platform := fs.Args()[0]
	body, err := c.apiGet("/signing/discover/"+platform, nil)
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

// GenerateKey generates Linux GPG key.
func (c *Commands) GenerateKey(args []string) error {
	fs := flag.NewFlagSet("signing-generate-key", flag.ContinueOnError)
	name := fs.String("name", "", "Key owner name")
	email := fs.String("email", "", "Key owner email")
	passphrase := fs.String("passphrase", "", "Key passphrase")
	passphraseEnv := fs.String("passphrase-env", "", "Environment variable for passphrase")
	force := fs.Bool("force", false, "Overwrite existing key")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 || *name == "" || *email == "" {
		return fmt.Errorf("usage: signing-generate-key <scenario> --name <name> --email <email> [--passphrase <pass>] [--passphrase-env <var>]")
	}

	scenario := fs.Args()[0]
	req := map[string]interface{}{
		"name":  *name,
		"email": *email,
	}
	if *passphrase != "" {
		req["passphrase"] = *passphrase
	}
	if *passphraseEnv != "" {
		req["passphrase_env"] = *passphraseEnv
	}
	if *force {
		req["force"] = true
	}

	body, err := c.apiPost("/signing/"+scenario+"/linux/generate-key", req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		Status      string `json:"status"`
		KeyID       string `json:"key_id"`
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("GPG key generated: %s\n", resp.Fingerprint)
	return nil
}
