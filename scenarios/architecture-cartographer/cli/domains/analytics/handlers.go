package analytics

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	analyticsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/analytics"
	analyticsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/analytics/analytics_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client analyticsconnect.AnalyticsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: analyticsconnect.NewAnalyticsServiceClient(httpClient, baseURL),
	}
}

// events pages the append-only event log for a scenario.
func (h *handlers) events(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	req := &analyticsv1.ListEventsRequest{
		Scenario:  scenario,
		PageToken: ctx.Flag("page-token"),
	}
	if raw := strings.TrimSpace(ctx.Flag("page-size")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("--page-size must be an integer: %w", err)
		}
		req.PageSize = int32(n)
	}
	if raw := strings.TrimSpace(ctx.Flag("since")); raw != "" {
		ts, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return fmt.Errorf("--since must be RFC3339 (e.g. 2026-05-25T00:00:00Z): %w", err)
		}
		req.Since = timestamppb.New(ts)
	}
	for _, k := range cliutil.ParseCSV(ctx.Flag("kind")) {
		kind, ok := parseEventKind(k)
		if !ok {
			return fmt.Errorf("unknown --kind %q", k)
		}
		req.Kinds = append(req.Kinds, kind)
	}
	resp, err := h.client.ListEvents(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("list events for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no events response")
	}
	results := make([]string, 0, len(resp.Msg.GetEvents()))
	for _, e := range resp.Msg.GetEvents() {
		when := ""
		if e.GetRecordedAt() != nil {
			when = e.GetRecordedAt().AsTime().Format(time.RFC3339)
		}
		results = append(results, fmt.Sprintf("%s %s actor=%s domain=%s conflict=%s chunk=%s",
			when, eventKindName(e.GetKind()), e.GetActor(), e.GetDomain(), e.GetConflictId(), e.GetChunkId()))
	}
	hint := "`analytics stats <scenario>` for the roll-up."
	if tok := resp.Msg.GetNextPageToken(); tok != "" {
		hint = fmt.Sprintf("More results: `analytics events %s --page-token %s`.", scenario, tok)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d event(s) for %q.", len(resp.Msg.GetEvents()), scenario)},
		ResultsHeading: "Events",
		Results:        results,
		RetrievalHints: []string{hint},
	})
}

// stats renders the analytics roll-up. The verdict-success rate is
// suppressed (no percentage) when the observation count is below the N>=5
// threshold; this handler surfaces the honest explanatory message rather
// than a fabricated rate.
func (h *handlers) stats(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.GetStats(context.Background(), connect.NewRequest(&analyticsv1.GetStatsRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get stats for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetStats() == nil {
		return fmt.Errorf("server returned no stats")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	s := resp.Msg.GetStats()

	successLine := fmt.Sprintf("verdict success rate: %.1f%% (N=%d)", s.GetVerdictSuccessRate()*100, s.GetVerdictObservationCount())
	if s.GetVerdictSuccessRateSuppressed() {
		successLine = fmt.Sprintf("verdict success rate: suppressed — only %d observation(s), below the N>=5 threshold; no rate shown until more data lands.",
			s.GetVerdictObservationCount())
	}
	results := []string{
		fmt.Sprintf("conflicts detected: %d", s.GetConflictsDetected()),
		fmt.Sprintf("conflicts resolved: %d", s.GetConflictsResolved()),
		fmt.Sprintf("conflicts force-resolved: %d", s.GetConflictsForceResolved()),
		fmt.Sprintf("placements auto: %d", s.GetPlacementsAuto()),
		fmt.Sprintf("placements suggest: %d", s.GetPlacementsSuggest()),
		fmt.Sprintf("overrides: %d", s.GetOverrides()),
		successLine,
	}
	return ctx.RenderList(cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Analytics roll-up for %q.", scenario)},
		ResultsHeading: "Stats",
		Results:        results,
	})
}

// placements pages the placement-outcome rows for a scenario.
func (h *handlers) placements(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	req := &analyticsv1.ListPlacementsRequest{
		Scenario:  scenario,
		Outcomes:  cliutil.ParseCSV(ctx.Flag("outcome")),
		PageToken: ctx.Flag("page-token"),
	}
	if raw := strings.TrimSpace(ctx.Flag("page-size")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("--page-size must be an integer: %w", err)
		}
		req.PageSize = int32(n)
	}
	resp, err := h.client.ListPlacements(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("list placements for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no placements response")
	}
	results := make([]string, 0, len(resp.Msg.GetPlacements()))
	for _, p := range resp.Msg.GetPlacements() {
		top := ""
		if v := p.GetVerdict(); v != nil {
			top = fmt.Sprintf("%s(%.2f)", v.GetTopDomain(), v.GetTopValue())
		}
		results = append(results, fmt.Sprintf("%s %s outcome=%s auto=%t verdict=%s",
			p.GetId(), p.GetChunkPath(), p.GetOutcome(), p.GetAutoActed(), top))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d placement(s) for %q.", len(resp.Msg.GetPlacements()), scenario)},
		ResultsHeading: "Placements",
		Results:        results,
	})
}

