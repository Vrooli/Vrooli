package measures

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"
	"agent-manager/internal/usageattribution"

	"connectrpc.com/connect"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

const (
	UsageBurners         = "usage.burners"
	UsageRecommendations = "usage.recommendations"
	UsageCoverage        = "usage.coverage"
	UsageCompare         = "usage.compare"

	defaultUsageRows    = 20
	coverageLookback    = 7 * 24 * time.Hour
	coverageComparisonM = "Independent scan (internal/transcriptindex/live_coverage_test.go, USAGE_COVERAGE_LIVE=1): walk the same roots, take every *.jsonl modified since `since`, sum sizes and count provider turns on disk (Codex turn_context lines; Claude assistant lines with usage). Files owned by an Agent Manager run count as captured. The disk bytes must equal bytes, captured_bytes/bytes must agree within 1 point, and captured turns (Codex turns, Claude usage_reports, owned files' disk turns) over disk turns is the A1 share."
)

// SetUsageAttribution connects the usage facts, the price book and the
// indexer's coverage. Without them the procedures report unavailable.
func (h *Handler) SetUsageAttribution(store usageattribution.Store, pricer usageattribution.ModelPricer, coverage transcriptindex.CoverageReader) {
	// One price book per handler: prices are reused across requests.
	h.usageStore, h.usagePricer, h.coverage = store, usageattribution.NewPriceBook(pricer), coverage
}

func (h *Handler) usageService() (usageattribution.Service, error) {
	if h.usageStore == nil || h.quotaReadings == nil {
		return usageattribution.Service{}, connect.NewError(connect.CodeUnavailable, fmt.Errorf("usage attribution is unavailable"))
	}
	return usageattribution.Service{Store: h.usageStore, Readings: h.quotaReadings, Pricer: h.usagePricer}, nil
}

func parseOptionalTime(field, value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	at, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s must be RFC 3339: %w", field, err))
	}
	return at, nil
}

func usageRows(limit int32) int {
	if limit <= 0 {
		return defaultUsageRows
	}
	return int(limit)
}

func usageAttributionProvenance(source string, rows int64, from, to time.Time, filters ...*measurepb.MeasureFilter) *measurepb.MeasureProvenance {
	provenance := &measurepb.MeasureProvenance{SourceTable: source, RowCount: rows, WindowStart: rfc3339(from), WindowEnd: rfc3339(to)}
	for _, filter := range filters {
		if filter.GetValue() != "" {
			provenance.AppliedFilters = append(provenance.AppliedFilters, filter)
		}
	}
	return provenance
}

func (h *Handler) UsageBurners(ctx context.Context, req *connect.Request[measurepb.UsageBurnersRequest]) (*connect.Response[measurepb.UsageBurnersResponse], error) {
	service, err := h.usageService()
	if err != nil {
		return nil, err
	}
	by := strings.TrimSpace(req.Msg.GetGroupBy())
	if by == "" {
		by = usageattribution.BySession
	}
	if !usageattribution.ValidDimension(by) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid group_by %q; accepted values: %s", by, strings.Join(usageattribution.Dimensions, ", ")))
	}
	since, err := parseOptionalTime("since", req.Msg.GetSince())
	if err != nil {
		return nil, err
	}
	now := h.now().UTC()
	attribution, err := service.Current(ctx, req.Msg.GetProvider(), since, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &measurepb.UsageBurnersResponse{GeneratedAt: now.Format(time.RFC3339), DefinitionId: definitionID(UsageBurners)}
	var turns int64
	from := now
	for _, result := range attribution.Results {
		response.Windows = append(response.Windows, protoWindowAttribution(result, by, attribution.Runs, strings.TrimSpace(req.Msg.GetRunId()), usageRows(req.Msg.GetLimit())))
		turns += int64(len(result.Turns))
		if result.From.Before(from) {
			from = result.From
		}
	}
	response.Validity = h.validityForSample(turns)
	response.Provenance = usageAttributionProvenance(definitionFor(UsageBurners).SourceTable, turns, from, now,
		&measurepb.MeasureFilter{Field: "provider", Value: req.Msg.GetProvider()}, &measurepb.MeasureFilter{Field: "group_by", Value: by},
		&measurepb.MeasureFilter{Field: "run_id", Value: req.Msg.GetRunId()}, &measurepb.MeasureFilter{Field: "since", Value: req.Msg.GetSince()})
	return connect.NewResponse(response), nil
}

