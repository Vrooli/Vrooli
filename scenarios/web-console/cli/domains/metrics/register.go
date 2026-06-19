package metrics

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/metrics"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/metrics/metrics_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client metricsconnect.MetricsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: metricsconnect.NewMetricsServiceClient(httpClient, baseURL),
	}
}

// Register exposes `web-console metrics` as a flat read-only command over
// MetricsService.Get. Built from the embedded manifest; DefaultSubcommand
// preserves the flat invocation.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"MetricsService.Get": h.run,
	}
	group, err := cliapp.LoadFromManifest(manifest, "metrics", bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("metrics: load from manifest: %w", err)
	}
	group.DefaultSubcommand = "metrics"
	return group, nil
}

func (h *handlers) run(rc cliapp.RunContext) error {
	resp, err := h.client.Get(context.Background(), connect.NewRequest(&metricsv1.GetRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("metrics get", err, nil)
	}

	report := cliapp.ListReport{
		Summary:        []string{"Runtime metrics"},
		ResultsHeading: "Values",
		Results:        metricRows(resp.Msg),
		RetrievalHints: []string{fmt.Sprintf("%s events", support.CLIName)},
	}
	if rc.JSON() {
		return cliapp.PrintReportJSON(rc.Stdout(), report)
	}
	return cliapp.RenderListReport(rc.Stdout(), report)
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