// overrideRecord appends an Override row. Mutating → honors --dry-run via
// the X-Dry-Run header; the response echoes the dry-run state.
func (h *handlers) overrideRecord(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.RecordOverride(context.Background(), connect.NewRequest(&analyticsv1.RecordOverrideRequest{
		Scenario:       scenario,
		ChunkId:        ctx.Flag("chunk-id"),
		VerdictDomain:  ctx.Flag("verdict-domain"),
		ChosenDomain:   ctx.Flag("chosen-domain"),
		Note:           ctx.Flag("note"),
		VerdictEventId: ctx.Flag("verdict-event-id"),
		IdempotencyKey: ctx.Flag("idempotency-key"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("record override for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetOverride() == nil {
		return fmt.Errorf("server returned no override")
	}
	o := resp.Msg.GetOverride()
	result := fmt.Sprintf("Recorded override %s: %q → %q for chunk %s.", o.GetId(), o.GetVerdictDomain(), o.GetChosenDomain(), o.GetChunkId())
	if resp.Msg.GetDryRun() {
		result += " (dry-run: no row persisted)"
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{result},
		Changes:     []string{fmt.Sprintf("override: scenario=%s chunk=%s verdict=%s chosen=%s", o.GetScenario(), o.GetChunkId(), o.GetVerdictDomain(), o.GetChosenDomain())},
		NextCommand: []string{fmt.Sprintf("`analytics stats %s` to see the override reflected in the roll-up.", scenario)},
	})
}

// -------------------------- enum mapping --------------------------

var eventKinds = map[string]analyticsv1.EventKind{
	"conflict_detected":       analyticsv1.EventKind_EVENT_KIND_CONFLICT_DETECTED,
	"conflict_assigned":       analyticsv1.EventKind_EVENT_KIND_CONFLICT_ASSIGNED,
	"conflict_resolved":       analyticsv1.EventKind_EVENT_KIND_CONFLICT_RESOLVED,
	"conflict_reopened":       analyticsv1.EventKind_EVENT_KIND_CONFLICT_REOPENED,
	"conflict_force_resolved": analyticsv1.EventKind_EVENT_KIND_CONFLICT_FORCE_RESOLVED,
	"verdict_produced":        analyticsv1.EventKind_EVENT_KIND_VERDICT_PRODUCED,
	"placement_auto":          analyticsv1.EventKind_EVENT_KIND_PLACEMENT_AUTO,
	"placement_suggest":       analyticsv1.EventKind_EVENT_KIND_PLACEMENT_SUGGEST,
	"override_recorded":       analyticsv1.EventKind_EVENT_KIND_OVERRIDE_RECORDED,
	"apply_planned":           analyticsv1.EventKind_EVENT_KIND_APPLY_PLANNED,
	"apply_ran":               analyticsv1.EventKind_EVENT_KIND_APPLY_RAN,
	"apply_build_green":       analyticsv1.EventKind_EVENT_KIND_APPLY_BUILD_GREEN,
	"apply_build_red":         analyticsv1.EventKind_EVENT_KIND_APPLY_BUILD_RED,
	"apply_reverted":          analyticsv1.EventKind_EVENT_KIND_APPLY_REVERTED,
}

func parseEventKind(token string) (analyticsv1.EventKind, bool) {
	k, ok := eventKinds[strings.ToLower(strings.TrimSpace(token))]
	return k, ok
}

func eventKindName(k analyticsv1.EventKind) string {
	for name, kind := range eventKinds {
		if kind == k {
			return name
		}
	}
	return "unspecified"
}
