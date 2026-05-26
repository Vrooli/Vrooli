package conflicts

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	conflictsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts"
	conflictsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts/conflicts_v1connect"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each
// RunCtx-func has typed access to the Connect client without re-resolving it.
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
		fmt.Sprintf("`conflicts list %s` shows the persisted set with lifecycle status.", scenario))
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
		n, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("--page-size must be an integer: %w", err)
		}
		req.PageSize = int32(n)
	}
	for _, s := range cliutil.ParseCSV(ctx.Flag("status")) {
		status, ok := parseStatus(s)
		if !ok {
			return fmt.Errorf("unknown --status %q (want one of: %s)", s, strings.Join(statusNames(), ", "))
		}
		req.Statuses = append(req.Statuses, status)
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

// show returns one conflict by id with full evidence + suggested fixes.
func (h *handlers) show(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetConflict(context.Background(), connect.NewRequest(&conflictsv1.GetConflictRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get conflict %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetConflict() == nil {
		return fmt.Errorf("server returned no conflict")
	}
	c := resp.Msg.GetConflict()
	results := []string{conflictLine(c)}
	for _, e := range c.GetEvidence() {
		results = append(results, fmt.Sprintf("  evidence[%s]: %s", e.GetKind(), e.GetSummary()))
	}
	for _, f := range c.GetSuggestedFixes() {
		results = append(results, fmt.Sprintf("  fix[%s] via %s: %s (confidence %.2f)",
			fixKindName(f.GetKind()), f.GetResolver(), f.GetSummary(), f.GetConfidence()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Conflict %s (%s).", c.GetId(), statusName(c.GetStatus()))},
		ResultsHeading: "Detail",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("`conflicts assign %s --domain <domain>` to record the owning domain.", c.GetId()),
			fmt.Sprintf("`conflicts resolve %s` to mark it resolved (add --force to ignore).", c.GetId()),
		},
	})
}

// assign moves a conflict to ASSIGNED with the operator's chosen domain.
func (h *handlers) assign(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.AssignConflict(context.Background(), connect.NewRequest(&conflictsv1.AssignConflictRequest{
		Id:     id,
		Domain: ctx.Flag("domain"),
		Note:   ctx.Flag("note"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("assign conflict %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetConflict() == nil {
		return fmt.Errorf("server returned no conflict")
	}
	c := resp.Msg.GetConflict()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{mutationResult(resp.Msg.GetDryRun(),
			fmt.Sprintf("Assigned conflict %s to %q.", c.GetId(), c.GetAssignedDomain()))},
		Changes:     []string{conflictLine(c)},
		NextCommand: []string{fmt.Sprintf("`conflicts resolve %s` once the move is planned.", c.GetId())},
	})
}

// resolve moves a conflict to RESOLVED (or FORCE_RESOLVED with --force).
func (h *handlers) resolve(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.ResolveConflict(context.Background(), connect.NewRequest(&conflictsv1.ResolveConflictRequest{
		Id:    id,
		Note:  ctx.Flag("note"),
		Force: ctx.BoolFlag("force"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("resolve conflict %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetConflict() == nil {
		return fmt.Errorf("server returned no conflict")
	}
	c := resp.Msg.GetConflict()
	changes := []string{conflictLine(c)}
	if resp.Msg.GetApplyDeferred() {
		changes = append(changes, "Note: the chosen fix requires file movement, which `apply` defers in v0.1 (execution is unimplemented).")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{mutationResult(resp.Msg.GetDryRun(),
			fmt.Sprintf("Resolved conflict %s (%s).", c.GetId(), statusName(c.GetStatus())))},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("`conflicts validate %s` to confirm closure.", c.GetScenario())},
	})
}

// reopen moves a resolved/force-resolved conflict back to DETECTED.
func (h *handlers) reopen(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.ReopenConflict(context.Background(), connect.NewRequest(&conflictsv1.ReopenConflictRequest{
		Id:   id,
		Note: ctx.Flag("note"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("reopen conflict %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetConflict() == nil {
		return fmt.Errorf("server returned no conflict")
	}
	c := resp.Msg.GetConflict()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{mutationResult(resp.Msg.GetDryRun(),
			fmt.Sprintf("Reopened conflict %s (%s).", c.GetId(), statusName(c.GetStatus())))},
		Changes:     []string{conflictLine(c)},
		NextCommand: []string{fmt.Sprintf("`conflicts show %s` to review it again.", c.GetId())},
	})
}

// validate re-runs detection against the current resolution state and
// reports the residual conflict set with a clean/dirty gate.
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
			fmt.Sprintf("`conflicts list %s --status detected` to work the open set.", scenario),
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

// renderConflictList is the shared list-rendering path for detect + list:
// human consumers see one line per conflict; --json consumers see the
// proto-typed wire shape (the full response message).
func renderConflictList(ctx cliapp.RunContext, payload proto.Message, conflicts []*conflictsv1.Conflict, summary, hint string) error {
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

func conflictLine(c *conflictsv1.Conflict) string {
	domains := strings.Join(c.GetDomains(), ",")
	if c.GetAssignedDomain() != "" {
		domains = c.GetAssignedDomain()
	}
	loc := ""
	if len(c.GetLocations()) > 0 {
		loc = c.GetLocations()[0]
		if len(c.GetLocations()) > 1 {
			loc = fmt.Sprintf("%s (+%d more)", loc, len(c.GetLocations())-1)
		}
	}
	return fmt.Sprintf("%s [%s/%s] %s domain=%s loc=%s status=%s",
		c.GetId(), c.GetType(), severityName(c.GetSeverity()), c.GetSubtype(), domains, loc, statusName(c.GetStatus()))
}

func mutationResult(dryRun bool, msg string) string {
	if dryRun {
		return msg + " (dry-run: no changes persisted)"
	}
	return msg
}

func severityName(s conflictsv1.Severity) string {
	switch s {
	case conflictsv1.Severity_SEVERITY_INFO:
		return "info"
	case conflictsv1.Severity_SEVERITY_WARN:
		return "warn"
	case conflictsv1.Severity_SEVERITY_ERROR:
		return "error"
	case conflictsv1.Severity_SEVERITY_BLOCKER:
		return "blocker"
	default:
		return "unspecified"
	}
}

func statusName(s conflictsv1.ResolutionStatus) string {
	switch s {
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_DETECTED:
		return "detected"
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_ASSIGNED:
		return "assigned"
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_SPLIT:
		return "split"
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_RESOLVED:
		return "resolved"
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_VALIDATED:
		return "validated"
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_COMMITTED:
		return "committed"
	case conflictsv1.ResolutionStatus_RESOLUTION_STATUS_FORCE_RESOLVED:
		return "force_resolved"
	default:
		return "unspecified"
	}
}

func fixKindName(k conflictsv1.FixKind) string {
	switch k {
	case conflictsv1.FixKind_FIX_KIND_MOVE_FILE:
		return "move_file"
	case conflictsv1.FixKind_FIX_KIND_REASSIGN_DOMAIN:
		return "reassign_domain"
	case conflictsv1.FixKind_FIX_KIND_BREAK_CYCLE:
		return "break_cycle"
	case conflictsv1.FixKind_FIX_KIND_ADD_DEPENDENCY:
		return "add_dependency"
	case conflictsv1.FixKind_FIX_KIND_ADD_TRANSITIONAL:
		return "add_transitional"
	default:
		return "unspecified"
	}
}

// statusFilters maps the CLI's bare status tokens onto the proto enum,
// so `conflicts list <scenario> --status detected,assigned` works.
var statusFilters = map[string]conflictsv1.ResolutionStatus{
	"detected":       conflictsv1.ResolutionStatus_RESOLUTION_STATUS_DETECTED,
	"assigned":       conflictsv1.ResolutionStatus_RESOLUTION_STATUS_ASSIGNED,
	"split":          conflictsv1.ResolutionStatus_RESOLUTION_STATUS_SPLIT,
	"resolved":       conflictsv1.ResolutionStatus_RESOLUTION_STATUS_RESOLVED,
	"validated":      conflictsv1.ResolutionStatus_RESOLUTION_STATUS_VALIDATED,
	"committed":      conflictsv1.ResolutionStatus_RESOLUTION_STATUS_COMMITTED,
	"force_resolved": conflictsv1.ResolutionStatus_RESOLUTION_STATUS_FORCE_RESOLVED,
}

func parseStatus(token string) (conflictsv1.ResolutionStatus, bool) {
	s, ok := statusFilters[strings.ToLower(strings.TrimSpace(token))]
	return s, ok
}

func statusNames() []string {
	return []string{"detected", "assigned", "split", "resolved", "validated", "committed", "force_resolved"}
}
