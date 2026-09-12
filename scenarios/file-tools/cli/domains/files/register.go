package files

import (
	"fmt"
	"os"
	"strings"

	"file-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `files` subcommand group covering metadata, checksum,
// operation, search, organize, duplicates, and batch metadata extraction.
// Commands with rich nested options (operation, organize, duplicates,
// metadata-extract) take `--body-file PATH` so callers supply canonical JSON
// rather than reconstructing nested payloads on the command line.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "files",
		Description: "File metadata, search, checksums, and direct file operations",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "metadata", Description: "Get file metadata (size, mime, checksums)", Run: func(args []string) error { return runMetadata(core, args) }},
			{Name: "metadata-extract", Description: "Extract batch metadata from a set of files (--body-file PATH)", Run: func(args []string) error { return runMetadataExtract(core, args) }},
			{Name: "checksum", Description: "Calculate file checksums", Run: func(args []string) error { return runChecksum(core, args) }},
			{Name: "operation", Description: "Perform copy/move/rename/delete (--body-file PATH)", Run: func(args []string) error { return runOperation(core, args) }},
			{Name: "search", Description: "Search files by name", Run: func(args []string) error { return runSearch(core, args) }},
			{Name: "organize", Description: "Organize files by rules (--body-file PATH)", Run: func(args []string) error { return runOrganize(core, args) }},
			{Name: "duplicates", Description: "Detect duplicate files (--body-file PATH)", Run: func(args []string) error { return runDuplicates(core, args) }},
		},
	}
}

func runMetadata(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("files metadata")
	path := fs.String("path", "", "File path (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*path) == "" {
		return fmt.Errorf("--path PATH is required")
	}

	query := support.BuildQuery(map[string]string{"path": *path})
	body, err := core.Get("/files/metadata", query)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Metadata for %s", *path)},
		ResultsHeading: "Fields",
		Results:        support.MapRows(resp),
		RetrievalHints: []string{fmt.Sprintf("%s files checksum %s", support.CLIName, *path)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runMetadataExtract(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("files metadata-extract")
	bodyFile := fs.String("body-file", "", "JSON body file with file_paths/extraction_types/options (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}
	body, err := core.Request("POST", "/files/metadata/extract", nil, payload)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{}
	if total, ok := resp["total_processed"].(float64); ok {
		summary = append(summary, fmt.Sprintf("Processed: %d", int(total)))
	}
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Results",
		Results:        support.MapRows(resp),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runChecksum(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("files checksum")
	algorithm := fs.String("algorithm", "sha256", "Hash algorithm: md5|sha1|sha256")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	targets := fs.Args()
	if len(targets) == 0 {
		return fmt.Errorf("usage: files checksum <file>... [--algorithm sha256]")
	}

	payload := map[string]interface{}{
		"files":     targets,
		"algorithm": *algorithm,
	}
	body, err := core.Request("POST", "/files/checksum", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ChecksumResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	rows := make([]string, 0, len(resp.Results))
	for _, r := range resp.Results {
		rows = append(rows, fmt.Sprintf("%s  %s (%s)", r.Checksum, r.File, r.Algorithm))
	}
	if len(rows) == 0 {
		rows = []string{"(no checksums computed)"}
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Checksums: %d", resp.Total)},
		ResultsHeading: "Results",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runOperation(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("files operation")
	bodyFile := fs.String("body-file", "", "JSON body file with operation/source/target/options (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}
	body, err := core.Request("POST", "/files/operation", nil, payload)
	if err != nil {
		return err
	}
	var resp support.FileOperationResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	changes := []string{
		fmt.Sprintf("Operation: %s", resp.Operation),
		fmt.Sprintf("Source: %s", resp.Source),
	}
	if resp.Target != "" {
		changes = append(changes, fmt.Sprintf("Target: %s", resp.Target))
	}
	changes = append(changes, fmt.Sprintf("Status: %s", resp.Status))

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s complete (op %s)", resp.Operation, resp.OperationID)},
		Changes: changes,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("files search")
	queryText := fs.String("query", "", "Search query (filename substring)")
	searchType := fs.String("type", "", "Search type: filename|all")
	path := fs.String("path", "", "Directory to search (defaults to API working dir)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"query":       *queryText,
		"search_type": *searchType,
		"path":        *path,
	})
	body, err := core.Get("/files/search", query)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{}
	if total, ok := resp["total_matches"].(float64); ok {
		summary = append(summary, fmt.Sprintf("Matches: %d", int(total)))
	}
	if elapsed, ok := resp["search_time_ms"].(float64); ok {
		summary = append(summary, fmt.Sprintf("Search time: %dms", int(elapsed)))
	}
	rows := searchRows(resp["results"])
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Hits",
		Results:        rows,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func searchRows(results interface{}) []string {
	list, ok := results.([]interface{})
	if !ok || len(list) == 0 {
		return []string{"(no matches)"}
	}
	rows := make([]string, 0, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		path, _ := entry["file_path"].(string)
		score, _ := entry["relevance_score"].(float64)
		rows = append(rows, fmt.Sprintf("%s (score %.2f)", path, score))
	}
	if len(rows) == 0 {
		return []string{"(no matches)"}
	}
	return rows
}

func runOrganize(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("files organize")
	bodyFile := fs.String("body-file", "", "JSON body file with source_path/destination_path/organization_rules/options (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}
	body, err := core.Request("POST", "/files/organize", nil, payload)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{"Organize plan generated"},
		ResultsHeading: "Fields",
		Results:        support.MapRows(resp),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runDuplicates(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("files duplicates")
	bodyFile := fs.String("body-file", "", "JSON body file with scan_paths/detection_method/options (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}
	body, err := core.Request("POST", "/files/duplicates/detect", nil, payload)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{"Duplicate detection complete"},
		ResultsHeading: "Fields",
		Results:        support.MapRows(resp),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