func protoWindowAttribution(result usageattribution.Result, by string, runs map[string]usageattribution.Run, runID string, limit int) *measurepb.UsageWindowAttribution {
	w := result.Window
	out := &measurepb.UsageWindowAttribution{
		Provider: w.Provider, Pool: w.Pool, Window: w.Name, WindowMinutes: w.Minutes, WindowStart: rfc3339(w.Start), ResetsAt: rfc3339(w.ResetAt),
		PeriodStart: rfc3339(result.From), PeriodEnd: rfc3339(result.To), UsedPercent: w.UsedPercent,
		ObservedIncreasePoints: result.Observed, AttributedPoints: result.Attributed(), UnattributedPoints: result.Unattributed,
		GroupBy: by, PendingTokens: protoTokens(result.Pending), Runner: usageattribution.RunnerFor(w.Provider),
		Calibration: protoCalibration(result.Calibration),
	}
	burners := usageattribution.Group(result, by, runs, runID)
	for i, burner := range burners {
		if i >= limit {
			out.OtherBurners++
			out.OtherPoints += burner.Points
			continue
		}
		out.Burners = append(out.Burners, protoBurner(burner))
	}
	if runID != "" {
		// A run's rows are part of the window; the rest of the window is
		// neither "other" nor unattributed for the run panel.
		out.OtherPoints = result.Attributed()
		for _, burner := range burners {
			out.OtherPoints -= burner.Points
		}
		out.OtherPoints = max(0, out.OtherPoints)
	}
	for _, line := range usageattribution.MeteredLines(result) {
		out.MeteredLines = append(out.MeteredLines, &measurepb.UsageMeteredLine{Kind: line.Kind, Count: int64(line.Count), State: line.State, Amount: line.Amount, Note: line.Note})
	}
	return out
}

func protoTokens(c domain.UsageCounters) *measurepb.UsageTokens {
	return &measurepb.UsageTokens{Input: int64(c.InputTokens), CachedInput: int64(c.CacheReadTokens), CacheCreation: int64(c.CacheCreationTokens), Output: int64(c.OutputTokens), Total: c.Total()}
}

func protoCalibration(c usageattribution.Calibration) *measurepb.UsageCalibration {
	out := &measurepb.UsageCalibration{Method: c.Method, RSquared: c.RSquared, Slices: int64(c.Slices), OffDiskPointsPerHour: c.OffDiskPerHour, Note: c.Note}
	for _, w := range c.Weights {
		out.Weights = append(out.Weights, &measurepb.UsageModelWeight{Runner: w.Runner, Model: w.Model, CachedInputTokensPerPoint: w.CachedPerPoint, InputWeight: w.InputWeight, CacheCreationWeight: w.CreationWeight, OutputWeight: w.OutWeight, Source: w.Source, Slices: int64(w.Slices)})
	}
	return out
}

func protoBurner(b usageattribution.Burner) *measurepb.UsageBurner {
	out := &measurepb.UsageBurner{
		Key: b.Key, Label: b.Label, Points: b.Points, Share: b.Share, Tokens: protoTokens(b.Tokens), WeightedTokens: b.Weighted,
		Sessions: int64(b.Sessions), Runs: int64(b.Runs), Turns: int64(b.Turns), RunIds: b.RunIDs, Runner: b.Runner, Model: b.Model,
		ProfileId: b.ProfileID, Workload: b.Workload, FirstAt: rfc3339(b.FirstAt), LastAt: rfc3339(b.LastAt), Source: b.Source,
		Modality: &measurepb.UsageModality{
			ImagesIn: int64(b.Facts.ImagesIn), ImagesInUser: int64(b.Facts.ImagesInUser), ImagesInHighDetail: int64(b.Facts.ImagesInHighDetail),
			ImageBytesIn: b.Facts.ImageBytesIn, EstimatedImageInputTokens: usageattribution.EstimatedImageTokens(b.Facts), ImagesOut: int64(b.Facts.ImagesOut),
			WebSearches: int64(b.Facts.WebSearches), WebFetches: int64(b.Facts.WebFetches), AudioIn: int64(b.Facts.AudioIn), ToolCalls: int64(b.Facts.ToolCalls),
		},
	}
	for _, kind := range sortedKinds(b.Facts.ToolKinds) {
		out.Modality.ToolKinds = append(out.Modality.ToolKinds, &measurepb.UsageToolKindCount{Kind: kind, Calls: int64(b.Facts.ToolKinds[kind])})
	}
	return out
}

func sortedKinds(kinds map[string]int) []string {
	out := make([]string, 0, len(kinds))
	for kind := range kinds {
		out = append(out, kind)
	}
	sort.Slice(out, func(i, j int) bool {
		if kinds[out[i]] != kinds[out[j]] {
			return kinds[out[i]] > kinds[out[j]]
		}
		return out[i] < out[j]
	})
	return out
}

