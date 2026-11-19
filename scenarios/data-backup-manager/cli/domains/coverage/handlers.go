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
	results := make([]string, 0, len(rep.RecommendedTargets)+len(rep.SensitiveTargets)+len(rep.ReacquirableTargets)+len(rep.RegisteredTargets)+4)

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
	results = append(results, fmt.Sprintf("Reacquirable — excluded from backup coverage and bulk registration (%d):", len(rep.ReacquirableTargets)))
	if len(rep.ReacquirableTargets) == 0 {
		results = append(results, "  (none)")
	}
	for _, t := range rep.ReacquirableTargets {
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
			fmt.Sprintf("Coverage: %d registered, %d recommended-unregistered, %d sensitive, %d planned, %d backed-up, %d verified; %d destination copies backed-up, %d targets backed-up on every destination, %d targets verified on every destination, %d targets on multiple destinations, %d protected by two or more devices.",
				s.RegisteredCount, s.RecommendedCount, s.SensitiveCount, s.PlannedCount, s.BackedUpCount, s.VerifiedCount,
				s.DestinationCopyCount, s.AllDestinationBackedUpCount, s.AllDestinationVerifiedCount, s.MultiDestinationTargetCount, s.ProtectedTargetCount),
			coverageStatusLine(s),
			driveLossLine(s),
			reacquisitionStatusLine(rep.ReacquirableTargets),
		},
		ResultsHeading: "Coverage detail",
		Results:        results,
		RetrievalHints: nextSteps(s),
	})
}

func (h *handlers) acceptDefaults(ctx cliapp.RunContext) error {
	dryRun := ctx.BoolFlag("dry-run")

	resp, err := h.client.AcceptDefaultTargets(context.Background(), connect.NewRequest(&coveragev1.AcceptDefaultTargetsRequest{
		IncludeSensitive: false,
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
		changes = append(changes, fmt.Sprintf("⚠ skipped (sensitive) %s/%s — use `coverage review-sensitive` for the verified operator decision", sk.Owner, sk.Name))
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
		next = append(next, "`coverage review-sensitive` — raise one verified review for sensitive targets")
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
	case s.AllDestinationBackedUpCount < s.RegisteredCount:
		return "Default coverage complete, but some registered targets have no successful copy at every enabled destination. " + sensitive
	case s.AllDestinationVerifiedCount < s.RegisteredCount:
		return "Some enabled target destinations lack a verified scratch restore. A verified restore is what proves the backup is usable. " + sensitive
	default:
		return "Every registered target has a successful copy. Single-device copies are a warning, not a failure: add a second storage device for drive-loss protection; off-host placement is a separate, stronger level. " + sensitive
	}
}

// driveLossLine says plainly what a backup copy does and does not protect
// against. A copy on the same drive as its source (the local baseline-safety
// destination) survives a deleted or corrupted file; it does not survive losing
// the drive. Only a target with an independent second copy does.
func driveLossLine(s *coveragev1.CoverageSummary) string {
	// The server classifies every target once. Reporting the same counts keeps
	// the CLI from re-deriving protection (which previously mis-counted a
	// target whose second destination had failed but whose first copy was good).
	return fmt.Sprintf("Drive loss: %d of %d targets are protected by two or more devices; %d have a single-device (or same-device) copy — a warning, not a failure; %d have no successful copy. A second device protects against losing one drive; off-host placement protects against losing this host.",
		s.ProtectedTargetCount, s.RegisteredCount, s.SingleCopyTargetCount, s.UnprotectedTargetCount)
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
		steps = append(steps, "`coverage review-sensitive` — raise one verified review for sensitive targets")
	}
	if s.ProtectedTargetCount < s.RegisteredCount {
		steps = append(steps,
			"`destinations create --name second-drive --location /media/<drive>/vrooli-backups` — add another storage device",
			"`plans add-destination --destination <new-id> --apply` — attach it to every enabled plan at once",
		)
	}
	steps = append(steps, "`plans create --name <n> --targets <ids> --destinations <ids>` — protect targets with a plan")
	return steps
}

func (h *handlers) reviewSensitive(ctx cliapp.RunContext) error {
	resp, err := h.client.RequestSensitiveReview(context.Background(), connect.NewRequest(&coveragev1.RequestSensitiveReviewRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("request sensitive coverage review", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no sensitive review response")
	}
	if !resp.Msg.Pending {
		// A standing rule covers the candidates: nothing needed asking.
		return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
			Result:      []string{fmt.Sprintf("The standing rule is on, so nothing was asked. It just added %d place(s) that hold secrets.", resp.Msg.CandidateCount)},
			NextCommand: []string{"`coverage sensitive-policy` — see what the rule has added and kept out"},
		})
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Raised sensitive coverage review %s for %d current target(s). No target was registered by this command.", resp.Msg.AskId, resp.Msg.CandidateCount)},
		NextCommand: []string{"Open the Notification Hub decision and record one disposition for each listed target.", "`coverage report` — re-check coverage after the callback completes"},
	})
}

