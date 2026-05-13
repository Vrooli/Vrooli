package metrics

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"

	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/metrics"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/metrics/metrics_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"web-console/cli/internal/support"
)

// Register exposes `web-console metrics` as a flat read-only command
// over MetricsService.Get.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Metrics",
		Commands: []cliapp.Command{
			{
				Name:        "metrics",
				Description: "Show runtime metrics (active sessions, totals, TTS/voice counters)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return run(core, args) },
			},
		},
	}
}

func newClient(core *cliapp.ScenarioApp) metricsconnect.MetricsServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return metricsconnect.NewMetricsServiceClient(httpClient, baseURL)
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("metrics")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	resp, err := newClient(core).Get(context.Background(), connect.NewRequest(&metricsv1.GetRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("metrics get", err, nil)
	}

	report := cliapp.ListReport{
		Summary:        []string{"Runtime metrics"},
		ResultsHeading: "Values",
		Results:        metricRows(resp.Msg),
		RetrievalHints: []string{fmt.Sprintf("%s events", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func metricRows(m *metricsv1.GetResponse) []string {
	s, c, msg, r, rec := m.GetSessions(), m.GetConnections(), m.GetMessages(), m.GetReattach(), m.GetRecovery()
	return []string{
		fmt.Sprintf("uptime: %s", m.GetUptime()),
		fmt.Sprintf("sessions: created=%d deleted=%d active=%d resizes=%d", s.GetCreated(), s.GetDeleted(), s.GetActive(), s.GetResizes()),
		fmt.Sprintf("connections: total=%d active=%d", c.GetTotal(), c.GetActive()),
		fmt.Sprintf("messages: sent=%d received=%d", msg.GetSent(), msg.GetReceived()),
		fmt.Sprintf("reattach: attempts=%d successes=%d failures=%d", r.GetAttempts(), r.GetSuccesses(), r.GetFailures()),
		fmt.Sprintf("recovery: recovered=%d orphaned_meta=%d orphaned_tmux=%d attach_retries=%d preserved=%d", rec.GetRecovered(), rec.GetOrphanedMetadata(), rec.GetOrphanedTmux(), rec.GetAttachRetries(), rec.GetPreservedForFutureRecovery()),
		fmt.Sprintf("ai: generations=%d suggestions=%d", m.GetAiGenerations(), m.GetAiSuggestions()),
		fmt.Sprintf("stdin_before_ready_total: %d", m.GetStdinBeforeReadyTotal()),
		fmt.Sprintf("voice_skip_verification_total: %d", m.GetVoiceSkipVerificationTotal()),
	}
}
