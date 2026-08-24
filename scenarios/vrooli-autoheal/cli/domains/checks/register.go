package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"connectrpc.com/connect"

	"vrooli-autoheal/cli/domains/watchdog"
	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	checksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks"
	checksconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks/checks_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

func LegacyRegister(core *cliapp.ScenarioApp, deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Recovery",
		Commands: []cliapp.Command{
			{
				Name:        "checks",
				NeedsAPI:    true,
				Description: "List registered health checks",
				Run: func(args []string) error {
					return runList(core, args)
				},
			},
			{
				Name:        "orphans",
				NeedsAPI:    true,
				Description: "Inspect or clean orphaned Vrooli processes",
				Run: func(args []string) error {
					return runCompatAction(core, "vrooli-orphans", args, "kill")
				},
			},
			{
				Name:        "locks",
				NeedsAPI:    true,
				Description: "Inspect or clean stale Vrooli port lock files",
				Run: func(args []string) error {
					return runCompatAction(core, "vrooli-stale-locks", args, "clean")
				},
			},
			{
				Name:        "watchdog",
				NeedsAPI:    true,
				Description: "Show OS watchdog status",
				Run: func(args []string) error {
					return watchdogStatus(core, args)
				},
			},
			{
				Name:        "install",
				NeedsAPI:    false,
				Description: "Delegate autoheal protection installation to project setup",
				Run: func(args []string) error {
					return watchdogInstall(core, args)
				},
			},
			{
				Name:        "uninstall",
				NeedsAPI:    true,
				Description: "Remove the OS watchdog service",
				Run: func(args []string) error {
					return watchdogUninstall(core, args)
				},
			},
		},
	}
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "check",
		Description: "Inspect checks, history, and recovery actions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List registered health checks", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Description: "Show the latest result for one check", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "history", Description: "Show recent history for one check", Run: func(args []string) error { return runHistory(core, args) }},
			{Name: "actions", Description: "List recovery actions for one check", Run: func(args []string) error { return runActions(core, args) }},
			{Name: "run-action", Description: "Execute a recovery action for one check", Run: func(args []string) error { return runAction(core, args) }},
			{Name: "reconcile", Description: "Compare registered scenario checks with the derived core set", Run: func(args []string) error { return runReconcile(core, args) }},
			{Name: "shelve", Description: "Shelve a check until a mandatory expiry", Run: func(args []string) error { return runShelve(core, args) }},
			{Name: "unshelve", Description: "Remove a check shelf", Run: func(args []string) error { return runUnshelve(core, args) }},
			{Name: "shelved", Description: "List active check shelves", Run: func(args []string) error { return runShelved(core, args) }},
			{Name: "saturation", Description: "Tally transitions for every registered check in one read", Run: func(args []string) error { return runSaturation(core, args) }},
		},
	}
}

func runReconcile(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("check reconcile")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := checksconnect.NewChecksServiceClient(httpClient, baseURL)
	response, err := client.GetReconcile(context.Background(), connect.NewRequest(&checksv1.GetReconcileRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("reconcile checks", err, nil)
	}
	if *jsonOutput {
		encoded, marshalErr := protojson.MarshalOptions{Indent: "  "}.Marshal(response.Msg)
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(os.Stdout, string(encoded))
		return nil
	}
	if !response.Msg.Reconcile.Available {
		return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{Status: []string{"Reconcile unavailable: " + response.Msg.Reconcile.UnavailableReason}})
	}
	status := []string{
		fmt.Sprintf("Ghost checks: %d", len(response.Msg.Reconcile.GhostCheckIds)),
		fmt.Sprintf("Out-of-scope checks: %d", len(response.Msg.Reconcile.OutOfScopeCheckIds)),
		fmt.Sprintf("Unsupervised plant: %d", len(response.Msg.Reconcile.UnsupervisedPlant)),
	}
	if !response.Msg.Reconcile.GhostDetectionAvailable {
		status = append(status, "Ghost detection unavailable: "+response.Msg.Reconcile.GhostUnavailableReason)
	}
	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: status,
		Triage: []cliapp.TriageGroup{
			{Heading: "Ghost checks (target no longer exists)", Items: response.Msg.Reconcile.GhostCheckIds},
			{Heading: "Out-of-scope checks (target exists, outside the core set)", Items: response.Msg.Reconcile.OutOfScopeCheckIds},
			{Heading: "Unsupervised plant", Items: response.Msg.Reconcile.UnsupervisedPlant},
		},
	})
}

