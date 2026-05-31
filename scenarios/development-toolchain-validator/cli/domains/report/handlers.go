package report

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	reportv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/report"
	reportconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/report/report_v1connect"
	vrv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client reportconnect.ReportServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: reportconnect.NewReportServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) goldenSummary(ctx cliapp.RunContext) error {
	slug := ctx.Positional("slug")
	resp, err := h.client.GetGoldenSummary(context.Background(), connect.NewRequest(&reportv1.GetGoldenSummaryRequest{GoldenSlug: slug}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("golden summary %q", slug), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Summary == nil {
		return fmt.Errorf("server returned no summary")
	}
	s := resp.Msg.Summary
	lines := []string{
		fmt.Sprintf("Skills: %d", len(s.SkillVerdicts)),
		fmt.Sprintf("Tools:  %d", len(s.ToolVerdicts)),
		fmt.Sprintf("Stale:  %d", s.StaleCount),
	}
	for _, v := range s.SkillVerdicts {
		lines = append(lines, "  "+formatTupleVerdict(v))
	}
	for _, v := range s.ToolVerdicts {
		lines = append(lines, "  "+formatTupleVerdict(v))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Summary for golden %q.", slug)},
		ResultsHeading: "Verdicts",
		Results:        lines,
	})
}

func (h *handlers) tupleHistory(ctx cliapp.RunContext) error {
	skill := strings.TrimSpace(ctx.Flag("skill"))
	tool := strings.TrimSpace(ctx.Flag("tool"))
	if (skill == "") == (tool == "") {
		return fmt.Errorf("exactly one of --skill or --tool must be provided")
	}
	kind := vrv1.TupleKind_TUPLE_KIND_SKILL
	subject := skill
	if tool != "" {
		kind = vrv1.TupleKind_TUPLE_KIND_TOOL
		subject = tool
	}
	pageSize := 0
	if raw := strings.TrimSpace(ctx.Flag("page-size")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			pageSize = n
		}
	}
	resp, err := h.client.GetTupleHistory(context.Background(), connect.NewRequest(&reportv1.GetTupleHistoryRequest{
		TupleKind: kind, SubjectId: subject, GoldenSlug: ctx.Flag("golden"),
		PageSize: int32(pageSize), PageToken: ctx.Flag("page-token"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("tuple history", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.History == nil {
		return fmt.Errorf("server returned no history")
	}
	hist := resp.Msg.History
	results := make([]string, 0, len(hist.Records))
	for _, r := range hist.Records {
		results = append(results, formatRecord(r))
	}
	hints := []string{}
	if hist.NextPageToken != "" {
		hints = append(hints, fmt.Sprintf("`report tuple-history ... --page-token %s`", hist.NextPageToken))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d record(s) for %s/%s/%s.", len(hist.Records), kindLabel(kind), subject, ctx.Flag("golden"))},
		ResultsHeading: "History",
		Results:        results,
		RetrievalHints: hints,
	})
}

func (h *handlers) coverage(ctx cliapp.RunContext) error {
	slug := ctx.Positional("slug")
	resp, err := h.client.GetCoverage(context.Background(), connect.NewRequest(&reportv1.GetCoverageRequest{GoldenSlug: slug}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("coverage %q", slug), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Coverage == nil {
		return fmt.Errorf("server returned no coverage")
	}
	cov := resp.Msg.Coverage
	results := make([]string, 0, len(cov.Rows))
	for _, r := range cov.Rows {
		results = append(results, fmt.Sprintf("%s/%s — verdict=%s manifest=%v stale=%v",
			kindLabel(r.TupleKind), r.SubjectId, verdictLabel(r.Verdict), r.HasManifest, r.Stale))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Coverage for golden %q (%d row(s)).", slug, len(cov.Rows))},
		ResultsHeading: "Coverage",
		Results:        results,
	})
}

func (h *handlers) skillFitness(ctx cliapp.RunContext) error {
	skillID := ctx.Positional("skill_id")
	resp, err := h.client.GetSkillFitness(context.Background(), connect.NewRequest(&reportv1.GetSkillFitnessRequest{SkillId: skillID}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("skill fitness %q", skillID), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Fitness == nil {
		return fmt.Errorf("server returned no fitness")
	}
	f := resp.Msg.Fitness
	lines := []string{
		fmt.Sprintf("Verdict:       %s", fitnessVerdictLabel(f.Verdict)),
		fmt.Sprintf("Latest:        %s", verdictLabel(f.LatestVerdict)),
		fmt.Sprintf("Runs:          %d (pass %d, mutation %d, run-failure %d, tool-failure %d)",
			f.TotalRuns, f.PassCount, f.UnexpectedMutationCount, f.RunFailureCount, f.ToolFailureCount),
		fmt.Sprintf("Pass rate:     %.2f", f.PassRate),
		fmt.Sprintf("Avg tokens:    %.0f (total %d)", f.AvgTokens, f.TotalTokens),
		fmt.Sprintf("Avg cost:      %.0f µ$ (total %d)", f.AvgCostUsdMicro, f.TotalCostUsdMicro),
		fmt.Sprintf("Avg duration:  %.0fms (total %dms)", f.AvgDurationMs, f.TotalDurationMs),
		fmt.Sprintf("Convergence:   %.2f (%d unique diff(s))", f.ConvergenceRatio, f.UniqueDiffHashes),
		fmt.Sprintf("Any stale:     %v", f.AnyStale),
	}
	for _, slug := range sortedGoldenSlugs(f.ByGolden) {
		snap := f.ByGolden[slug]
		lines = append(lines, fmt.Sprintf("  %s — verdict=%s runs=%d stale=%v",
			slug, verdictLabel(snap.LatestVerdict), snap.RunCount, snap.Stale))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fitness for skill %q.", skillID)},
		ResultsHeading: "Fitness",
		Results:        lines,
	})
}

func sortedGoldenSlugs(m map[string]*reportv1.GoldenSkillSnapshot) []string {
	out := make([]string, 0, len(m))
	for slug := range m {
		out = append(out, slug)
	}
	sort.Strings(out)
	return out
}

func fitnessVerdictLabel(v reportv1.SkillFitnessVerdict) string {
	switch v {
	case reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_UNKNOWN:
		return "unknown"
	case reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_GREEN:
		return "green"
	case reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_YELLOW:
		return "yellow"
	case reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_RED:
		return "red"
	default:
		return "unspecified"
	}
}

func formatTupleVerdict(v *reportv1.TupleVerdict) string {
	if v == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s/%s — verdict=%s stale=%v record=%s",
		kindLabel(v.TupleKind), v.SubjectId, verdictLabel(v.LatestVerdict), v.Stale, v.LatestRecordId)
}

func formatRecord(r *vrv1.ValidationRecord) string {
	if r == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — verdict=%s duration=%dms err=%q",
		r.Id, verdictLabel(r.Verdict), r.DurationMs, r.ErrorMessage)
}

func kindLabel(k vrv1.TupleKind) string {
	switch k {
	case vrv1.TupleKind_TUPLE_KIND_SKILL:
		return "skill"
	case vrv1.TupleKind_TUPLE_KIND_TOOL:
		return "tool"
	default:
		return "unspecified"
	}
}

func verdictLabel(v vrv1.Verdict) string {
	switch v {
	case vrv1.Verdict_VERDICT_PASS:
		return "pass"
	case vrv1.Verdict_VERDICT_UNEXPECTED_MUTATION:
		return "unexpected_mutation"
	case vrv1.Verdict_VERDICT_RUN_FAILURE:
		return "run_failure"
	case vrv1.Verdict_VERDICT_TOOL_FAILURE:
		return "tool_failure"
	default:
		return "none"
	}
}