func (h *Handler) UsageRecommendations(ctx context.Context, req *connect.Request[measurepb.UsageRecommendationsRequest]) (*connect.Response[measurepb.UsageRecommendationsResponse], error) {
	service, err := h.usageService()
	if err != nil {
		return nil, err
	}
	now := h.now().UTC()
	report, attribution, err := service.Recommendations(ctx, req.Msg.GetProvider(), now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	runID := strings.TrimSpace(req.Msg.GetRunId())
	response := &measurepb.UsageRecommendationsResponse{GeneratedAt: now.Format(time.RFC3339), DominantConsumer: report.Dominant, DominantConsumerLabel: report.DominantLabel, DefinitionId: definitionID(UsageRecommendations)}
	limit := usageRows(req.Msg.GetLimit())
	for _, row := range report.Recommendations {
		if runID != "" && !namesRun(row, runID) {
			continue
		}
		if len(response.Recommendations) >= limit {
			break
		}
		response.Recommendations = append(response.Recommendations, protoRecommendation(row))
	}
	for _, rule := range report.Rules {
		response.Rules = append(response.Rules, &measurepb.UsageRuleStatus{Rule: rule.Rule, Fired: rule.Fired, Rows: int64(rule.Rows), Candidates: int64(rule.Candidates), Reason: rule.Reason})
	}
	var turns int64
	from := now
	for _, result := range attribution.Results {
		turns += int64(len(result.Turns))
		if result.From.Before(from) {
			from = result.From
		}
	}
	response.Validity = h.validityForSample(turns)
	response.Provenance = usageAttributionProvenance(definitionFor(UsageRecommendations).SourceTable, int64(len(report.Recommendations)), from, now,
		&measurepb.MeasureFilter{Field: "provider", Value: req.Msg.GetProvider()}, &measurepb.MeasureFilter{Field: "run_id", Value: runID})
	return connect.NewResponse(response), nil
}

func namesRun(row usageattribution.Recommendation, runID string) bool {
	if row.TargetID == runID {
		return true
	}
	for _, ref := range row.Evidence {
		if ref.ID == runID || ref.Label == "run "+runID {
			return true
		}
	}
	return false
}

func protoRecommendation(row usageattribution.Recommendation) *measurepb.UsageRecommendation {
	out := &measurepb.UsageRecommendation{
		Rank: int32(row.Rank), Rule: row.Rule, Lever: row.Lever, TargetKind: row.TargetKind, TargetId: row.TargetID, TargetLabel: row.TargetLabel,
		Provider: row.Provider, Pool: row.Pool, Window: row.Window, EstWeeklyQuotaPointsSaved: row.WeeklySaved, Confidence: row.Confidence,
		Action: row.Action, Rationale: row.Rationale, ConsumerPoints: row.ConsumerPoints,
	}
	for _, ref := range row.Evidence {
		out.EvidenceRefs = append(out.EvidenceRefs, &measurepb.UsageEvidenceRef{Kind: ref.Kind, Id: ref.ID, Label: ref.Label})
	}
	for _, metric := range row.Metrics {
		out.Metrics = append(out.Metrics, &measurepb.UsageEvidenceMetric{Name: metric.Name, Value: metric.Value, Unit: metric.Unit})
	}
	return out
}

func (h *Handler) UsageCoverage(ctx context.Context, req *connect.Request[measurepb.UsageCoverageRequest]) (*connect.Response[measurepb.UsageCoverageResponse], error) {
	if h.coverage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("transcript coverage is unavailable"))
	}
	now := h.now().UTC()
	since, err := parseOptionalTime("since", req.Msg.GetSince())
	if err != nil {
		return nil, err
	}
	if since.IsZero() {
		since = now.Add(-coverageLookback)
	}
	coverage, err := h.coverage.Coverage(ctx, since)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &measurepb.UsageCoverageResponse{Since: rfc3339(since), Totals: protoCoverageRoot(coverage.Totals), Method: coverageComparisonM, GeneratedAt: now.Format(time.RFC3339), DefinitionId: definitionID(UsageCoverage)}
	for _, root := range coverage.Roots {
		response.Roots = append(response.Roots, protoCoverageRoot(root))
	}
	response.Provenance = usageAttributionProvenance(definitionFor(UsageCoverage).SourceTable, int64(coverage.Totals.Files), since, now)
	return connect.NewResponse(response), nil
}

