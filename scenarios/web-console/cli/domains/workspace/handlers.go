package workspace

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/workspace/workspace_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client workspaceconnect.WorkspaceServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: workspaceconnect.NewWorkspaceServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) layoutGet(ctx cliapp.RunContext) error {
	resp, err := h.client.GetLayout(context.Background(), connect.NewRequest(&workspacev1.GetLayoutRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("workspace layout-get", err, nil)
	}

	rows := []string{
		fmt.Sprintf("Active pane: %s", resp.Msg.GetActivePane()),
		fmt.Sprintf("Panes: %d", len(resp.Msg.GetPanes())),
		fmt.Sprintf("Groups: %d", len(resp.Msg.GetGroups())),
	}
	for _, p := range resp.Msg.GetPanes() {
		rows = append(rows, fmt.Sprintf("  pane %s | name=%s | group=%s | sort=%d", support.ShortID(p.GetSessionId()), p.GetName(), p.GetGroupId(), p.GetSortOrder()))
	}
	for _, g := range resp.Msg.GetGroups() {
		rows = append(rows, fmt.Sprintf("  group %s | name=%s | color=%s | collapsed=%t", support.ShortID(g.GetId()), g.GetName(), g.GetColor(), g.GetIsCollapsed()))
	}

	report := cliapp.ListReport{
		Summary:        []string{"Workspace layout"},
		ResultsHeading: "Layout",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s workspace layout-save --body-file layout.json", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

type layoutSaveBody struct {
	ActivePane string   `json:"active_pane"`
	PaneOrder  []string `json:"pane_order"`
}

func (h *handlers) layoutSave(ctx cliapp.RunContext) error {
	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var body layoutSaveBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	if _, err := h.client.SaveLayout(context.Background(), connect.NewRequest(&workspacev1.SaveLayoutRequest{
		ActivePane: body.ActivePane,
		PaneOrder:  body.PaneOrder,
	})); err != nil {
		return cliapp.WrapAPIError("workspace layout-save", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{"Workspace layout saved"},
		NextCommand: []string{fmt.Sprintf("%s workspace layout-get", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

// paneUpdateBody mirrors UpdatePaneRequest. Any field present in the JSON
// flips the corresponding has_* flag server-side. Pointer fields let us
// distinguish "absent" from "zero value".
type paneUpdateBody struct {
	Name                 *string `json:"name,omitempty"`
	HeaderColor          *string `json:"header_color,omitempty"`
	ThemeID              *string `json:"theme_id,omitempty"`
	FontSize             *int32  `json:"font_size,omitempty"`
	SortOrder            *int32  `json:"sort_order,omitempty"`
	GroupID              *string `json:"group_id,omitempty"`
	SupportsMessagesView *bool   `json:"supports_messages_view,omitempty"`
}

func (h *handlers) paneUpdate(ctx cliapp.RunContext) error {
	sessionID := ctx.Positional("session-id")
	if sessionID == "" {
		return fmt.Errorf("usage: workspace pane-update <session-id> --body-file PATH")
	}

	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var body paneUpdateBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	req := &workspacev1.UpdatePaneRequest{SessionId: sessionID}
	if body.Name != nil {
		req.Name = *body.Name
		req.HasName = true
	}
	if body.HeaderColor != nil {
		req.HeaderColor = *body.HeaderColor
		req.HasHeaderColor = true
	}
	if body.ThemeID != nil {
		req.ThemeId = *body.ThemeID
		req.HasThemeId = true
	}
	if body.FontSize != nil {
		req.FontSize = *body.FontSize
		req.HasFontSize = true
	}
	if body.SortOrder != nil {
		req.SortOrder = *body.SortOrder
		req.HasSortOrder = true
	}
	if body.GroupID != nil {
		req.GroupId = *body.GroupID
		req.HasGroupId = true
	}
	if body.SupportsMessagesView != nil {
		req.SupportsMessagesView = *body.SupportsMessagesView
		req.HasSupportsMessagesView = true
	}

	if _, err := h.client.UpdatePane(context.Background(), connect.NewRequest(req)); err != nil {
		return cliapp.WrapAPIError("workspace pane-update", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated pane for session %s", sessionID)},
		NextCommand: []string{fmt.Sprintf("%s workspace layout-get", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func (h *handlers) paneDelete(ctx cliapp.RunContext) error {
	sessionID := ctx.Positional("session-id")
	if sessionID == "" {
		return fmt.Errorf("usage: workspace pane-delete <session-id>")
	}

	if _, err := h.client.DeletePane(context.Background(), connect.NewRequest(&workspacev1.DeletePaneRequest{SessionId: sessionID})); err != nil {
		return cliapp.WrapAPIError("workspace pane-delete", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Removed pane for session %s", sessionID)},
		NextCommand: []string{fmt.Sprintf("%s workspace layout-get", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

type groupCreateBody struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func (h *handlers) groupCreate(ctx cliapp.RunContext) error {
	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var body groupCreateBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	resp, err := h.client.CreateGroup(context.Background(), connect.NewRequest(&workspacev1.CreateGroupRequest{
		Name: body.Name, Color: body.Color,
	}))
	if err != nil {
		return cliapp.WrapAPIError("workspace group-create", err, nil)
	}
	g := resp.Msg.GetGroup()

	report := cliapp.MutationReport{
		Result: []string{"Created workspace group"},
		Changes: []string{
			fmt.Sprintf("ID: %s", g.GetId()),
			fmt.Sprintf("Name: %s", g.GetName()),
			fmt.Sprintf("Color: %s", g.GetColor()),
		},
		NextCommand: []string{fmt.Sprintf("%s workspace layout-get", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

type groupUpdateBody struct {
	Name        *string `json:"name,omitempty"`
	Color       *string `json:"color,omitempty"`
	IsCollapsed *bool   `json:"is_collapsed,omitempty"`
}

func (h *handlers) groupUpdate(ctx cliapp.RunContext) error {
	id := ctx.Positional("group-id")
	if id == "" {
		return fmt.Errorf("usage: workspace group-update <group-id> --body-file PATH")
	}

	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var body groupUpdateBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	req := &workspacev1.UpdateGroupRequest{Id: id}
	if body.Name != nil {
		req.Name = *body.Name
		req.HasName = true
	}
	if body.Color != nil {
		req.Color = *body.Color
		req.HasColor = true
	}
	if body.IsCollapsed != nil {
		req.IsCollapsed = *body.IsCollapsed
		req.HasIsCollapsed = true
	}

	if _, err := h.client.UpdateGroup(context.Background(), connect.NewRequest(req)); err != nil {
		return cliapp.WrapAPIError("workspace group-update", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated workspace group %s", id)},
		NextCommand: []string{fmt.Sprintf("%s workspace layout-get", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func (h *handlers) groupDelete(ctx cliapp.RunContext) error {
	id := ctx.Positional("group-id")
	if id == "" {
		return fmt.Errorf("usage: workspace group-delete <group-id>")
	}

	if _, err := h.client.DeleteGroup(context.Background(), connect.NewRequest(&workspacev1.DeleteGroupRequest{Id: id})); err != nil {
		return cliapp.WrapAPIError("workspace group-delete", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted workspace group %s", id)},
		NextCommand: []string{fmt.Sprintf("%s workspace layout-get", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

// ---------------------------------------------------------------------------
// Roles
// ---------------------------------------------------------------------------

func (h *handlers) roleList(ctx cliapp.RunContext) error {
	// The group id is optional here: omitting it lists every role, which is
	// the view you want when you are looking for a role and not sure which
	// group holds it.
	resp, err := h.client.ListRoles(context.Background(), connect.NewRequest(&workspacev1.ListRolesRequest{
		GroupId: ctx.Positional("group-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("workspace role-list", err, nil)
	}

	roles := resp.Msg.GetRoles()
	rows := make([]string, 0, len(roles))
	for _, r := range roles {
		// "waiting" is the whole point of the role table, so it is the first
		// thing each line reports.
		state := "waiting"
		if r.GetSessionId() != "" {
			state = "running " + support.ShortID(r.GetSessionId())
		}
		rows = append(rows, fmt.Sprintf("  role %s | group=%s | label=%s | %s | sort=%d | command=%s",
			support.ShortID(r.GetId()), support.ShortID(r.GetGroupId()), r.GetLabel(), state, r.GetSortOrder(), r.GetCommand()))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Roles: %d", len(roles))},
		ResultsHeading: "Roles",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s workspace role-create <group-id> --body-file role.json", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

// roleCreateBody mirrors CreateRoleRequest. session_id is omitted for a
// waiting role, which is the ordinary case.
type roleCreateBody struct {
	Label          string `json:"label"`
	Command        string `json:"command"`
	WorkingDir     string `json:"working_dir"`
	IncomingPrompt string `json:"incoming_prompt"`
	Backend        string `json:"backend"`
	TargetID       string `json:"target_id"`
	SessionID      string `json:"session_id"`
	SortOrder      *int32 `json:"sort_order,omitempty"`
}

func (h *handlers) roleCreate(ctx cliapp.RunContext) error {
	groupID := ctx.Positional("group-id")
	if groupID == "" {
		return fmt.Errorf("usage: workspace role-create <group-id> --body-file PATH")
	}

	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var body roleCreateBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	req := &workspacev1.CreateRoleRequest{
		GroupId:        groupID,
		Label:          body.Label,
		Command:        body.Command,
		WorkingDir:     body.WorkingDir,
		IncomingPrompt: body.IncomingPrompt,
		Backend:        body.Backend,
		TargetId:       body.TargetID,
		SessionId:      body.SessionID,
	}
	if body.SortOrder != nil {
		req.SortOrder = *body.SortOrder
	}

	resp, err := h.client.CreateRole(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("workspace role-create", err, nil)
	}
	r := resp.Msg.GetRole()

	report := cliapp.MutationReport{
		Result: []string{"Created workspace role"},
		Changes: []string{
			fmt.Sprintf("ID: %s", r.GetId()),
			fmt.Sprintf("Label: %s", r.GetLabel()),
			fmt.Sprintf("Group: %s", r.GetGroupId()),
			fmt.Sprintf("Waiting: %t", r.GetSessionId() == ""),
		},
		NextCommand: []string{fmt.Sprintf("%s workspace role-list %s", support.CLIName, groupID)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

// roleUpdateBody mirrors UpdateRoleRequest. Any field present in the JSON
// flips the corresponding has_* flag server-side, so a pointer distinguishes
// "absent" from "set to empty" — which is what clearing session_id (returning
// a role to waiting) needs.
type roleUpdateBody struct {
	Label          *string `json:"label,omitempty"`
	Command        *string `json:"command,omitempty"`
	WorkingDir     *string `json:"working_dir,omitempty"`
	IncomingPrompt *string `json:"incoming_prompt,omitempty"`
	SessionID      *string `json:"session_id,omitempty"`
	SortOrder      *int32  `json:"sort_order,omitempty"`
	Backend        *string `json:"backend,omitempty"`
	TargetID       *string `json:"target_id,omitempty"`
	GroupID        *string `json:"group_id,omitempty"`
}

func (h *handlers) roleUpdate(ctx cliapp.RunContext) error {
	id := ctx.Positional("role-id")
	if id == "" {
		return fmt.Errorf("usage: workspace role-update <role-id> --body-file PATH")
	}

	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var body roleUpdateBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	req := &workspacev1.UpdateRoleRequest{Id: id}
	if body.Label != nil {
		req.Label = *body.Label
		req.HasLabel = true
	}
	if body.Command != nil {
		req.Command = *body.Command
		req.HasCommand = true
	}
	if body.WorkingDir != nil {
		req.WorkingDir = *body.WorkingDir
		req.HasWorkingDir = true
	}
	if body.IncomingPrompt != nil {
		req.IncomingPrompt = *body.IncomingPrompt
		req.HasIncomingPrompt = true
	}
	if body.SessionID != nil {
		req.SessionId = *body.SessionID
		req.HasSessionId = true
	}
	if body.SortOrder != nil {
		req.SortOrder = *body.SortOrder
		req.HasSortOrder = true
	}
	if body.Backend != nil {
		req.Backend = *body.Backend
		req.HasBackend = true
	}
	if body.TargetID != nil {
		req.TargetId = *body.TargetID
		req.HasTargetId = true
	}
	if body.GroupID != nil {
		req.GroupId = *body.GroupID
		req.HasGroupId = true
	}

	if _, err := h.client.UpdateRole(context.Background(), connect.NewRequest(req)); err != nil {
		return cliapp.WrapAPIError("workspace role-update", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated workspace role %s", id)},
		NextCommand: []string{fmt.Sprintf("%s workspace role-list", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func (h *handlers) roleDelete(ctx cliapp.RunContext) error {
	id := ctx.Positional("role-id")
	if id == "" {
		return fmt.Errorf("usage: workspace role-delete <role-id>")
	}

	if _, err := h.client.DeleteRole(context.Background(), connect.NewRequest(&workspacev1.DeleteRoleRequest{Id: id})); err != nil {
		return cliapp.WrapAPIError("workspace role-delete", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted workspace role %s", id)},
		NextCommand: []string{fmt.Sprintf("%s workspace role-list", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}
