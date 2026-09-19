package investigations

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	investigationspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/investigations"
	investigationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/investigations/investigationsconnect"
	scriptspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/scripts"
	scriptsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/scripts/scriptsconnect"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"system-monitor/cli/internal/support"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "investigations",
		Description: "List, inspect, trigger, and tune anomaly investigations",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "catalog", Description: "List the portable investigation catalog", RunCtx: h.catalog},
			{Name: "run", Description: "Execute a catalog investigation", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "Catalog entry id"}}}, RunCtx: h.runScript},
			{Name: "runs", Description: "List persisted investigation runs", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "entry-id", Description: "Filter by catalog entry"}, {Name: "limit", Description: "Maximum runs", Default: "20"}}}, RunCtx: h.runs},
			{Name: "runs-get", Description: "Get one persisted investigation run", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "Persisted run id"}}}, RunCtx: h.runGet},
			{Name: "runs-prune", Description: "Prune investigation history", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "dry-run", Description: "Report rows without deleting", Bool: true}}}, RunCtx: h.pruneRuns},
			{Name: "list", Description: "List recent investigations", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "limit", Description: "Maximum number of investigations to return", Default: "20"}}}, RunCtx: h.list},
			{Name: "latest", Description: "Get the latest investigation", RunCtx: h.latest},
			{Name: "get", Description: "Get an investigation by ID", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "Investigation ID"}}}, RunCtx: h.get},
			{Name: "trigger", Description: "Trigger a new investigation", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "auto-fix", Description: "Request automatic remediation", Bool: true}, {Name: "note", Description: "Context for the investigation"}}}, RunCtx: h.trigger},
			{Name: "cooldown", Description: "Show cooldown status", RunCtx: h.cooldown},
			{Name: "cooldown-reset", Description: "Reset the investigation cooldown", RunCtx: h.cooldownReset},
			{Name: "cooldown-set", Description: "Update the cooldown duration", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "seconds", Description: "Cooldown duration in seconds", Default: "0"}}}, RunCtx: h.cooldownSet},
			{Name: "triggers", Description: "List trigger thresholds", RunCtx: h.triggers},
		},
	}
}

type handlers struct {
	client  investigationsconnect.InvestigationsServiceClient
	scripts scriptsconnect.ScriptsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client:  investigationsconnect.NewInvestigationsServiceClient(httpClient, baseURL),
		scripts: scriptsconnect.NewScriptsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) catalog(ctx cliapp.RunContext) error {
	resp, err := h.scripts.ListScripts(context.Background(), connect.NewRequest(&scriptspb.ListScriptsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list investigation catalog", err, nil)
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	rows := make([]string, 0, len(resp.Msg.GetScripts()))
	for _, item := range resp.Msg.GetScripts() {
		status := "available"
		if item.GetSkipReason() != "" {
			status = "skipped: " + item.GetSkipReason()
		}
		rows = append(rows, fmt.Sprintf("%s mode=%s source=%s %s", item.GetId(), item.GetExecutionMode(), item.GetSource(), status))
	}
	return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{fmt.Sprintf("Catalog entries: %d", len(rows))}, Triage: []cliapp.TriageGroup{{Heading: "Investigation catalog", Items: rows}}, NextSteps: []string{"system-monitor investigations run <id>", "system-monitor investigations runs --limit 5"}})
}

func (h *handlers) runScript(ctx cliapp.RunContext) error {
	id := strings.TrimSpace(ctx.Positional("id"))
	resp, err := h.scripts.ExecuteScript(context.Background(), connect.NewRequest(&scriptspb.ExecuteScriptRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("run investigation", err, nil)
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	exec := resp.Msg.GetExecution()
	return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{fmt.Sprintf("%s: %s", exec.GetScriptId(), exec.GetStatus().String()), fmt.Sprintf("Duration: %.2fs", exec.GetDurationSeconds())}, Triage: []cliapp.TriageGroup{{Heading: "Result", Items: []string{exec.GetStdout(), exec.GetStderr()}}}, NextSteps: []string{"system-monitor investigations runs --limit 5"}})
}