func (h *handlers) sensitivePolicy(ctx cliapp.RunContext) error {
	resp, err := h.client.GetSensitivePolicy(context.Background(), connect.NewRequest(&coveragev1.GetSensitivePolicyRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get sensitive coverage policy", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Policy == nil {
		return fmt.Errorf("server returned no sensitive coverage policy")
	}
	p := resp.Msg.Policy
	summary := policySummary(p)
	results := make([]string, 0, len(p.Excluded)+len(p.Unreported)+len(p.History))
	results = append(results, fmt.Sprintf("Kept out of the rule (%d):", len(p.Excluded)))
	for _, e := range p.Excluded {
		results = append(results, fmt.Sprintf("  %s/%s — %s (%s, %s)", e.Owner, e.Name, e.Reason, e.ExcludedBy, formatTimestamp(e.ExcludedAt)))
	}
	results = append(results, fmt.Sprintf("Added since the last weekly note (%d):", len(p.Unreported)))
	for _, a := range p.Unreported {
		results = append(results, fmt.Sprintf("  %s/%s [%s] %s", a.Owner, a.Name, a.Group, humanizeBytes(a.ApproxBytes)))
	}
	results = append(results, fmt.Sprintf("History (%d):", len(p.History)))
	for _, e := range p.History {
		results = append(results, fmt.Sprintf("  %s %s %s — %s", formatTimestamp(e.At), e.Kind, e.By, e.Detail))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Standing rule detail",
		Results:        results,
		RetrievalHints: []string{"`coverage revoke-sensitive-policy` — turn the rule off", "`coverage exclude-sensitive --owner O --name N --reason R` — keep one item out"},
	})
}

func policySummary(p *coveragev1.SensitivePolicy) string {
	switch {
	case p.Active:
		basis := "chosen from the decision's menu"
		if p.GrantBasis == "free_text" {
			basis = "given in their own words"
		}
		return fmt.Sprintf("Standing rule ON: consent from %s (%s via %s) on %s, %s, recorded by decision %s. It has added %d place(s).", p.GrantedBy, p.GrantedByIdentityKind, p.GrantedByIdentitySource, formatTimestamp(p.GrantedAt), basis, p.GrantAskId, p.AddedTotal)
	case p.RevokedAt != nil:
		return fmt.Sprintf("Standing rule OFF: turned off by %s on %s. New places that hold secrets wait for a decision.", p.RevokedBy, formatTimestamp(p.RevokedAt))
	}
	return "No standing rule: places that hold secrets wait for a decision."
}

