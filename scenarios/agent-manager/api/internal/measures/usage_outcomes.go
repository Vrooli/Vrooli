package measures

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/outcomeledger"
	"agent-manager/internal/pricing"
	"agent-manager/internal/usageattribution"

	"connectrpc.com/connect"
	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

const (
	UsageParity   = "usage.parity"
	UsageOutcomes = "usage.outcomes"
)

// SetOutcomeLedger connects the run ledger. Without it parity and outcomes
// report unavailable. The ledger's reader carries the read model's input cache,
// which lives as long as this connection.
func (h *Handler) SetOutcomeLedger(reader outcomeledger.Reader) {
	h.ledger = &cachedLedger{Reader: reader, inputs: newInputCache()}
}

func (h *Handler) ledgerReader() (outcomeledger.Reader, error) {
	if h.ledger == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("the run ledger is unavailable"))
	}
	return h.ledger, nil
}

// ledgerWindow resolves a request window, default the last 7 days, ending by now.
func (h *Handler) ledgerWindow(window *sharedmeasurepb.TimeWindow) (from, to time.Time, err error) {
	now := h.now().UTC()
	if window == nil || window.GetWindow() == nil || window.GetToken() == sharedmeasurepb.TimeWindowToken_TIME_WINDOW_TOKEN_UNSPECIFIED {
		return now.Add(-7 * 24 * time.Hour), now, nil
	}
	span, err := measurelib.ResolveTimeWindow(window, now, time.UTC)
	if err != nil {
		return time.Time{}, time.Time{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if span.To.After(now) {
		span.To = now
	}
	return span.From, span.To, nil
}

func (h *Handler) UsageParity(ctx context.Context, req *connect.Request[measurepb.UsageParityRequest]) (*connect.Response[measurepb.UsageParityResponse], error) {
	ledger, err := h.ledgerReader()
	if err != nil {
		return nil, err
	}
	from, to, err := h.ledgerWindow(req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	runs, err := ledger.FactsInWindow(ctx, from, to)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	backfill, err := ledger.BackfillCount(ctx, from, to)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pending, err := ledger.PendingCount(ctx, h.now().UTC().Add(-outcomeledger.Lookback))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &measurepb.UsageParityResponse{GeneratedAt: h.now().UTC().Format(time.RFC3339), WindowStart: rfc3339(from), WindowEnd: rfc3339(to), BackfillRuns: backfill, LedgerPending: pending, DefinitionId: definitionID(UsageParity)}
	var eligible int64
	for _, row := range outcomeledger.BuildParity(runs) {
		out := &measurepb.UsageParityRow{Path: row.Path, Harness: row.Harness, Runs: row.Runs, EligibleRuns: row.Eligible, NoRequests: row.NoRequests}
		eligible += row.Eligible
		for _, name := range row.ExcludedNames() {
			out.Excluded = append(out.Excluded, &measurepb.UsageParityCount{Name: name, Count: row.Excluded[name]})
		}
		for _, m := range row.Measures {
			out.Measures = append(out.Measures, &measurepb.UsageParityMeasure{Name: m.Name, Numerator: m.Numerator, Denominator: m.Denominator, Share: m.Share(), NotApplicable: m.NotApplicable, Reason: m.Reason})
		}
		response.Rows = append(response.Rows, out)
	}
	response.Gaps = h.parityGaps(ctx, runs, backfill, from, response)
	response.Validity = h.validityForSample(eligible)
	response.Provenance = usageAttributionProvenance(definitionFor(UsageParity).SourceTable, int64(len(runs)), from, to)
	return connect.NewResponse(response), nil
}

// parityGaps states each known difference between paths and why it remains.
func (h *Handler) parityGaps(ctx context.Context, runs []outcomeledger.RunFacts, backfill int64, from time.Time, response *measurepb.UsageParityResponse) []*measurepb.UsageParityGap {
	var gaps []*measurepb.UsageParityGap
	add := func(subject, status string, count int64, reason string) {
		gaps = append(gaps, &measurepb.UsageParityGap{Subject: subject, Status: status, Count: count, Reason: reason})
	}
	add("old sessions imported this window", outcomeledger.GapExplained, backfill, "first imported in this window, last active before it. Excluded by activity time; their events are outside the recall window, so facts exist only for those re-read on demand")
	var helper, linked, older, olderCommands int64
	for _, run := range runs {
		switch run.AttachedClass {
		case outcomeledger.AttachedHelper:
			helper++
		case outcomeledger.AttachedLinked:
			linked++
		}
		if run.OlderParser && run.Commands > 0 {
			older++
			olderCommands += run.Commands
		}
	}
	add("attached rows that are helpers", outcomeledger.GapExplained, helper, "lived under 3 minutes with no transcript: the shim attaches any launch not on its informational list. Left out of denominators")
	add("attached rows that duplicate a transcript", outcomeledger.GapClosed, linked, "linked to the imported run that holds the facts. Left out of denominators")
	add("commands read by an older parser", outcomeledger.GapExplained, olderCommands, fmt.Sprintf("%d runs were read before the parser recorded outcomes. Interactive history cannot be re-derived (SPEC AM-12a) and leaves this window by 2026-11-04; imported files re-read within the budget", older))
	for _, gap := range outcomeledger.NoReaderGaps(runs) {
		add(gap.Subject, gap.Status, gap.Count, gap.Reason)
	}
	if h.coverage != nil {
		if coverage, err := h.coverage.Coverage(ctx, from); err == nil {
			for _, root := range coverage.Roots {
				response.TranscriptRoots = append(response.TranscriptRoots, protoCoverageRoot(root))
			}
			var waiting, kept, other int64
			for _, root := range coverage.Roots {
				for _, issue := range root.Issues {
					switch {
					case strings.Contains(issue.Reason, "exceeds replacement staging"):
						kept++
					case strings.Contains(issue.Reason, "not indexed yet"), strings.Contains(issue.Reason, "source changed since"), strings.Contains(issue.Reason, "partially indexed"):
						waiting++
					case issue.Reason != "":
						other++
					}
				}
			}
			add("transcripts still being read", outcomeledger.GapExplained, waiting, "live or new files between index ticks; transient")
			add("transcripts too large to re-read", outcomeledger.GapExplained, kept, "an older parser read them and they exceed the replacement staging budget: prior usage and evidence are kept and stay readable")
			add("transcripts needing reconciliation", outcomeledger.GapOpen, other, "the cursor no longer matches the file or a retry failed. An interrupted capture now retries by itself: see `usage coverage`")
		}
	}
	return gaps
}

func (h *Handler) UsageOutcomes(ctx context.Context, req *connect.Request[measurepb.UsageOutcomesRequest]) (*connect.Response[measurepb.UsageOutcomesResponse], error) {
	ledger, err := h.ledgerReader()
	if err != nil {
		return nil, err
	}
	by := strings.TrimSpace(req.Msg.GetBy())
	if by == "" {
		by = outcomeledger.ByModel
	}
	if by != outcomeledger.ByModel && by != outcomeledger.ByRole && by != outcomeledger.ByMode && by != outcomeledger.ByArm {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid by %q; accepted values: model, role, mode, arm", by))
	}
	from, to, err := h.ledgerWindow(req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	inputs, err := h.loadOutcomeInputs(ctx, from, to)
	if err != nil {
		return nil, err
	}
	runs, period, use := inputs.runs, inputs.period, inputs.use
	if by == outcomeledger.ByArm { // only units a comparison assigned to an arm, whole
		runs = outcomeledger.ArmWork(runs, "")
	}
	units, samples := outcomeledger.Assemble(runs, use, by)
	spanning := 0
	if by == outcomeledger.ByArm {
		units, samples, spanning = outcomeledger.DropSpanning(units, samples)
	}
	config := outcomeledger.Config{Baseline: strings.TrimSpace(req.Msg.GetBaseline())}
	if by == outcomeledger.ByModel {
		config.PriceSources = period.Sources
		config.ModelPrices, config.ReferenceMix = modelPrices(runs, period)
	}
	report := outcomeledger.Analyze(units, samples, config)
	if spanning > 0 {
		report.Notes = append(report.Notes, fmt.Sprintf("%d units ran in more than one arm for a reason other than a retry; they measure neither arm and are left out.", spanning))
	}
	response := &measurepb.UsageOutcomesResponse{
		GeneratedAt: h.now().UTC().Format(time.RFC3339), WindowStart: rfc3339(from), WindowEnd: rfc3339(to), By: by, Units: int64(report.Units), UnknownUnits: int64(report.Unknown),
		CompletedUnits: int64(report.Completed), SpanningUnits: int64(spanning),
		LinkConfidence: report.LinkConfidence, LinkedUnits: int64(report.LinkedUnits), Notes: report.Notes, DefinitionId: definitionID(UsageOutcomes),
		Evidence: protoEvidence(report.Evidence), Paths: protoEvidence(report.Paths),
	}
	for _, row := range report.Rows {
		response.Rows = append(response.Rows, protoOutcomeRow(row))
	}
	periods, err := ledger.ModePeriods(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	labels := map[string]string{}
	if h.usageModes != nil {
		for _, active := range h.usageModes.List() {
			labels[active.ID] = active.Label
		}
	}
	response.ModePeriods = protoModePeriods(periods, labels, from, to)
	response.Design = outcomeledger.DesignObservational
	if by == outcomeledger.ByArm { // read from each unit's assignment basis, not assumed
		response.Design = outcomeledger.DesignOf(units)
	}
	if by == outcomeledger.ByModel {
		response.Series = modelSeries(runs, use, from, to)
	}
	spans := make([]usageattribution.Span, 0, len(periods))
	for _, mode := range periods {
		end := mode.EndedAt
		if end.IsZero() {
			end = mode.ExpiresAt
		}
		if end.IsZero() || end.After(to) {
			end = to
		}
		spans = append(spans, usageattribution.Span{From: mode.StartedAt, To: end})
	}
	for _, row := range usageattribution.Refit(period, spans) {
		status := "insufficient"
		if row.Usable {
			status = "usable"
		}
		response.Refit = append(response.Refit, &measurepb.UsageRefitRow{
			Runner: row.Runner, Model: row.Model, PriceSource: row.PriceSource, DominantBlocks: int64(row.DominantBlocks), ModeBlocks: int64(row.ModeBlocks),
			ImpliedPointsPerMillion: &measurepb.UsageRange{Available: row.DominantBlocks > 0, Value: row.ImpliedPerMillion, Low: row.ImpliedLow, High: row.ImpliedHigh, Samples: int64(row.DominantBlocks)},
			PriorPointsPerMillion:   row.PriorPerMillion, Status: status, Reason: row.Reason,
		})
	}
	response.Validity = h.validityForSample(int64(report.Units - report.Unknown))
	response.Provenance = usageAttributionProvenance(definitionFor(UsageOutcomes).SourceTable, int64(report.Units), from, to, &measurepb.MeasureFilter{Field: "by", Value: by})
	return connect.NewResponse(response), nil
}

// modelPrices reads each model's price from the price book and the token mix of
// each runner's work: one mix for every model of the runner, so the ratio is the
// price alone. A price that omits its cache categories takes the pricing package's
// ratios, as attribution does.
func modelPrices(runs []outcomeledger.RunFacts, period usageattribution.PeriodUse) (map[string]outcomeledger.ModelPrice, map[string]outcomeledger.TokenMix) {
	prices, mixes := map[string]outcomeledger.ModelPrice{}, map[string]outcomeledger.TokenMix{}
	if period.Prices == nil {
		return prices, mixes
	}
	for _, run := range runs {
		for model, use := range period.Runs[run.RunID] {
			mix := mixes[run.Harness]
			mix.Input += float64(use.Tokens.InputTokens)
			mix.CacheRead += float64(use.Tokens.CacheReadTokens)
			mix.CacheCreation += float64(use.Tokens.CacheCreationTokens)
			mix.Output += float64(use.Tokens.OutputTokens)
			mixes[run.Harness] = mix
			if _, seen := prices[model]; seen {
				continue
			}
			price := period.Prices(run.Harness, model)
			if !price.Known || price.Input <= 0 {
				continue
			}
			filled := pricing.TokenPrices{Input: price.Input, Output: price.Output, CacheRead: price.CacheRead, CacheCreation: price.CacheCreation}.WithDefaultCacheRatios()
			source := period.Sources[model]
			if source == "" {
				source = priceOrigin(price)
			}
			prices[model] = outcomeledger.ModelPrice{Input: filled.Input, CacheRead: filled.CacheRead, CacheCreation: filled.CacheCreation, Output: filled.Output, Source: source}
		}
	}
	return prices, mixes
}

func priceOrigin(price usageattribution.Price) string {
	if price.Prior {
		return "tier_prior"
	}
	return "price_book"
}

func protoEvidence(counts map[string]int) []*measurepb.UsageEvidenceCount {
	out := make([]*measurepb.UsageEvidenceCount, 0, len(counts))
	for _, name := range sortedKinds(counts) {
		out = append(out, &measurepb.UsageEvidenceCount{Source: name, Units: int64(counts[name])})
	}
	return out
}

func protoRange(r outcomeledger.Range) *measurepb.UsageRange {
	return &measurepb.UsageRange{Available: r.Available, Value: r.Value, Low: r.Low, High: r.High, Samples: int64(r.Samples)}
}

func protoOutcomeRow(row outcomeledger.Row) *measurepb.UsageOutcomeRow {
	return &measurepb.UsageOutcomeRow{
		Key: row.Key, Runner: row.Runner, Status: row.Status, StatusReason: row.Reason, Units: int64(row.Units), Runs: int64(row.Runs),
		Stratum: row.Stratum, Completed: int64(row.Completed), PointsPerCompleted: protoRange(row.PointsPerCompleted), CompletedBaseline: row.CompletedBaseline, CompletedReading: row.CompletedReading, CheaperByCompleted: protoRange(row.CheaperByCompleted),
		ModelPriceRatio: row.ModelPriceRatio, ModelPriceSource: row.ModelPriceSource,
		Accepted: int64(row.Accepted), Corrected: int64(row.Corrected), Failed: int64(row.Failed), Abandoned: int64(row.Abandoned), Unknown: int64(row.Unknown),
		UnknownRate: row.UnknownRate, ReworkRate: protoRange(row.ReworkRate), PointsTotal: row.PointsTotal, PointsKnown: row.PointsKnown,
		PointsPerAccepted: protoRange(row.PointsPerAccepted), ConsumptionPerAccepted: protoRange(row.ConsumptionPerAccepted),
		PointsPerRun: protoRange(row.PointsPerRun), ConsumptionPerRun: protoRange(row.ConsumptionPerRun), ActiveMinutesPerRun: protoRange(row.ActiveMinutesPerRun),
		WallHoursMean: row.WallHoursMean, WallHoursMedian: row.WallHoursMedian, RunsOver_24H: int64(row.RunsOver24h), IdleShare: row.IdleShare,
		Baseline: row.Baseline, Verdict: row.Verdict, CheaperBy: protoRange(row.CheaperBy), ConsumptionRatio: protoRange(row.ConsumptionRatio), RunLengthRatio: protoRange(row.RunLengthRatio),
		BreakEvenRatio: row.BreakEvenRatio, PriceSource: row.PriceSource, Evidence: protoEvidence(row.Evidence),
	}
}
