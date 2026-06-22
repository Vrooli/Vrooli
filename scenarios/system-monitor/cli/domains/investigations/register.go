package investigations

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"system-monitor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "investigations",
		Description: "List, inspect, trigger, and tune anomaly investigations",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List recent investigations", Run: func(args []string) error { return runList(core, args) }},
			{Name: "latest", Description: "Get the latest investigation", Run: func(args []string) error { return runGet(core, args, true) }},
			{Name: "get", Description: "Get an investigation by ID", Run: func(args []string) error { return runGet(core, args, false) }},
			{Name: "trigger", Description: "Trigger a new investigation", Run: func(args []string) error { return runTrigger(core, args) }},
			{Name: "cooldown", Description: "Show cooldown status", Run: func(args []string) error { return runCooldown(core, args) }},
			{Name: "cooldown-reset", Description: "Reset the investigation cooldown", Run: func(args []string) error { return runCooldownReset(core, args) }},
			{Name: "cooldown-set", Description: "Update the cooldown duration", Run: func(args []string) error { return runCooldownSet(core, args) }},
			{Name: "triggers", Description: "List trigger thresholds", Run: func(args []string) error { return runTriggers(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("investigations list")
	limit := fs.Int("limit", 20, "Maximum number of investigations to return")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *limit <= 0 {
		return fmt.Errorf("--limit must be greater than 0")
	}

	body, err := core.Get("/investigations", map[string][]string{"limit": {fmt.Sprintf("%d", *limit)}})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var response apipb.ListInvestigationsResponse
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Investigations returned: %d", len(response.GetInvestigations())),
			fmt.Sprintf("Limit applied: %d", *limit),
		},
		ResultsHeading: "Investigations",
		Results:        investigationRows(response.GetInvestigations()),
		RetrievalHints: []string{"system-monitor investigations latest", "system-monitor investigations get <id>", "system-monitor investigations trigger --note \"describe the issue\""},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string, latest bool) error {
	fs := support.NewFlagSet("investigations get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	path := "/investigations/latest"
	if !latest {
		if fs.NArg() != 1 {
			return fmt.Errorf("usage: system-monitor investigations get <id>")
		}
		path = "/investigations/" + strings.TrimSpace(fs.Arg(0))
	}

	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var response domainpb.Investigation
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}
	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Investigation ID: %s", response.GetId()),
			fmt.Sprintf("Status: %s", strings.TrimPrefix(response.GetStatus().String(), "INVESTIGATION_STATUS_")),
			fmt.Sprintf("Progress: %d%%", response.GetProgress()),
			fmt.Sprintf("Started: %s", support.FormatTimestamp(response.GetStartTime())),
			fmt.Sprintf("Ended: %s", support.FormatTimestamp(response.GetEndTime())),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Findings", Items: []string{support.FormatMaybeString(response.GetFindings(), "No findings recorded yet.")}},
			{Heading: "Steps", Items: investigationStepRows(response.GetSteps())},
		},
		NextSteps: []string{
			"system-monitor investigations list --limit 10",
			fmt.Sprintf("system-monitor investigations get %s --json", response.GetId()),
		},
	}
	if details := response.GetDetails(); details != nil && len(details.GetFields()) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Details",
			Items:   []string{structSummary(details)},
		})
	}
	if response.GetStatus() != domainpb.InvestigationStatus_INVESTIGATION_STATUS_COMPLETED {
		report.NextSteps = append(report.NextSteps, "system-monitor investigations latest")
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runTrigger(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("investigations trigger")
	autoFix := fs.Bool("auto-fix", false, "Request automatic remediation")
	note := fs.String("note", "", "Context for the investigation")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	request := &apipb.TriggerInvestigationRequest{
		AutoFix: *autoFix,
		Note:    strings.TrimSpace(*note),
	}
	body, err := core.Request("POST", "/investigations/trigger", nil, request)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var response apipb.TriggerInvestigationResponse
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Investigation %s queued.", response.GetInvestigationId()),
			fmt.Sprintf("Status: %s", response.GetStatus()),
		},
		Changes: []string{
			fmt.Sprintf("Auto-fix requested: %s", support.BoolString(response.GetAutoFix(), "yes", "no")),
			fmt.Sprintf("Note: %s", support.FormatMaybeString(response.GetNote(), "none")),
		},
		NextCommand: []string{
			"system-monitor investigations latest",
			fmt.Sprintf("system-monitor investigations get %s", response.GetInvestigationId()),
		},
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runCooldown(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("investigations cooldown")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/investigations/cooldown", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var response apipb.GetCooldownStatusResponse
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}
	cooldown := response.GetCooldown()
	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Ready: %s", support.BoolString(cooldown.GetIsReady(), "yes", "no")),
			fmt.Sprintf("Cooldown period: %ds", cooldown.GetCooldownPeriodSeconds()),
			fmt.Sprintf("Remaining: %ds", cooldown.GetRemainingSeconds()),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Timing", Items: []string{fmt.Sprintf("Last trigger time: %s", support.FormatTimestamp(cooldown.GetLastTriggerTime()))}},
		},
		NextSteps: []string{"system-monitor investigations cooldown-reset", "system-monitor investigations cooldown-set --seconds 120"},
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runCooldownReset(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("investigations cooldown-reset")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Request("POST", "/investigations/cooldown/reset", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{"Investigation cooldown reset."},
		Changes:     []string{"The next investigation can run immediately."},
		NextCommand: []string{"system-monitor investigations cooldown", "system-monitor investigations trigger --note \"run a fresh diagnostic\""},
	})
}

