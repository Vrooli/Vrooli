package signals

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	signalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals"
	signalsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals/signals_v1connect"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client signalsconnect.SignalsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: signalsconnect.NewSignalsServiceClient(httpClient, baseURL),
	}
}

// score runs every enabled signal against a file's chunk and renders the
// aggregated verdict plus per-signal scores.
func (h *handlers) score(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	fileID := ctx.Positional("file_id")
	resp, err := h.client.ScoreChunk(context.Background(), connect.NewRequest(&signalsv1.ScoreChunkRequest{
		Scenario: scenario,
		FileId:   fileID,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("score %q in %q", fileID, scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetVerdict() == nil {
		return fmt.Errorf("server returned no verdict")
	}
	return renderVerdict(ctx, resp.Msg, resp.Msg.GetVerdict(), false)
}

// explain re-runs scoring and renders the verdict with the full
// per-signal evidence breakdown (the explainability contract).
func (h *handlers) explain(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	fileID := ctx.Positional("file_id")
	resp, err := h.client.ExplainVerdict(context.Background(), connect.NewRequest(&signalsv1.ExplainVerdictRequest{
		Scenario: scenario,
		FileId:   fileID,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("explain %q in %q", fileID, scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetVerdict() == nil {
		return fmt.Errorf("server returned no verdict")
	}
	return renderVerdict(ctx, resp.Msg, resp.Msg.GetVerdict(), true)
}

// list returns the registered signals with default weights and current
// enable/disable state (manifest-overlaid when --scenario is supplied).
func (h *handlers) list(ctx cliapp.RunContext) error {
	scenario := ctx.Flag("scenario")
	resp, err := h.client.ListSignals(context.Background(), connect.NewRequest(&signalsv1.ListSignalsRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError("list signals", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no signals response")
	}
	results := make([]string, 0, len(resp.Msg.GetSignals()))
	for _, s := range resp.Msg.GetSignals() {
		state := "enabled"
		if s.GetDisabled() {
			state = "disabled"
			if reason := s.GetDisabledReason(); reason != "" {
				state = fmt.Sprintf("disabled (%s)", reason)
			}
		}
		results = append(results, fmt.Sprintf("%s — weight=%.2f (%s) [%s] — %s",
			s.GetName(), s.GetDefaultWeight(), s.GetStability(), state, s.GetDescription()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d registered signal(s).", len(resp.Msg.GetSignals()))},
		ResultsHeading: "Signals",
		Results:        results,
	})
}

// renderVerdict is the shared rendering path for score + explain. When
// withEvidence is true (explain), each per-signal score expands its
// Evidence entries. --json consumers get the proto-typed response.
func renderVerdict(ctx cliapp.RunContext, payload proto.Message, v *signalsv1.Verdict, withEvidence bool) error {
	tieNote := ""
	if v.GetTied() {
		tieNote = " (TIED)"
	}
	summary := []string{
		fmt.Sprintf("%s → tier=%s top=%s (%.3f) runner-up=%s (%.3f)%s",
			v.GetChunkPath(), tierName(v.GetTier()),
			v.GetTopDomain(), v.GetTopValue(),
			v.GetRunnerUpDomain(), v.GetRunnerUpValue(), tieNote),
	}

	var results []string
	for _, dv := range v.GetDomainValues() {
		results = append(results, fmt.Sprintf("domain %s = %.3f", dv.GetDomain(), dv.GetValue()))
	}
	for _, s := range v.GetScores() {
		results = append(results, fmt.Sprintf("signal %s → %s = %.3f (%s)",
			s.GetSignal(), s.GetDomain(), s.GetValue(), s.GetReason()))
		if withEvidence {
			for _, e := range s.GetEvidence() {
				loc := ""
				if e.GetLocator() != "" {
					loc = " @ " + e.GetLocator()
				}
				results = append(results, fmt.Sprintf("    evidence[%s] w=%.2f: %s%s",
					e.GetKind(), e.GetWeight(), e.GetSummary(), loc))
			}
		}
	}

	heading := "Aggregated domain values + per-signal scores"
	if withEvidence {
		heading = "Aggregated values, per-signal scores, and evidence"
	}
	return cliapp.RenderProtoList(ctx, payload, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: heading,
		Results:        results,
		RetrievalHints: []string{
			"`signals explain <scenario> <file_id>` for the full per-signal evidence breakdown.",
		},
	})
}

func tierName(t signalsv1.Tier) string {
	switch t {
	case signalsv1.Tier_TIER_AUTO_PLACE:
		return "auto_place"
	case signalsv1.Tier_TIER_SUGGEST:
		return "suggest"
	case signalsv1.Tier_TIER_CONFLICT:
		return "conflict"
	default:
		return "unspecified"
	}
}
