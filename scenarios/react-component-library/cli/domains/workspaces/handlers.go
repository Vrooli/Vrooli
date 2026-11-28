package workspaces

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace/workspace_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"react-component-library/cli/internal/runtimeenv"
)

// handlers bundles the closure over *cliapp.ScenarioApp + the generated
// Connect-Go client. Mirrors cli/domains/adoptions/.
type handlers struct {
	core   *cliapp.ScenarioApp
	client workspaceconnect.WorkspaceServiceClient
	deps   runtimeenv.Dependencies
}

func newHandlers(core *cliapp.ScenarioApp, deps runtimeenv.Dependencies) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: workspaceconnect.NewWorkspaceServiceClient(httpClient, baseURL),
		deps:   deps,
	}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*workspacev1.ListWorkspacesResponse, error) {
	provenance, err := parseProvenance(ctx.Flag("provenance"))
	if err != nil {
		return nil, err
	}
	req := &workspacev1.ListWorkspacesRequest{
		Provenance: provenance,
		Query:      ctx.Flag("query"),
	}
	if req.Limit, err = parseLimit(ctx.Flag("limit")); err != nil {
		return nil, err
	}
	resp, err := h.client.ListWorkspaces(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("list workspaces", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no workspace list")
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, msg *workspacev1.ListWorkspacesResponse) cliapp.ListReport {
	rows := make([]string, 0, len(msg.GetWorkspaces()))
	for _, workspace := range msg.GetWorkspaces() {
		rows = append(rows, formatWorkspaceSummary(workspace))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d workspace(s).", len(rows))},
		ResultsHeading: "Workspaces",
		Results:        rows,
		RetrievalHints: []string{
			"`workspaces get <id>` — open the full composition",
			"`workspaces create --name <name>` — author a new workspace",
		},
	}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*workspacev1.GetWorkspaceResponse, error) {
	id := ctx.Positional("id")
	resp, err := h.client.GetWorkspace(context.Background(), connect.NewRequest(&workspacev1.GetWorkspaceRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("get workspace %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetWorkspace() == nil {
		return nil, fmt.Errorf("server returned no workspace %q", id)
	}
	return resp.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, msg *workspacev1.GetWorkspaceResponse) cliapp.ListReport {
	workspace := msg.GetWorkspace()
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched workspace %s (%s) at revision %d.", workspace.GetName(), workspace.GetId(), workspace.GetRevision())},
		ResultsHeading: "Workspace",
		Results:        formatWorkspaceLines(workspace),
		RetrievalHints: []string{
			fmt.Sprintf("`workspaces save %s --base-revision %d --workspace <file>` — save an edit from this revision", workspace.GetId(), workspace.GetRevision()),
			"`workspaces validate --workspace <file>` — check an unsaved composition",
			fmt.Sprintf("`workspaces revisions %s` — inspect saved snapshots", workspace.GetId()),
		},
	}
}

func (h *handlers) createCall(ctx cliapp.OperationContext) (*workspacev1.CreateWorkspaceResponse, error) {
	provenance, err := parseProvenance(ctx.Flag("provenance"))
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(ctx.Flag("name"))
	if name == "" {
		return nil, fmt.Errorf("--name is required to create a workspace")
	}
	req := &workspacev1.CreateWorkspaceRequest{
		Name:            name,
		Summary:         ctx.Flag("summary"),
		Provenance:      provenance,
		Tags:            cleanValues(ctx.FlagValues("tags")),
		BindingScenario: ctx.Flag("binding-scenario"),
		BindingPage:     ctx.Flag("binding-page"),
		ThemeJson:       ctx.Flag("theme-json"),
		OptionsJson:     ctx.Flag("options-json"),
	}
	if path := strings.TrimSpace(ctx.Flag("pages")); path != "" {
		pages, readErr := readWorkspacePages(path)
		if readErr != nil {
			return nil, readErr
		}
		req.Pages = pages
	}
	resp, err := h.client.CreateWorkspace(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("create workspace", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetWorkspace() == nil {
		return nil, fmt.Errorf("server returned no created workspace")
	}
	return resp.Msg, nil
}

func (h *handlers) createReport(_ cliapp.OperationContext, msg *workspacev1.CreateWorkspaceResponse) cliapp.MutationReport {
	workspace := msg.GetWorkspace()
	return cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created workspace %s (%s) at revision %d.", workspace.GetName(), workspace.GetId(), workspace.GetRevision())},
		Changes:     formatWorkspaceLines(workspace),
		NextCommand: []string{fmt.Sprintf("`workspaces get %s` — open the new workspace", workspace.GetId())},
	}
}

func (h *handlers) saveCall(ctx cliapp.OperationContext) (*workspacev1.SaveWorkspaceResponse, error) {
	id := strings.TrimSpace(ctx.Positional("id"))
	if id == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	baseRevision, err := parseRequiredRevision(ctx.Flag("base-revision"))
	if err != nil {
		return nil, err
	}
	path := strings.TrimSpace(ctx.Flag("workspace"))
	if path == "" {
		return nil, fmt.Errorf("--workspace is required to save a workspace")
	}
	workspace, err := readWorkspace(path)
	if err != nil {
		return nil, err
	}
	if workspace.GetId() != "" && workspace.GetId() != id {
		return nil, fmt.Errorf("--workspace document is for %q, but the command targets %q", workspace.GetId(), id)
	}
	workspace.Id = id

	resp, err := h.client.SaveWorkspace(context.Background(), connect.NewRequest(&workspacev1.SaveWorkspaceRequest{
		Workspace:    workspace,
		BaseRevision: baseRevision,
		Actor:        ctx.Flag("actor"),
	}))
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeAborted {
			return nil, fmt.Errorf("workspace %s changed since base revision %d; re-read it with `workspaces get %s` and retry", id, baseRevision, id)
		}
		return nil, cliapp.WrapAPIError(fmt.Sprintf("save workspace %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetWorkspace() == nil {
		return nil, fmt.Errorf("server returned no saved workspace")
	}
	return resp.Msg, nil
}

func (h *handlers) saveReport(_ cliapp.OperationContext, msg *workspacev1.SaveWorkspaceResponse) cliapp.MutationReport {
	workspace := msg.GetWorkspace()
	return cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Saved workspace %s at revision %d.", workspace.GetId(), workspace.GetRevision())},
		Changes:     formatWorkspaceLines(workspace),
		NextCommand: []string{fmt.Sprintf("`workspaces revisions %s` — inspect the appended snapshot", workspace.GetId())},
	}
}

func (h *handlers) deleteCall(ctx cliapp.OperationContext) (*workspacev1.DeleteWorkspaceResponse, error) {
	id := ctx.Positional("id")
	resp, err := h.client.DeleteWorkspace(context.Background(), connect.NewRequest(&workspacev1.DeleteWorkspaceRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("delete workspace %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no delete response for workspace %q", id)
	}
	return resp.Msg, nil
}

func (h *handlers) deleteReport(ctx cliapp.OperationContext, msg *workspacev1.DeleteWorkspaceResponse) cliapp.MutationReport {
	id := ctx.Positional("id")
	state := "was not present"
	if msg.GetDeleted() {
		state = "and its revisions were removed"
	}
	return cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted workspace %s (%s).", id, state)},
		NextCommand: []string{"`workspaces list` — show remaining workspaces"},
	}
}

func (h *handlers) forkCall(ctx cliapp.OperationContext) (*workspacev1.ForkWorkspaceResponse, error) {
	resp, err := h.client.ForkWorkspace(context.Background(), connect.NewRequest(&workspacev1.ForkWorkspaceRequest{
		Id:   ctx.Positional("id"),
		Name: ctx.Flag("name"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("fork workspace", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetWorkspace() == nil {
		return nil, fmt.Errorf("server returned no forked workspace")
	}
	return resp.Msg, nil
}

func (h *handlers) forkReport(_ cliapp.OperationContext, msg *workspacev1.ForkWorkspaceResponse) cliapp.MutationReport {
	workspace := msg.GetWorkspace()
	return cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Forked workspace %s into %s (%s).", workspace.GetOriginWorkspaceId(), workspace.GetName(), workspace.GetId())},
		Changes: []string{
			fmt.Sprintf("origin=%s revision=%d", workspace.GetOriginWorkspaceId(), workspace.GetRevision()),
			formatWorkspaceSummaryLine(workspace),
		},
		NextCommand: []string{fmt.Sprintf("`workspaces get %s` — open the editable fork", workspace.GetId())},
	}
}

func (h *handlers) revisionsCall(ctx cliapp.OperationContext) (*workspacev1.ListWorkspaceRevisionsResponse, error) {
	req := &workspacev1.ListWorkspaceRevisionsRequest{Id: ctx.Positional("id")}
	limit, err := parseLimit(ctx.Flag("limit"))
	if err != nil {
		return nil, err
	}
	req.Limit = limit
	resp, err := h.client.ListWorkspaceRevisions(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("list workspace revisions", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no workspace revisions")
	}
	return resp.Msg, nil
}

func (h *handlers) revisionsReport(ctx cliapp.OperationContext, msg *workspacev1.ListWorkspaceRevisionsResponse) cliapp.ListReport {
	id := ctx.Positional("id")
	rows := make([]string, 0, len(msg.GetRevisions()))
	for _, revision := range msg.GetRevisions() {
		if revision == nil {
			continue
		}
		rows = append(rows, fmt.Sprintf("revision=%d actor=%s created=%s snapshot=%d bytes",
			revision.GetRevision(), nonEmpty(revision.GetActor(), "(unattributed)"), nonEmpty(revision.GetCreatedAt(), "(unknown)"), len(revision.GetSnapshotJson())))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d revision(s) for workspace %s.", len(rows), id)},
		ResultsHeading: "Revisions (newest first)",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("`workspaces get %s` — open the current revision", id)},
	}
}

func (h *handlers) validateCall(ctx cliapp.OperationContext) (*workspacev1.ValidateWorkspaceResponse, error) {
	path := strings.TrimSpace(ctx.Flag("workspace"))
	if path == "" {
		return nil, fmt.Errorf("--workspace is required to validate a workspace")
	}
	workspace, err := readWorkspace(path)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ValidateWorkspace(context.Background(), connect.NewRequest(&workspacev1.ValidateWorkspaceRequest{Workspace: workspace}))
	if err != nil {
		return nil, cliapp.WrapAPIError("validate workspace", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no workspace validation")
	}
	return resp.Msg, nil
}

func (h *handlers) validateReport(_ cliapp.OperationContext, msg *workspacev1.ValidateWorkspaceResponse) cliapp.ListReport {
	rows := make([]string, 0, len(msg.GetIssues()))
	for _, issue := range msg.GetIssues() {
		if issue == nil {
			continue
		}
		rows = append(rows, fmt.Sprintf("%s: %s", nonEmpty(issue.GetPath(), "(workspace)"), issue.GetMessage()))
	}
	summary := "Workspace is valid."
	if !msg.GetValid() {
		summary = fmt.Sprintf("Workspace has %d validation issue(s).", len(msg.GetIssues()))
	}
	return cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Validation issues",
		Results:        rows,
		RetrievalHints: []string{"`workspaces save <id> --base-revision <n> --workspace <file>` — save after fixing the issues"},
	}
}

func readWorkspace(path string) (*workspacev1.Workspace, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read --workspace %s: %w", path, err)
	}
	workspace := &workspacev1.Workspace{}
	if err := protojson.Unmarshal(bytes.TrimSpace(raw), workspace); err != nil {
		return nil, fmt.Errorf("decode --workspace %s as proto JSON: %w", path, err)
	}
	return workspace, nil
}

// readWorkspacePages accepts either a bare JSON array of WorkspacePage objects
// or an object with a "pages" array, so an operator can pass a page list or a
// partial CreateWorkspaceRequest.
func readWorkspacePages(path string) ([]*workspacev1.WorkspacePage, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read --pages %s: %w", path, err)
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("--pages %s is empty", path)
	}
	var entries []json.RawMessage
	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			return nil, fmt.Errorf("decode --pages %s: %w", path, err)
		}
	} else {
		var wrapper struct {
			Pages []json.RawMessage `json:"pages"`
		}
		if err := json.Unmarshal(trimmed, &wrapper); err != nil {
			return nil, fmt.Errorf("decode --pages %s: %w", path, err)
		}
		entries = wrapper.Pages
	}
	pages := make([]*workspacev1.WorkspacePage, 0, len(entries))
	for i, entry := range entries {
		page := &workspacev1.WorkspacePage{}
		if err := protojson.Unmarshal(entry, page); err != nil {
			return nil, fmt.Errorf("decode --pages %s entry %d: %w", path, i, err)
		}
		pages = append(pages, page)
	}
	return pages, nil
}

func parseLimit(raw string) (int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("--limit must be a non-negative integer (got %q)", raw)
	}
	return int32(value), nil
}

func parseRequiredRevision(raw string) (int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("--base-revision is required; read the workspace first and pass its revision")
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("--base-revision must be a non-negative integer (got %q)", raw)
	}
	return int32(value), nil
}

func parseProvenance(raw string) (workspacev1.WorkspaceProvenance, error) {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(raw)), "_", "-")
	normalized = strings.TrimPrefix(normalized, "workspace-provenance-")
	switch normalized {
	case "", "unspecified":
		return workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED, nil
	case "authored":
		return workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_AUTHORED, nil
	case "curated":
		return workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED, nil
	case "imported":
		return workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED, nil
	case "forked":
		return workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED, nil
	default:
		return 0, fmt.Errorf("--provenance must be one of authored, curated, imported, forked (got %q)", raw)
	}
}

func cleanValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func formatWorkspaceSummary(workspace *workspacev1.WorkspaceSummary) string {
	if workspace == nil {
		return "(nil workspace)"
	}
	var parts []string
	parts = append(parts, fmt.Sprintf("%s (%s)", nonEmpty(workspace.GetName(), "(unnamed)"), workspace.GetId()))
	parts = append(parts, provenanceLabel(workspace.GetProvenance()))
	parts = append(parts, fmt.Sprintf("rev=%d", workspace.GetRevision()))
	parts = append(parts, fmt.Sprintf("pages=%d displays=%d states=%d", workspace.GetPageCount(), workspace.GetDisplayCount(), workspace.GetStateCount()))
	if binding := bindingLabel(workspace.GetBindingScenario(), workspace.GetBindingPage()); binding != "" {
		parts = append(parts, "binding="+binding)
	}
	if tags := workspace.GetTags(); len(tags) > 0 {
		parts = append(parts, "tags="+strings.Join(tags, ","))
	}
	if updated := workspace.GetUpdatedAt(); updated != "" {
		parts = append(parts, "updated="+updated)
	}
	return strings.Join(parts, " | ")
}

func formatWorkspaceSummaryLine(workspace *workspacev1.Workspace) string {
	if workspace == nil {
		return "(nil workspace)"
	}
	summary := &workspacev1.WorkspaceSummary{
		Id:              workspace.GetId(),
		Name:            workspace.GetName(),
		Summary:         workspace.GetSummary(),
		Provenance:      workspace.GetProvenance(),
		Tags:            workspace.GetTags(),
		BindingScenario: workspace.GetBindingScenario(),
		BindingPage:     workspace.GetBindingPage(),
		PageCount:       int32(len(workspace.GetPages())),
		Revision:        workspace.GetRevision(),
		CreatedAt:       workspace.GetCreatedAt(),
		UpdatedAt:       workspace.GetUpdatedAt(),
	}
	for _, page := range workspace.GetPages() {
		summary.DisplayCount += int32(len(page.GetDisplays()))
		for _, display := range page.GetDisplays() {
			summary.StateCount += int32(len(display.GetStates()))
		}
	}
	return formatWorkspaceSummary(summary)
}