func (h *handlers) runs(ctx cliapp.RunContext) error {
	limit, err := positiveIntFlag(ctx, "limit")
	if err != nil {
		return err
	}
	resp, err := h.scripts.ListRuns(context.Background(), connect.NewRequest(&scriptspb.ListRunsRequest{EntryId: strings.TrimSpace(ctx.Flag("entry-id")), Limit: int32(limit)}))
	if err != nil {
		return cliapp.WrapAPIError("list investigation runs", err, nil)
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	rows := make([]string, 0, len(resp.Msg.GetRuns()))
	for _, run := range resp.Msg.GetRuns() {
		rows = append(rows, fmt.Sprintf("%s entry=%s status=%s started=%s duration=%.2fs", run.GetId(), run.GetEntryId(), run.GetStatus(), support.FormatTimestamp(run.GetStartedAt()), run.GetDurationSeconds()))
	}
	return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{fmt.Sprintf("Runs returned: %d", len(rows))}, Triage: []cliapp.TriageGroup{{Heading: "Investigation runs", Items: rows}}})
}

func (h *handlers) runGet(ctx cliapp.RunContext) error {
	id := strings.TrimSpace(ctx.Positional("id"))
	resp, err := h.scripts.GetRun(context.Background(), connect.NewRequest(&scriptspb.GetRunRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("get investigation run", err, nil)
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	if resp.Msg.GetRun() == nil {
		return fmt.Errorf("server returned no investigation run")
	}
	run := resp.Msg.GetRun()
	return ctx.RenderOperational(cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("%s: %s", run.GetEntryId(), run.GetStatus()),
			fmt.Sprintf("Started: %s", support.FormatTimestamp(run.GetStartedAt())),
			fmt.Sprintf("Duration: %.2fs", run.GetDurationSeconds()),
		},
		Triage: []cliapp.TriageGroup{{Heading: "Result", Items: []string{run.GetResultJson(), run.GetStderrTail()}}},
	})
}

func (h *handlers) pruneRuns(ctx cliapp.RunContext) error {
	resp, err := h.scripts.PruneRuns(context.Background(), connect.NewRequest(&scriptspb.PruneRunsRequest{DryRun: ctx.BoolFlag("dry-run")}))
	if err != nil {
		return cliapp.WrapAPIError("prune investigation runs", err, nil)
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	mode := "deleted"
	if resp.Msg.GetDryRun() {
		mode = "would delete"
	}
	return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{fmt.Sprintf("Investigation runs %s: %d", mode, resp.Msg.GetDeleted())}})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	limit, err := positiveIntFlag(ctx, "limit")
	if err != nil {
		return err
	}

	resp, err := h.client.ListInvestigations(context.Background(), connect.NewRequest(&investigationspb.ListInvestigationsRequest{
		Limit: protoInt32(limit),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list investigations", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no investigations list")
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Investigations returned: %d", len(resp.Msg.GetInvestigations())),
			fmt.Sprintf("Limit applied: %d", limit),
		},
		ResultsHeading: "Investigations",
		Results:        investigationRows(resp.Msg.GetInvestigations()),
		RetrievalHints: []string{"system-monitor investigations latest", "system-monitor investigations get <id>", "system-monitor investigations trigger --note \"describe the issue\""},
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, report)
}

