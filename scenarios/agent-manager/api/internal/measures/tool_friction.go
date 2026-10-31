package measures

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/toolfriction"
	"agent-manager/internal/usageattribution"

	"connectrpc.com/connect"
	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

const defaultToolRows = 50

// ToolFriction profiles every command path agents ran in the window.
func (h *Handler) ToolFriction(ctx context.Context, req *connect.Request[measurepb.ToolFrictionRequest]) (*connect.Response[measurepb.ToolFrictionResponse], error) {
	if h.tools == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("tool friction profiles are unavailable"))
	}
	window := req.Msg.GetWindow()
	if window == nil {
		window = &sharedmeasurepb.TimeWindow{Window: &sharedmeasurepb.TimeWindow_Token{Token: sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK}}
	}
	span, err := measurelib.ResolveTimeWindow(window, h.now(), time.UTC)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// The window every tool read gets: it ends by now and starts at most
	// usageattribution.MaxLookback before its end.
	if now := h.now(); span.To.After(now) {
		span.To = now
	}
	if span.From, err = usageattribution.ToolWindowStart(span.From, span.To); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	tools, err := h.tools.ToolCalls(ctx, span.From, span.To)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	report := toolfriction.Analyze(tools)
	owner, limit := strings.TrimSpace(req.Msg.GetOwner()), int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = defaultToolRows
	}
	response := &measurepb.ToolFrictionResponse{Summary: protoToolSummary(report), WindowStart: rfc3339(span.From), WindowEnd: rfc3339(span.To), DefinitionId: definitionID(ToolFriction)}
	for _, p := range report.Profiles {
		if len(response.Profiles) < limit && (owner == "" || p.Owner == owner || owner == toolfriction.OwnerExternal && p.Owner == "") {
			response.Profiles = append(response.Profiles, protoToolProfile(p))
		}
	}
	for _, c := range report.Clusters {
		if len(response.Clusters) < limit && (owner == "" || c.Owner == owner) {
			response.Clusters = append(response.Clusters, &measurepb.ToolErrorCluster{Signature: c.Signature, Kind: c.Kind, Owner: c.Owner, OwnerKind: c.OwnerKind, CommandPaths: c.CommandPaths, Calls: int64(c.Calls), Runs: int64(c.Runs), DurationMs: c.DurationMS, Line: c.Line, ExampleCommands: c.Examples, RunIds: c.RunIDs, RootCause: c.Root, Subs: protoSubs(c.Subs)})
		}
	}
	statuses := map[string]usageattribution.ProblemStatus{}
	if h.frictionStatuses != nil {
		if statuses, err = h.frictionStatuses.FrictionStatuses(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	for _, f := range report.Findings {
		if len(response.Findings) < limit && (owner == "" || f.Owner == owner) {
			finding := usageattribution.ToolFinding{Finding: f, Status: usageattribution.StatusOpen}
			if status, ok := statuses[f.Fingerprint]; ok {
				finding.Status, finding.BugID, finding.StatusSetAt = status.Status, status.BugID, status.At
			}
			response.Findings = append(response.Findings, protoToolFinding(finding))
		}
	}
	response.Validity = h.validityForSample(int64(report.Coverage.Commands))
	response.Provenance = h.provenanceWithQuery(ctx, invocationreadmodel.Filter{From: &span.From, To: &span.To}, definitionFor(ToolFriction).SourceTable, int64(report.Coverage.Commands), "SELECT command facts WITH stored tool features FROM invocation_read_model_facts WHERE tool_profiled = 1 AND classifier_version = <current> AND occurred_at in window; older versions counted as stale")
	return connect.NewResponse(response), nil
}

func protoToolSummary(report toolfriction.Report) *measurepb.ToolFrictionSummary {
	t := report.Totals
	return &measurepb.ToolFrictionSummary{
		Coverage: protoToolCoverage(report.Coverage), Calls: int64(t.Calls), FailedCalls: int64(t.FailedCalls), VrooliFailureRate: t.VrooliFailureRate(),
		ShapeGuesses: int64(t.ShapeGuesses), ShapeGuessRate: t.VrooliShapeGuessRate(), PolledCalls: int64(t.Polled), PolledShare: t.PolledShare(),
		BypassCalls: int64(t.BypassCalls), ContextTaxTokens: t.ContextTax, DurationMs: t.DurationMS, FilteredShare: t.FilteredShare, Profiles: int64(len(report.Profiles)),
	}
}

func protoToolCoverage(c toolfriction.Coverage) *measurepb.ToolFrictionCoverage {
	return &measurepb.ToolFrictionCoverage{Commands: int64(c.Commands), OutcomeKnown: int64(c.OutcomeKnown), OutcomeKnownShare: c.KnownShare(), VrooliCommands: int64(c.VrooliCommands), VrooliOutcomeKnown: int64(c.VrooliOutcomeKnown), OlderParserCommands: int64(c.OlderParser), NoTranscriptCommands: int64(c.NoTranscript), StaleRuns: int64(c.Stale.Runs), StaleCalls: int64(c.Stale.Calls), UnknownTokenCommands: int64(c.UnknownTokens), Sentence: c.Sentence()}
}

func protoToolProfile(p toolfriction.Profile) *measurepb.ToolProfile {
	return &measurepb.ToolProfile{
		CommandPath: p.CommandPath, Owner: p.Owner, OwnerKind: p.OwnerKind, Calls: int64(p.Calls), Runs: int64(p.Runs),
		Decided: int64(p.Decided), Failed: int64(p.Failed), NoMatch: int64(p.NoMatch), Refused: int64(p.Refused), Unknown: int64(p.Unknown), FailureRate: p.FailureRate,
		ShapeGuesses: int64(p.ShapeGuesses), MedianResultTokens: p.MedianTokens, P90ResultTokens: p.P90Tokens, MaxResultTokens: p.MaxTokens,
		TruncatedShare: p.TruncatedShare, ContextTaxTokens: p.ContextTax, FilteredShare: p.FilteredShare, JsonPaths: p.JSONPaths,
		TotalDurationMs: p.TotalMS, P90DurationMs: p.P90MS, PolledCalls: int64(p.Polled), WrappedPolls: int64(p.Wrapped), CostShare: p.CostShare,
	}
}

func protoToolFinding(f usageattribution.ToolFinding) *measurepb.UsageToolFinding {
	out := &measurepb.UsageToolFinding{
		Fingerprint: f.Fingerprint, Kind: f.Kind, Owner: f.Owner, OwnerKind: f.OwnerKind, CommandPath: f.CommandPath, Description: f.Description,
		Calls: int64(f.Calls), Runs: int64(f.Runs), CostUnit: f.Unit, Cost: f.Cost, CostShare: f.CostShare, Signature: f.ErrorLine,
		ExampleCommands: f.Examples, Status: f.Status, BugId: f.BugID, TopRunShare: f.TopRunShare, RootCause: f.RootCause, Subs: protoSubs(f.Subs),
		Guesses: protoGuesses(f.Guesses), PrintedTokens: f.PrintedTokens, ReadTokens: f.ReadTokens, Shares: protoShares(f.Shares), Score: f.Score,
	}
	if !f.StatusSetAt.IsZero() {
		out.StatusAt = rfc3339(f.StatusSetAt)
	}
	for _, id := range f.RunIDs {
		out.EvidenceRefs = append(out.EvidenceRefs, &measurepb.UsageEvidenceRef{Kind: "run", Id: id})
	}
	return out
}

// protoShares states each share of a finding with its base, as every reader
// prints it.
func protoShares(shares []toolfriction.Share) []*measurepb.UsageToolShare {
	out := make([]*measurepb.UsageToolShare, 0, len(shares))
	for _, s := range shares {
		out = append(out, &measurepb.UsageToolShare{Kind: s.Kind, Unit: s.Unit, Cost: s.Cost, Share: s.Share, Base: s.Base, Sentence: s.Sentence()})
	}
	return out
}

func protoSubs(subs []toolfriction.SubSignature) []*measurepb.UsageToolSub {
	var out []*measurepb.UsageToolSub
	for _, s := range subs {
		out = append(out, &measurepb.UsageToolSub{Kind: s.Kind, Signature: s.Signature, Line: s.Line, Calls: int64(s.Calls)})
	}
	return out
}

func protoGuesses(guesses []toolfriction.Guess) []*measurepb.UsageToolGuess {
	var out []*measurepb.UsageToolGuess
	for _, g := range guesses {
		out = append(out, &measurepb.UsageToolGuess{Token: g.Token, Suggestion: g.Suggestion, Calls: int64(g.Calls)})
	}
	return out
}

// tokenWindow is the measure parameter token (this_week) as a TimeWindow.
func tokenWindow(token string) *sharedmeasurepb.TimeWindow {
	if token == "" {
		return nil
	}
	value := sharedmeasurepb.TimeWindowToken_value["TIME_WINDOW_TOKEN_"+strings.ToUpper(strings.ReplaceAll(token, "-", "_"))]
	return &sharedmeasurepb.TimeWindow{Window: &sharedmeasurepb.TimeWindow_Token{Token: sharedmeasurepb.TimeWindowToken(value)}}
}