func formatWorkspaceLines(workspace *workspacev1.Workspace) []string {
	if workspace == nil {
		return []string{"(nil workspace)"}
	}
	header := formatWorkspaceSummaryLine(workspace)
	rows := []string{header}
	if summary := strings.TrimSpace(workspace.GetSummary()); summary != "" {
		rows = append(rows, "summary: "+summary)
	}
	if theme := strings.TrimSpace(workspace.GetThemeJson()); theme != "" {
		rows = append(rows, "theme_json: "+theme)
	}
	if options := strings.TrimSpace(workspace.GetOptionsJson()); options != "" {
		rows = append(rows, "options_json: "+options)
	}
	for _, page := range workspace.GetPages() {
		if page == nil {
			continue
		}
		rows = append(rows, fmt.Sprintf("  page %s %q route=%s type=%s order=%d",
			page.GetId(), page.GetName(), nonEmpty(page.GetRoute(), "(none)"), nonEmpty(page.GetPageTypeId(), "(none)"), page.GetSortOrder()))
		for _, display := range page.GetDisplays() {
			if display == nil {
				continue
			}
			rows = append(rows, fmt.Sprintf("    display %s device=%s %dx%d order=%d",
				display.GetId(), deviceLabel(display.GetDevice()), display.GetWidth(), display.GetHeight(), display.GetSortOrder()))
			for _, state := range display.GetStates() {
				if state == nil {
					continue
				}
				rows = append(rows, fmt.Sprintf("      state %s %q status=%s order=%d",
					state.GetId(), state.GetName(), stateStatusLabel(state.GetStatus()), state.GetSortOrder()))
			}
		}
	}
	return rows
}