func runCooldownSet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("investigations cooldown-set")
	seconds := fs.Int("seconds", 0, "Cooldown duration in seconds")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *seconds <= 0 {
		return fmt.Errorf("--seconds must be greater than 0")
	}

	body, err := core.Request("PUT", "/investigations/cooldown/period", nil, map[string]int{"cooldown_period_seconds": *seconds})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:      []string{"Investigation cooldown updated."},
		Changes:     []string{fmt.Sprintf("New cooldown period: %ds", *seconds)},
		NextCommand: []string{"system-monitor investigations cooldown"},
	})
}

func runTriggers(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("investigations triggers")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/investigations/triggers", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var response apipb.GetTriggersResponse
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Triggers configured: %d", len(response.GetTriggers())),
		},
		ResultsHeading: "Triggers",
		Results:        triggerRows(response.GetTriggers()),
		RetrievalHints: []string{"system-monitor investigations cooldown", "system-monitor status"},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func investigationRows(items []*domainpb.Investigation) []string {
	if len(items) == 0 {
		return []string{"No investigations were returned."}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("%s status=%s progress=%d%% started=%s findings=%s", item.GetId(), strings.TrimPrefix(item.GetStatus().String(), "INVESTIGATION_STATUS_"), item.GetProgress(), support.FormatTimestamp(item.GetStartTime()), support.FormatMaybeString(item.GetFindings(), "pending")))
	}
	return rows
}

func investigationStepRows(steps []*domainpb.InvestigationStep) []string {
	if len(steps) == 0 {
		return []string{"No investigation steps have been recorded."}
	}
	rows := make([]string, 0, len(steps))
	for _, step := range steps {
		rows = append(rows, fmt.Sprintf("%s status=%s started=%s findings=%s", step.GetName(), strings.TrimPrefix(step.GetStatus().String(), "INVESTIGATION_STEP_STATUS_"), support.FormatTimestamp(step.GetStartTime()), support.FormatMaybeString(step.GetFindings(), "none")))
	}
	return rows
}

func triggerRows(items map[string]*domainpb.TriggerConfig) []string {
	if len(items) == 0 {
		return []string{"No investigation triggers are configured."}
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([]string, 0, len(keys))
	for _, key := range keys {
		item := items[key]
		rows = append(rows, fmt.Sprintf("%s enabled=%t autoFix=%t threshold=%.2f%s condition=%s", key, item.GetEnabled(), item.GetAutoFix(), item.GetThreshold(), item.GetUnit(), item.GetCondition()))
	}
	return rows
}

func structSummary(data *structpb.Struct) string {
	body, err := json.MarshalIndent(data.AsMap(), "", "  ")
	if err != nil {
		return "Unable to render details."
	}
	return string(body)
}
