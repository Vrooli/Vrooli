package sync

import (
	"fmt"
	"os"
	"strings"

	"calendar/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

var supportedProviders = map[string]struct{}{
	"google":  {},
	"outlook": {},
}

// Register builds the `sync` subcommand group for external calendar integration.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "sync",
		Description: "Manage external calendar connections (Google, Outlook)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "connect", Description: "Start OAuth flow for a provider", Run: func(args []string) error { return runConnect(core, args) }},
			{Name: "disconnect", Description: "Disconnect an external calendar", Run: func(args []string) error { return runDisconnect(core, args) }},
			{Name: "status", Description: "Show external calendar connection status", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "run", Description: "Trigger a manual sync across connected calendars", Run: func(args []string) error { return runSync(core, args) }},
		},
	}
}

func runConnect(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("sync connect")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	provider, err := providerArg(fs, "connect")
	if err != nil {
		return err
	}

	body, err := core.Get("/external-sync/oauth/"+provider, nil)
	if err != nil {
		return err
	}
	var resp support.OAuthInitiateResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Started %s Calendar connection", provider)},
		Changes: []string{
			fmt.Sprintf("Authorization URL: %s", resp.AuthURL),
			fmt.Sprintf("State: %s", resp.State),
		},
		NextCommand: []string{
			"Visit the authorization URL above to complete the OAuth flow",
			fmt.Sprintf("%s sync status", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDisconnect(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("sync disconnect")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	provider, err := providerArg(fs, "disconnect")
	if err != nil {
		return err
	}

	body, err := core.Request("DELETE", "/external-sync/disconnect/"+provider, nil, nil)
	if err != nil {
		return err
	}
	var resp support.DisconnectResponse
	if err := support.Decode(body, &resp); err != nil {
		// Non-envelope error shape; fall back to a generic confirmation.
		resp = support.DisconnectResponse{Provider: provider, Disconnected: true, Success: true}
	}

	message := fmt.Sprintf("Disconnected %s Calendar", provider)
	if !resp.Disconnected {
		message = fmt.Sprintf("No active %s Calendar connection to disconnect", provider)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		NextCommand: []string{fmt.Sprintf("%s sync status", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("sync status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/external-sync/status", nil)
	if err != nil {
		return err
	}
	var resp support.SyncStatusResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	rows := make([]string, 0, len(resp.Connections))
	for _, c := range resp.Connections {
		state := "connected"
		if !c.Connected {
			state = "disconnected"
		}
		enabled := "enabled"
		if !c.SyncEnabled {
			enabled = "disabled"
		}
		row := fmt.Sprintf("%s | %s | sync=%s", strings.ToUpper(c.Provider), state, enabled)
		if len(c.LastSync) > 0 && string(c.LastSync) != "null" {
			row += fmt.Sprintf(" | last_sync=%s", strings.Trim(string(c.LastSync), "\""))
		}
		if c.SyncDirection != "" {
			row += fmt.Sprintf(" | direction=%s", c.SyncDirection)
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		rows = []string{"No external calendars connected"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("External calendar connections: %d", len(resp.Connections))},
		ResultsHeading: "Connections",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s sync connect google", support.CLIName),
			fmt.Sprintf("%s sync run", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSync(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("sync run")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Request("POST", "/external-sync/sync", nil, map[string]interface{}{})
	if err != nil {
		return err
	}
	var resp support.SyncRunResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	if len(resp.Results) == 0 {
		report := cliapp.MutationReport{
			Result:      []string{"No calendars to sync"},
			NextCommand: []string{fmt.Sprintf("%s sync connect google", support.CLIName)},
		}
		if *jsonOutput {
			return cliapp.PrintReportJSON(os.Stdout, report)
		}
		return cliapp.RenderMutationReport(os.Stdout, report)
	}

	changes := make([]string, 0, len(resp.Results))
	for _, r := range resp.Results {
		changes = append(changes, fmt.Sprintf("%s: %s | created=%d updated=%d synced=%d",
			strings.ToUpper(r.Provider), r.Status, r.EventsCreated, r.EventsUpdated, r.EventsSynced))
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Sync completed across %d provider(s)", len(resp.Results))},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s sync status", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func providerArg(fs interface {
	Arg(int) string
	NArg() int
}, verb string,
) (string, error) {
	if fs.NArg() < 1 {
		return "", fmt.Errorf("usage: sync %s <provider> (google|outlook)", verb)
	}
	provider := strings.ToLower(strings.TrimSpace(fs.Arg(0)))
	if _, ok := supportedProviders[provider]; !ok {
		return "", fmt.Errorf("invalid provider %q — must be google or outlook", provider)
	}
	return provider, nil
}
