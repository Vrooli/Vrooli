package host

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "host",
		Description: "Inspect host inventory and host-integrity drift",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "inventory", Description: "Show latest host inventory snapshot", Run: func(args []string) error { return inventory(core, args) }},
			{Name: "collect", Description: "Collect and persist host inventory now", Run: func(args []string) error { return collect(core, args) }},
			{Name: "changes", Description: "Show recent host inventory changes", Run: func(args []string) error { return changes(core, args) }},
		},
	}
}

func inventory(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("host inventory")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get("/host/inventory", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	var resp support.HostInventoryResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	if resp.Snapshot == nil {
		return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
			Summary:        []string{"No host inventory snapshot has been collected."},
			RetrievalHints: []string{"vrooli-autoheal host collect"},
		})
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Snapshot: %s", resp.Snapshot.ID),
			fmt.Sprintf("Platform: %s/%s %s", resp.Snapshot.Platform, resp.Snapshot.OS, resp.Snapshot.Arch),
			fmt.Sprintf("Kernel: %s", resp.Snapshot.KernelRelease),
			fmt.Sprintf("Fresh: %s (%ds old)", support.BoolWord(resp.Fresh), resp.AgeSeconds),
		},
		ResultsHeading: "Probe Status",
		Results:        probeStatusLines(resp.ProbeStatus),
		RetrievalHints: []string{"vrooli-autoheal host changes", "vrooli-autoheal check get host-capability-drift"},
	})
}

func collect(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("host collect")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Request("POST", "/host/inventory/collect", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	var resp struct {
		Snapshot *support.HostInventorySnapshot `json:"snapshot"`
		Changes  []support.HostInventoryChange  `json:"changes"`
	}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	summary := []string{"Host inventory collected."}
	if resp.Snapshot != nil {
		summary = append(summary, fmt.Sprintf("Snapshot: %s", resp.Snapshot.ID), fmt.Sprintf("Fingerprint: %s", resp.Snapshot.Fingerprint))
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Changes",
		Results:        changeLines(resp.Changes),
		RetrievalHints: []string{"vrooli-autoheal host inventory --json", "vrooli-autoheal host changes"},
	})
}

func changes(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("host changes")
	limit := fs.Int("limit", 50, "Maximum changes")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	query := url.Values{"limit": []string{strconv.Itoa(*limit)}}
	body, err := core.Get("/host/inventory/changes", query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	var resp support.HostInventoryChangesResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Host inventory changes: %d", resp.Total)},
		ResultsHeading: "Changes",
		Results:        changeLines(resp.Changes),
		RetrievalHints: []string{"vrooli-autoheal host collect"},
	})
}

func probeStatusLines(probes map[string]string) []string {
	lines := make([]string, 0, len(probes))
	for name, status := range probes {
		lines = append(lines, fmt.Sprintf("%s: %s", name, status))
	}
	return lines
}

func changeLines(changes []support.HostInventoryChange) []string {
	lines := make([]string, 0, len(changes))
	for _, change := range changes {
		lines = append(lines, fmt.Sprintf("%s %s %s: %s", change.CreatedAt, change.Severity, change.ChangeType, change.Summary))
	}
	if len(lines) == 0 {
		return []string{"No host inventory changes recorded."}
	}
	return lines
}
