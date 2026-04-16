package smells

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"code-smell/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes the six code-smell verbs as flat commands. Every command is
// a thin wrapper over `/api/v1/code-smell/*`: the API is the source of truth
// for analysis, fix application, queueing, statistics, rule configuration, and
// pattern learning. Commands that need complex nested payloads take
// `--body-file PATH` rather than hand-assembling JSON in Go.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Code Smell",
		Commands: []cliapp.Command{
			{
				Name:        "analyze",
				Description: "Analyze files/directories for code smells",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runAnalyze(core, args) },
			},
			{
				Name:        "fix",
				Description: "Apply or reject a specific fix",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runFix(core, args) },
			},
			{
				Name:        "queue",
				Description: "Show violations awaiting review",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runQueue(core, args) },
			},
			{
				Name:        "stats",
				Description: "Show code smell statistics",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runStats(core, args) },
			},
			{
				Name:        "rules",
				Description: "List smell detection rules",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runRules(core, args) },
			},
			{
				Name:        "learn",
				Description: "Submit a pattern for learning (--body-file PATH required)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runLearn(core, args) },
			},
		},
	}
}

func runAnalyze(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analyze")
	bodyFile := fs.String("body-file", "", "Path to analyze request JSON (overrides positional/flags)")
	rules := fs.String("rules", "", "Comma-separated rule ids to apply")
	autoFix := fs.Bool("auto-fix", false, "Automatically apply safe fixes")
	risk := fs.String("risk", "", "Maximum risk threshold (safe|moderate|dangerous)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	switch {
	case strings.TrimSpace(*bodyFile) != "":
		loaded, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = loaded
	default:
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: analyze <path> [--rules a,b] [--auto-fix] [--risk LEVEL] | --body-file PATH")
		}
		paths := append([]string{}, fs.Args()...)
		req := map[string]interface{}{
			"paths":    paths,
			"auto_fix": *autoFix,
		}
		if r := strings.TrimSpace(*rules); r != "" {
			list := splitCSV(r)
			if len(list) > 0 {
				req["rules"] = list
			}
		}
		if rl := strings.TrimSpace(*risk); rl != "" {
			req["risk_threshold"] = rl
		}
		payload = req
	}

	body, err := core.Request("POST", "/code-smell/analyze", nil, payload)
	if err != nil {
		return err
	}
	var resp support.AnalyzeResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Files analyzed: %d", resp.TotalFiles),
		fmt.Sprintf("Violations: %d", len(resp.Violations)),
		fmt.Sprintf("Auto-fixed: %d", resp.AutoFixed),
		fmt.Sprintf("Needs review: %d", resp.NeedsReview),
		fmt.Sprintf("Duration: %dms", resp.DurationMs),
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Violations",
		Results:        violationRows(resp.Violations),
		RetrievalHints: []string{
			fmt.Sprintf("%s queue", support.CLIName),
			fmt.Sprintf("%s fix <violation-id> --action approve", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runFix(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("fix")
	action := fs.String("action", "approve", "Action to take: approve|reject|ignore")
	bodyFile := fs.String("body-file", "", "Path to fix request JSON (overrides flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	var violationID string
	switch {
	case strings.TrimSpace(*bodyFile) != "":
		loaded, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = loaded
		// Best-effort: pull the violation id out of the body for reporting.
		var peek struct {
			ViolationID string `json:"violation_id"`
		}
		_ = json.Unmarshal(loaded, &peek)
		violationID = peek.ViolationID
	default:
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: fix <violation-id> [--action approve|reject|ignore] | --body-file PATH")
		}
		violationID = fs.Arg(0)
		payload = map[string]interface{}{
			"violation_id": violationID,
			"action":       strings.TrimSpace(*action),
		}
	}

	body, err := core.Request("POST", "/code-smell/fix", nil, payload)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Fix %s processed for %s", strings.TrimSpace(*action), violationID)
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: support.MapRows(result),
		NextCommand: []string{
			fmt.Sprintf("%s queue", support.CLIName),
			fmt.Sprintf("%s stats", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runQueue(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("queue")
	severity := fs.String("severity", "", "Filter by severity (error|warning|info)")
	file := fs.String("file", "", "Filter by file pattern")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"severity": *severity,
		"file":     *file,
	})
	body, err := core.Get("/code-smell/queue", query)
	if err != nil {
		return err
	}
	var resp support.QueueResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Violations awaiting review: %d", resp.Total)}
	if len(resp.BySeverity) > 0 {
		summary = append(summary,
			fmt.Sprintf("Errors: %d", resp.BySeverity["error"]),
			fmt.Sprintf("Warnings: %d", resp.BySeverity["warning"]),
			fmt.Sprintf("Info: %d", resp.BySeverity["info"]),
		)
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Queue",
		Results:        violationRows(resp.Violations),
		RetrievalHints: []string{
			fmt.Sprintf("%s fix <violation-id> --action approve", support.CLIName),
			fmt.Sprintf("%s queue --severity error", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runStats(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("stats")
	period := fs.String("period", "", "Time period filter (day|week|month|all)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{"period": *period})
	body, err := core.Get("/code-smell/stats", query)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	summary := []string{"Code smell statistics"}
	if p, ok := data["period"].(string); ok && p != "" {
		summary = append(summary, fmt.Sprintf("Period: %s", p))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Metrics",
		Results:        support.MapRows(data),
		RetrievalHints: []string{
			fmt.Sprintf("%s queue", support.CLIName),
			fmt.Sprintf("%s rules", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRules(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("rules")
	category := fs.String("category", "", "Filter by category")
	vrooli := fs.Bool("vrooli", false, "Show only Vrooli-specific rules")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{"category": *category})
	if *vrooli {
		query.Set("vrooli", "true")
	}
	body, err := core.Get("/code-smell/rules", query)
	if err != nil {
		return err
	}
	var resp support.RulesResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Rules: %d", len(resp.Rules)),
		fmt.Sprintf("Vrooli-specific: %d", resp.VrooliSpecificCount),
	}
	if len(resp.Categories) > 0 {
		cats := append([]string{}, resp.Categories...)
		sort.Strings(cats)
		summary = append(summary, fmt.Sprintf("Categories: %s", strings.Join(cats, ", ")))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Rules",
		Results:        ruleRows(resp.Rules),
		RetrievalHints: []string{
			fmt.Sprintf("%s rules --vrooli", support.CLIName),
			fmt.Sprintf("%s analyze <path> --rules <rule-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runLearn(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("learn")
	bodyFile := fs.String("body-file", "", "Path to learn request JSON (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/code-smell/learn", nil, payload)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = "Pattern submitted for learning"
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     support.MapRows(data),
		NextCommand: []string{fmt.Sprintf("%s stats", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func violationRows(violations []support.Violation) []string {
	if len(violations) == 0 {
		return []string{"(no violations)"}
	}
	rows := make([]string, 0, len(violations))
	for _, v := range violations {
		rows = append(rows, fmt.Sprintf("%s | %s:%d | %s | %s | %s",
			support.ShortID(v.ID),
			v.FilePath,
			v.LineNumber,
			v.Severity,
			v.RuleName,
			truncate(v.Message, 80),
		))
	}
	return rows
}

func ruleRows(rules []support.Rule) []string {
	if len(rules) == 0 {
		return []string{"(no rules)"}
	}
	rows := make([]string, 0, len(rules))
	for _, r := range rules {
		vrooli := ""
		if r.VrooliSpecific {
			vrooli = " [vrooli]"
		}
		enabled := ""
		if !r.Enabled {
			enabled = " [disabled]"
		}
		rows = append(rows, fmt.Sprintf("%s | %s | category=%s | risk=%s%s%s",
			r.ID, r.Name, r.Category, r.RiskLevel, vrooli, enabled))
	}
	return rows
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
