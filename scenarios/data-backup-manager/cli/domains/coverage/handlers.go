package coverage

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage"
	coverageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage/coverage_v1connect"
	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/sources"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client coverageconnect.CoverageServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: coverageconnect.NewCoverageServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) report(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCoverageReport(context.Background(), connect.NewRequest(&coveragev1.GetCoverageReportRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get coverage report", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Report == nil {
		return fmt.Errorf("server returned no coverage report")
	}
	rep := resp.Msg.Report
	s := rep.Summary

	// Decision-oriented ordering: unregistered defaults first (the action), then
	// sensitive review, then registered targets with their planned/verified
	// posture last.
	results := make([]string, 0, len(rep.RecommendedTargets)+len(rep.SensitiveTargets)+len(rep.RegisteredTargets)+3)

	results = append(results, fmt.Sprintf("Unregistered recommended defaults (%d):", len(rep.RecommendedTargets)))
	if len(rep.RecommendedTargets) == 0 {
		results = append(results, "  (none — default coverage complete)")
	}
	for _, t := range rep.RecommendedTargets {
		results = append(results, "  "+formatSuggestion(t))
	}

	results = append(results, fmt.Sprintf("Sensitive — review only (%d):", len(rep.SensitiveTargets)))
	if len(rep.SensitiveTargets) == 0 {
		results = append(results, "  (none)")
	}
	for _, t := range rep.SensitiveTargets {
		results = append(results, "  "+formatSuggestion(t))
	}

	results = append(results, fmt.Sprintf("Registered targets (%d):", len(rep.RegisteredTargets)))
	if len(rep.RegisteredTargets) == 0 {
		results = append(results, "  (none registered yet)")
	}
	for _, t := range rep.RegisteredTargets {
		results = append(results, "  "+formatRegistered(t))
	}

	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Coverage: %d registered, %d recommended-unregistered, %d sensitive, %d planned, %d backed-up, %d verified.",
				s.RegisteredCount, s.RecommendedCount, s.SensitiveCount, s.PlannedCount, s.BackedUpCount, s.VerifiedCount),
			coverageStatusLine(s),
		},
		ResultsHeading: "Coverage detail",
		Results:        results,
		RetrievalHints: nextSteps(s),
	})
}

func (h *handlers) acceptDefaults(ctx cliapp.RunContext) error {
	includeSensitive := ctx.BoolFlag("include-sensitive")
	dryRun := ctx.BoolFlag("dry-run")

	resp, err := h.client.AcceptDefaultTargets(context.Background(), connect.NewRequest(&coveragev1.AcceptDefaultTargetsRequest{
		IncludeSensitive: includeSensitive,
		DryRun:           dryRun,
	}))
	if err != nil {
		return cliapp.WrapAPIError("accept default coverage targets", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no accept-defaults response")
	}
	msg := resp.Msg

	verb := "Registered"
	if msg.DryRun {
		verb = "Would register"
	}
	changes := make([]string, 0, len(msg.Accepted)+len(msg.SkippedSensitive)+len(msg.Failed))
	for _, a := range msg.Accepted {
		id := a.TargetId
		if id == "" {
			id = "(dry-run)"
		}
		changes = append(changes, fmt.Sprintf("+ %s/%s [kind=%s locator=%s] -> %s", a.Owner, a.Name, kindLabel(a.SourceKind), a.Locator, id))
	}
	for _, sk := range msg.SkippedSensitive {
		changes = append(changes, fmt.Sprintf("⚠ skipped (sensitive) %s/%s — pass --include-sensitive to register", sk.Owner, sk.Name))
	}
	for _, f := range msg.Failed {
		changes = append(changes, fmt.Sprintf("✗ failed %s/%s — %s", f.Owner, f.Name, f.Message))
	}

	result := []string{fmt.Sprintf("%s %d target(s); skipped %d sensitive; %d failed.", verb, len(msg.Accepted), len(msg.SkippedSensitive), len(msg.Failed))}
	if msg.DryRun {
		result = append(result, "Dry run — nothing was registered. Re-run without --dry-run to apply.")
	}

	next := []string{"`coverage report` — re-check coverage"}
	if !msg.DryRun && len(msg.Accepted) > 0 {
		next = append(next, "`plans create --name <n> --targets <ids> --destinations <ids>` — bind the new targets into a plan")
	}
	if len(msg.SkippedSensitive) > 0 {
		next = append(next, "`coverage accept-defaults --include-sensitive` — register sensitive credential targets too")
	}

	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
		Result:      result,
		Changes:     changes,
		NextCommand: next,
	})
}