func provenanceLabel(value workspacev1.WorkspaceProvenance) string {
	switch value {
	case workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_AUTHORED:
		return "authored"
	case workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED:
		return "curated"
	case workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED:
		return "imported"
	case workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED:
		return "forked"
	default:
		return "unspecified"
	}
}

func deviceLabel(value workspacev1.DisplayDevice) string {
	switch value {
	case workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP:
		return "desktop"
	case workspacev1.DisplayDevice_DISPLAY_DEVICE_TABLET:
		return "tablet"
	case workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE:
		return "phone"
	default:
		return "unspecified"
	}
}

func stateStatusLabel(value workspacev1.StateStatus) string {
	switch value {
	case workspacev1.StateStatus_STATE_STATUS_DRAFT:
		return "draft"
	case workspacev1.StateStatus_STATE_STATUS_READY:
		return "ready"
	case workspacev1.StateStatus_STATE_STATUS_CAPTURED:
		return "captured"
	default:
		return "unspecified"
	}
}

func bindingLabel(scenario, page string) string {
	switch {
	case scenario == "" && page == "":
		return ""
	case scenario == "":
		return page
	case page == "":
		return scenario
	default:
		return scenario + "/" + page
	}
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// assistCall asks the server for a validated patch. It never writes: the CLI
// prints the proposal so an operator can inspect or apply it deliberately.
func (h *handlers) assistCall(ctx cliapp.OperationContext) (*workspacev1.AssistWorkspaceResponse, error) {
	id := strings.TrimSpace(ctx.Positional("id"))
	if id == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	prompt := strings.TrimSpace(ctx.Flag("prompt"))
	if prompt == "" {
		return nil, fmt.Errorf("--prompt is required")
	}
	kind := strings.ToLower(strings.TrimSpace(ctx.Flag("scope")))
	if kind == "" {
		// Infer from the selector the operator named, so the common cases do
		// not require two flags that must agree.
		switch {
		case strings.TrimSpace(ctx.Flag("node")) != "":
			kind = "layer"
		case strings.TrimSpace(ctx.Flag("display")) != "":
			kind = "frame"
		default:
			kind = "design"
		}
	}
	switch kind {
	case "design", "frame", "layer":
	default:
		return nil, fmt.Errorf("--scope must be design, frame or layer")
	}

	// Validate the request shape before any network call: a typo in --state
	// should name the flag, not surface as a confusing revision read.
	active := make([]*workspacev1.WorkspaceAssistState, 0)
	for _, raw := range ctx.FlagValues("state") {
		displayID, stateID, ok := strings.Cut(strings.TrimSpace(raw), "=")
		if !ok || strings.TrimSpace(displayID) == "" {
			return nil, fmt.Errorf("--state must be display=<state> (got %q)", raw)
		}
		active = append(active, &workspacev1.WorkspaceAssistState{
			DisplayId: strings.TrimSpace(displayID),
			StateId:   strings.TrimSpace(stateID),
		})
	}

	baseRevision, err := parseLimit(ctx.Flag("base-revision"))
	if err != nil {
		return nil, fmt.Errorf("--base-revision must be a non-negative integer")
	}
	if baseRevision == 0 {
		// A proposal is only meaningful against the revision it saw. Reading the
		// current one here lets an operator ask without first copying a number.
		got, err := h.client.GetWorkspace(context.Background(), connect.NewRequest(&workspacev1.GetWorkspaceRequest{Id: id}))
		if err != nil {
			return nil, cliapp.WrapAPIError("read workspace revision", err, nil)
		}
		baseRevision = got.Msg.GetWorkspace().GetRevision()
	}

	key := strings.TrimSpace(ctx.Flag("idempotency-key"))
	if key == "" {
		key = h.defaultIdempotencyKey()
	}
	resp, err := h.client.AssistWorkspace(context.Background(), connect.NewRequest(&workspacev1.AssistWorkspaceRequest{
		Id:           id,
		BaseRevision: baseRevision,
		PageId:       strings.TrimSpace(ctx.Flag("page")),
		ActiveStates: active,
		Prompt:       prompt,
		Scope: &workspacev1.WorkspaceAssistScope{
			Kind:      kind,
			DisplayId: strings.TrimSpace(ctx.Flag("display")),
			NodeId:    strings.TrimSpace(ctx.Flag("node")),
		},
		IdempotencyKey: key,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("assist workspace", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no proposal")
	}
	return resp.Msg, nil
}

func (h *handlers) assistReport(ctx cliapp.OperationContext, msg *workspacev1.AssistWorkspaceResponse) cliapp.MutationReport {
	var proposal struct {
		Summary    string `json:"summary"`
		Operations []struct {
			Kind   string `json:"kind"`
			NodeID string `json:"nodeId"`
			Parent string `json:"parentId"`
		} `json:"operations"`
	}
	_ = json.Unmarshal([]byte(msg.GetProposalJson()), &proposal)

	result := []string{fmt.Sprintf("Proposed %d change(s) against revision %d; nothing was written.",
		len(proposal.Operations), msg.GetRevision())}
	if summary := strings.TrimSpace(proposal.Summary); summary != "" {
		result = append(result, summary)
	}
	modelParts := make([]string, 0, 2)
	for _, part := range []string{msg.GetOperation().GetProvider(), msg.GetOperation().GetModel()} {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			modelParts = append(modelParts, trimmed)
		}
	}
	if model := strings.Join(modelParts, " · "); model != "" {
		result = append(result, "model: "+model)
	}
	changes := make([]string, 0, len(proposal.Operations))
	for index, operation := range proposal.Operations {
		target := nonEmpty(operation.NodeID, nonEmpty(operation.Parent, "the page"))
		changes = append(changes, fmt.Sprintf("%d. %s %s", index+1, operation.Kind, target))
	}
	return cliapp.MutationReport{
		Result:  result,
		Changes: changes,
		NextCommand: []string{
			"apply it in the studio (`design?workspace=" + strings.TrimSpace(ctx.Positional("id")) + "`) or discard it",
			"`workspaces get <id>` — inspect the stored composition",
		},
	}
}

// randomSuffix makes a default idempotency key distinct without depending on
// crypto/rand being available in a stripped environment.
func (h *handlers) defaultIdempotencyKey() string {
	timestamp := h.deps.Clock.Now().UnixNano()
	suffix := randomSuffix(h.deps.Clock.Now())
	return fmt.Sprintf("assist-%d-%s", timestamp, suffix)
}

func randomSuffix(now time.Time) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	var builder strings.Builder
	seed := now.UnixNano()
	for index := 0; index < 6; index++ {
		builder.WriteByte(alphabet[int(seed>>(index*5))%len(alphabet)])
	}
	return builder.String()
}