func runSaturation(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("check saturation")
	windowHours := fs.Int("window-hours", 24, "Transition window in hours")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := checksconnect.NewChecksServiceClient(httpClient, baseURL)
	response, err := client.ListSaturation(context.Background(), connect.NewRequest(&checksv1.ListSaturationRequest{WindowHours: int32(*windowHours)}))
	if err != nil {
		return cliapp.WrapAPIError("list check saturation", err, nil)
	}
	if *jsonOutput {
		encoded, marshalErr := protojson.MarshalOptions{Indent: "  "}.Marshal(response.Msg)
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(os.Stdout, string(encoded))
		return nil
	}
	saturated := make([]string, 0)
	quiet := 0
	for _, item := range response.Msg.GetSaturations() {
		if item.GetSaturated() {
			saturated = append(saturated, item.GetCheckId())
			continue
		}
		if !item.GetTransitioned() {
			quiet++
		}
	}
	status := []string{
		fmt.Sprintf("Checks tallied: %d over %dh", len(response.Msg.GetSaturations()), response.Msg.GetWindowHours()),
		fmt.Sprintf("Saturated (pinned in a non-normal state): %d", len(saturated)),
		fmt.Sprintf("Quiet but healthy (no transition, currently OK): %d", quiet),
	}
	if response.Msg.GetTruncated() {
		status = append(status, "Transition window was truncated; saturation cannot be derived from this tally.")
	}
	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: status,
		Triage: []cliapp.TriageGroup{{Heading: "Saturated checks", Items: saturated}},
	})
}

func runShelve(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("check shelve")
	reason := fs.String("reason", "", "Why this check is temporarily shelved")
	expiry := fs.String("expiry", "", "Mandatory duration or future RFC3339 expiry")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 || strings.TrimSpace(*reason) == "" || strings.TrimSpace(*expiry) == "" {
		return fmt.Errorf("usage: vrooli-autoheal check shelve <check-id> --reason <text> --expiry <duration|timestamp>")
	}
	body, err := core.Request("POST", "/checks/"+url.PathEscape(fs.Arg(0))+"/shelve", nil, map[string]string{"reason": *reason, "expiry": *expiry})
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{Status: []string{fmt.Sprintf("Shelved %s until the requested expiry.", fs.Arg(0))}})
}

func runUnshelve(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("check unshelve")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: vrooli-autoheal check unshelve <check-id>")
	}
	body, err := core.Request("DELETE", "/checks/"+url.PathEscape(fs.Arg(0))+"/shelve", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{Status: []string{fmt.Sprintf("Unshelved %s.", fs.Arg(0))}})
}

