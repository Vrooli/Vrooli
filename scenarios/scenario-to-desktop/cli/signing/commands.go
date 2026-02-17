// Package signing provides CLI commands for code signing configuration.
package signing

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"scenario-to-desktop/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

// Commands provides signing CLI commands.
type Commands struct {
	api *cliutil.APIClient
}

// New creates a new signing Commands instance.
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
		fmt.Println("Signing configuration is valid")
	} else {
		fmt.Println("Signing configuration has issues:")
		for _, e := range resp.Errors {
			fmt.Printf("  - %s\n", e.Message)
		}
	}
	return nil
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

	if resp.Ready {
		fmt.Println("Signing is ready")
	} else {
		fmt.Println("Signing is not ready")
	}
	fmt.Println("Platforms:")
	for platform, status := range resp.Platforms {
		icon := "x"
		if status.Ready {
			icon = "+"
		}
		reason := status.Reason
		if reason == "" {
			reason = "Ready"
		}
		fmt.Printf("  %s %-8s %s\n", icon, platform+":", reason)
	}
	return nil
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
