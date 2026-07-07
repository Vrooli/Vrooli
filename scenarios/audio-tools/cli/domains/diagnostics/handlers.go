package diagnostics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
	diagconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics/diagnostics_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client diagconnect.DiagnosticsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: diagconnect.NewDiagnosticsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	caps, err := parseCapabilities(ctx.Flag("capability"))
	if err != nil {
		return err
	}
	resp, err := h.client.RunSuite(context.Background(), connect.NewRequest(&diagv1.RunSuiteRequest{
		Capabilities: caps,
	}))
	if err != nil {
		return cliapp.WrapAPIError("diagnostics run", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, mutationReportForRun(resp.Msg.GetRun()))
}

func (h *handlers) last(ctx cliapp.RunContext) error {
	resp, err := h.client.GetLastRun(context.Background(), connect.NewRequest(&diagv1.GetLastRunRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("diagnostics last", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, mutationReportForRun(resp.Msg.GetRun()))
}

func parseCapabilities(raw string) ([]diagv1.Capability, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]diagv1.Capability, 0, len(parts))
	for _, p := range parts {
		switch strings.ToLower(strings.TrimSpace(p)) {
		case "":
			continue
		case "stt":
			out = append(out, diagv1.Capability_CAPABILITY_STT)
		case "tts":
			out = append(out, diagv1.Capability_CAPABILITY_TTS)
		case "summarize":
			out = append(out, diagv1.Capability_CAPABILITY_SUMMARIZE)
		case "transcode":
			out = append(out, diagv1.Capability_CAPABILITY_TRANSCODE)
		default:
			return nil, fmt.Errorf("unknown capability %q (allowed: stt,tts,summarize,transcode)", p)
		}
	}
	return out, nil
}

func mutationReportForRun(run *diagv1.RunSuiteResult) cliapp.MutationReport {
	if run == nil || run.GetRunId() == "" {
		return cliapp.MutationReport{
			Result: []string{"Overall: NEVER (no suite has executed since the API started)"},
			NextCommand: []string{
				"audio-tools diagnostics run",
			},
		}
	}
	overall := run.GetOverall()
	rep := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Overall: %s  (%d pass / %d fail / %d total)  run_id=%s  duration=%dms",
				overallLabel(overall.GetStatus()),
				overall.GetPassCount(), overall.GetFailCount(), overall.GetTotalCount(),
				run.GetRunId(),
				run.GetFinishedAtUnixMs()-run.GetStartedAtUnixMs(),
			),
		},
	}
	for _, s := range run.GetSteps() {
		rep.Changes = append(rep.Changes, formatStep(s))
	}
	rep.NextCommand = []string{
		"audio-tools diagnostics last         # re-print the same envelope",
		"audio-tools diagnostics run --json   # machine-readable proto JSON",
	}
	return rep
}

func formatStep(s *diagv1.SuiteStepResult) string {
	cap := capabilityLabel(s.GetCapability())
	if s.GetOk() {
		line := fmt.Sprintf("%-10s OK     tier=%-7s provider=%-20s model=%-20s latency=%dms",
			cap, providerTierLabel(s.GetProviderTier()), truncate(s.GetProviderId(), 20),
			truncate(s.GetModelId(), 20), int(s.GetLatencyMs()))
		if s.GetCapability() == diagv1.Capability_CAPABILITY_STT {
			line += " " + sttQualitySummary(s.GetDetails())
		}
		return line
	}
	line := fmt.Sprintf("%-10s FAIL   code=%-22s message=%s",
		cap, s.GetErrorCode(), truncate(s.GetErrorMessage(), 80))
	if s.GetCapability() == diagv1.Capability_CAPABILITY_STT && s.GetDetails()["quality_assessed"] == "true" {
		// A quality-smoke failure flips the STT step, but readiness (provider
		// reachability) stayed its own signal — surface it so operators know
		// the provider is reachable and the fault is transcript safety.
		line += "\n           readiness=pass " + sttQualitySummary(s.GetDetails())
	}
	return line
}

// cliFixtureResult is the CLI-side view of one quality-smoke fixture,
// decoded from the opaque quality_fixtures detail JSON so human output can
// summarize per-fixture verdicts without the operator parsing prose.
type cliFixtureResult struct {
	FixtureID             string  `json:"fixture_id"`
	ExpectedKind          string  `json:"expected_kind"`
	Status                string  `json:"status"`
	WER                   float64 `json:"wer"`
	WERThreshold          float64 `json:"wer_threshold"`
	Filtered              bool    `json:"transcript_filtered"`
	HallucinationDetected bool    `json:"hallucination_detected"`
}

// sttQualitySummary renders the layer-2 quality-smoke verdict for the STT
// step: "quality=not-assessed" when readiness ran alone, otherwise the
// aggregate status plus a compact per-fixture breakdown.
func sttQualitySummary(details map[string]string) string {
	if details["quality_assessed"] != "true" {
		return "quality=not-assessed"
	}
	summary := "quality=" + orDash(details["quality_status"])
	var fixtures []cliFixtureResult
	if raw := details["quality_fixtures"]; raw != "" {
		_ = json.Unmarshal([]byte(raw), &fixtures)
	}
	parts := make([]string, 0, len(fixtures))
	for _, f := range fixtures {
		switch f.ExpectedKind {
		case "speech":
			parts = append(parts, fmt.Sprintf("%s:%s wer=%.2f/%.2f", f.FixtureID, f.Status, f.WER, f.WERThreshold))
		default:
			tag := f.Status
			if f.HallucinationDetected {
				tag += " hallucination"
			} else if f.Filtered {
				tag += " filtered"
			}
			parts = append(parts, fmt.Sprintf("%s:%s", f.FixtureID, tag))
		}
	}
	if len(parts) > 0 {
		summary += " [" + strings.Join(parts, ", ") + "]"
	}
	return summary
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func overallLabel(s diagv1.SuiteOverall_Status) string {
	switch s {
	case diagv1.SuiteOverall_STATUS_PASS:
		return "PASS"
	case diagv1.SuiteOverall_STATUS_PARTIAL:
		return "PARTIAL"
	case diagv1.SuiteOverall_STATUS_FAIL:
		return "FAIL"
	case diagv1.SuiteOverall_STATUS_NEVER:
		return "NEVER"
	}
	return "UNKNOWN"
}

func capabilityLabel(c diagv1.Capability) string {
	switch c {
	case diagv1.Capability_CAPABILITY_STT:
		return "stt"
	case diagv1.Capability_CAPABILITY_TTS:
		return "tts"
	case diagv1.Capability_CAPABILITY_SUMMARIZE:
		return "summarize"
	case diagv1.Capability_CAPABILITY_TRANSCODE:
		return "transcode"
	}
	return "unknown"
}

func providerTierLabel(t commonv1.ProviderTier) string {
	switch t {
	case commonv1.ProviderTier_PROVIDER_TIER_LOCAL:
		return "local"
	case commonv1.ProviderTier_PROVIDER_TIER_BYOK:
		return "byok"
	case commonv1.ProviderTier_PROVIDER_TIER_VROOLI:
		return "vrooli"
	}
	return "-"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
