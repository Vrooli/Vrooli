package data

import (
	"fmt"
	"io"
	"os"
	"strings"

	"data-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `data` subcommand group for the data processing surface
// exposed under /api/v1/data (parse/transform/validate/query/profile). The bash
// CLI exposed each verb at the top level; the REST paths already group them
// under /data, and this CLI mirrors that.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "data",
		Description: "Parse, transform, validate, and query data",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "parse", Description: "Parse and analyze a data payload", Run: func(args []string) error { return runParse(core, args) }},
			{Name: "transform", Description: "Transform a dataset via SQL or operations", Run: func(args []string) error { return runTransform(core, args) }},
			{Name: "validate", Description: "Validate a dataset against a schema or quality rules", Run: func(args []string) error { return runValidate(core, args) }},
			{Name: "query", Description: "Execute a SQL query against registered datasets", Run: func(args []string) error { return runQuery(core, args) }},
			{Name: "profile", Description: "Profile a dataset", Run: func(args []string) error { return runProfile(core, args) }},
		},
	}
}

func runParse(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("data parse")
	format := fs.String("format", "auto", "Data format (csv, json, xml, excel, auto)")
	delimiter := fs.String("delimiter", ",", "CSV delimiter character")
	headers := fs.Bool("headers", false, "First row contains headers")
	sample := fs.Int("sample", 1000, "Number of rows to sample for type inference")
	bodyFile := fs.String("body-file", "", "Path to JSON file containing the full request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		input := ""
		if fs.NArg() > 0 {
			input = fs.Arg(0)
		}
		content, err := readDataInput(input)
		if err != nil {
			return err
		}
		payload = map[string]interface{}{
			"data":   content,
			"format": *format,
			"options": map[string]interface{}{
				"delimiter":   *delimiter,
				"headers":     *headers,
				"infer_types": true,
				"sample_size": *sample,
			},
		}
	}

	body, err := core.Request("POST", "/data/parse", nil, payload)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Parse result"},
		ResultsHeading: "Fields",
		Results:        support.MapRows(result),
		RetrievalHints: []string{fmt.Sprintf("%s data parse <file> --format csv --headers", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runTransform(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("data transform")
	validate := fs.Bool("validate", false, "Validate output against inferred schema")
	cache := fs.Bool("cache", true, "Cache the result server-side")
	bodyFile := fs.String("body-file", "", "Path to JSON file containing the full request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 2 {
			return fmt.Errorf("usage: data transform <dataset-id> <sql|operation> [--validate] [--cache=false] (or pass --body-file PATH)")
		}
		dataset := fs.Arg(0)
		operation := fs.Arg(1)

		transformation, err := buildTransformation(operation)
		if err != nil {
			return err
		}

		payload = map[string]interface{}{
			"dataset_id":      dataset,
			"transformations": []interface{}{transformation},
			"options": map[string]interface{}{
				"validate_output": *validate,
				"cache_result":    *cache,
			},
		}
	}

	body, err := core.Request("POST", "/data/transform", nil, payload)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Transform result"},
		ResultsHeading: "Fields",
		Results:        support.MapRows(result),
		RetrievalHints: []string{fmt.Sprintf("%s data transform <dataset-id> 'SELECT * WHERE age > 25'", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runValidate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("data validate")
	schemaFile := fs.String("schema-file", "", "Path to JSON schema file")
	rulesFile := fs.String("rules-file", "", "Path to quality rules JSON file")
	bodyFile := fs.String("body-file", "", "Path to JSON file containing the full request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: data validate <dataset-id> [--schema-file PATH] [--rules-file PATH] (or pass --body-file PATH)")
		}
		dataset := fs.Arg(0)

		built := map[string]interface{}{"dataset_id": dataset}
		schema, err := support.ReadJSONFile(*schemaFile, false)
		if err != nil {
			return err
		}
		if schema != nil {
			built["schema"] = schema
		}
		rules, err := support.ReadJSONFile(*rulesFile, false)
		if err != nil {
			return err
		}
		if rules != nil {
			built["quality_rules"] = rules
		}
		payload = built
	}

	body, err := core.Request("POST", "/data/validate", nil, payload)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	summary := []string{"Validation result"}
	if valid, ok := result["is_valid"].(bool); ok {
		if valid {
			summary = append(summary, "Status: valid")
		} else {
			summary = append(summary, "Status: invalid")
		}
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Fields",
		Results:        support.MapRows(result),
		RetrievalHints: []string{fmt.Sprintf("%s data validate <dataset-id> --schema-file schema.json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runQuery(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("data query")
	datasets := fs.String("datasets", "", "Comma-separated dataset IDs (id or id:alias)")
	limit := fs.Int("limit", 100, "Maximum rows to return")
	explain := fs.Bool("explain", false, "Return query execution plan")
	cache := fs.Bool("cache", true, "Cache the result server-side")
	bodyFile := fs.String("body-file", "", "Path to JSON file containing the full request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: data query <sql> [--datasets id1,id2:alias2] [--limit N] [--explain] (or pass --body-file PATH)")
		}
		sql := fs.Arg(0)
		payload = map[string]interface{}{
			"sql":      sql,
			"datasets": parseDatasets(*datasets),
			"options": map[string]interface{}{
				"limit":   *limit,
				"explain": *explain,
				"cache":   *cache,
			},
		}
	}

	body, err := core.Request("POST", "/data/query", nil, payload)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Query result"},
		ResultsHeading: "Fields",
		Results:        support.MapRows(result),
		RetrievalHints: []string{fmt.Sprintf("%s data query 'SELECT COUNT(*) FROM dataset' --limit 10", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runProfile(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("data profile")
	bodyFile := fs.String("body-file", "", "Path to JSON file containing the full request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	switch {
	case strings.TrimSpace(*bodyFile) != "":
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	case fs.NArg() >= 1:
		payload = map[string]interface{}{"dataset_id": fs.Arg(0)}
	default:
		return fmt.Errorf("usage: data profile <dataset-id> (or pass --body-file PATH)")
	}

	body, err := core.Request("POST", "/data/profile", nil, payload)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Profile result"},
		ResultsHeading: "Fields",
		Results:        support.MapRows(result),
		RetrievalHints: []string{fmt.Sprintf("%s data profile <dataset-id>", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// readDataInput returns the textual content to send as the `data` field for a
// parse request. "-" or empty reads from stdin; a path reads that file.
func readDataInput(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" || input == "-" {
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		return string(raw), nil
	}
	raw, err := os.ReadFile(input)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", input, err)
	}
	return string(raw), nil
}

// buildTransformation mirrors the bash CLI's shorthand: SELECT ... is a SQL
// transformation; anything else is split on the first colon into type:condition.
func buildTransformation(operation string) (map[string]interface{}, error) {
	trimmed := strings.TrimSpace(operation)
	if trimmed == "" {
		return nil, fmt.Errorf("transformation operation is required")
	}
	upper := strings.ToUpper(trimmed)
	if strings.HasPrefix(upper, "SELECT") {
		return map[string]interface{}{
			"type": "sql",
			"sql":  trimmed,
		}, nil
	}
	parts := strings.SplitN(trimmed, ":", 2)
	opType := parts[0]
	var params string
	if len(parts) == 2 {
		params = parts[1]
	}
	return map[string]interface{}{
		"type": opType,
		"parameters": map[string]interface{}{
			"condition": params,
		},
	}, nil
}

// parseDatasets converts a comma-separated list (id or id:alias) into the
// request shape expected by /api/v1/data/query.
func parseDatasets(value string) []map[string]interface{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return []map[string]interface{}{}
	}
	out := []map[string]interface{}{}
	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		id, alias := entry, entry
		if idx := strings.Index(entry, ":"); idx >= 0 {
			id = strings.TrimSpace(entry[:idx])
			alias = strings.TrimSpace(entry[idx+1:])
			if alias == "" {
				alias = id
			}
		}
		out = append(out, map[string]interface{}{
			"id":    id,
			"alias": alias,
		})
	}
	return out
}