func (h *handlers) revokeSensitivePolicy(ctx cliapp.RunContext) error {
	resp, err := h.client.RevokeSensitivePolicy(context.Background(), connect.NewRequest(&coveragev1.RevokeSensitivePolicyRequest{RevokedBy: "operator via CLI"}))
	if err != nil {
		return cliapp.WrapAPIError("revoke sensitive coverage policy", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Policy == nil {
		return fmt.Errorf("server returned no sensitive coverage policy")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{policySummary(resp.Msg.Policy), "Items the rule already added stay backed up."},
		NextCommand: []string{"`coverage review-sensitive` — raise a decision for any place that holds secrets and is not backed up"},
	})
}

func (h *handlers) adoptSensitiveConsent(ctx cliapp.RunContext) error {
	resp, err := h.client.AdoptSensitiveConsent(context.Background(), connect.NewRequest(&coveragev1.AdoptSensitiveConsentRequest{HubAskId: ctx.Flag("ask")}))
	if err != nil {
		return cliapp.WrapAPIError("adopt sensitive coverage consent", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Policy == nil {
		return fmt.Errorf("server returned no sensitive coverage policy")
	}
	result := []string{policySummary(resp.Msg.Policy), fmt.Sprintf("Registered %d place(s) that hold secrets; %d failed and will be retried by the next sweep.", resp.Msg.RegisteredCount, resp.Msg.FailedCount)}
	if resp.Msg.Policy.GrantText != "" {
		result = append(result, "The operator's words, kept verbatim: "+resp.Msg.Policy.GrantText)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      result,
		NextCommand: []string{"`coverage sensitive-policy` — see the consent record", "`coverage revoke-sensitive-policy` — turn the rule off"},
	})
}

func (h *handlers) excludeSensitive(ctx cliapp.RunContext) error {
	resp, err := h.client.ExcludeSensitiveTarget(context.Background(), connect.NewRequest(&coveragev1.ExcludeSensitiveTargetRequest{
		Owner: ctx.Flag("owner"), Name: ctx.Flag("name"), Reason: ctx.Flag("reason"), ExcludedBy: "operator via CLI",
	}))
	if err != nil {
		return cliapp.WrapAPIError("exclude sensitive coverage target", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Policy == nil {
		return fmt.Errorf("server returned no sensitive coverage policy")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("%s/%s is kept out of the standing rule. If it is already registered, remove its target separately.", ctx.Flag("owner"), ctx.Flag("name"))},
		NextCommand: []string{"`coverage sensitive-policy` — see what the rule keeps out"},
	})
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
	if s.Reacquirable {
		line += "  REACQUIRABLE — excluded from backup coverage and bulk registration"
		line += formatReacquisitionProof(s.ReacquisitionCommand, s.ReacquisitionEvidence, s.ReacquisitionNeedsOperator, s.ReacquisitionAuthReason)
		if s.BudgetMaxBytes != "" {
			line += " max_bytes=" + s.BudgetMaxBytes
		}
		if s.BudgetMaxAge != "" {
			line += " max_age=" + s.BudgetMaxAge
		}
		if s.Retention != "" {
			line += " retention=" + s.Retention
		}
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
	line := fmt.Sprintf("%s — %s/%s [kind=%s locator=%s] %s backed-up=%s verified=%s",
		t.Id, t.Owner, t.Name, kindLabel(t.SourceKind), t.Locator, planned,
		formatTimestamp(t.LastSuccessAt), formatTimestamp(t.LastVerifiedAt))
	for _, destination := range t.Destinations {
		line += fmt.Sprintf("\n    destination=%s backed-up=%s verified=%s", destination.DestinationId,
			formatTimestamp(destination.LastSuccessAt), formatTimestamp(destination.LastVerifiedAt))
	}
	topology := "protection=unknown"
	switch t.ProtectionLevel {
	case "multi_device":
		topology = fmt.Sprintf("protection=two-or-more-devices (%d)", t.ProtectionDeviceCount)
	case "single_copy":
		topology = "protection=single-device (warning)"
	case "unprotected":
		topology = "protection=none (no successful copy)"
	case "identity_unknown":
		topology = "protection=unknown (device identity unproven)"
	}
	line += "\n    " + topology
	if t.Reacquirable {
		line += formatReacquisitionProof(t.ReacquisitionCommand, t.ReacquisitionEvidence, t.ReacquisitionNeedsOperator, t.ReacquisitionAuthReason)
	}
	for _, warning := range t.IndependenceWarnings {
		line += "\n    topology-warning=" + warning
	}
	return line
}

func formatReacquisitionProof(command, evidence string, needsOperator bool, authReason string) string {
	if command == "" || evidence == "" {
		return " reacquisition-proof=INCOMPLETE"
	}
	line := " reacquire=" + command + " evidence=" + evidence
	if needsOperator {
		line += " authorization-required=" + authReason
	}
	return line
}

func reacquisitionStatusLine(suggestions []*coveragev1.SuggestedTarget) string {
	complete, unbounded := 0, 0
	for _, suggestion := range suggestions {
		if suggestion == nil {
			continue
		}
		if suggestion.ReacquisitionCommand != "" && suggestion.ReacquisitionEvidence != "" {
			complete++
		}
		if suggestion.BudgetMaxBytes == "" && suggestion.BudgetMaxAge == "" {
			unbounded++
		}
	}
	return fmt.Sprintf("Reacquisition evidence: %d/%d suggestions have both an owner command and evidence; %d lack both a byte and age bound.", complete, len(suggestions), unbounded)
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
	case sourcesv1.SourceKind_SOURCE_KIND_WORKSPACE_CHECKPOINT:
		return "workspace-checkpoint"
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
