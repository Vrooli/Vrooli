package process

import (
	"fmt"
	"os"
	"strings"

	"data-structurer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `process` subcommand group covering POST /api/v1/process
// and GET /api/v1/process/:id. Result listings per-schema live under `data`
// because the upstream endpoint is /api/v1/data/:schema_id.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "process",
		Description: "Run data processing and inspect results",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "run", Description: "Submit text/file/url content for extraction against a schema", Run: func(args []string) error { return runProcess(core, args) }},
			{Name: "result", Aliases: []string{"get"}, Description: "Fetch the result of a previous processing request", Run: func(args []string) error { return runResult(core, args) }},
		},
	}
}

func runProcess(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("process run")
	inputType := fs.String("input-type", "auto", "Input type: auto|text|file|url")
	batch := fs.Bool("batch", false, "Treat input as a comma-separated batch")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full processing request; overrides other flags")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}

	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 2 {
			return fmt.Errorf("usage: process run <schema-id> <input> [--input-type auto|text|file|url] [--batch] | --body-file PATH")
		}
		schemaID := fs.Arg(0)
		input := strings.Join(fs.Args()[1:], " ")

		detected := detectInputType(*inputType, input)
		body := map[string]interface{}{
			"schema_id":  schemaID,
			"input_type": detected,
			"input_data": input,
			"batch_mode": *batch,
		}
		if *batch {
			items := strings.Split(input, ",")
			trimmed := make([]string, 0, len(items))
			for _, it := range items {
				it = strings.TrimSpace(it)
				if it != "" {
					trimmed = append(trimmed, it)
				}
			}
			body["batch_items"] = trimmed
		}
		payload = body
	}

	respBody, err := core.Request("POST", "/process", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ProcessingResponse
	if err := support.Decode(respBody, &resp); err != nil {
		return err
	}

	if resp.BatchID != "" {
		return renderBatchResult(resp, *jsonOutput)
	}
	return renderSingleResult(resp, *jsonOutput)
}

func runResult(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("process result")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: process result <processing-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/process/"+id, nil)
	if err != nil {
		return err
	}
	var detail support.ProcessingResultDetail
	if err := support.Decode(body, &detail); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Processing ID: %s", detail.ID),
		fmt.Sprintf("Schema ID: %s", detail.SchemaID),
		fmt.Sprintf("Status: %s", detail.ProcessingStatus),
		fmt.Sprintf("Confidence: %s", support.FormatConfidence(detail.ConfidenceScore)),
	}
	if detail.ProcessingTimeMs != nil {
		results = append(results, fmt.Sprintf("Processing time: %dms", *detail.ProcessingTimeMs))
	}
	if detail.SourceFileName != "" {
		results = append(results, fmt.Sprintf("Source: %s", detail.SourceFileName))
	}
	if detail.CreatedAt != nil {
		results = append(results, fmt.Sprintf("Created: %s", support.FormatTimePtr(detail.CreatedAt)))
	}
	if detail.ProcessedAt != nil {
		results = append(results, fmt.Sprintf("Processed: %s", support.FormatTimePtr(detail.ProcessedAt)))
	}
	if detail.ErrorMessage != "" {
		results = append(results, fmt.Sprintf("Error: %s", detail.ErrorMessage))
	}
	if len(detail.StructuredData) > 0 {
		results = append(results, "Structured data:")
		results = append(results, support.MapRows(detail.StructuredData)...)
	}
	if len(detail.Metadata) > 0 {
		results = append(results, "Metadata:")
		results = append(results, support.MapRows(detail.Metadata)...)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Processing %s (%s)", detail.ID, detail.ProcessingStatus)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s data %s", support.CLIName, detail.SchemaID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderSingleResult(resp support.ProcessingResponse, jsonOutput bool) error {
	result := []string{
		fmt.Sprintf("Processing ID: %s", resp.ProcessingID),
		fmt.Sprintf("Status: %s", resp.Status),
		fmt.Sprintf("Confidence: %s", support.FormatConfidence(resp.ConfidenceScore)),
	}
	if len(resp.Errors) > 0 {
		result = append(result, "Errors:")
		for _, e := range resp.Errors {
			result = append(result, "  "+e)
		}
	}
	if len(resp.StructuredData) > 0 {
		result = append(result, "Structured data:")
		result = append(result, support.MapRows(resp.StructuredData)...)
	}

	change := fmt.Sprintf("Stored processing record %s (status=%s)", resp.ProcessingID, resp.Status)
	report := cliapp.MutationReport{
		Result:  result,
		Changes: []string{change},
		NextCommand: []string{
			fmt.Sprintf("%s process result %s", support.CLIName, resp.ProcessingID),
		},
	}
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderBatchResult(resp support.ProcessingResponse, jsonOutput bool) error {
	summary := []string{
		fmt.Sprintf("Batch %s (%s)", resp.BatchID, resp.Status),
		fmt.Sprintf("Processed %d/%d (failed: %d)", resp.Completed, resp.TotalItems, resp.Failed),
	}
	if resp.AvgConfidence != nil {
		summary = append(summary, fmt.Sprintf("Avg confidence: %.2f", *resp.AvgConfidence))
	}
	summary = append(summary, fmt.Sprintf("Total time: %dms", resp.ProcessingTimeMs))

	rows := make([]string, 0, len(resp.Results))
	for _, r := range resp.Results {
		line := fmt.Sprintf("%s | %s | confidence=%s",
			support.ShortID(r.ProcessingID), r.Status, support.FormatConfidence(r.ConfidenceScore))
		if r.Error != "" {
			line += " | error=" + r.Error
		}
		rows = append(rows, line)
	}
	if len(rows) == 0 {
		rows = []string{"(no batch results)"}
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Batch items",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s process result <processing-id>", support.CLIName),
		},
	}
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// detectInputType mirrors the bash CLI's auto-detection rules: if the user
// passes --input-type auto (default), infer from whether the input is an
// existing file path or a URL; otherwise default to text.
func detectInputType(requested, input string) string {
	requested = strings.TrimSpace(strings.ToLower(requested))
	if requested != "" && requested != "auto" {
		return requested
	}
	if info, err := os.Stat(input); err == nil && !info.IsDir() {
		return "file"
	}
	lower := strings.ToLower(input)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return "url"
	}
	return "text"
}
