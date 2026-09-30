package reports

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	reportspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/reports"
	reportsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/reports/reportsconnect"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"

	"system-monitor/cli/internal/support"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "reports",
		Description: "Generate and inspect system health reports",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "generate", Description: "Generate a daily or weekly report", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "type", Required: true, Description: "Report type: daily or weekly"}}}, RunCtx: h.generate},
			{Name: "list", Description: "List generated reports", RunCtx: h.list},
			{Name: "get", Description: "Get a report by ID", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "Report ID"}}}, RunCtx: h.get},
		},
	}
}

type handlers struct {
	client reportsconnect.ReportsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client: reportsconnect.NewReportsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) generate(ctx cliapp.RunContext) error {
	reportType := strings.ToLower(strings.TrimSpace(ctx.Positional("type")))
	if reportType != "daily" && reportType != "weekly" {
		return fmt.Errorf("report type must be daily or weekly")
	}

	resp, err := h.client.GenerateReport(context.Background(), connect.NewRequest(&reportspb.GenerateReportRequest{Type: reportType}))
	if err != nil {
		return cliapp.WrapAPIError("generate report", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetReport() == nil {
		return fmt.Errorf("server returned no generated report")
	}
	return renderReport(ctx, resp.Msg, resp.Msg.GetReport(), true)
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListReports(context.Background(), connect.NewRequest(&reportspb.ListReportsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list reports", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no reports list")
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Reports available: %d", resp.Msg.GetCount()),
		},
		ResultsHeading: "Reports",
		Results:        reportRows(resp.Msg.GetReports()),
		RetrievalHints: []string{"system-monitor reports generate daily", "system-monitor reports get <id>"},
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, report)
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := strings.TrimSpace(ctx.Positional("id"))
	resp, err := h.client.GetReport(context.Background(), connect.NewRequest(&reportspb.GetReportRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get report %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetReport() == nil {
		return fmt.Errorf("server returned no report")
	}
	return renderReport(ctx, resp.Msg, resp.Msg.GetReport(), false)
}

func renderReport(ctx cliapp.RunContext, payload proto.Message, report *reportspb.EnhancedSystemReport, generated bool) error {
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), payload)
	}

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

	return ctx.RenderOperational(cliapp.OperationalReport{
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

func reportRows(items []*reportspb.EnhancedSystemReport) []string {
	if len(items) == 0 {
		return []string{"No reports have been generated yet."}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("%s type=%s generated=%s health=%s metrics=%d alerts=%d investigations=%d", item.GetReportId(), item.GetReportType(), support.FormatTimestamp(item.GetGeneratedAt()), item.GetExecutiveSummary().GetOverallHealth(), item.GetMetricsCount(), item.GetAlertsCount(), item.GetInvestigationsCount()))
	}
	return rows
}
