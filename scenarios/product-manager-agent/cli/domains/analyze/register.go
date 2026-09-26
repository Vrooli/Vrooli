package analyze

import (
	"fmt"
	"os"
	"strings"

	"product-manager-agent/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `analyze` subcommand group wrapping the five AI-backed
// analysis endpoints. Each takes a JSON body via `--body-file` because payload
// shapes vary (product name, competitor name, feedback items, feature, decision).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "analyze",
		Description: "AI-backed analyses: market, competitor, feedback, ROI, decision",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "market", Description: "Run a market analysis (--body-file PATH)", Run: func(args []string) error { return runMarket(core, args) }},
			{Name: "competitor", Description: "Run a competitor analysis (--body-file PATH)", Run: func(args []string) error { return runCompetitor(core, args) }},
			{Name: "feedback", Description: "Analyze user feedback (--body-file PATH)", Run: func(args []string) error { return runFeedback(core, args) }},
			{Name: "roi", Description: "Compute ROI for a feature (--body-file PATH)", Run: func(args []string) error { return runROI(core, args) }},
			{Name: "decision", Description: "Analyze a product decision (--body-file PATH)", Run: func(args []string) error { return runDecision(core, args) }},
		},
	}
}

func runMarket(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analyze market")
	bodyFile := fs.String("body-file", "", `Path to JSON body: {"product_name":"..."}`)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/market/analyze", nil, payload)
	if err != nil {
		return err
	}
	var m support.MarketAnalysis
	if err := support.Decode(body, &m); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Product: %s", m.ProductName),
		fmt.Sprintf("Market size: %s", fallback(m.MarketSize, "unknown")),
		fmt.Sprintf("Growth rate: %s", fallback(m.GrowthRate, "unknown")),
		fmt.Sprintf("Demographics: %s", fallback(m.Demographics, "unknown")),
		fmt.Sprintf("Competitors: %s", stringList(m.Competitors)),
		fmt.Sprintf("Opportunities: %s", stringList(m.Opportunities)),
		fmt.Sprintf("Challenges: %s", stringList(m.Challenges)),
		fmt.Sprintf("Analyzed at: %s", support.FormatTimeValue(m.Timestamp)),
	}
	return renderList("Market analysis", results, *jsonOutput)
}

func runCompetitor(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analyze competitor")
	bodyFile := fs.String("body-file", "", `Path to JSON body: {"competitor_name":"...","depth":"standard"}`)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/competitor/analyze", nil, payload)
	if err != nil {
		return err
	}
	var c support.CompetitorAnalysis
	if err := support.Decode(body, &c); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Competitor: %s", c.CompetitorName),
		fmt.Sprintf("Pricing: %s", fallback(c.Pricing, "unknown")),
		fmt.Sprintf("Target market: %s", fallback(c.TargetMarket, "unknown")),
		fmt.Sprintf("Market share: %s", fallback(c.MarketShare, "unknown")),
		fmt.Sprintf("Features: %s", stringList(c.Features)),
		fmt.Sprintf("Strengths: %s", stringList(c.Strengths)),
		fmt.Sprintf("Weaknesses: %s", stringList(c.Weaknesses)),
		fmt.Sprintf("Analyzed at: %s", support.FormatTimeValue(c.AnalyzedAt)),
	}
	return renderList("Competitor analysis", results, *jsonOutput)
}

func runFeedback(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analyze feedback")
	bodyFile := fs.String("body-file", "", `Path to JSON body: {"feedback_items":[...]}`)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/feedback/analyze", nil, payload)
	if err != nil {
		return err
	}
	var fa support.FeedbackAnalysis
	if err := support.Decode(body, &fa); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Items analyzed: %d", fa.TotalItems),
		fmt.Sprintf("Sentiment: %s (score %.2f)", fallback(fa.Sentiment, "unknown"), fa.SentimentScore),
		fmt.Sprintf("Key themes: %s", stringList(fa.KeyThemes)),
		fmt.Sprintf("Feature requests: %s", stringList(fa.FeatureRequests)),
		fmt.Sprintf("Pain points: %s", stringList(fa.PainPoints)),
		fmt.Sprintf("Analyzed at: %s", support.FormatTimeValue(fa.AnalyzedAt)),
	}
	return renderList("Feedback analysis", results, *jsonOutput)
}

func runROI(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analyze roi")
	bodyFile := fs.String("body-file", "", "Path to JSON body: a Feature object")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/roi/calculate", nil, payload)
	if err != nil {
		return err
	}
	var r support.ROICalculation
	if err := support.Decode(body, &r); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Feature: %s", fallback(r.FeatureID, "(unnamed)")),
		fmt.Sprintf("Revenue impact: $%.2f", r.RevenueImpact),
		fmt.Sprintf("Cost estimate: $%.2f", r.CostEstimate),
		fmt.Sprintf("ROI: %.2f%%", r.ROI),
		fmt.Sprintf("Payback period: %.2f months", r.PaybackPeriod),
		fmt.Sprintf("Assumptions: %s", stringList(r.Assumptions)),
		fmt.Sprintf("Calculated at: %s", support.FormatTimeValue(r.CalculatedAt)),
	}
	return renderList("ROI calculation", results, *jsonOutput)
}

func runDecision(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analyze decision")
	bodyFile := fs.String("body-file", "", "Path to JSON body: a Decision object with options")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/decision/analyze", nil, payload)
	if err != nil {
		return err
	}
	var d support.DecisionAnalysis
	if err := support.Decode(body, &d); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Decision ID: %s", fallback(d.DecisionID, "(unnamed)")),
		fmt.Sprintf("Recommendation: %s", fallback(d.Recommendation, "(none)")),
		fmt.Sprintf("Analyzed at: %s", support.FormatTimeValue(d.AnalyzedAt)),
		fmt.Sprintf("Options: %d", len(d.Options)),
	}
	for _, opt := range d.Options {
		results = append(results, fmt.Sprintf("  - %s | risk=%s complexity=%s timeline=%s success=%.2f score=%.2f",
			opt.Name, fallback(opt.RiskLevel, "?"), fallback(opt.Complexity, "?"),
			fallback(opt.Timeline, "?"), opt.SuccessProbability, opt.Score))
	}
	return renderList("Decision analysis", results, *jsonOutput)
}

func renderList(heading string, results []string, asJSON bool) error {
	report := cliapp.ListReport{
		Summary:        []string{heading},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s dashboard", support.CLIName)},
	}
	if asJSON {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func fallback(value, def string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return value
}

func stringList(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}
