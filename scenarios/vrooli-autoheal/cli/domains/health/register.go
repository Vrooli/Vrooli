package health

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp, deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Operations",
		Commands: []cliapp.Command{
			{
				Name:        "status",
				NeedsAPI:    true,
				Description: "Show current health summary and triage",
				Run: func(args []string) error {
					return runStatus(core, args)
				},
			},
			{
				Name:        "tick",
				NeedsAPI:    true,
				Description: "Run a single health check cycle",
				Run: func(args []string) error {
					return runTick(core, args)
				},
			},
			{
				Name:        "loop",
				Description: "Run the dedicated autoheal loop binary",
				Run: func(args []string) error {
					if deps.RunLoop == nil {
						return fmt.Errorf("loop runner is not configured")
					}
					return deps.RunLoop(args)
				},
			},
			{
				Name:        "platform",
				NeedsAPI:    true,
				Description: "Show detected platform capabilities",
				Run: func(args []string) error {
					return runPlatform(core, args)
				},
			},
			{
				Name:        "diagnose-port",
				Description: "Diagnose a port conflict and stale lock state",
				Run: func(args []string) error {
					if deps.DiagnosePort == nil {
						return fmt.Errorf("port diagnostics are not configured")
					}
					return deps.DiagnosePort(args)
				},
			},
		},
	}
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/status", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}

	var resp support.StatusResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	status := []string{
		fmt.Sprintf("Overall status: %s", strings.ToUpper(resp.Status)),
		fmt.Sprintf("Checks: %d total, %d OK, %d warning, %d critical", resp.Summary.Total, resp.Summary.OK, resp.Summary.Warning, resp.Summary.Critical),
		fmt.Sprintf("Platform: %s", resp.Platform.Platform),
	}
	if !resp.Timestamp.IsZero() {
		status = append(status, fmt.Sprintf("Last update: %s", resp.Timestamp.Format("2006-01-02 15:04:05Z07:00")))
	}
	if resp.TickRunning {
		status = append(status, "A tick is currently running.")
	}

	triage := []cliapp.TriageGroup{
		{
			Heading: "Platform",
			Items: []string{
				fmt.Sprintf("Docker available: %s", support.BoolWord(resp.Platform.HasDocker)),
				fmt.Sprintf("systemd available: %s", support.BoolWord(resp.Platform.SupportsSystemd)),
				fmt.Sprintf("Cloudflared supported: %s", support.BoolWord(resp.Platform.SupportsCloudflared)),
				fmt.Sprintf("WSL detected: %s", support.BoolWord(resp.Platform.IsWsl)),
				fmt.Sprintf("Headless server: %s", support.BoolWord(resp.Platform.IsHeadlessServer)),
			},
		},
	}

	critical := filterChecks(resp.Checks, "critical")
	warnings := filterChecks(resp.Checks, "warning")
	if len(critical) > 0 {
		triage = append(triage, cliapp.TriageGroup{Heading: "Critical Checks", Items: critical})
	}
	if len(warnings) > 0 {
		triage = append(triage, cliapp.TriageGroup{Heading: "Warnings", Items: warnings})
	}
	if len(critical) == 0 && len(warnings) == 0 {
		triage = append(triage, cliapp.TriageGroup{Heading: "Health Checks", Items: []string{"No warning or critical checks are currently active."}})
	}

	nextSteps := []string{
		"vrooli-autoheal tick",
		"vrooli-autoheal check list",
		"vrooli-autoheal watchdog",
	}
	if len(critical) > 0 {
		nextSteps = append([]string{"vrooli-autoheal check actions <check-id>"}, nextSteps...)
	}

	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status:    status,
		Triage:    triage,
		NextSteps: nextSteps,
	})
}

func runTick(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tick")
	force := fs.Bool("force", false, "Ignore interval restrictions")
	compact := fs.Bool("compact", false, "Request compact API response")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if *force {
		query.Set("force", "true")
	}
	if *compact {
		query.Set("compact", "true")
	}

	body, err := core.Request("POST", "/tick", query, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}

	var resp support.TickResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	triage := []cliapp.TriageGroup{}
	critical := filterChecks(resp.Results, "critical")
	warnings := filterChecks(resp.Results, "warning")
	if len(critical) > 0 {
		triage = append(triage, cliapp.TriageGroup{Heading: "Critical Checks", Items: critical})
	}
	if len(warnings) > 0 {
		triage = append(triage, cliapp.TriageGroup{Heading: "Warnings", Items: warnings})
	}
	if len(resp.Warnings) > 0 {
		triage = append(triage, cliapp.TriageGroup{Heading: "Persistence Warnings", Items: resp.Warnings})
	}

	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Tick completed with overall status %s.", strings.ToUpper(resp.Status)),
			fmt.Sprintf("Results: %d total, %d OK, %d warning, %d critical", resp.Summary.Total, resp.Summary.OK, resp.Summary.Warning, resp.Summary.Critical),
		},
		Triage: triage,
		NextSteps: []string{
			"vrooli-autoheal status",
			"vrooli-autoheal check list --status critical",
			"vrooli-autoheal actions history",
		},
	})
}

func runPlatform(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("platform")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/platform", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}

	var platform support.PlatformInfo
	if err := support.Decode(body, &platform); err != nil {
		return err
	}

	capabilities := map[string]bool{
		"docker":      platform.HasDocker,
		"systemd":     platform.SupportsSystemd,
		"launchd":     platform.SupportsLaunchd,
		"windows svc": platform.SupportsWindowsServices,
		"rdp":         platform.SupportsRdp,
		"cloudflared": platform.SupportsCloudflared,
		"headless":    platform.IsHeadlessServer,
		"wsl":         platform.IsWsl,
	}
	keys := make([]string, 0, len(capabilities))
	for key := range capabilities {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	results := make([]string, 0, len(keys))
	for _, key := range keys {
		results = append(results, fmt.Sprintf("%s: %s", key, support.BoolWord(capabilities[key])))
	}

	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Platform: %s", platform.Platform)},
		ResultsHeading: "Capabilities",
		Results:        results,
		RetrievalHints: []string{"vrooli-autoheal watchdog", "vrooli-autoheal status"},
	})
}

func filterChecks(checks []support.CheckResult, target string) []string {
	lines := make([]string, 0)
	for _, check := range checks {
		if !strings.EqualFold(check.Status, target) {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s: %s", support.StatusGlyph(check.Status), check.CheckID, check.Message))
	}
	sort.Strings(lines)
	return lines
}
