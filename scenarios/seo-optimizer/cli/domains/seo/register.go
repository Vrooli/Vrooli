package seo

import (
	"fmt"
	"os"
	"strings"

	"seo-optimizer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes the four SEO Optimizer API surfaces as flat verbs. Each
// command is a thin wrapper over a single POST endpoint; the API is the source
// of truth for analysis logic. Every command accepts `--body-file <path>` for
// callers that want to submit a pre-built JSON payload instead of the typed
// flag form.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "SEO",
		Commands: []cliapp.Command{
			{
				Name:        "audit",
				Description: "Run an SEO audit on a URL (POST /api/seo-audit)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runAudit(core, args) },
			},
			{
				Name:        "keywords",
				Description: "Research keywords from a seed term (POST /api/keyword-research)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runKeywords(core, args) },
			},
			{
				Name:        "content",
				Description: "Optimize content against target keywords (POST /api/content-optimize)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runContent(core, args) },
			},
			{
				Name:        "competitors",
				Description: "Compare your URL against a competitor (POST /api/competitor-analysis)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runCompetitors(core, args) },
			},
		},
	}
}

func runAudit(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("audit")
	depth := fs.Int("depth", 0, "Crawl depth (optional)")
	bodyFile := fs.String("body-file", "", "Path to JSON file with full request body (overrides positional args)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if raw, err := support.ReadJSONFile(*bodyFile, false); err != nil {
		return err
	} else if raw != nil {
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: audit <url> [--depth N] | --body-file <path>")
		}
		payload = support.AuditRequest{
			URL:   fs.Arg(0),
			Depth: *depth,
		}
	}

	body, err := core.Request("POST", "/seo-audit", nil, payload)
	if err != nil {
		return err
	}

	result := decodeMap(body)
	report := cliapp.ListReport{
		Summary:        auditSummary(result, fs.Arg(0)),
		ResultsHeading: "Audit",
		Results:        support.MapRows(result),
		RetrievalHints: []string{
			fmt.Sprintf("%s audit %s --depth 2", support.CLIName, placeholder(fs.Arg(0), "<url>")),
			fmt.Sprintf("%s content --body-file payload.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runKeywords(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("keywords")
	location := fs.String("location", "", "Target location for keyword research")
	language := fs.String("language", "", "Language code (e.g. en)")
	bodyFile := fs.String("body-file", "", "Path to JSON file with full request body (overrides positional args)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if raw, err := support.ReadJSONFile(*bodyFile, false); err != nil {
		return err
	} else if raw != nil {
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: keywords <seed> [--location LOC] [--language LANG] | --body-file <path>")
		}
		payload = support.KeywordRequest{
			SeedKeyword:    fs.Arg(0),
			TargetLocation: strings.TrimSpace(*location),
			Language:       strings.TrimSpace(*language),
		}
	}

	body, err := core.Request("POST", "/keyword-research", nil, payload)
	if err != nil {
		return err
	}

	result := decodeMap(body)
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Keyword research for: %s", placeholder(fs.Arg(0), "(from body-file)"))},
		ResultsHeading: "Research",
		Results:        support.MapRows(result),
		RetrievalHints: []string{
			fmt.Sprintf("%s keywords %s --location US --language en", support.CLIName, placeholder(fs.Arg(0), "<seed>")),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runContent(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("content")
	targetKeywords := fs.String("keywords", "", "Target keywords (comma-separated)")
	contentType := fs.String("type", "", "Content type (e.g. blog, landing, product)")
	bodyFile := fs.String("body-file", "", "Path to JSON file with full request body (overrides positional args)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if raw, err := support.ReadJSONFile(*bodyFile, false); err != nil {
		return err
	} else if raw != nil {
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: content <content> [--keywords K1,K2] [--type TYPE] | --body-file <path>")
		}
		payload = support.ContentRequest{
			Content:        fs.Arg(0),
			TargetKeywords: strings.TrimSpace(*targetKeywords),
			ContentType:    strings.TrimSpace(*contentType),
		}
	}

	body, err := core.Request("POST", "/content-optimize", nil, payload)
	if err != nil {
		return err
	}

	result := decodeMap(body)
	report := cliapp.ListReport{
		Summary:        []string{"Content optimization"},
		ResultsHeading: "Analysis",
		Results:        support.MapRows(result),
		RetrievalHints: []string{
			fmt.Sprintf("%s content --body-file payload.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCompetitors(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("competitors")
	analysisType := fs.String("type", "", "Analysis type (optional)")
	bodyFile := fs.String("body-file", "", "Path to JSON file with full request body (overrides positional args)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if raw, err := support.ReadJSONFile(*bodyFile, false); err != nil {
		return err
	} else if raw != nil {
		payload = raw
	} else {
		if fs.NArg() < 2 {
			return fmt.Errorf("usage: competitors <your-url> <competitor-url> [--type TYPE] | --body-file <path>")
		}
		payload = support.CompetitorRequest{
			YourURL:       fs.Arg(0),
			CompetitorURL: fs.Arg(1),
			AnalysisType:  strings.TrimSpace(*analysisType),
		}
	}

	body, err := core.Request("POST", "/competitor-analysis", nil, payload)
	if err != nil {
		return err
	}

	result := decodeMap(body)
	report := cliapp.ListReport{
		Summary:        []string{"Competitor analysis"},
		ResultsHeading: "Comparison",
		Results:        support.MapRows(result),
		RetrievalHints: []string{
			fmt.Sprintf("%s competitors %s %s", support.CLIName,
				placeholder(fs.Arg(0), "<your-url>"),
				placeholder(fs.Arg(1), "<competitor-url>")),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// decodeMap unwraps the optional {success, data} envelope and decodes the
// inner payload as a generic map for human-friendly rendering. If decoding
// fails the caller gets an empty map, which MapRows renders as
// "(empty payload)" — the --json mode still emits the full upstream body.
func decodeMap(body []byte) map[string]interface{} {
	result := map[string]interface{}{}
	if err := support.Decode(body, &result); err != nil {
		return map[string]interface{}{}
	}
	return result
}

func auditSummary(result map[string]interface{}, url string) []string {
	label := placeholder(url, "(from body-file)")
	summary := []string{fmt.Sprintf("SEO audit for: %s", label)}
	if status, ok := result["status"].(string); ok && status != "" {
		summary = append(summary, fmt.Sprintf("Status: %s", status))
	}
	if score, ok := result["score"].(float64); ok {
		summary = append(summary, fmt.Sprintf("Score: %d", int64(score)))
	}
	return summary
}

func placeholder(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
