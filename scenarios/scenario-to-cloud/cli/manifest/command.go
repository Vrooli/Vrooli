package manifest

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	"scenario-to-cloud/cli/internal/flagutil"
	internalmanifest "scenario-to-cloud/cli/internal/manifest"
)

// Run executes manifest subcommands.
func Run(client *Client, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "validate":
		return runValidate(client, args[1:])
	case "schema":
		return runSchema(client, args[1:])
	case "init":
		return runInit(client, args[1:])
	case "template":
		return runTemplate(client, args[1:])
	case "doctor":
		return runDoctor(client, args[1:])
	case "fix":
		return runFix(client, args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'scenario-to-cloud manifest help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: scenario-to-cloud manifest <command> [arguments]

Commands:
  validate <manifest.json>    Validate a cloud manifest JSON file
  schema                      Print the canonical manifest JSON schema
  init                        Generate a starter manifest
  template                    Print built-in manifest template (minimal|full)
  doctor <manifest.json>      Analyze manifest issues and suggested fixes
  fix <manifest.json>         Return/write a normalized auto-fixed manifest

Run 'scenario-to-cloud manifest <command> -h' for command-specific options.`)
	return nil
}

func runValidate(client *Client, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: scenario-to-cloud manifest validate <manifest.json>")
	}
	manifestData, err := internalmanifest.ReadJSONFile(args[0])
	if err != nil {
		return err
	}
	body, _, err := client.Validate(manifestData)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func runSchema(client *Client, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("usage: scenario-to-cloud manifest schema")
	}
	body, _, err := client.Schema()
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func runInit(client *Client, args []string) error {
	fs := flag.NewFlagSet("manifest init", flag.ContinueOnError)
	scenarioID := fs.String("scenario", "", "Scenario ID to prefill dependencies and ports")
	host := fs.String("host", "", "VPS host")
	domain := fs.String("domain", "", "Edge domain")
	user := fs.String("user", "", "SSH user")
	port := fs.Int("port", 0, "SSH port")
	keyPath := fs.String("key-path", "", "SSH private key path")
	workdir := fs.String("workdir", "", "Remote Vrooli workdir")
	caddyEmail := fs.String("caddy-email", "", "ACME contact email")
	outPath := fs.String("out", "", "Write manifest JSON to this path")
	jsonOutput := fs.Bool("json", false, "Output raw JSON response")
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: scenario-to-cloud manifest init [--scenario <id>] [--host <host>] [--domain <domain>] [--out <path>]")
	}

	req := InitRequest{
		ScenarioID: strings.TrimSpace(*scenarioID),
		Host:       strings.TrimSpace(*host),
		Domain:     strings.TrimSpace(*domain),
		User:       strings.TrimSpace(*user),
		Port:       *port,
		KeyPath:    strings.TrimSpace(*keyPath),
		Workdir:    strings.TrimSpace(*workdir),
		CaddyEmail: strings.TrimSpace(*caddyEmail),
	}
	body, resp, err := client.Init(req)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	manifestJSON, err := json.MarshalIndent(resp.Manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if *outPath != "" {
		if err := os.WriteFile(*outPath, append(manifestJSON, '\n'), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", *outPath, err)
		}
		fmt.Printf("Wrote manifest to %s\n", *outPath)
	}
	fmt.Printf("Initialized manifest (source: %s, issues: %d)\n", resp.Source, len(resp.Issues))
	fmt.Println(string(manifestJSON))
	return nil
}

func runTemplate(client *Client, args []string) error {
	fs := flag.NewFlagSet("manifest template", flag.ContinueOnError)
	variant := fs.String("variant", "minimal", "Template variant: minimal|full")
	jsonOutput := fs.Bool("json", false, "Output raw JSON response")
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: scenario-to-cloud manifest template [--variant minimal|full]")
	}

	body, resp, err := client.Template(*variant)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	manifestJSON, err := json.MarshalIndent(resp.Manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	fmt.Printf("Template: %s\n", resp.Variant)
	fmt.Println(string(manifestJSON))
	return nil
}

func runDoctor(client *Client, args []string) error {
	fs := flag.NewFlagSet("manifest doctor", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output raw JSON response")
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud manifest doctor <manifest.json>")
	}

	manifestData, err := internalmanifest.ReadJSONFile(fs.Arg(0))
	if err != nil {
		return err
	}
	body, resp, err := client.Doctor(manifestData)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Valid: %t\n", resp.Valid)
	fmt.Printf("Issues: %d\n", len(resp.Issues))
	for _, issue := range resp.Issues {
		fmt.Printf("- [%s] %s: %s\n", issue.Severity, issue.Path, issue.Message)
	}
	return nil
}

func runFix(client *Client, args []string) error {
	fs := flag.NewFlagSet("manifest fix", flag.ContinueOnError)
	writeInPlace := fs.Bool("write", false, "Write fixes back to input file")
	outPath := fs.String("out", "", "Write fixed manifest to this output path")
	jsonOutput := fs.Bool("json", false, "Output raw JSON response")
	if err := flagutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud manifest fix <manifest.json> [--write|--out <path>]")
	}
	if *writeInPlace && *outPath != "" {
		return fmt.Errorf("--write and --out cannot be combined")
	}

	inputPath := fs.Arg(0)
	manifestData, err := internalmanifest.ReadJSONFile(inputPath)
	if err != nil {
		return err
	}
	body, resp, err := client.Fix(manifestData)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fixedJSON, err := json.MarshalIndent(resp.Manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal fixed manifest: %w", err)
	}

	switch {
	case *writeInPlace:
		if err := os.WriteFile(inputPath, append(fixedJSON, '\n'), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", inputPath, err)
		}
		fmt.Printf("Wrote fixed manifest to %s\n", inputPath)
	case *outPath != "":
		if err := os.WriteFile(*outPath, append(fixedJSON, '\n'), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", *outPath, err)
		}
		fmt.Printf("Wrote fixed manifest to %s\n", *outPath)
	default:
		fmt.Println(string(fixedJSON))
	}

	fmt.Printf("Valid: %t | Issues: %d\n", resp.Valid, len(resp.Issues))
	return nil
}
