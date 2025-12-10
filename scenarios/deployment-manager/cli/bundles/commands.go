package bundles

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

// BundleManifest represents the core fields we need from the API response.
type BundleManifest struct {
	SchemaVersion string `json:"schema_version"`
	Target        string `json:"target"`
	App           struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description,omitempty"`
	} `json:"app"`
}

// AssembleResponse is the response from the assemble endpoint.
type AssembleResponse struct {
	Status   string          `json:"status"`
	Schema   string          `json:"schema"`
	Manifest json.RawMessage `json:"manifest"`
}

// ExportResponse is the response from the export endpoint.
type ExportResponse struct {
	Status      string          `json:"status"`
	Schema      string          `json:"schema"`
	Scenario    string          `json:"scenario"`
	Tier        string          `json:"tier"`
	Manifest    json.RawMessage `json:"manifest"`
	Checksum    string          `json:"checksum"`
	GeneratedAt string          `json:"generated_at"`
}

// ValidateResponse is the response from the validate endpoint.
type ValidateResponse struct {
	Status string `json:"status"`
	Schema string `json:"schema"`
}

// Commands provides bundle-related CLI commands.
type Commands struct {
	api *cliutil.APIClient
}

// New creates a new bundle commands instance.
func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

// Run dispatches to the appropriate bundle subcommand.
func (c *Commands) Run(args []string) error {
	if len(args) == 0 {
		return c.printHelp()
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "assemble":
		return c.Assemble(rest)
	case "export":
		return c.Export(rest)
	case "validate":
		return c.Validate(rest)
	case "help", "-h", "--help":
		return c.printHelp()
	default:
		return fmt.Errorf("unknown bundle subcommand: %s\n\nRun 'deployment-manager bundle help' for usage", sub)
	}
}

func (c *Commands) printHelp() error {
	help := `Bundle Commands - Generate and validate desktop bundle manifests

Usage:
  deployment-manager bundle <command> [arguments]

Commands:
  assemble    Assemble a bundle manifest from a scenario
  export      Export a production-ready bundle manifest with checksum
  validate    Validate a bundle manifest file

Examples:
  # Assemble a bundle manifest for a scenario
  deployment-manager bundle assemble picker-wheel --tier desktop

  # Export a signed bundle manifest to a file
  deployment-manager bundle export picker-wheel --output bundle.json

  # Validate an existing bundle manifest
  deployment-manager bundle validate ./bundle.json

Run 'deployment-manager bundle <command> --help' for command-specific help.
`
	fmt.Print(help)
	return nil
}

