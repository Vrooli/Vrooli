package reports

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/domain"

	"system-monitor/cli/internal/support"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "reports",
		Description: "Generate and inspect system health reports",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "generate", Description: "Generate a daily or weekly report", Run: func(args []string) error { return runGenerate(core, args) }},
			{Name: "list", Description: "List generated reports", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Description: "Get a report by ID", Run: func(args []string) error { return runGet(core, args) }},
		},
	}
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("reports generate")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: system-monitor reports generate <daily|weekly>")
	}
	reportType := strings.ToLower(strings.TrimSpace(fs.Arg(0)))
	if reportType != "daily" && reportType != "weekly" {
		return fmt.Errorf("report type must be daily or weekly")
	}

	body, err := core.Request("POST", "/reports/generate", nil, &apipb.GenerateReportRequest{Type: reportType})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var report domainpb.EnhancedSystemReport
	if err := support.DecodeProto(body, &report); err != nil {
		return err
	}
	return renderReport(os.Stdout, &report, true)
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("reports list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/reports", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var response apipb.ListReportsResponse
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Reports available: %d", response.GetCount()),
		},
		ResultsHeading: "Reports",
		Results:        reportRows(response.GetReports()),
		RetrievalHints: []string{"system-monitor reports generate daily", "system-monitor reports get <id>"},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("reports get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: system-monitor reports get <id>")
	}

	body, err := core.Get("/reports/"+strings.TrimSpace(fs.Arg(0)), nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var report domainpb.EnhancedSystemReport
	if err := support.DecodeProto(body, &report); err != nil {
		return err
	}
	return renderReport(os.Stdout, &report, false)
}

func renderReport(stdout *os.File, report *domainpb.EnhancedSystemReport, generated bool) error {
	status := []string{
		fmt.Sprintf("Report ID: %s", report.GetReportId()),
		fmt.Sprintf("Report type: %s", report.GetReportType()),
		fmt.Sprintf("Generated at: %s", support.FormatTimestamp(report.GetGeneratedAt())),
		fmt.Sprintf("Overall health: %s", report.GetExecutiveSummary().GetOverallHealth()),
	}
	if generated {
		status = append(status, "Report generation completed successfully.")
	}

	highlights := append([]string{}, report.GetHighlights()...)
	if len(highlights) == 0 {
		highlights = []string{"No highlights were included in this report."}
	}
	recommendations := append([]string{}, report.GetRecommendations()...)
	if len(recommendations) == 0 {
		recommendations = []string{"No recommendations were included in this report."}
	}

	return cliapp.RenderOperationalReport(stdout, cliapp.OperationalReport{
		Status: status,
		Triage: []cliapp.TriageGroup{
			{Heading: "Highlights", Items: highlights},
			{
				Heading: "Counts",
				Items: []string{
					fmt.Sprintf("Metrics analyzed: %d", report.GetMetricsCount()),
					fmt.Sprintf("Alerts analyzed: %d", report.GetAlertsCount()),
					fmt.Sprintf("Investigations analyzed: %d", report.GetInvestigationsCount()),
				},
			},
			{Heading: "Recommendations", Items: recommendations},
		},
		NextSteps: []string{
			"system-monitor reports list",
			fmt.Sprintf("system-monitor reports get %s --json", report.GetReportId()),
		},
	})
}

func reportRows(items []*domainpb.EnhancedSystemReport) []string {
	if len(items) == 0 {
		return []string{"No reports have been generated yet."}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("%s type=%s generated=%s health=%s metrics=%d alerts=%d investigations=%d", item.GetReportId(), item.GetReportType(), support.FormatTimestamp(item.GetGeneratedAt()), item.GetExecutiveSummary().GetOverallHealth(), item.GetMetricsCount(), item.GetAlertsCount(), item.GetInvestigationsCount()))
	}
	return rows
}
