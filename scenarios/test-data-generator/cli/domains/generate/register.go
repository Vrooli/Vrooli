package generate

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"test-data-generator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `generate` subcommand group. One subcommand per
// predefined type (`users`, `companies`, `products`, `orders`) wraps
// `POST /api/generate/<type>`. The `custom` subcommand wraps
// `POST /api/generate/custom` and requires either a JSON `--schema` string or
// a complete `--body-file` payload.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "generate",
		Description: "Generate mock data by type",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "users", Description: "Generate user records", Run: func(args []string) error { return runType(core, args, "users") }},
			{Name: "companies", Description: "Generate company records", Run: func(args []string) error { return runType(core, args, "companies") }},
			{Name: "products", Description: "Generate product records", Run: func(args []string) error { return runType(core, args, "products") }},
			{Name: "orders", Description: "Generate order records", Run: func(args []string) error { return runType(core, args, "orders") }},
			{Name: "custom", Description: "Generate records from a custom schema (see --schema / --body-file)", Run: func(args []string) error { return runCustom(core, args) }},
		},
	}
}

type generateOptions struct {
	count    int
	format   string
	output   string
	fields   string
	seed     string
	pretty   bool
	bodyFile string
}

func registerCommonFlags(fs *flag.FlagSet, opts *generateOptions) {
	fs.IntVar(&opts.count, "count", 10, "Number of records to generate")
	fs.IntVar(&opts.count, "c", 10, "Number of records to generate (short)")
	fs.StringVar(&opts.format, "format", "json", "Output format: json|csv|xml|sql")
	fs.StringVar(&opts.format, "f", "json", "Output format (short)")
	fs.StringVar(&opts.output, "output", "", "Output file path (default: stdout)")
	fs.StringVar(&opts.output, "o", "", "Output file path (short)")
	fs.StringVar(&opts.fields, "fields", "", "Comma-separated list of fields to include")
	fs.StringVar(&opts.seed, "seed", "", "Seed for deterministic generation")
	fs.BoolVar(&opts.pretty, "pretty", false, "Pretty-print JSON output")
	fs.StringVar(&opts.bodyFile, "body-file", "", "Path to a JSON file whose contents are sent as the request body (overrides other flags)")
}

func runType(core *cliapp.ScenarioApp, args []string, dataType string) error {
	fs := support.NewFlagSet("generate " + dataType)
	opts := generateOptions{}
	registerCommonFlags(fs, &opts)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := buildPayload(opts, nil)
	if err != nil {
		return err
	}
	return postAndRender(core, dataType, payload, opts, *jsonOutput)
}

func runCustom(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("generate custom")
	opts := generateOptions{}
	registerCommonFlags(fs, &opts)
	schema := fs.String("schema", "", "JSON schema object describing fields and types (required unless --body-file is set)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var schemaRaw json.RawMessage
	if strings.TrimSpace(*schema) != "" {
		if err := json.Unmarshal([]byte(*schema), &schemaRaw); err != nil {
			return fmt.Errorf("parse --schema as JSON: %w", err)
		}
	}
	if opts.bodyFile == "" && len(schemaRaw) == 0 {
		return fmt.Errorf("generate custom requires --schema '<json>' or --body-file <path>")
	}

	payload, err := buildPayload(opts, schemaRaw)
	if err != nil {
		return err
	}
	return postAndRender(core, "custom", payload, opts, *jsonOutput)
}

// buildPayload assembles the request body. When --body-file is set, its raw
// contents are used verbatim so callers can supply payloads the CLI flags do
// not expose. Otherwise, fields are constructed from opts.
func buildPayload(opts generateOptions, schema json.RawMessage) ([]byte, error) {
	if strings.TrimSpace(opts.bodyFile) != "" {
		raw, err := support.ReadJSONFile(opts.bodyFile, true)
		if err != nil {
			return nil, err
		}
		return raw, nil
	}

	body := map[string]interface{}{
		"count":  opts.count,
		"format": opts.format,
	}
	if strings.TrimSpace(opts.seed) != "" {
		body["seed"] = opts.seed
	}
	if fields := support.SplitCSV(opts.fields); len(fields) > 0 {
		body["fields"] = fields
	}
	if len(schema) > 0 {
		body["schema"] = schema
	}
	return json.Marshal(body)
}

func postAndRender(core *cliapp.ScenarioApp, dataType string, payload []byte, opts generateOptions, jsonOutput bool) error {
	// json.RawMessage lets the underlying client marshal the pre-built body
	// without re-encoding through reflect-based path.
	body, err := core.Request("POST", "/generate/"+dataType, nil, json.RawMessage(payload))
	if err != nil {
		return err
	}

	var resp support.GenerateResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	outputBytes, err := formatData(resp, opts)
	if err != nil {
		return err
	}
	if opts.output != "" {
		if err := support.WriteOutput(opts.output, outputBytes); err != nil {
			return err
		}
	}

	result := []string{
		fmt.Sprintf("Generated %d %s record(s) in %s format", resp.Count, resp.Type, resp.Format),
	}
	if resp.Timestamp != "" {
		result = append(result, fmt.Sprintf("Server timestamp: %s", resp.Timestamp))
	}
	if resp.Note != "" {
		result = append(result, fmt.Sprintf("Note: %s", resp.Note))
	}

	changes := []string{}
	if opts.output != "" {
		changes = append(changes, fmt.Sprintf("Wrote %d byte(s) to %s", len(outputBytes), opts.output))
	} else if !jsonOutput {
		// Human mode with no --output: emit the payload to stdout before the
		// report so the generated data is visible inline.
		if _, err := os.Stdout.Write(outputBytes); err != nil {
			return err
		}
		if len(outputBytes) > 0 && outputBytes[len(outputBytes)-1] != '\n' {
			fmt.Println()
		}
	}

	report := cliapp.MutationReport{
		Result:  result,
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s types", support.CLIName),
			fmt.Sprintf("%s generate %s --count 25 --output %s.json", support.CLIName, dataType, dataType),
		},
	}
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

// formatData renders the `data` field from the response according to format.
// For json, optionally pretty-prints; for string formats (xml/sql/csv), the
// API returns a JSON string that we unwrap to raw text.
func formatData(resp support.GenerateResponse, opts generateOptions) ([]byte, error) {
	if len(resp.Data) == 0 {
		return []byte{}, nil
	}
	switch strings.ToLower(resp.Format) {
	case "json":
		if opts.pretty {
			var value interface{}
			if err := json.Unmarshal(resp.Data, &value); err != nil {
				return nil, fmt.Errorf("parse data as JSON: %w", err)
			}
			return json.MarshalIndent(value, "", "  ")
		}
		return []byte(resp.Data), nil
	default:
		var asString string
		if err := json.Unmarshal(resp.Data, &asString); err == nil {
			return []byte(asString), nil
		}
		return []byte(resp.Data), nil
	}
}
