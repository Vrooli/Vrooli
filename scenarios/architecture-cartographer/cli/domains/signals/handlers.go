package signals

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
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
// aggregated verdict plus per-signal scores. The <file> positional is
// either a snapshot file id (starts with "file:") or a repo-relative
// path; the CLI routes to the matching API field and the service
// resolves either form to a chunk.
func (h *handlers) score(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	fileID, repoPath := splitFileArg(ctx.Positional("file"))
	resp, err := h.client.ScoreChunk(context.Background(), connect.NewRequest(&signalsv1.ScoreChunkRequest{
		Scenario: scenario,
		FileId:   fileID,
		RepoPath: repoPath,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("score %q in %q", ctx.Positional("file"), scenario), err, nil)
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
	fileID, repoPath := splitFileArg(ctx.Positional("file"))
	resp, err := h.client.ExplainVerdict(context.Background(), connect.NewRequest(&signalsv1.ExplainVerdictRequest{
		Scenario: scenario,
		FileId:   fileID,
		RepoPath: repoPath,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("explain %q in %q", ctx.Positional("file"), scenario), err, nil)
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

// boundaries renders per-domain coupling/boundary-health scores.
func (h *handlers) boundaries(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.BoundaryHealth(context.Background(), connect.NewRequest(&signalsv1.BoundaryHealthRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("boundary health for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no boundary-health report")
	}
	results := make([]string, 0, len(resp.Msg.GetDomains()))
	for _, d := range resp.Msg.GetDomains() {
		kernel := ""
		if d.GetStableKernel() {
			kernel = " [stable-kernel]"
		}
		line := fmt.Sprintf("%s — health=%.2f Ce=%d Ca=%d I=%.2f fan_out=%.2f%s",
			d.GetDomain(), d.GetHealthScore(), d.GetEfferent(), d.GetAfferent(), d.GetInstability(), d.GetFanOut(), kernel)
		for _, s := range d.GetSmells() {
			line += fmt.Sprintf("\n      [%s] %s — %s", couplingSeverityName(s.GetSeverity()), s.GetKind(), s.GetMessage())
		}
		results = append(results, line)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Boundary health for %q (%d domains).", scenario, resp.Msg.GetTotalDomains())},
		ResultsHeading: "Per-domain coupling",
		Results:        results,
		RetrievalHints: []string{"Health is graded [0,1]; smells are advisory. Stable kernels (shared substrate) are exempt from the god-domain smell."},
	})
}

func couplingSeverityName(s signalsv1.CouplingSeverity) string {
	switch s {
	case signalsv1.CouplingSeverity_COUPLING_SEVERITY_WARN:
		return "warn"
	case signalsv1.CouplingSeverity_COUPLING_SEVERITY_INFO:
		return "info"
	default:
		return "unspecified"
	}
}

// renderVerdict is the shared rendering path for score + explain. When
// withEvidence is true (explain), each per-signal score expands its
// Evidence entries. --json consumers get the proto-typed response.
func renderVerdict(ctx cliapp.RunContext, payload proto.Message, v *sharedv1.Verdict, withEvidence bool) error {
	tieNote := ""
	if v.GetTied() {
		tieNote = " (TIED)"
	}
	// Hide blank runner-up: only render the runner-up clause when a real
	// runner-up domain is present (avoids `runner-up= (0.000)` cosmetic
	// noise when only one domain has any signal weight).
	summaryLine := fmt.Sprintf("%s → tier=%s top=%s (%.3f)",
		v.GetChunkPath(), tierName(v.GetTier()),
		v.GetTopDomain(), v.GetTopValue())
	if v.GetRunnerUpDomain() != "" {
		summaryLine += fmt.Sprintf(" runner-up=%s (%.3f)", v.GetRunnerUpDomain(), v.GetRunnerUpValue())
	}
	summaryLine += tieNote
	summary := []string{summaryLine}

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
	if abst := v.GetAbstentions(); len(abst) > 0 {
		results = append(results, "Abstentions:")
		for _, a := range abst {
			results = append(results, fmt.Sprintf("  signal %s — %s", a.GetSignal(), a.GetReason()))
			if withEvidence {
				for _, e := range a.GetEvidence() {
					loc := ""
					if e.GetLocator() != "" {
						loc = " @ " + e.GetLocator()
					}
					results = append(results, fmt.Sprintf("      evidence[%s]: %s%s",
						e.GetKind(), e.GetSummary(), loc))
				}
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
			"`signals explain <scenario> <file>` for the full per-signal evidence breakdown (<file> = snapshot file id or repo-relative path).",
		},
	})
}

// splitFileArg routes the <file> positional to one of the two API
// fields. Snapshot file ids are emitted by the graph extractor as
// "file:<path>", so a leading "file:" prefix is the unambiguous
// discriminator. Anything else is treated as a repo-relative path.
func splitFileArg(arg string) (fileID, repoPath string) {
	if strings.HasPrefix(arg, "file:") {
		return arg, ""
	}
	return "", arg
}

func tierName(t sharedv1.Tier) string {
	switch t {
	case sharedv1.Tier_TIER_AUTO_PLACE:
		return "auto_place"
	case sharedv1.Tier_TIER_SUGGEST:
		return "suggest"
	case sharedv1.Tier_TIER_CONFLICT:
		return "conflict"
	default:
		return "unspecified"
	}
}
