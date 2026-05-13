package shortcuts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"connectrpc.com/connect"

	shortcutsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shortcuts"
	shortcutsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shortcuts/shortcuts_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"web-console/cli/internal/support"
)

// Register builds the `shortcuts` subcommand group covering the effective
// shortcut view and per-profile CRUD. Calls Connect-RPC ShortcutsService
// directly via the generated client.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "shortcuts",
		Description: "Inspect and manage shortcut profiles",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "effective", Description: "Show the effective shortcut list", Run: func(args []string) error { return runEffective(core, args) }},
			{Name: "list", Description: "List all shortcut profiles", Run: func(args []string) error { return runList(core, args) }},
			{Name: "upsert", Description: "Create or update a shortcut profile (--body-file PATH)", Run: func(args []string) error { return runUpsert(core, args) }},
			{Name: "delete", Description: "Delete a shortcut profile", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func newClient(core *cliapp.ScenarioApp) shortcutsconnect.ShortcutsServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return shortcutsconnect.NewShortcutsServiceClient(httpClient, baseURL)
}

func runEffective(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("shortcuts effective")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	resp, err := newClient(core).GetEffective(context.Background(), connect.NewRequest(&shortcutsv1.GetEffectiveRequest{}))
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
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("shortcuts list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	resp, err := newClient(core).ListProfiles(context.Background(), connect.NewRequest(&shortcutsv1.ListProfilesRequest{}))
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
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// upsertBody is the on-disk JSON shape the `upsert` command consumes.
// Mirrors UpsertProfileRequest field-for-field with snake_case keys to
// match the rest of the CLI's body-file conventions.
type upsertBody struct {
	ID        string        `json:"id"`
	Scope     string        `json:"scope"`
	Name      string        `json:"name"`
	Shortcuts []bodyEntry   `json:"shortcuts"`
}

type bodyEntry struct {
	Label       string `json:"label"`
	Command     string `json:"command"`
	Description string `json:"description,omitempty"`
}

func runUpsert(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("shortcuts upsert")
	bodyFile := fs.String("body-file", "", "Path to a JSON profile body (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
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

	resp, err := newClient(core).UpsertProfile(context.Background(), connect.NewRequest(req))
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
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("shortcuts delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: shortcuts delete <profile-id>")
	}
	id := fs.Arg(0)

	if _, err := newClient(core).DeleteProfile(context.Background(), connect.NewRequest(&shortcutsv1.DeleteProfileRequest{Id: id})); err != nil {
		return cliapp.WrapAPIError("shortcuts delete", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted shortcut profile %s", id)},
		NextCommand: []string{fmt.Sprintf("%s shortcuts list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