// Assemble generates a bundle manifest for a scenario.
func (c *Commands) Assemble(args []string) error {
	fs := flag.NewFlagSet("bundle assemble", flag.ContinueOnError)
	tier := fs.String("tier", "desktop", "target tier (desktop)")
	includeSecrets := fs.Bool("include-secrets", true, "include secret configuration in manifest")
	format := fs.String("format", "", "output format (json)")
	output := fs.String("output", "", "write manifest to file instead of stdout")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Assemble a bundle manifest from a scenario.

Usage:
  deployment-manager bundle assemble <scenario> [flags]

Arguments:
  scenario    Name of the scenario to bundle

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  deployment-manager bundle assemble picker-wheel
  deployment-manager bundle assemble picker-wheel --tier desktop --output manifest.json
  deployment-manager bundle assemble my-app --include-secrets=false
`)
	}

	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		fs.Usage()
		return errors.New("scenario name is required")
	}
	scenario := remaining[0]

	tierValue := normalizeTier(*tier)

	payload := map[string]interface{}{
		"scenario":        scenario,
		"tier":            tierValue,
		"include_secrets": *includeSecrets,
	}

	body, err := c.api.Request("POST", "/api/v1/bundles/assemble", nil, payload)
	if err != nil {
		return fmt.Errorf("failed to assemble bundle: %w", err)
	}

	// If output file specified, extract just the manifest and write it
	if *output != "" {
		var resp AssembleResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Pretty-print the manifest
		var manifest interface{}
		if err := json.Unmarshal(resp.Manifest, &manifest); err != nil {
			return fmt.Errorf("failed to parse manifest: %w", err)
		}
		prettyManifest, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format manifest: %w", err)
		}

		if err := os.WriteFile(*output, prettyManifest, 0o644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Bundle manifest assembled and written to %s\n", *output)
		fmt.Printf("  Schema: %s\n", resp.Schema)
		fmt.Printf("  Status: %s\n", resp.Status)
		return nil
	}

	cmdutil.PrintByFormat(*format, body)
	return nil
}

// Export generates a production-ready bundle manifest with checksum.
func (c *Commands) Export(args []string) error {
	fs := flag.NewFlagSet("bundle export", flag.ContinueOnError)
	tier := fs.String("tier", "desktop", "target tier (desktop)")
	includeSecrets := fs.Bool("include-secrets", true, "include secret configuration in manifest")
	format := fs.String("format", "", "output format (json)")
	output := fs.String("output", "", "write manifest to file (recommended)")
	manifestOnly := fs.Bool("manifest-only", false, "output only the manifest without metadata")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Export a production-ready bundle manifest with checksum.

This command generates a complete, validated bundle manifest suitable for
packaging into a desktop application. The output includes:
  - The full manifest (bundle.json content)
  - SHA256 checksum for integrity verification
  - Generation timestamp
  - Schema version

Usage:
  deployment-manager bundle export <scenario> [flags]

Arguments:
  scenario    Name of the scenario to export

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # Export to file (recommended for bundled apps)
  deployment-manager bundle export picker-wheel --output bundle.json

  # Export manifest only (for piping to other tools)
  deployment-manager bundle export picker-wheel --manifest-only > bundle.json

  # Export with full metadata to stdout
  deployment-manager bundle export picker-wheel
`)
	}

	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		fs.Usage()
		return errors.New("scenario name is required")
	}
	scenario := remaining[0]

	tierValue := normalizeTier(*tier)

	payload := map[string]interface{}{
		"scenario":        scenario,
		"tier":            tierValue,
		"include_secrets": *includeSecrets,
	}

	body, err := c.api.Request("POST", "/api/v1/bundles/export", nil, payload)
	if err != nil {
		return fmt.Errorf("failed to export bundle: %w", err)
	}

	var resp ExportResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Pretty-print the manifest
	var manifest interface{}
	if err := json.Unmarshal(resp.Manifest, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}
	prettyManifest, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format manifest: %w", err)
	}

	// Write to file if specified
	if *output != "" {
		if err := os.WriteFile(*output, prettyManifest, 0o644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Bundle manifest exported to %s\n", *output)
		fmt.Printf("  Scenario:    %s\n", resp.Scenario)
		fmt.Printf("  Tier:        %s\n", resp.Tier)
		fmt.Printf("  Schema:      %s\n", resp.Schema)
		fmt.Printf("  Checksum:    %s\n", resp.Checksum)
		fmt.Printf("  Generated:   %s\n", resp.GeneratedAt)
		return nil
	}

	// Output manifest only (for piping)
	if *manifestOnly {
		fmt.Println(string(prettyManifest))
		return nil
	}

	// Output full response
	cmdutil.PrintByFormat(*format, body)
	return nil
}

// Validate checks a bundle manifest file against the schema.
func (c *Commands) Validate(args []string) error {
	fs := flag.NewFlagSet("bundle validate", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Validate a bundle manifest file against the schema.

This command checks that a bundle.json file:
  - Conforms to the v0.1 schema structure
  - Has all required fields populated
  - Has valid service, secret, and IPC configurations

Usage:
  deployment-manager bundle validate <file> [flags]

Arguments:
  file    Path to the bundle.json manifest file

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  deployment-manager bundle validate ./bundle.json
  deployment-manager bundle validate /path/to/scenarios/picker-wheel/platforms/electron/bundle/bundle.json
`)
	}

	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		fs.Usage()
		return errors.New("manifest file path is required")
	}
	filePath := remaining[0]

	// Read the manifest file
	manifestData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Validate JSON structure and parse into a map
	var manifestMap map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifestMap); err != nil {
		return fmt.Errorf("invalid JSON in manifest file: %w", err)
	}

	// Send to API for validation (the API expects the manifest as the body)
	body, err := c.api.Request("POST", "/api/v1/bundles/validate", nil, manifestMap)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	var resp ValidateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		cmdutil.PrintByFormat(*format, body)
		return nil
	}

	// Success output
	if strings.ToLower(cmdutil.ResolveFormat(*format)) == "json" {
		cmdutil.PrintByFormat(*format, body)
	} else {
		fmt.Printf("Manifest is valid\n")
		fmt.Printf("  File:   %s\n", filePath)
		fmt.Printf("  Schema: %s\n", resp.Schema)
	}
	return nil
}

// normalizeTier converts user-friendly tier names to API format.
func normalizeTier(tier string) string {
	tier = strings.ToLower(strings.TrimSpace(tier))
	switch tier {
	case "desktop", "2", "tier-2", "tier2":
		return "tier-2-desktop"
	case "mobile", "3", "tier-3", "tier3":
		return "tier-3-mobile"
	case "saas", "cloud", "4", "tier-4", "tier4":
		return "tier-4-saas"
	case "enterprise", "5", "tier-5", "tier5":
		return "tier-5-enterprise"
	default:
		// If already in correct format, return as-is
		if strings.HasPrefix(tier, "tier-") {
			return tier
		}
		return "tier-2-desktop"
	}
}
