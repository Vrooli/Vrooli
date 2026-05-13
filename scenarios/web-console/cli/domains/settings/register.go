package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"connectrpc.com/connect"

	settingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/settings"
	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/settings/settings_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"web-console/cli/internal/support"
)

// Register builds the `settings` subcommand group for scenario-wide
// configuration (currently: session defaults). Calls Connect-RPC
// SettingsService directly via the generated client.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "settings",
		Description: "Inspect and update web-console settings",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "session-defaults-get", Aliases: []string{"session-defaults"}, Description: "Show default values applied to new sessions", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "session-defaults-set", Description: "Update default session settings (--body-file PATH)", Run: func(args []string) error { return runSet(core, args) }},
		},
	}
}

func newClient(core *cliapp.ScenarioApp) settingsconnect.SettingsServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return settingsconnect.NewSettingsServiceClient(httpClient, baseURL)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("settings session-defaults-get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	client := newClient(core)
	resp, err := client.GetSessionDefaults(context.Background(), connect.NewRequest(&settingsv1.GetSessionDefaultsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get session defaults", err, nil)
	}

	report := cliapp.ListReport{
		Summary:        []string{"Session defaults"},
		ResultsHeading: "Values",
		Results:        defaultsRows(resp.Msg.GetDefaults()),
		RetrievalHints: []string{fmt.Sprintf("%s settings session-defaults-set --body-file defaults.json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("settings session-defaults-set")
	bodyFile := fs.String("body-file", "", "Path to a JSON body with session defaults (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	// Map the legacy JSON body shape (default_backend + default_policy)
	// onto the proto request. Unknown fields are ignored; the wire-level
	// validation is server-side.
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	var legacy struct {
		DefaultBackend *string `json:"default_backend,omitempty"`
		DefaultPolicy  *struct {
			Mode     string `json:"mode"`
			Duration string `json:"duration,omitempty"`
		} `json:"default_policy,omitempty"`
	}
	if err := json.Unmarshal(body, &legacy); err != nil {
		return fmt.Errorf("decode body: %w", err)
	}

	req := &settingsv1.UpdateSessionDefaultsRequest{}
	if legacy.DefaultBackend != nil {
		v := *legacy.DefaultBackend
		req.DefaultBackend = &v
	}
	if legacy.DefaultPolicy != nil {
		req.DefaultPolicy = &settingsv1.ExpirationPolicy{
			Mode:     legacy.DefaultPolicy.Mode,
			Duration: legacy.DefaultPolicy.Duration,
		}
	}

	client := newClient(core)
	if _, err := client.UpdateSessionDefaults(context.Background(), connect.NewRequest(req)); err != nil {
		return cliapp.WrapAPIError("update session defaults", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{"Updated session defaults"},
		NextCommand: []string{fmt.Sprintf("%s settings session-defaults-get", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func defaultsRows(d *settingsv1.SessionDefaults) []string {
	if d == nil {
		return nil
	}
	rows := []string{
		fmt.Sprintf("default_backend: %s", d.GetDefaultBackend()),
	}
	if p := d.GetDefaultPolicy(); p != nil {
		rows = append(rows,
			fmt.Sprintf("default_policy.mode: %s", p.GetMode()),
			fmt.Sprintf("default_policy.duration: %s", p.GetDuration()),
		)
	}
	return rows
}
