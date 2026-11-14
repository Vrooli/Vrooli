package text

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"text-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `text` subcommand group covering diff/search/transform/
// extract/analyze. The API is the source of truth for processing logic; this
// package is a thin wrapper that builds request bodies, calls the API, and
// formats responses through the standard output contracts.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "text",
		Description: "Diff, search, transform, extract, and analyze text",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "diff", Description: "Compare two text inputs", Run: func(args []string) error { return runDiff(core, args) }},
			{Name: "search", Description: "Search a text input for a pattern", Run: func(args []string) error { return runSearch(core, args) }},
			{Name: "transform", Description: "Apply transformations to a text input", Run: func(args []string) error { return runTransform(core, args) }},
			{Name: "extract", Description: "Extract text from a document or URL", Run: func(args []string) error { return runExtract(core, args) }},
			{Name: "analyze", Description: "Run NLP analyses on a text input", Run: func(args []string) error { return runAnalyze(core, args) }},
		},
	}
}

// ---------- diff ----------

func runDiff(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("text diff")
	diffType := fs.String("type", "line", "Diff algorithm: line|word|character|semantic")
	ignoreWhitespace := fs.Bool("ignore-whitespace", false, "Ignore whitespace differences")
	ignoreCase := fs.Bool("ignore-case", false, "Ignore case differences")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request body (overrides other flags)")
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
			return fmt.Errorf("usage: text diff <input1> <input2> [--type line|word|character|semantic] [--ignore-whitespace] [--ignore-case] [--body-file PATH]")
		}
		text1, err := support.ReadTextInput(fs.Arg(0))
		if err != nil {
			return err
		}
		text2, err := support.ReadTextInput(fs.Arg(1))
		if err != nil {
			return err
		}
		payload = map[string]interface{}{
			"text1": text1,
			"text2": text2,
			"options": map[string]interface{}{
				"type":              *diffType,
				"ignore_whitespace": *ignoreWhitespace,
				"ignore_case":       *ignoreCase,
			},
		}
	}

	body, err := core.Request("POST", "/text/diff", nil, payload)
	if err != nil {
		return err
	}
	var resp support.DiffResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{}
	if resp.Summary != "" {
		summary = append(summary, resp.Summary)
	}
	summary = append(summary,
		fmt.Sprintf("Changes: %d", len(resp.Changes)),
		fmt.Sprintf("Similarity: %.2f%%", resp.SimilarityScore*100),
	)

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Changes",
		Results:        diffRows(resp.Changes),
		RetrievalHints: []string{
			fmt.Sprintf("%s text diff <a> <b> --type word", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func diffRows(changes []support.Change) []string {
	if len(changes) == 0 {
		return []string{"(no changes)"}
	}
	rows := make([]string, 0, len(changes))
	for _, c := range changes {
		rows = append(rows, fmt.Sprintf("[%s] Line %d: %s", c.Type, c.LineStart, c.Content))
	}
	return rows
}

// ---------- search ----------

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("text search")
	regex := fs.Bool("regex", false, "Use regex pattern matching")
	caseSensitive := fs.Bool("case-sensitive", false, "Case sensitive search")
	wholeWord := fs.Bool("whole-word", false, "Match whole words only")
	fuzzy := fs.Bool("fuzzy", false, "Enable fuzzy matching")
	semantic := fs.Bool("semantic", false, "Use semantic search (requires Ollama)")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request body (overrides other flags)")
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
			return fmt.Errorf("usage: text search <pattern> [<source>] [--regex] [--case-sensitive] [--whole-word] [--fuzzy] [--semantic] [--body-file PATH]")
		}
		pattern := fs.Arg(0)
		var text string
		if fs.NArg() >= 2 {
			combined := make([]string, 0, fs.NArg()-1)
			for i := 1; i < fs.NArg(); i++ {
				part, err := support.ReadTextInput(fs.Arg(i))
				if err != nil {
					return err
				}
				combined = append(combined, part)
			}
			text = strings.Join(combined, "\n")
		} else {
			stdinText, err := support.ReadTextInput("-")
			if err != nil {
				return err
			}
			text = stdinText
		}
		payload = map[string]interface{}{
			"text":    text,
			"pattern": pattern,
			"options": map[string]interface{}{
				"regex":          *regex,
				"case_sensitive": *caseSensitive,
				"whole_word":     *wholeWord,
				"fuzzy":          *fuzzy,
				"semantic":       *semantic,
			},
		}
	}

	body, err := core.Request("POST", "/text/search", nil, payload)
	if err != nil {
		return err
	}
	var resp support.SearchResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d matches", resp.TotalMatches)},
		ResultsHeading: "Matches",
		Results:        searchRows(resp.Matches),
		RetrievalHints: []string{
			fmt.Sprintf("%s text search <pattern> <file> --regex", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func searchRows(matches []support.Match) []string {
	if len(matches) == 0 {
		return []string{"(no matches)"}
	}
	rows := make([]string, 0, len(matches))
	for _, m := range matches {
		rows = append(rows, fmt.Sprintf("Line %d: %s", m.Line, m.Context))
	}
	return rows
}

// ---------- transform ----------

type transformSpec struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

func runTransform(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("text transform")
	upper := fs.Bool("upper", false, "Convert to uppercase")
	lower := fs.Bool("lower", false, "Convert to lowercase")
	title := fs.Bool("title", false, "Convert to title case")
	base64Flag := fs.Bool("base64", false, "Encode/decode base64")
	formatFlag := fs.String("format", "", "Format structured data: json|xml|yaml")
	sanitize := fs.Bool("sanitize", false, "Remove HTML and normalize whitespace")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request body (overrides other flags)")
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
		source := "-"
		if fs.NArg() >= 1 {
			source = fs.Arg(0)
		}
		text, err := support.ReadTextInput(source)
		if err != nil {
			return err
		}

		transformations := []transformSpec{}
		if *upper {
			transformations = append(transformations, transformSpec{Type: "case", Parameters: map[string]interface{}{"type": "upper"}})
		}
		if *lower {
			transformations = append(transformations, transformSpec{Type: "case", Parameters: map[string]interface{}{"type": "lower"}})
		}
		if *title {
			transformations = append(transformations, transformSpec{Type: "case", Parameters: map[string]interface{}{"type": "title"}})
		}
		if *base64Flag {
			transformations = append(transformations, transformSpec{Type: "encode", Parameters: map[string]interface{}{"type": "base64"}})
		}
		if strings.TrimSpace(*formatFlag) != "" {
			transformations = append(transformations, transformSpec{Type: "format", Parameters: map[string]interface{}{"type": *formatFlag}})
		}
		if *sanitize {
			transformations = append(transformations, transformSpec{Type: "sanitize", Parameters: map[string]interface{}{}})
		}
		if len(transformations) == 0 {
			return fmt.Errorf("at least one transformation flag is required (see --help); use --body-file for advanced payloads")
		}

		payload = map[string]interface{}{
			"text":            text,
			"transformations": transformations,
		}
	}

	body, err := core.Request("POST", "/text/transform", nil, payload)
	if err != nil {
		return err
	}
	var resp support.TransformResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{}
	if len(resp.TransformationsApplied) > 0 {
		summary = append(summary, fmt.Sprintf("Applied: %s", strings.Join(resp.TransformationsApplied, ", ")))
	}
	if len(resp.Warnings) > 0 {
		summary = append(summary, fmt.Sprintf("Warnings: %d", len(resp.Warnings)))
	}
	if len(summary) == 0 {
		summary = []string{"Transformation complete"}
	}

	results := []string{"=== Result ==="}
	if resp.Result == "" {
		results = append(results, "(empty result)")
	} else {
		results = append(results, resp.Result)
	}
	if len(resp.Warnings) > 0 {
		results = append(results, "=== Warnings ===")
		results = append(results, resp.Warnings...)
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Transform",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// ---------- extract ----------

func runExtract(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("text extract")
	format := fs.String("format", "auto", "Source format: pdf|html|docx|auto")
	ocr := fs.Bool("ocr", false, "Use OCR for images")
	metadata := fs.Bool("metadata", false, "Extract metadata")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request body (overrides other flags)")
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
			return fmt.Errorf("usage: text extract <source> [--format pdf|html|docx|auto] [--ocr] [--metadata] [--body-file PATH]")
		}
		source := fs.Arg(0)

		var sourceBlock map[string]interface{}
		switch {
		case strings.HasPrefix(source, "http://"), strings.HasPrefix(source, "https://"):
			sourceBlock = map[string]interface{}{"url": source}
		default:
			info, err := os.Stat(source)
			if err != nil || info.IsDir() {
				return fmt.Errorf("source must be a URL or an existing file: %s", source)
			}
			raw, err := os.ReadFile(source)
			if err != nil {
				return fmt.Errorf("read %s: %w", source, err)
			}
			sourceBlock = map[string]interface{}{"file": encodeBase64(raw)}
		}

		payload = map[string]interface{}{
			"source": sourceBlock,
			"format": *format,
			"options": map[string]interface{}{
				"ocr":              *ocr,
				"extract_metadata": *metadata,
			},
		}
	}

	body, err := core.Request("POST", "/text/extract", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ExtractResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Extracted %d characters", len(resp.Text))}
	if len(resp.Warnings) > 0 {
		summary = append(summary, fmt.Sprintf("Warnings: %d", len(resp.Warnings)))
	}

	results := []string{"=== Text ==="}
	if resp.Text == "" {
		results = append(results, "(no text extracted)")
	} else {
		results = append(results, resp.Text)
	}
	if len(resp.Metadata) > 0 {
		results = append(results, "=== Metadata ===")
		results = append(results, support.MapRows(resp.Metadata)...)
	}
	if len(resp.Warnings) > 0 {
		results = append(results, "=== Warnings ===")
		results = append(results, resp.Warnings...)
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Extraction",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// ---------- analyze ----------

func runAnalyze(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("text analyze")
	entities := fs.Bool("entities", false, "Extract named entities")
	sentiment := fs.Bool("sentiment", false, "Analyze sentiment")
	summary := fs.Bool("summary", false, "Generate summary")
	summaryLength := fs.Int("summary-length", 0, "Target summary length (used with --summary)")
	keywords := fs.Bool("keywords", false, "Extract keywords")
	language := fs.Bool("language", false, "Detect language")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request body (overrides other flags)")
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
		source := "-"
		if fs.NArg() >= 1 {
			source = fs.Arg(0)
		}
		text, err := support.ReadTextInput(source)
		if err != nil {
			return err
		}

		analyses := []string{}
		if *entities {
			analyses = append(analyses, "entities")
		}
		if *sentiment {
			analyses = append(analyses, "sentiment")
		}
		if *summary {
			analyses = append(analyses, "summary")
		}
		if *keywords {
			analyses = append(analyses, "keywords")
		}
		if *language {
			analyses = append(analyses, "language")
		}
		if len(analyses) == 0 {
			return fmt.Errorf("at least one analysis flag is required (see --help); use --body-file for advanced payloads")
		}

		payload = map[string]interface{}{
			"text":     text,
			"analyses": analyses,
			"options": map[string]interface{}{
				"summary_length": *summaryLength,
			},
		}
	}

	body, err := core.Request("POST", "/text/analyze", nil, payload)
	if err != nil {
		return err
	}
	var resp support.AnalyzeResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := []string{}
	if len(resp.Entities) > 0 {
		results = append(results, "=== Entities ===")
		for _, e := range resp.Entities {
			results = append(results, fmt.Sprintf("%s [%s] (%.2f)", e.Value, e.Type, e.Confidence))
		}
	}
	if resp.Sentiment.Label != "" || resp.Sentiment.Score != 0 {
		results = append(results, "=== Sentiment ===",
			fmt.Sprintf("Label: %s", resp.Sentiment.Label),
			fmt.Sprintf("Score: %.2f", resp.Sentiment.Score))
	}
	if resp.Summary != "" {
		results = append(results, "=== Summary ===", resp.Summary)
	}
	if len(resp.Keywords) > 0 {
		results = append(results, "=== Keywords ===")
		for _, k := range resp.Keywords {
			results = append(results, fmt.Sprintf("%s (%.2f)", k.Word, k.Score))
		}
	}
	if resp.Language.Code != "" || resp.Language.Name != "" {
		results = append(results, "=== Language ===",
			fmt.Sprintf("Code: %s", resp.Language.Code),
			fmt.Sprintf("Name: %s", resp.Language.Name),
			fmt.Sprintf("Confidence: %.2f", resp.Language.Confidence))
	}
	if len(results) == 0 {
		results = []string{"(no analysis results)"}
	}

	summaryLine := []string{"Analysis complete"}
	if raw := support.EnvelopeMessage(body); raw != "" {
		summaryLine = append(summaryLine, raw)
	}

	report := cliapp.ListReport{
		Summary:        summaryLine,
		ResultsHeading: "Analysis",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
