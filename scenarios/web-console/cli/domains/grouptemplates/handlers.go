package grouptemplates

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"

	grouptemplatesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/grouptemplates"
	grouptemplatesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/grouptemplates/grouptemplates_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client grouptemplatesconnect.GroupTemplatesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: grouptemplatesconnect.NewGroupTemplatesServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListTemplates(context.Background(), connect.NewRequest(&grouptemplatesv1.ListTemplatesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("group-template list", err, nil)
	}

	templates := resp.Msg.GetTemplates()
	rows := make([]string, 0, len(templates))
	for _, t := range templates {
		rows = append(rows, fmt.Sprintf("  %s | %s | roles=%d | used=%d",
			support.ShortID(t.GetId()), t.GetName(), len(t.GetRoles()), t.GetUseCount()))
		for _, r := range t.GetRoles() {
			// start_mode is the field that decides whether creating a group
			// from this template costs a process, so name it on every line.
			rows = append(rows, fmt.Sprintf("      role %s | %s | %s", r.GetLabel(), r.GetStartMode(), r.GetCommand()))
		}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Group templates: %d", len(templates))},
		ResultsHeading: "Templates",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s group-template upsert --body-file template.json", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

type templateRoleBody struct {
	Label          string `json:"label"`
	Command        string `json:"command"`
	WorkingDir     string `json:"working_dir"`
	IncomingPrompt string `json:"incoming_prompt"`
	Backend        string `json:"backend"`
	TargetID       string `json:"target_id"`
	StartMode      string `json:"start_mode"`
}

type templateUpsertBody struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Color    string             `json:"color"`
	Roles    []templateRoleBody `json:"roles"`
	UseCount *int32             `json:"use_count,omitempty"`
}

func (h *handlers) upsert(ctx cliapp.RunContext) error {
	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var body templateUpsertBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	roles := make([]*grouptemplatesv1.TemplateRole, 0, len(body.Roles))
	for _, r := range body.Roles {
		roles = append(roles, &grouptemplatesv1.TemplateRole{
			Label:          r.Label,
			Command:        r.Command,
			WorkingDir:     r.WorkingDir,
			IncomingPrompt: r.IncomingPrompt,
			Backend:        r.Backend,
			TargetId:       r.TargetID,
			StartMode:      r.StartMode,
		})
	}

	req := &grouptemplatesv1.UpsertTemplateRequest{
		Id:    body.ID,
		Name:  body.Name,
		Color: body.Color,
		Roles: roles,
	}
	// Editing content must not reset the counter, so it moves only when the
	// caller explicitly names it.
	if body.UseCount != nil {
		req.UseCount = *body.UseCount
		req.HasUseCount = true
	}

	resp, err := h.client.UpsertTemplate(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("group-template upsert", err, nil)
	}
	t := resp.Msg.GetTemplate()

	report := cliapp.MutationReport{
		Result: []string{"Saved group template"},
		Changes: []string{
			fmt.Sprintf("ID: %s", t.GetId()),
			fmt.Sprintf("Name: %s", t.GetName()),
			fmt.Sprintf("Roles: %d", len(t.GetRoles())),
		},
		NextCommand: []string{fmt.Sprintf("%s group-template list", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	id := ctx.Positional("template-id")
	if id == "" {
		return fmt.Errorf("usage: group-template delete <template-id>")
	}

	if _, err := h.client.DeleteTemplate(context.Background(), connect.NewRequest(&grouptemplatesv1.DeleteTemplateRequest{Id: id})); err != nil {
		return cliapp.WrapAPIError("group-template delete", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted group template %s", id)},
		NextCommand: []string{fmt.Sprintf("%s group-template list", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}
