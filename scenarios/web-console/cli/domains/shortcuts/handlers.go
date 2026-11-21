package shortcuts

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"

	shortcutsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shortcuts"
	shortcutsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shortcuts/shortcuts_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client shortcutsconnect.ShortcutsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: shortcutsconnect.NewShortcutsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) effective(ctx cliapp.RunContext) error {
	resp, err := h.client.GetEffective(context.Background(), connect.NewRequest(&shortcutsv1.GetEffectiveRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("shortcuts effective", err, nil)
	}

	rows := make([]string, 0, len(resp.Msg.GetShortcuts()))
	for _, s := range resp.Msg.GetShortcuts() {
		line := fmt.Sprintf("%s — %s", s.GetLabel(), s.GetCommand())
		if d := s.GetDescription(); d != "" {
			line = line + " (" + d + ")"
		}
		rows = append(rows, line)
	}
	if len(rows) == 0 {
		rows = []string{"(no shortcuts)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{"Effective shortcuts"},
		ResultsHeading: "Bindings",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s shortcuts list", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListProfiles(context.Background(), connect.NewRequest(&shortcutsv1.ListProfilesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("shortcuts list", err, nil)
	}

	rows := make([]string, 0, len(resp.Msg.GetProfiles()))
	for _, p := range resp.Msg.GetProfiles() {
		rows = append(rows, fmt.Sprintf("%s | %s | scope=%s | shortcuts=%d",
			support.ShortID(p.GetId()), p.GetName(), p.GetScope(), len(p.GetShortcuts())))
	}
	if len(rows) == 0 {
		rows = []string{"(no profiles)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Shortcut profiles: %d", len(resp.Msg.GetProfiles()))},
		ResultsHeading: "Profiles",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s shortcuts upsert --body-file profile.json", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

// upsertBody is the on-disk JSON shape the `upsert` command consumes.
// Mirrors UpsertProfileRequest field-for-field with snake_case keys to
// match the rest of the CLI's body-file conventions.
type upsertBody struct {
	ID        string      `json:"id"`
	Scope     string      `json:"scope"`
	Name      string      `json:"name"`
	Shortcuts []bodyEntry `json:"shortcuts"`
}

type bodyEntry struct {
	Label       string `json:"label"`
	Command     string `json:"command"`
	Description string `json:"description,omitempty"`
}

func (h *handlers) upsert(ctx cliapp.RunContext) error {
	raw, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}
	var body upsertBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("decode --body-file: %w", err)
	}

	req := &shortcutsv1.UpsertProfileRequest{
		Id:        body.ID,
		Scope:     body.Scope,
		Name:      body.Name,
		Shortcuts: make([]*shortcutsv1.Shortcut, 0, len(body.Shortcuts)),
	}
	for _, e := range body.Shortcuts {
		req.Shortcuts = append(req.Shortcuts, &shortcutsv1.Shortcut{
			Label:       e.Label,
			Command:     e.Command,
			Description: e.Description,
		})
	}

	resp, err := h.client.UpsertProfile(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("shortcuts upsert", err, nil)
	}
	p := resp.Msg.GetProfile()

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Upserted shortcut profile %s", p.GetName())},
		Changes: []string{
			fmt.Sprintf("ID: %s", p.GetId()),
			fmt.Sprintf("Scope: %s", p.GetScope()),
			fmt.Sprintf("Shortcuts: %d", len(p.GetShortcuts())),
		},
		NextCommand: []string{fmt.Sprintf("%s shortcuts list", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	id := ctx.Positional("profile-id")
	if id == "" {
		return fmt.Errorf("usage: shortcuts delete <profile-id>")
	}

	if _, err := h.client.DeleteProfile(context.Background(), connect.NewRequest(&shortcutsv1.DeleteProfileRequest{Id: id})); err != nil {
		return cliapp.WrapAPIError("shortcuts delete", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted shortcut profile %s", id)},
		NextCommand: []string{fmt.Sprintf("%s shortcuts list", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}
