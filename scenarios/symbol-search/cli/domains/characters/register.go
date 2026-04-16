package characters

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"symbol-search/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `characters` subcommand group. It covers name-based search,
// codepoint detail lookup, and bulk range export — each a thin wrapper over the
// API's GET/POST routes.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "characters",
		Description: "Search, inspect, and export Unicode characters",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "search", Description: "Search characters by name or properties", Run: func(args []string) error { return runSearch(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show details for one character by codepoint", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "range", Description: "List characters between two codepoints", Run: func(args []string) error { return runRange(core, args) }},
		},
	}
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("characters search")
	query := fs.String("q", "", "Query string (alternative to positional)")
	category := fs.String("category", "", "Filter by Unicode category code (e.g. So, Sm)")
	block := fs.String("block", "", "Filter by Unicode block name")
	unicodeVersion := fs.String("unicode-version", "", "Filter by Unicode version")
	limit := fs.Int("limit", 50, "Maximum results (1-1000)")
	offset := fs.Int("offset", 0, "Pagination offset")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	q := support.PtrString(query)
	if q == "" && fs.NArg() > 0 {
		q = fs.Arg(0)
	}
	if q == "" {
		return fmt.Errorf("usage: characters search <query> [--category CODE] [--block NAME] [--unicode-version V] [--limit N] [--offset N]")
	}

	params := support.BuildQuery(map[string]string{
		"q":               q,
		"category":        *category,
		"block":           *block,
		"unicode_version": *unicodeVersion,
		"limit":           strconv.Itoa(*limit),
		"offset":          strconv.Itoa(*offset),
	})
	body, err := core.Get("/search", params)
	if err != nil {
		return err
	}

	var resp support.SearchResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Query: %q", q),
		fmt.Sprintf("Matches: %d (showing %d, offset %d)", resp.Total, len(resp.Characters), *offset),
		fmt.Sprintf("Elapsed: %.2fms", resp.QueryTimeMs),
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Characters",
		Results:        characterRows(resp.Characters),
		RetrievalHints: []string{
			fmt.Sprintf("%s characters get <codepoint>", support.CLIName),
			fmt.Sprintf("%s catalog blocks", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("characters get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: characters get <codepoint>")
	}
	codepoint := fs.Arg(0)

	body, err := core.Get("/character/"+codepoint, nil)
	if err != nil {
		return err
	}
	var resp support.CharacterDetailResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	char := resp.Character
	results := []string{
		fmt.Sprintf("Codepoint: %s", char.Codepoint),
		fmt.Sprintf("Decimal: %d", char.Decimal),
		fmt.Sprintf("Name: %s", char.Name),
		fmt.Sprintf("Category: %s", char.Category),
		fmt.Sprintf("Block: %s", char.Block),
		fmt.Sprintf("Unicode version: %s", char.UnicodeVersion),
	}
	if desc := support.PtrString(char.Description); desc != "" {
		results = append(results, fmt.Sprintf("Description: %s", desc))
	}
	if entity := support.PtrString(char.HTMLEntity); entity != "" {
		results = append(results, fmt.Sprintf("HTML entity: %s", entity))
	}
	if css := support.PtrString(char.CSSContent); css != "" {
		results = append(results, fmt.Sprintf("CSS content: %s", css))
	}
	if len(resp.UsageExamples) > 0 {
		results = append(results, "Usage examples:")
		for _, ex := range resp.UsageExamples {
			results = append(results, "  "+ex)
		}
	}
	if len(resp.RelatedCharacters) > 0 {
		results = append(results, fmt.Sprintf("Related (%d):", len(resp.RelatedCharacters)))
		for _, r := range resp.RelatedCharacters {
			results = append(results, fmt.Sprintf("  %s %s", r.Codepoint, r.Name))
		}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Character %s (%s)", char.Codepoint, char.Name)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s characters search %q", support.CLIName, char.Block),
			fmt.Sprintf("%s catalog blocks", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRange(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("characters range")
	format := fs.String("format", "", "Per-range output format hint (unicode|decimal|html)")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with a full {ranges:[...]} payload (overrides positional args)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload map[string]interface{}
	var summaryHead string
	if path := support.PtrString(bodyFile); path != "" {
		raw, err := readJSONFile(path)
		if err != nil {
			return err
		}
		payload = raw
		summaryHead = fmt.Sprintf("Range request from %s", path)
	} else {
		if fs.NArg() < 2 {
			return fmt.Errorf("usage: characters range <start> <end> [--format TYPE] | characters range --body-file FILE")
		}
		start := fs.Arg(0)
		end := fs.Arg(1)
		entry := map[string]interface{}{"start": start, "end": end}
		if f := support.PtrString(format); f != "" {
			entry["format"] = f
		}
		payload = map[string]interface{}{"ranges": []interface{}{entry}}
		summaryHead = fmt.Sprintf("Range %s .. %s", start, end)
	}

	body, err := core.Request("POST", "/bulk/range", nil, payload)
	if err != nil {
		return err
	}
	var resp support.BulkRangeResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			summaryHead,
			fmt.Sprintf("Characters: %d (ranges processed: %d)", resp.TotalCharacters, resp.RangesProcessed),
		},
		ResultsHeading: "Characters",
		Results:        characterRows(resp.Characters),
		RetrievalHints: []string{
			fmt.Sprintf("%s characters get <codepoint>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func characterRows(chars []support.Character) []string {
	if len(chars) == 0 {
		return []string{"(no characters)"}
	}
	rows := make([]string, 0, len(chars))
	for _, c := range chars {
		rows = append(rows, fmt.Sprintf("%s (%d) | %s | %s / %s",
			c.Codepoint, c.Decimal, c.Name, c.Category, c.Block))
	}
	return rows
}

func readJSONFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parse %s as JSON: %w", path, err)
	}
	return parsed, nil
}