func protoCoverageRoot(root transcriptindex.RootCoverage) *measurepb.UsageCoverageRoot {
	out := &measurepb.UsageCoverageRoot{
		Kind: root.Kind, Harness: root.Harness, Files: int64(root.Files), Indexed: int64(root.Indexed), WithUsage: int64(root.WithUsage), Owned: int64(root.Owned),
		Excluded: int64(root.Excluded), Pending: int64(root.Pending), Unreadable: int64(root.Unreadable), Bytes: root.Bytes, CapturedBytes: root.IndexedBytes,
		CapturedShare: root.CapturedShare(), Turns: root.Turns, UsageReports: root.UsageReports,
	}
	for _, issue := range root.Issues {
		out.Issues = append(out.Issues, &measurepb.UsageCoverageIssue{Path: issue.Path, Reason: issue.Reason})
	}
	return out
}

func (h *Handler) UsageCompare(ctx context.Context, req *connect.Request[measurepb.UsageCompareRequest]) (*connect.Response[measurepb.UsageCompareResponse], error) {
	service, err := h.usageService()
	if err != nil {
		return nil, err
	}
	left, err := h.compareSide(ctx, service, req.Msg.GetProvider(), req.Msg.GetLeft())
	if err != nil {
		return nil, err
	}
	right, err := h.compareSide(ctx, service, req.Msg.GetProvider(), req.Msg.GetRight())
	if err != nil {
		return nil, err
	}
	response := &measurepb.UsageCompareResponse{Left: protoCompareSide(left), Right: protoCompareSide(right), ChangeBinding: req.Msg.GetChangeBinding(), DefinitionId: definitionID(UsageCompare)}
	response.Caveats = []string{
		"Points are each run's attributed share of the observed window increase (estimates); runs outside any window with readings are unpriced.",
		"Cohorts are observational: compare comparable workloads, and read the success counts before the ratio.",
	}
	sample := int64(min(left.Succeeded, right.Succeeded))
	response.Validity = h.validityForSample(sample)
	switch {
	case left.Succeeded < 5 || right.Succeeded < 5 || left.PointsPerSuccess() == 0:
		response.Verdict = "insufficient"
		response.Caveats = append(response.Caveats, "each side needs at least five successful runs with priced usage")
	default:
		response.PointsPerSuccessChange = right.PointsPerSuccess()/left.PointsPerSuccess() - 1
		switch {
		case response.PointsPerSuccessChange <= -0.1:
			response.Verdict = "improved"
		case response.PointsPerSuccessChange >= 0.1:
			response.Verdict = "regressed"
		default:
			response.Verdict = "unchanged"
		}
	}
	response.Provenance = usageAttributionProvenance(definitionFor(UsageCompare).SourceTable, int64(left.Runs+right.Runs), left.From, right.To, &measurepb.MeasureFilter{Field: "provider", Value: req.Msg.GetProvider()})
	return connect.NewResponse(response), nil
}

func (h *Handler) compareSide(ctx context.Context, service usageattribution.Service, provider string, selector *measurepb.CohortSelector) (usageattribution.Side, error) {
	selection, err := h.resolveCohort(ctx, selector.GetWindow(), selector.GetFilter(), selector.GetCohort(), selector.GetRunIds())
	if err != nil {
		return usageattribution.Side{}, err
	}
	cohort, err := h.runs(ctx, selection, maxCohortRunIDs*10)
	if err != nil {
		return usageattribution.Side{}, connect.NewError(connect.CodeInternal, err)
	}
	now := h.now().UTC()
	from, to := now.Add(-7*24*time.Hour), now
	if selection.filter.From != nil {
		from = *selection.filter.From
	}
	if selection.filter.To != nil {
		to = *selection.filter.To
	}
	side, err := service.Compare(ctx, provider, cohort.RunIDs, from, to)
	if err != nil {
		return side, connect.NewError(connect.CodeInternal, err)
	}
	return side, nil
}

func protoCompareSide(s usageattribution.Side) *measurepb.UsageCompareSide {
	out := &measurepb.UsageCompareSide{
		Runs: int64(s.Runs), SuccessfulRuns: int64(s.Succeeded), PricedRuns: int64(s.Priced), Tokens: protoTokens(s.Tokens),
		WeightedTokens: s.Weighted, Points: s.Points, PointsPerSuccess: s.PointsPerSuccess(), WindowStart: rfc3339(s.From), WindowEnd: rfc3339(s.To),
	}
	if s.Succeeded > 0 {
		out.TokensPerSuccess = float64(s.Tokens.Total()) / float64(s.Succeeded)
	}
	if s.Runs > 0 {
		out.SuccessRate = float64(s.Succeeded) / float64(s.Runs)
	}
	return out
}