func (h *handlers) latest(ctx cliapp.RunContext) error {
	resp, err := h.client.GetLatestInvestigation(context.Background(), connect.NewRequest(&investigationspb.GetLatestInvestigationRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get latest investigation", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetInvestigation() == nil {
		return fmt.Errorf("server returned no latest investigation")
	}
	return renderInvestigation(ctx, resp.Msg, resp.Msg.GetInvestigation())
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := strings.TrimSpace(ctx.Positional("id"))
	resp, err := h.client.GetInvestigation(context.Background(), connect.NewRequest(&investigationspb.GetInvestigationRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get investigation %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetInvestigation() == nil {
		return fmt.Errorf("server returned no investigation")
	}
	return renderInvestigation(ctx, resp.Msg, resp.Msg.GetInvestigation())
}

func renderInvestigation(ctx cliapp.RunContext, payload proto.Message, response *investigationspb.Investigation) error {
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), payload)
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
	if response.GetStatus() != investigationspb.InvestigationStatus_INVESTIGATION_STATUS_COMPLETED {
		report.NextSteps = append(report.NextSteps, "system-monitor investigations latest")
	}
	return ctx.RenderOperational(report)
}

func (h *handlers) trigger(ctx cliapp.RunContext) error {
	resp, err := h.client.TriggerInvestigation(context.Background(), connect.NewRequest(&investigationspb.TriggerInvestigationRequest{
		AutoFix: ctx.BoolFlag("auto-fix"),
		Note:    strings.TrimSpace(ctx.Flag("note")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("trigger investigation", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no trigger response")
	}
	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Investigation %s queued.", resp.Msg.GetInvestigationId()),
			fmt.Sprintf("Status: %s", resp.Msg.GetStatus()),
		},
		Changes: []string{
			fmt.Sprintf("Auto-fix requested: %s", support.BoolString(resp.Msg.GetAutoFix(), "yes", "no")),
			fmt.Sprintf("Note: %s", support.FormatMaybeString(resp.Msg.GetNote(), "none")),
		},
		NextCommand: []string{
			"system-monitor investigations latest",
			fmt.Sprintf("system-monitor investigations get %s", resp.Msg.GetInvestigationId()),
		},
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, report)
}

func (h *handlers) cooldown(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCooldownStatus(context.Background(), connect.NewRequest(&investigationspb.GetCooldownStatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get investigation cooldown", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetCooldown() == nil {
		return fmt.Errorf("server returned no cooldown status")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	cooldown := resp.Msg.GetCooldown()
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
	return ctx.RenderOperational(report)
}

func (h *handlers) cooldownReset(ctx cliapp.RunContext) error {
	resp, err := h.client.ResetCooldown(context.Background(), connect.NewRequest(&investigationspb.ResetCooldownRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("reset investigation cooldown", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no cooldown reset response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{"Investigation cooldown reset."},
		Changes:     []string{"The next investigation can run immediately."},
		NextCommand: []string{"system-monitor investigations cooldown", "system-monitor investigations trigger --note \"run a fresh diagnostic\""},
	})
}

func (h *handlers) cooldownSet(ctx cliapp.RunContext) error {
	seconds, err := positiveIntFlag(ctx, "seconds")
	if err != nil {
		return fmt.Errorf("--seconds must be greater than 0")
	}

	resp, err := h.client.UpdateCooldownPeriod(context.Background(), connect.NewRequest(&investigationspb.UpdateCooldownPeriodRequest{CooldownPeriodSeconds: int32(seconds)}))
	if err != nil {
		return cliapp.WrapAPIError("update investigation cooldown", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no cooldown update response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{"Investigation cooldown updated."},
		Changes:     []string{fmt.Sprintf("New cooldown period: %ds", seconds)},
		NextCommand: []string{"system-monitor investigations cooldown"},
	})
}

func (h *handlers) triggers(ctx cliapp.RunContext) error {
	resp, err := h.client.GetTriggers(context.Background(), connect.NewRequest(&investigationspb.GetTriggersRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get investigation triggers", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no investigation triggers")
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Triggers configured: %d", len(resp.Msg.GetTriggers())),
		},
		ResultsHeading: "Triggers",
		Results:        triggerRows(resp.Msg.GetTriggers()),
		RetrievalHints: []string{"system-monitor investigations cooldown", "system-monitor status"},
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, report)
}

func positiveIntFlag(ctx cliapp.RunContext, name string) (int, error) {
	value := strings.TrimSpace(ctx.Flag(name))
	if value == "" {
		return 0, fmt.Errorf("--%s is required", name)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("--%s must be a positive integer", name)
	}
	return parsed, nil
}

func protoInt32(value int) *int32 {
	v := int32(value)
	return &v
}

func investigationRows(items []*investigationspb.Investigation) []string {
	if len(items) == 0 {
		return []string{"No investigations were returned."}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("%s status=%s progress=%d%% started=%s findings=%s", item.GetId(), strings.TrimPrefix(item.GetStatus().String(), "INVESTIGATION_STATUS_"), item.GetProgress(), support.FormatTimestamp(item.GetStartTime()), support.FormatMaybeString(item.GetFindings(), "pending")))
	}
	return rows
}

func investigationStepRows(steps []*investigationspb.InvestigationStep) []string {
	if len(steps) == 0 {
		return []string{"No investigation steps have been recorded."}
	}
	rows := make([]string, 0, len(steps))
	for _, step := range steps {
		rows = append(rows, fmt.Sprintf("%s status=%s started=%s findings=%s", step.GetName(), strings.TrimPrefix(step.GetStatus().String(), "INVESTIGATION_STEP_STATUS_"), support.FormatTimestamp(step.GetStartTime()), support.FormatMaybeString(step.GetFindings(), "none")))
	}
	return rows
}

func triggerRows(items map[string]*investigationspb.TriggerConfig) []string {
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