func runShelved(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("check shelved")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get("/checks/shelves", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	var response struct {
		Shelves []struct {
			CheckID string `json:"checkId"`
			Reason  string `json:"reason"`
		} `json:"shelves"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}
	items := make([]string, 0, len(response.Shelves))
	for _, shelf := range response.Shelves {
		items = append(items, fmt.Sprintf("%s: %s", shelf.CheckID, shelf.Reason))
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{Summary: []string{fmt.Sprintf("Active shelves: %d", len(items))}, ResultsHeading: "Shelves", Results: items})
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("check list")
	statusFilter := fs.String("status", "", "Filter by current status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/checks", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}

	var checks []support.CheckInfo
	if err := support.Decode(body, &checks); err != nil {
		return err
	}
	sort.SliceStable(checks, func(i, j int) bool { return checks[i].ID < checks[j].ID })

	results := make([]string, 0, len(checks))
	for _, check := range checks {
		line := fmt.Sprintf("%s (%s, every %ds): %s", check.ID, check.Category, check.IntervalSeconds, check.Description)
		if len(check.Platforms) > 0 {
			line += fmt.Sprintf(" | platforms: %s", strings.Join(check.Platforms, ", "))
		}
		results = append(results, line)
	}
	if strings.TrimSpace(*statusFilter) != "" {
		filtered, err := filterByStatus(core, checks, *statusFilter)
		if err != nil {
			return err
		}
		results = filtered
	}

	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Registered checks: %d", len(results))},
		ResultsHeading: "Checks",
		Results:        results,
		RetrievalHints: []string{"vrooli-autoheal check get <check-id>", "vrooli-autoheal check actions <check-id>"},
	})
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("check get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: vrooli-autoheal check get <check-id>")
	}
	checkID := fs.Arg(0)

	body, err := core.Get("/checks/"+url.PathEscape(checkID), nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}

	var result support.CheckResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}
	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("%s %s", support.StatusGlyph(result.Status), result.CheckID),
			result.Message,
			fmt.Sprintf("Last run: %s", result.Timestamp.Format("2006-01-02 15:04:05Z07:00")),
			fmt.Sprintf("Duration: %s", result.Duration),
		},
		NextSteps: []string{
			fmt.Sprintf("vrooli-autoheal check history %s", checkID),
			fmt.Sprintf("vrooli-autoheal check actions %s", checkID),
		},
	})
}

func runHistory(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("check history")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: vrooli-autoheal check history <check-id>")
	}
	checkID := fs.Arg(0)

	body, err := core.Get("/checks/"+url.PathEscape(checkID)+"/history", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}

	var history support.CheckHistoryResponse
	if err := support.Decode(body, &history); err != nil {
		return err
	}
	results := make([]string, 0, len(history.History))
	for _, item := range history.History {
		results = append(results, fmt.Sprintf("%s %s %s: %s", item.Timestamp.Format("2006-01-02 15:04:05"), support.StatusGlyph(item.Status), item.CheckID, item.Message))
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("History entries for %s: %d", checkID, history.Count)},
		ResultsHeading: "History",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("vrooli-autoheal check get %s", checkID)},
	})
}

func runActions(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("check actions")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: vrooli-autoheal check actions <check-id>")
	}
	checkID := fs.Arg(0)

	body, err := core.Get("/checks/"+url.PathEscape(checkID)+"/actions", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}

	var resp support.CheckActionsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	results := make([]string, 0, len(resp.Actions))
	for _, action := range resp.Actions {
		results = append(results, fmt.Sprintf("%s (available=%s dangerous=%s): %s", action.ID, support.BoolWord(action.Available), support.BoolWord(action.Dangerous), action.Description))
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Recovery actions for %s: %d", checkID, len(resp.Actions))},
		ResultsHeading: "Actions",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("vrooli-autoheal check run-action %s <action-id>", checkID)},
	})
}

func runAction(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("check run-action")
	yes := fs.Bool("yes", false, "Skip confirmation for dangerous actions")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("usage: vrooli-autoheal check run-action <check-id> <action-id>")
	}
	checkID := fs.Arg(0)
	actionID := fs.Arg(1)

	actionsBody, err := core.Get("/checks/"+url.PathEscape(checkID)+"/actions", nil)
	if err != nil {
		return err
	}
	var actions support.CheckActionsResponse
	if err := support.Decode(actionsBody, &actions); err != nil {
		return err
	}
	var selected *support.RecoveryAction
	for i := range actions.Actions {
		if actions.Actions[i].ID == actionID {
			selected = &actions.Actions[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("action %q is not available for %s", actionID, checkID)
	}
	if selected.Dangerous && !*yes {
		confirmed, err := support.Confirm(fmt.Sprintf("Run dangerous action %s on %s?", actionID, checkID))
		if err != nil {
			return err
		}
		if !confirmed {
			return fmt.Errorf("aborted")
		}
	}

	body, err := core.Request("POST", "/checks/"+url.PathEscape(checkID)+"/actions/"+url.PathEscape(actionID), nil, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}

	var result support.ActionResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Action %s on %s completed with success=%s", result.ActionID, result.CheckID, support.BoolWord(result.Success)),
			result.Message,
		},
		Changes: []string{
			fmt.Sprintf("Duration: %s", result.Duration),
			strings.TrimSpace(result.Output),
			strings.TrimSpace(result.Error),
		},
		NextCommand: []string{
			fmt.Sprintf("vrooli-autoheal check get %s", checkID),
			fmt.Sprintf("vrooli-autoheal check history %s", checkID),
		},
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runCompatAction(core *cliapp.ScenarioApp, checkID string, args []string, dangerousAction string) error {
	action := "list"
	if len(args) > 0 {
		action = strings.TrimSpace(args[0])
	}
	switch action {
	case "list":
		return runActionByID(core, checkID, "list", false, false)
	case dangerousAction:
		return runActionByID(core, checkID, dangerousAction, true, false)
	default:
		return fmt.Errorf("unsupported action %q", action)
	}
}

func runActionByID(core *cliapp.ScenarioApp, checkID, actionID string, dangerous bool, jsonOutput bool) error {
	if dangerous {
		confirmed, err := support.Confirm(fmt.Sprintf("Run %s against %s?", actionID, checkID))
		if err != nil {
			return err
		}
		if !confirmed {
			return fmt.Errorf("aborted")
		}
	}
	body, err := core.Request("POST", "/checks/"+url.PathEscape(checkID)+"/actions/"+url.PathEscape(actionID), nil, nil)
	if err != nil {
		return err
	}
	if jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	var result support.ActionResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}
	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("%s %s on %s", support.StatusGlyph(boolStatus(result.Success)), result.ActionID, result.CheckID),
			result.Message,
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Output", Items: []string{support.TruncateLines(result.Output, 20)}},
		},
		NextSteps: []string{
			fmt.Sprintf("vrooli-autoheal check get %s", checkID),
			fmt.Sprintf("vrooli-autoheal check history %s", checkID),
		},
	})
}

func filterByStatus(core *cliapp.ScenarioApp, checks []support.CheckInfo, target string) ([]string, error) {
	target = strings.ToLower(strings.TrimSpace(target))
	var results []string
	for _, check := range checks {
		body, err := core.Get("/checks/"+url.PathEscape(check.ID), nil)
		if err != nil {
			return nil, err
		}
		var result support.CheckResult
		if err := support.Decode(body, &result); err != nil {
			return nil, err
		}
		if strings.ToLower(result.Status) != target {
			continue
		}
		results = append(results, fmt.Sprintf("%s (%s): %s", result.CheckID, result.Status, result.Message))
	}
	sort.Strings(results)
	return results, nil
}

func boolStatus(value bool) string {
	if value {
		return "ok"
	}
	return "critical"
}

func watchdogStatus(core *cliapp.ScenarioApp, args []string) error {
	return watchdog.RenderStatus(core, args)
}

func watchdogInstall(core *cliapp.ScenarioApp, args []string) error {
	return watchdog.Install(core, args)
}

func watchdogUninstall(core *cliapp.ScenarioApp, args []string) error {
	return watchdog.Uninstall(core, args)
}
