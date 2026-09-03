package versions

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
	versionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions/versions_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core            *cliapp.ScenarioApp
	client          versionsconnect.VersionsServiceClient
	lifecycleClient versionsconnect.VersionLifecycleServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:            core,
		client:          versionsconnect.NewVersionsServiceClient(httpClient, baseURL),
		lifecycleClient: versionsconnect.NewVersionLifecycleServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) retireCandidates(ctx cliapp.RunContext) error {
	resp, err := h.lifecycleClient.ListRetireCandidates(context.Background(), connect.NewRequest(&versionsv1.ListRetireCandidatesRequest{ComponentId: ctx.Flag("component-id")}))
	if err != nil {
		return cliapp.WrapAPIError("list retire candidates", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Candidates))
	for _, item := range resp.Msg.Candidates {
		results = append(results, fmt.Sprintf("%s %s@%s status=%s", item.ComponentId, item.LibraryId, item.Version, item.Status))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d retire candidate(s).", len(results))}, ResultsHeading: "Retire candidates", Results: results})
}

func cleanupScope(ctx cliapp.RunContext) (*versionsv1.CleanupScope, error) {
	scope := &versionsv1.CleanupScope{ComponentId: ctx.Flag("component-id"), LibraryId: ctx.Flag("library-id")}
	if raw := ctx.Flag("older-than-days"); raw != "" {
		days, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || days < 0 {
			return nil, fmt.Errorf("--older-than-days must be a non-negative integer (got %q)", raw)
		}
		scope.OlderThanDays = int32(days)
	}
	if scope.ComponentId != "" && scope.LibraryId != "" {
		return nil, fmt.Errorf("use only one of --component-id or --library-id")
	}
	return scope, nil
}

func renderCleanupItems(items []*versionsv1.CleanupItem) []string {
	results := make([]string, 0, len(items))
	for _, item := range items {
		if item == nil || item.Version == nil {
			continue
		}
		state := "blocked"
		if item.Eligible {
			state = "eligible"
		}
		results = append(results, fmt.Sprintf("%s@%s %s age=%dd adopters=%d dependencies=%d references=%d (%s)", item.Version.LibraryId, item.Version.Version, state, item.AgeDays, item.AdoptionCount, item.DependencyCount, len(item.References), item.Reason))
		for _, ref := range item.References {
			owner := fmt.Sprintf("%s@%s %s", ref.OwnerLibraryId, ref.OwnerVersion, ref.OwnerPath)
			if ref.OwnerScenario != "" {
				owner = fmt.Sprintf("%s [%s adoption=%s]", owner, ref.OwnerScenario, ref.AdoptionId)
			}
			results = append(results, fmt.Sprintf("  <- %s:%s imports %q (%s)", owner, ref.Kind, ref.ImportSpecifier, ref.Evidence))
		}
	}
	return results
}

func (h *handlers) planCleanup(ctx cliapp.RunContext) error {
	scope, err := cleanupScope(ctx)
	if err != nil {
		return err
	}
	resp, err := h.lifecycleClient.PlanCleanup(context.Background(), connect.NewRequest(&versionsv1.PlanCleanupRequest{Scope: scope}))
	if err != nil {
		return cliapp.WrapAPIError("plan version cleanup", err, nil)
	}
	results := renderCleanupItems(resp.Msg.Items)
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Cleanup plan: %d eligible item(s); plan-hash=%s.", countEligible(resp.Msg.Items), resp.Msg.PlanHash)}, ResultsHeading: "Version cleanup plan", Results: results, RetrievalHints: []string{"`versions reap --plan-hash <hash> --confirm` — apply this exact plan"}})
}

func countEligible(items []*versionsv1.CleanupItem) int {
	count := 0
	for _, item := range items {
		if item != nil && item.Eligible {
			count++
		}
	}
	return count
}

func (h *handlers) cleanupVersions(ctx cliapp.RunContext) error {
	scope, err := cleanupScope(ctx)
	if err != nil {
		return err
	}
	confirm := ctx.BoolFlag("confirm")
	resp, err := h.lifecycleClient.CleanupVersions(context.Background(), connect.NewRequest(&versionsv1.CleanupVersionsRequest{Scope: scope, PlanHash: ctx.Flag("plan-hash"), Confirm: confirm}))
	if err != nil {
		return cliapp.WrapAPIError("cleanup versions", err, nil)
	}
	mode := "dry-run"
	if resp.Msg.Applied {
		mode = "applied"
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Version cleanup %s: retired %d; plan-hash=%s.", mode, resp.Msg.RetiredCount, resp.Msg.PlanHash)}, ResultsHeading: "Version cleanup", Results: renderCleanupItems(resp.Msg.Items)})
}

// reap composes the lifecycle's preview-and-confirm protocol into one
// governed command. A dry run only plans; mutation requires the exact plan
// hash returned by that preview and an explicit confirmation flag.
func (h *handlers) reap(ctx cliapp.RunContext) error {
	if !ctx.BoolFlag("confirm") {
		return h.planCleanup(ctx)
	}
	return h.cleanupVersions(ctx)
}

func (h *handlers) cleanupDraft(ctx cliapp.RunContext) error {
	days := int64(0)
	if raw := ctx.Flag("older-than-days"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || parsed < 0 {
			return fmt.Errorf("--older-than-days must be a non-negative integer (got %q)", raw)
		}
		days = parsed
	}
	resp, err := h.lifecycleClient.CleanupDraft(context.Background(), connect.NewRequest(&versionsv1.CleanupDraftRequest{ComponentId: ctx.Positional("component-id"), OlderThanDays: int32(days), Confirm: ctx.BoolFlag("confirm")}))
	if err != nil {
		return cliapp.WrapAPIError("cleanup draft", err, nil)
	}
	item := resp.Msg.Item
	if item == nil {
		return fmt.Errorf("server returned no draft cleanup result")
	}
	mode := "dry-run"
	if resp.Msg.Applied {
		mode = "applied"
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Draft cleanup %s: %s@%s (%s).", mode, item.Version.LibraryId, item.Version.Version, item.Reason)}, ResultsHeading: "Draft cleanup", Results: []string{fmt.Sprintf("eligible=%t age=%dd", item.Eligible, item.AgeDays)}})
}

func (h *handlers) progression(ctx cliapp.RunContext) error {
	resp, err := h.lifecycleClient.ListVersionLedger(context.Background(), connect.NewRequest(&versionsv1.ListVersionLedgerRequest{LibraryId: ctx.Flag("library-id"), Window: ctx.Flag("window")}))
	if err != nil {
		return cliapp.WrapAPIError("list version progression", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Rows))
	for _, row := range resp.Msg.Rows {
		results = append(results, fmt.Sprintf("%s@%s state=%s gates=%d/%d tests=%d pass=%.2f adopters=%d size=%d LOC", row.LibraryId, row.Version, row.LifecycleState, row.GatePassCount, row.GateFailCount, row.TestRuns, row.TestPassRate, row.AdoptionCurrent, row.LinesOfCode))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d ledger row(s).", len(results))}, ResultsHeading: "Version progression", Results: results})
}

func (h *handlers) transition(ctx cliapp.RunContext, method string) error {
	confirm := false
	if method == "retire" || method == "archive" {
		confirm = ctx.Flag("confirm") != ""
	}
	planHash := ""
	if ctx.FlagDeclared("plan-hash") {
		planHash = ctx.Flag("plan-hash")
	}
	req := &versionsv1.VersionLifecycleRequest{ComponentId: ctx.Positional("component-id"), Version: ctx.Positional("version"), Confirm: confirm, PlanHash: planHash}
	var resp *connect.Response[versionsv1.VersionLifecycleResponse]
	var err error
	switch method {
	case "deprecate":
		resp, err = h.lifecycleClient.DeprecateVersion(context.Background(), connect.NewRequest(req))
	case "archive":
		resp, err = h.lifecycleClient.ArchiveVersion(context.Background(), connect.NewRequest(req))
	default:
		resp, err = h.lifecycleClient.RetireVersion(context.Background(), connect.NewRequest(req))
	}
	if err != nil {
		return cliapp.WrapAPIError(method+" version", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("%s %s@%s.", method, req.ComponentId, req.Version)}, ResultsHeading: "Version lifecycle", Results: []string{fmt.Sprintf("%s state=%s", resp.Msg.Version.GetVersion(), resp.Msg.LifecycleState)}})
}

func (h *handlers) materialize(ctx cliapp.RunContext) error {
	componentID := ctx.Positional("component-id")
	version := ctx.Positional("version")
	all := ctx.BoolFlag("all")
	if !all && (componentID == "" || version == "") {
		return fmt.Errorf("provide <component-id> <version>, or --all")
	}
	resp, err := h.lifecycleClient.MaterializeVersion(context.Background(), connect.NewRequest(&versionsv1.MaterializeVersionRequest{ComponentId: componentID, Version: version, All: all, Into: ctx.Flag("into")}))
	if err != nil {
		return cliapp.WrapAPIError("materialize versions", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Versions))
	for _, item := range resp.Msg.Versions {
		results = append(results, fmt.Sprintf("%s@%s directory=%s already-present=%t", item.LibraryId, item.Version, item.Directory, item.AlreadyPresent))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Materialized %d version(s).", len(results))}, ResultsHeading: "Materialized versions", Results: results})
}

func (h *handlers) reconcilePresence(ctx cliapp.RunContext) error {
	componentID := ""
	if ctx.FlagDeclared("component-id") {
		componentID = ctx.Flag("component-id")
	}
	resp, err := h.lifecycleClient.ReconcilePresence(context.Background(), connect.NewRequest(&versionsv1.ReconcilePresenceRequest{ComponentId: componentID, Apply: ctx.BoolFlag("apply")}))
	if err != nil {
		return cliapp.WrapAPIError("reconcile version presence", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Presence reconciliation: evict=%d materialize=%d unchanged=%d applied=%t.", len(resp.Msg.Evict), len(resp.Msg.Materialize), len(resp.Msg.Unchanged), resp.Msg.Applied)}, ResultsHeading: "Presence reconciliation", Results: append(renderCandidates(resp.Msg.Evict, "evict"), append(renderCandidates(resp.Msg.Materialize, "materialize"), renderCandidates(resp.Msg.Unchanged, "unchanged")...)...)})
}

func (h *handlers) exportArchive(ctx cliapp.RunContext) error {
	resp, err := h.lifecycleClient.ExportArchive(context.Background(), connect.NewRequest(&versionsv1.ArchiveRequest{Path: ctx.Flag("out")}))
	if err != nil {
		return cliapp.WrapAPIError("export version archive", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Exported version archive to %s (checksum %s).", resp.Msg.Path, resp.Msg.Checksum)}, ResultsHeading: "Version archive", Results: []string{fmt.Sprintf("schema=%d rows=%v", resp.Msg.SchemaVersion, resp.Msg.RowCounts)}})
}

func (h *handlers) importArchive(ctx cliapp.RunContext) error {
	resp, err := h.lifecycleClient.ImportArchive(context.Background(), connect.NewRequest(&versionsv1.ImportArchiveRequest{Path: ctx.Flag("in"), Overwrite: ctx.BoolFlag("overwrite")}))
	if err != nil {
		return cliapp.WrapAPIError("import version archive", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Imported version archive from %s (checksum %s).", resp.Msg.Path, resp.Msg.Checksum)}, ResultsHeading: "Version archive", Results: []string{fmt.Sprintf("schema=%d rows=%v", resp.Msg.SchemaVersion, resp.Msg.RowCounts)}})
}

func (h *handlers) doctorCall(_ cliapp.OperationContext) (*versionsv1.DoctorResponse, error) {
	resp, err := h.lifecycleClient.Doctor(context.Background(), connect.NewRequest(&versionsv1.DoctorRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("doctor version ledger", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no doctor response")
	}
	return resp.Msg, nil
}

func (h *handlers) doctorReport(_ cliapp.OperationContext, msg *versionsv1.DoctorResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Issues))
	for _, issue := range msg.Issues {
		results = append(results, fmt.Sprintf("%s@%s %s expected=%s actual=%s (%s)", issue.LibraryId, issue.Version, issue.Path, issue.ExpectedSha256, issue.ActualSha256, issue.Reason))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Version ledger doctor found %d issue(s).", len(results))}, ResultsHeading: "Version ledger issues", Results: results}
}

func renderCandidates(items []*versionsv1.RetireCandidate, action string) []string {
	results := make([]string, 0, len(items))
	for _, item := range items {
		if item != nil {
			results = append(results, fmt.Sprintf("%s %s@%s status=%s", action, item.LibraryId, item.Version, item.Status))
		}
	}
	return results
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*versionsv1.ListVersionsResponse, error) {
	all := ctx.FlagDeclared("all") && ctx.BoolFlag("all")
	req := &versionsv1.ListVersionsRequest{ComponentId: ctx.Positional("component-id"), All: all}
	if raw := ctx.Flag("limit"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("--limit must be an integer (got %q)", raw)
		}
		req.Limit = int32(n)
	}
	resp, err := h.client.ListVersions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("list versions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no list response")
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, msg *versionsv1.ListVersionsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Versions))
	for _, v := range msg.Versions {
		results = append(results, formatVersion(v))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d version(s).", len(msg.Versions))},
		ResultsHeading: "Versions",
		Results:        results,
		RetrievalHints: []string{
			"`versions show <component-id> <version> --with-content` — full body",
			"`versions diff <component-id> <from> <to>` — line-by-line diff",
		},
	}
}

func (h *handlers) showCall(ctx cliapp.OperationContext) (*versionsv1.GetVersionResponse, error) {
	req := &versionsv1.GetVersionRequest{
		ComponentId:    ctx.Positional("component-id"),
		Version:        ctx.Positional("version"),
		IncludeContent: ctx.Flag("with-content") != "",
	}
	resp, err := h.client.GetVersion(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("get version", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Version == nil {
		return nil, fmt.Errorf("server returned no version")
	}
	return resp.Msg, nil
}

func (h *handlers) showReport(_ cliapp.OperationContext, msg *versionsv1.GetVersionResponse) cliapp.ListReport {
	results := []string{formatVersion(msg.Version)}
	if msg.Content != "" {
		results = append(results, "--- content ---", msg.Content)
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Version %s for %s.", msg.Version.Version, msg.Version.ComponentId)},
		ResultsHeading: "Version",
		Results:        results,
	}
}

func (h *handlers) diffCall(ctx cliapp.OperationContext) (*versionsv1.DiffVersionsResponse, error) {
	req := &versionsv1.DiffVersionsRequest{
		ComponentId: ctx.Positional("component-id"),
		From:        ctx.Positional("from"),
		To:          ctx.Positional("to"),
	}
	resp, err := h.client.DiffVersions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("diff versions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no diff response")
	}
	return resp.Msg, nil
}

func (h *handlers) diffReport(ctx cliapp.OperationContext, msg *versionsv1.DiffVersionsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Rows))
	for _, r := range msg.Rows {
		results = append(results, formatDiffRow(r))
	}
	summary := fmt.Sprintf("%s → %s : +%d / -%d (%d rows)",
		ctx.Positional("from"), ctx.Positional("to"), msg.Additions, msg.Removals, len(msg.Rows))
	return cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Diff",
		Results:        results,
	}
}

func formatVersion(v *versionsv1.Version) string {
	if v == nil {
		return "(nil)"
	}
	created := "?"
	if v.CreatedAt != nil {
		created = v.CreatedAt.AsTime().Format(time.RFC3339)
	}
	sha := v.ContentSha256
	if len(sha) > 12 {
		sha = sha[:12]
	}
	return fmt.Sprintf("%s — v=%s sha=%s first-seen=%s required-tokens=%v required-token-patterns=%v changelog=%q",
		v.Id, v.Version, sha, created, v.RequiredTokens, v.RequiredTokenPatterns, v.ChangelogMd)
}

func formatDiffRow(r *versionsv1.DiffRow) string {
	if r == nil {
		return ""
	}
	left := formatDiffCell(r.Left)
	right := formatDiffCell(r.Right)
	sep := "  |  "
	return strings.Join([]string{left, sep, right}, "")
}

func formatDiffCell(c *versionsv1.DiffCell) string {
	if c == nil {
		return ""
	}
	marker := opMarker(c.Op)
	if c.Op == versionsv1.DiffOp_DIFF_OP_EMPTY {
		return fmt.Sprintf("%s %4s %s", marker, "", "")
	}
	return fmt.Sprintf("%s %4d %s", marker, c.LineNumber, c.Text)
}

func opMarker(op versionsv1.DiffOp) string {
	switch op {
	case versionsv1.DiffOp_DIFF_OP_ADD:
		return "+"
	case versionsv1.DiffOp_DIFF_OP_REMOVE:
		return "-"
	case versionsv1.DiffOp_DIFF_OP_EQUAL:
		return " "
	case versionsv1.DiffOp_DIFF_OP_EMPTY:
		return " "
	}
	return "?"
}