func coverageStatusLine(s *coveragev1.CoverageSummary) string {
	sensitive := "Sensitive coverage complete — no unregistered sensitive entries."
	if s.HasSensitiveUnreviewed {
		sensitive = "Sensitive coverage incomplete — review the sensitive entries before registering them."
	}
	switch {
	case !s.DefaultCoverageComplete:
		return "⚠ Default coverage INCOMPLETE — recommended targets are unregistered. Run `coverage accept-defaults`. " + sensitive
	case s.HasUnplannedRegisteredTargets:
		return "Default coverage complete, but some registered targets are bound to no plan. " + sensitive
	case s.HasUnverifiedTargets:
		return "Default coverage complete; some targets have never been verify-restored. " + sensitive
	default:
		return "Default coverage complete; all registered targets are planned and verified. " + sensitive
	}
}

func nextSteps(s *coveragev1.CoverageSummary) []string {
	steps := make([]string, 0, 3)
	if !s.DefaultCoverageComplete {
		steps = append(steps,
			"`coverage accept-defaults --dry-run` — preview default registration",
			"`coverage accept-defaults` — register recommended non-sensitive targets",
		)
	}
	if s.HasSensitiveUnreviewed {
		steps = append(steps, "`coverage accept-defaults --include-sensitive` — register sensitive targets after review")
	}
	steps = append(steps, "`plans create --name <n> --targets <ids> --destinations <ids>` — protect targets with a plan")
	return steps
}

func formatSuggestion(s *coveragev1.SuggestedTarget) string {
	if s == nil {
		return "(nil)"
	}
	line := fmt.Sprintf("%s — %s/%s [kind=%s locator=%s size=%s] — %s",
		s.Id, s.Owner, s.Name, kindLabel(s.SourceKind), s.Locator, humanizeBytes(s.ApproxBytes), s.Rationale)
	if s.Sensitive {
		warning := s.Warning
		if warning == "" {
			warning = "includes credentials/tokens — review before backing up"
		}
		line += "  ⚠ SENSITIVE: " + warning
	}
	return line
}

func formatRegistered(t *coveragev1.RegisteredTarget) string {
	if t == nil {
		return "(nil)"
	}
	planned := "UNPLANNED"
	if t.Planned {
		planned = "planned"
	}
	return fmt.Sprintf("%s — %s/%s [kind=%s locator=%s] %s backed-up=%s verified=%s",
		t.Id, t.Owner, t.Name, kindLabel(t.SourceKind), t.Locator, planned,
		formatTimestamp(t.LastSuccessAt), formatTimestamp(t.LastVerifiedAt))
}

func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return "never"
	}
	t := ts.AsTime()
	if t.IsZero() {
		return "never"
	}
	return t.UTC().Format(time.RFC3339)
}

func kindLabel(k sourcesv1.SourceKind) string {
	switch k {
	case sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM:
		return "filesystem"
	case sourcesv1.SourceKind_SOURCE_KIND_SQLITE:
		return "sqlite"
	case sourcesv1.SourceKind_SOURCE_KIND_POSTGRES:
		return "postgres"
	case sourcesv1.SourceKind_SOURCE_KIND_REDIS:
		return "redis"
	case sourcesv1.SourceKind_SOURCE_KIND_QDRANT:
		return "qdrant"
	case sourcesv1.SourceKind_SOURCE_KIND_OBJECT_STORAGE:
		return "object-storage"
	default:
		return "unspecified"
	}
}

func humanizeBytes(b int64) string {
	if b <= 0 {
		return "?"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
