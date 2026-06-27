package conflicts

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	conflictsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts"
	conflictsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts/conflicts_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
	"google.golang.org/protobuf/proto"

	"architecture-cartographer/cli/internal/attestrender"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each
// RunCtx-func has typed access to the Connect client without re-resolving it.
//
// The conflicts domain is DETECTION-ONLY: detect / list / show / explain /
// validate read the current photograph. Walking findings through a
// lifecycle lives in the `campaign` command group (CampaignService).
type handlers struct {
	core   *cliapp.ScenarioApp
	client conflictsconnect.ConflictsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: conflictsconnect.NewConflictsServiceClient(httpClient, baseURL),
	}
}

// detect runs every registered Detector against the scenario's current
// snapshot + manifest, persists the emitted conflicts, and lists them.
func (h *handlers) detect(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.DetectConflicts(context.Background(), connect.NewRequest(&conflictsv1.DetectConflictsRequest{
		Scenario:       scenario,
		SnapshotId:     ctx.Flag("snapshot-id"),
		IdempotencyKey: ctx.Flag("idempotency-key"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("detect conflicts for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no detect response")
	}
	return renderConflictList(ctx, resp.Msg, resp.Msg.GetConflicts(),
		fmt.Sprintf("Detected %d conflict(s) for %q.", len(resp.Msg.GetConflicts()), scenario),
		fmt.Sprintf("`campaign create %s --from-audit <report.json>` to start tracking them toward zero.", scenario))
}

// list paginates the persisted conflicts for a scenario.
func (h *handlers) list(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	req := &conflictsv1.ListConflictsRequest{
		Scenario:  scenario,
		Types:     cliutil.ParseCSV(ctx.Flag("type")),
		PageToken: ctx.Flag("page-token"),
	}
	if raw := strings.TrimSpace(ctx.Flag("page-size")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return fmt.Errorf("--page-size must be a 32-bit integer: %w", err)
		}
		if n <= 0 {
			return fmt.Errorf("--page-size must be greater than zero")
		}
		req.PageSize = int32(n)
	}
	resp, err := h.client.ListConflicts(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("list conflicts for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no list response")
	}
	summary := fmt.Sprintf("%d conflict(s) for %q.", len(resp.Msg.GetConflicts()), scenario)
	hint := "`conflicts show <id>` for one conflict's evidence and suggested fixes."
	if tok := resp.Msg.GetNextPageToken(); tok != "" {
		hint = fmt.Sprintf("More results: `conflicts list %s --page-token %s`.", scenario, tok)
	}
	return renderConflictList(ctx, resp.Msg, resp.Msg.GetConflicts(), summary, hint)
}

// show returns one conflict by id (the stable_id) with the operator-
// focused "what / where / why / next" layout. The positional argument
// accepts either the v0.2 csid: stable_id, a bare 16-hex short form
// (CLI normalizes to csid:<hex>), or the legacy UUID; the API only
// accepts the canonical csid: form, so normalization happens here.
func (h *handlers) show(ctx cliapp.RunContext) error {
	id := normalizeConflictID(ctx.Positional("id"))
	resp, err := h.client.GetConflict(context.Background(), connect.NewRequest(&conflictsv1.GetConflictRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get conflict %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetConflict() == nil {
		return fmt.Errorf("server returned no conflict")
	}
	c := resp.Msg.GetConflict()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Conflict %s.", c.GetId())},
		ResultsHeading: "Detail",
		Results:        conflictDetailLines(c),
		RetrievalHints: []string{
			fmt.Sprintf("`campaign create %s --from-audit <report.json>` to track this and its siblings toward zero.", c.GetScenario()),
		},
	})
}

// validate re-lists the current detected conflicts and reports a
// clean/dirty gate (clean ↔ zero error-severity, suppressed excluded).
func (h *handlers) validate(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.ValidateConflicts(context.Background(), connect.NewRequest(&conflictsv1.ValidateConflictsRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate conflicts for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validate response")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	outstanding := resp.Msg.GetConflicts()
	status := fmt.Sprintf("%q is cartographer-clean: zero error-severity conflicts outstanding.", scenario)
	if !resp.Msg.GetClean() {
		status = fmt.Sprintf("%q is NOT clean: %d outstanding conflict(s).", scenario, len(outstanding))
	}
	var triage []cliapp.TriageGroup
	if len(outstanding) > 0 {
		items := make([]string, 0, len(outstanding))
		for _, c := range outstanding {
			items = append(items, conflictLine(c))
		}
		triage = append(triage, cliapp.TriageGroup{Heading: "Outstanding conflicts", Items: items})
	}
	return ctx.RenderOperational(cliapp.OperationalReport{
		Status: []string{status},
		Triage: triage,
		NextSteps: []string{
			fmt.Sprintf("`campaign create %s --from-audit <report.json>` to track the open set toward zero.", scenario),
		},
	})
}

// detectors lists the registered Detector plug-ins.
func (h *handlers) detectors(ctx cliapp.RunContext) error {
	resp, err := h.client.ListDetectors(context.Background(), connect.NewRequest(&conflictsv1.ListDetectorsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list detectors", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no detectors response")
	}
	results := make([]string, 0, len(resp.Msg.GetDetectors()))
	for _, d := range resp.Msg.GetDetectors() {
		results = append(results, fmt.Sprintf("%s (%s) — emits: %s — %s",
			d.GetName(), d.GetStability(), strings.Join(d.GetEmitsTypes(), ", "), d.GetDescription()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d registered detector(s).", len(resp.Msg.GetDetectors()))},
		ResultsHeading: "Detectors",
		Results:        results,
	})
}

// resolvers lists the registered Resolver plug-ins.
func (h *handlers) resolvers(ctx cliapp.RunContext) error {
	resp, err := h.client.ListResolvers(context.Background(), connect.NewRequest(&conflictsv1.ListResolversRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list resolvers", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no resolvers response")
	}
	results := make([]string, 0, len(resp.Msg.GetResolvers()))
	for _, r := range resp.Msg.GetResolvers() {
		kinds := make([]string, 0, len(r.GetHandlesKinds()))
		for _, k := range r.GetHandlesKinds() {
			kinds = append(kinds, fixKindName(k))
		}
		deferred := ""
		if r.GetRequiresApply() {
			deferred = " [requires apply]"
		}
		results = append(results, fmt.Sprintf("%s (%s) — handles: %s%s — %s",
			r.GetName(), r.GetStability(), strings.Join(kinds, ", "), deferred, r.GetDescription()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d registered resolver(s).", len(resp.Msg.GetResolvers()))},
		ResultsHeading: "Resolvers",
		Results:        results,
	})
}

// -------------------------- render helpers --------------------------

// conflictDetailLines renders the operator-focused "what / where / why /
// next" layout shared by show and explain.
func conflictDetailLines(c *sharedv1.Conflict) []string {
	lines := []string{
		fmt.Sprintf("what     : %s (%s)", c.GetType(), c.GetSubtype()),
		fmt.Sprintf("severity : %s", severityName(c.GetSeverity())),
		fmt.Sprintf("class    : %s", findingClassName(c.GetFindingClass())),
		fmt.Sprintf("stable_id: %s", c.GetStableId()),
	}
	if att := c.GetAttestation(); att != nil {
		lines = append(lines, fmt.Sprintf("basis    : %s (%s)", attestrender.Basis(att.GetBasis()), attestrender.Sufficiency(att.GetSufficiency())))
	}
	if iid := c.GetInstanceId(); iid != "" {
		lines = append(lines, fmt.Sprintf("instance : %s (this run)", iid))
	}
	if locs := c.GetLocations(); len(locs) > 0 {
		lines = append(lines, "where    : "+strings.Join(locs, ", "))
	}
	if doms := c.GetDomains(); len(doms) > 0 {
		lines = append(lines, "domains  : "+strings.Join(doms, ", "))
	}
	if c.GetSuppressed() {
		lines = append(lines, fmt.Sprintf("suppress : sanctioned (%s)", c.GetSuppressionReason()))
	}
	lines = append(lines, "", "why:")
	for _, e := range c.GetEvidence() {
		lines = append(lines, fmt.Sprintf("  - [%s] %s", e.GetKind(), e.GetSummary()))
	}
	if fixes := c.GetSuggestedFixes(); len(fixes) > 0 {
		lines = append(lines, "", "next:")
		for _, f := range fixes {
			lines = append(lines, fmt.Sprintf("  - %s via %s — %s (confidence %.2f)",
				fixKindName(f.GetKind()), f.GetResolver(), f.GetSummary(), f.GetConfidence()))
		}
	}
	return lines
}

// renderConflictList is the shared list-rendering path for detect + list:
// human consumers see one line per conflict; --json consumers see the
// proto-typed wire shape (the full response message).
func renderConflictList(ctx cliapp.RunContext, payload proto.Message, conflicts []*sharedv1.Conflict, summary, hint string) error {
	results := make([]string, 0, len(conflicts))
	for _, c := range conflicts {
		results = append(results, conflictLine(c))
	}
	return cliapp.RenderProtoList(ctx, payload, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Conflicts",
		Results:        results,
		RetrievalHints: []string{hint},
	})
}

func conflictLine(c *sharedv1.Conflict) string {
	domains := strings.Join(c.GetDomains(), ",")
	loc := ""
	if len(c.GetLocations()) > 0 {
		loc = c.GetLocations()[0]
		if len(c.GetLocations()) > 1 {
			loc = fmt.Sprintf("%s (+%d more)", loc, len(c.GetLocations())-1)
		}
	}
	return fmt.Sprintf("%s [%s/%s/%s] %s domain=%s loc=%s",
		c.GetId(), c.GetType(), severityName(c.GetSeverity()), findingClassName(c.GetFindingClass()), c.GetSubtype(), domains, loc)
}

func severityName(s sharedv1.Severity) string {
	switch s {
	case sharedv1.Severity_SEVERITY_INFO:
		return "info"
	case sharedv1.Severity_SEVERITY_WARN:
		return "warn"
	case sharedv1.Severity_SEVERITY_ERROR:
		return "error"
	case sharedv1.Severity_SEVERITY_BLOCKER:
		return "blocker"
	default:
		return "unspecified"
	}
}

func findingClassName(c sharedv1.FindingClass) string {
	switch c {
	case sharedv1.FindingClass_FINDING_CLASS_DETERMINISTIC:
		return "deterministic"
	case sharedv1.FindingClass_FINDING_CLASS_HEURISTIC:
		return "heuristic"
	default:
		return "unspecified"
	}
}

// normalizeConflictID accepts the user's positional ID argument and
// returns the canonical form the API expects. A bare 16-hex string
// (the short form printed elsewhere as csid:<hex>) is prefixed with
// "csid:". Anything else (UUIDs, already-prefixed IDs, malformed
// input) passes through unchanged — the API surfaces the not_found
// error if it doesn't match a row.
func normalizeConflictID(id string) string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, "csid:") {
		return id
	}
	if len(id) == 16 && isHex(id) {
		return "csid:" + id
	}
	return id
}

func isHex(s string) bool {
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

func fixKindName(k sharedv1.FixKind) string {
	switch k {
	case sharedv1.FixKind_FIX_KIND_MOVE_FILE:
		return "move_file"
	case sharedv1.FixKind_FIX_KIND_REASSIGN_DOMAIN:
		return "reassign_domain"
	case sharedv1.FixKind_FIX_KIND_BREAK_CYCLE:
		return "break_cycle"
	case sharedv1.FixKind_FIX_KIND_ADD_DEPENDENCY:
		return "add_dependency"
	case sharedv1.FixKind_FIX_KIND_ADD_TRANSITIONAL:
		return "add_transitional"
	default:
		return "unspecified"
	}
}
