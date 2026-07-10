package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// Per-initiative `swarm-manager initiatives mode-*` commands operate on a
// specific initiative's operating-mode workspace. Like the top-level
// operating-mode commands, they speak the typed OperatingModeService Connect
// contract via the generated client.

func printOperatingModeCapabilities(header, prefix string, capabilities *apipb.OperatingModeCapabilities) {
	labels := make([]string, 0, 8)
	if capabilities.GetSupportsPhases() {
		labels = append(labels, "phases")
	}
	if capabilities.GetCanStartPhases() {
		labels = append(labels, "start phases")
	}
	if capabilities.GetCanCompleteItems() {
		labels = append(labels, "complete items")
	}
	if capabilities.GetCanApplyBacklogSyncProposals() {
		labels = append(labels, "apply backlog proposals")
	}
	if capabilities.GetRequiresAcceptanceCriteria() {
		labels = append(labels, "acceptance criteria")
	}
	if capabilities.GetSupportsArtifacts() {
		labels = append(labels, "artifacts")
	}
	if capabilities.GetSupportsHandoffs() {
		labels = append(labels, "handoffs")
	}
	if capabilities.GetUsesItemExecutionFlow() {
		labels = append(labels, "item execution flow")
	}
	if len(labels) == 0 {
		return
	}
	fmt.Println(header)
	for _, label := range labels {
		fmt.Println(prefix + label)
	}
}

func (a *App) cmdInitiativesModeList(args []string) error {
	fs := flag.NewFlagSet("initiatives mode-list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	resp, err := a.operatingModeClient().Catalog(context.Background(), connect.NewRequest(&apipb.OperatingModeCatalogRequest{}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printSection("Operating Modes")
	modes := resp.Msg.GetModes()
	if len(modes) == 0 {
		fmt.Println("  (none)")
		return nil
	}
	for _, mode := range modes {
		defaultMark := ""
		if mode.GetDefault() {
			defaultMark = " [default]"
		}
		fmt.Printf("  - %s%s (%s)\n", mode.GetMode(), defaultMark, mode.GetLabel())
		if mode.GetTargetKind() != "" {
			fmt.Printf("    target: %s\n", mode.GetTargetKind())
		}
		if mode.GetRunStrategy() != "" {
			fmt.Printf("    strategy: %s\n", mode.GetRunStrategy())
		}
		printOperatingModeCapabilities("    capabilities:", "      - ", mode.GetCapabilities())
		if len(mode.GetPhases()) > 0 {
			fmt.Println("    phases:")
			for _, phase := range mode.GetPhases() {
				writeAccess := "read-only"
				if phase.GetWritesRepo() {
					writeAccess = "writes repo"
				}
				fmt.Printf("      - %s (%s, %s)\n", phase.GetPhase(), phase.GetProfileKey(), writeAccess)
			}
		}
	}
	return nil
}

func (a *App) cmdInitiativesModeWorkspace(args []string) error {
	fs := flag.NewFlagSet("initiatives mode-workspace", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives mode-workspace --name NAME [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)
	resp, err := a.operatingModeClient().GetWorkspace(context.Background(), connect.NewRequest(&apipb.OperatingModeWorkspaceRequest{
		InitiativeName: name,
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	ws := resp.Msg
	def := ws.GetDefinition()

	printSection("Operating Mode")
	fmt.Printf("  Initiative: %s\n", ws.GetInitiativeName())
	fmt.Printf("  Mode:       %s\n", ws.GetMode())
	if def.GetLabel() != "" {
		fmt.Printf("  Label:      %s\n", def.GetLabel())
	}
	if def.GetRunStrategy() != "" {
		fmt.Printf("  Strategy:   %s\n", def.GetRunStrategy())
	}
	printOperatingModeCapabilities("  Capabilities:", "    - ", def.GetCapabilities())
	printSection("Executions")
	if len(ws.GetExecutions()) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, execution := range ws.GetExecutions() {
			fmt.Printf("  - %s: %s definition=%s", execution.GetExecutionId(), execution.GetStatus(), execution.GetDefinitionDigest())
			if execution.GetInputContractDigest() != "" {
				fmt.Printf(" inputs=%s", execution.GetInputContractDigest())
			}
			if len(execution.GetReachablePromptSources()) > 0 {
				fmt.Printf(" prompts=%d", len(execution.GetReachablePromptSources()))
			}
			fmt.Println()
		}
	}
	printSection("Phases")
	if len(def.GetPhases()) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, phase := range def.GetPhases() {
			writeAccess := "read-only"
			if phase.GetWritesRepo() {
				writeAccess = "writes repo"
			}
			fmt.Printf("  - %s (%s, %s)\n", phase.GetPhase(), phase.GetProfileKey(), writeAccess)
		}
	}
	printSection("Artifacts")
	if len(ws.GetArtifacts()) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, artifact := range ws.GetArtifacts() {
			status := "optional"
			if artifact.GetRequired() {
				status = "required"
			}
			if artifact.GetSizeBytes() > 0 {
				status = fmt.Sprintf("%d bytes", artifact.GetSizeBytes())
			}
			fmt.Printf("  - %s (%s)\n", artifact.GetPath(), status)
		}
	}
	printSection("Rounds")
	if len(ws.GetRounds()) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, round := range ws.GetRounds() {
			fmt.Printf("  - round %d: %s/%s", round.GetRound(), round.GetPhase(), round.GetStatus())
			if round.GetExecutionId() != "" {
				fmt.Printf(" execution=%s", round.GetExecutionId())
			}
			if round.GetRunId() != "" {
				fmt.Printf(" run=%s", round.GetRunId())
			}
			if summary := operatingModeResolutionSummary(round.GetResolution()); summary != "" {
				fmt.Printf(" resolution=%s", summary)
			}
			fmt.Println()
			if round.GetStatus() == "needs_attention" && strings.TrimSpace(round.GetError()) != "" {
				fmt.Printf("    reason: %s\n", round.GetError())
			}
		}
	}
	return nil
}

func (a *App) cmdInitiativesModeSwitch(args []string) error {
	fs := flag.NewFlagSet("initiatives mode-switch", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	modeFlag := fs.String("mode", "", "Operating mode")
	cancelFlag := fs.Bool("cancel-active-item-executions", false, "Cancel active member item executions before switching")
	requestedByFlag := fs.String("requested-by", "", "Actor recorded for the switch")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "mode", *modeFlag); err != nil {
		return fmt.Errorf("usage: initiatives mode-switch --name NAME --mode MODE [--cancel-active-item-executions] [--requested-by WHO] [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)
	resp, err := a.operatingModeClient().SwitchMode(context.Background(), connect.NewRequest(&apipb.OperatingModeSwitchRequest{
		InitiativeName:             name,
		Mode:                       strings.TrimSpace(*modeFlag),
		CancelActiveItemExecutions: *cancelFlag,
		RequestedBy:                defaultString(strings.TrimSpace(*requestedByFlag), "swarm-manager-cli"),
	}))
	if err != nil {
		return operatingModeSwitchError(err)
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	result := resp.Msg
	printSection("Mode Switch")
	fmt.Printf("  Initiative: %s\n", result.GetInitiativeName())
	fmt.Printf("  Mode:       %s -> %s\n", result.GetFromMode(), result.GetToMode())
	if len(result.GetCanceledItemExecutions()) > 0 {
		fmt.Printf("  Canceled item executions: %d\n", len(result.GetCanceledItemExecutions()))
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "mode-workspace", "--name", name),
	})
	return nil
}

// operatingModeSwitchError enriches a failed SwitchMode with the structured
// active-item-executions conflict detail (the Connect equivalent of the REST
// `active_item_executions` conflict body) so the operator sees which executions
// block the switch and can re-run with --cancel-active-item-executions.
func operatingModeSwitchError(err error) error {
	conflict := activeItemExecutionsConflictDetail(err)
	if conflict == nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "cannot switch %q from %s to %s: %d active item execution(s) must be canceled first",
		conflict.GetInitiativeName(), conflict.GetFromMode(), conflict.GetToMode(), len(conflict.GetExecutions()))
	for _, exec := range conflict.GetExecutions() {
		fmt.Fprintf(&b, "\n  - %s", exec.GetItemRef())
		if exec.GetStatus() != "" {
			fmt.Fprintf(&b, " (%s)", exec.GetStatus())
		}
		if exec.GetRunId() != "" {
			fmt.Fprintf(&b, " run=%s", exec.GetRunId())
		}
	}
	b.WriteString("\nRe-run with --cancel-active-item-executions to cancel them and switch.")
	return fmt.Errorf("%s", b.String())
}

// activeItemExecutionsConflictDetail extracts the structured conflict detail
// from a Connect error, returning nil when the error is not a switch conflict.
func activeItemExecutionsConflictDetail(err error) *apipb.OperatingModeActiveItemExecutionsConflict {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return nil
	}
	for _, detail := range connectErr.Details() {
		msg, valueErr := detail.Value()
		if valueErr != nil {
			continue
		}
		if conflict, ok := msg.(*apipb.OperatingModeActiveItemExecutionsConflict); ok {
			return conflict
		}
	}
	return nil
}

func (a *App) cmdInitiativesModeStart(args []string) error {
	fs := flag.NewFlagSet("initiatives mode-start", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	phaseFlag := fs.String("phase", "", "Phase name")
	noteFlag := fs.String("note", "", "Operator note")
	inputsFlag := fs.String("inputs", "", "Caller inputs as a JSON object keyed by logical input ID")
	overrideFlag := fs.Bool("override", false, "Acquire the initiative lock even if it is held")
	requestedByFlag := fs.String("requested-by", "", "Actor recorded for the phase start")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "phase", *phaseFlag); err != nil {
		return fmt.Errorf("usage: initiatives mode-start --name NAME --phase PHASE [--note MSG] [--inputs JSON] [--override] [--requested-by WHO] [--json]\n\n%s", err)
	}
	inputs, err := parseProtoStructJSON(*inputsFlag)
	if err != nil {
		return err
	}
	resp, err := a.operatingModeClient().StartPhase(context.Background(), connect.NewRequest(&apipb.OperatingModeStartPhaseRequest{
		InitiativeName: strings.TrimSpace(*nameFlag),
		Phase:          strings.TrimSpace(*phaseFlag),
		Note:           strings.TrimSpace(*noteFlag),
		Inputs:         inputs,
		Override:       *overrideFlag,
		RequestedBy:    defaultString(strings.TrimSpace(*requestedByFlag), "swarm-manager-cli"),
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printModeRound("Started Round", resp.Msg)
	return nil
}

func (a *App) cmdInitiativesModeRefresh(args []string) error {
	return a.runModeRoundCommand(args, "refresh")
}

func (a *App) cmdInitiativesModeCancel(args []string) error {
	return a.runModeRoundCommand(args, "cancel")
}

func (a *App) runModeRoundCommand(args []string, action string) error {
	fs := flag.NewFlagSet("initiatives mode-"+action, flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	modeFlag := fs.String("mode", "", "Operating mode")
	roundFlag := parseRoundFlag(fs, "round", "Round number")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "mode", *modeFlag); err != nil {
		return fmt.Errorf("usage: initiatives mode-%s --name NAME --mode MODE --round N [--json]\n\n%s", action, err)
	}
	if *roundFlag <= 0 {
		return fmt.Errorf("--round must be a positive integer")
	}
	req := connect.NewRequest(&apipb.OperatingModeRoundActionRequest{
		InitiativeName: strings.TrimSpace(*nameFlag),
		Mode:           strings.TrimSpace(*modeFlag),
		Round:          int32(*roundFlag),
	})
	client := a.operatingModeClient()
	var (
		resp *connect.Response[apipb.OperatingModeRoundEnvelope]
		err  error
	)
	switch action {
	case "refresh":
		resp, err = client.RefreshRound(context.Background(), req)
	case "cancel":
		resp, err = client.CancelRound(context.Background(), req)
	default:
		return fmt.Errorf("unknown round action %q", action)
	}
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printModeRound(modeRoundActionTitle(action), resp.Msg)
	return nil
}

func (a *App) cmdInitiativesModeComplete(args []string) error {
	fs := flag.NewFlagSet("initiatives mode-complete-items", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	modeFlag := fs.String("mode", "", "Operating mode")
	roundFlag := parseRoundFlag(fs, "round", "Round number")
	runIDFlag := fs.String("run-id", "", "AgentManager run ID recorded on the round")
	itemsFlag := fs.String("items", "", "Comma-separated kind/name item refs")
	requestedByFlag := fs.String("requested-by", "", "Actor recorded for the backlog sync")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "mode", *modeFlag, "run-id", *runIDFlag, "items", *itemsFlag); err != nil {
		return fmt.Errorf("usage: initiatives mode-complete-items --name NAME --mode MODE --round N --run-id RUN --items kind/name,... [--requested-by WHO] [--json]\n\n%s", err)
	}
	if *roundFlag <= 0 {
		return fmt.Errorf("--round must be a positive integer")
	}
	resp, err := a.operatingModeClient().CompleteItems(context.Background(), connect.NewRequest(&apipb.OperatingModeCompleteItemsRequest{
		InitiativeName: strings.TrimSpace(*nameFlag),
		Mode:           strings.TrimSpace(*modeFlag),
		Round:          int32(*roundFlag),
		RunId:          strings.TrimSpace(*runIDFlag),
		ItemRefs:       cliutil.ParseCSV(*itemsFlag),
		RequestedBy:    defaultString(strings.TrimSpace(*requestedByFlag), "swarm-manager-cli"),
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printBacklogSyncResponse(resp.Msg)
	return nil
}

func (a *App) cmdInitiativesModeApplyBacklogSync(args []string) error {
	fs := flag.NewFlagSet("initiatives mode-apply-backlog-sync", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	modeFlag := fs.String("mode", "", "Operating mode")
	roundFlag := parseRoundFlag(fs, "round", "Round number")
	runIDFlag := fs.String("run-id", "", "AgentManager run ID recorded on the round")
	mutationsFlag := fs.String("mutations", "", "Comma-separated proposal mutation IDs to apply")
	requestedByFlag := fs.String("requested-by", "", "Actor recorded for the backlog sync")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "mode", *modeFlag, "run-id", *runIDFlag, "mutations", *mutationsFlag); err != nil {
		return fmt.Errorf("usage: initiatives mode-apply-backlog-sync --name NAME --mode MODE --round N --run-id RUN --mutations m1,m2 [--requested-by WHO] [--json]\n\n%s", err)
	}
	if *roundFlag <= 0 {
		return fmt.Errorf("--round must be a positive integer")
	}
	resp, err := a.operatingModeClient().ApplyBacklogSync(context.Background(), connect.NewRequest(&apipb.OperatingModeApplyBacklogSyncRequest{
		InitiativeName:      strings.TrimSpace(*nameFlag),
		Mode:                strings.TrimSpace(*modeFlag),
		Round:               int32(*roundFlag),
		RunId:               strings.TrimSpace(*runIDFlag),
		AcceptedMutationIds: cliutil.ParseCSV(*mutationsFlag),
		RequestedBy:         defaultString(strings.TrimSpace(*requestedByFlag), "swarm-manager-cli"),
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printBacklogSyncResponse(resp.Msg)
	return nil
}

func printBacklogSyncResponse(resp *apipb.OperatingModeBacklogSyncResult) {
	printSection("Backlog Sync")
	fmt.Printf("  Initiative: %s\n", resp.GetInitiativeName())
	fmt.Printf("  Mode:       %s\n", resp.GetMode())
	fmt.Printf("  Round:      %d\n", resp.GetRound())
	proposal := resp.GetProposalResult()
	if len(resp.GetCompletedItems()) > 0 {
		fmt.Printf("  Completed:  %d item(s)\n", len(resp.GetCompletedItems()))
		for _, item := range resp.GetCompletedItems() {
			fmt.Printf("  - %s: %s -> %s\n", item.GetItemRef(), item.GetFromStatus(), item.GetToStatus())
		}
	} else if proposal == nil {
		fmt.Println("  Completed:  0 item(s)")
	}
	if proposal != nil {
		fmt.Printf("  Proposal:   %d applied, %d skipped, %d failed\n", proposal.GetApplied(), proposal.GetSkipped(), proposal.GetFailed())
		if proposal.GetCreated() > 0 || proposal.GetUpdated() > 0 {
			fmt.Printf("  Mutations:  %d created, %d updated\n", proposal.GetCreated(), proposal.GetUpdated())
		}
		for _, outcome := range proposal.GetOutcomes() {
			status := "applied"
			if outcome.GetSkipped() {
				status = "skipped"
			}
			if outcome.GetError() != "" {
				status = "failed: " + outcome.GetError()
			}
			target := outcome.GetTarget()
			if target == "" {
				target = outcome.GetOp()
			}
			fmt.Printf("  - %s (%s): %s\n", outcome.GetMutationId(), target, status)
		}
	}
}

func printModeRound(title string, round *apipb.OperatingModeRoundEnvelope) {
	printSection(title)
	fmt.Printf("  Round:   %d\n", round.GetRound())
	fmt.Printf("  Mode:    %s\n", round.GetMode())
	fmt.Printf("  Phase:   %s\n", round.GetPhase())
	fmt.Printf("  Status:  %s\n", round.GetStatus())
	if round.GetExecutionId() != "" {
		fmt.Printf("  Execution: %s\n", round.GetExecutionId())
	}
	if round.GetDefinitionDigest() != "" {
		fmt.Printf("  Definition: %s\n", round.GetDefinitionDigest())
	}
	if round.GetRunId() != "" {
		fmt.Printf("  Run ID:  %s\n", round.GetRunId())
	}
	if round.GetAgentProfileKey() != "" {
		fmt.Printf("  Profile: %s\n", round.GetAgentProfileKey())
	}
	printOperatingModeResolution(round.GetResolution())
	printOperatingModeResolvedEnvelope(round)
	if round.GetStatus() == "needs_attention" && strings.TrimSpace(round.GetError()) != "" {
		fmt.Printf("  Reason:  %s\n", round.GetError())
	}
}

func modeRoundActionTitle(action string) string {
	switch action {
	case "refresh":
		return "Refreshed Round"
	case "cancel":
		return "Canceled Round"
	default:
		return "Updated Round"
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func operatingModeResolutionSummary(record *apipb.OperatingModePhaseResolutionRecord) string {
	if record == nil || strings.TrimSpace(record.GetOutcome()) == "" {
		return ""
	}
	layer := strings.TrimSpace(record.GetLayer())
	if layer == "" {
		return record.GetOutcome()
	}
	return fmt.Sprintf("%s via %s", record.GetOutcome(), layer)
}

func printOperatingModeResolution(record *apipb.OperatingModePhaseResolutionRecord) {
	if record == nil || strings.TrimSpace(record.GetOutcome()) == "" {
		return
	}
	fmt.Printf("  Resolution: %s\n", operatingModeResolutionSummary(record))
	if record.GetMessagesScanned() > 0 {
		fmt.Printf("  Messages:   %d scanned", record.GetMessagesScanned())
		if record.GetChosenMessageIndex() >= 0 {
			fmt.Printf(", chose index %d", record.GetChosenMessageIndex())
		}
		fmt.Println()
	}
	if len(record.GetMissing()) > 0 {
		fmt.Printf("  Missing:    %s\n", strings.Join(record.GetMissing(), ", "))
	}
	if len(record.GetViolations()) > 0 {
		fmt.Printf("  Violations: %s\n", strings.Join(record.GetViolations(), ", "))
	}
	if len(record.GetNotes()) > 0 {
		fmt.Printf("  Notes:      %s\n", strings.Join(record.GetNotes(), "; "))
	}
	if selected := record.GetSelectedMessage(); selected != nil {
		if selected.GetEventId() != "" {
			fmt.Printf("  Event ID:   %s\n", selected.GetEventId())
		}
		if selected.GetSequence() != 0 {
			fmt.Printf("  Sequence:   %d\n", selected.GetSequence())
		}
		fmt.Printf("  Digest:     %s\n", selected.GetContentDigest())
		fmt.Printf("  Selector:   %s\n", selected.GetSelectionAlgorithmVersion())
		if selected.GetFallbackReason() != "" {
			fmt.Printf("  Fallback:   %s\n", selected.GetFallbackReason())
		}
	}
}

func printOperatingModeResolvedEnvelope(round *apipb.OperatingModeRoundEnvelope) {
	envelope := round.GetResolvedEnvelope()
	if envelope == nil || len(envelope.GetFields()) == 0 {
		return
	}
	data, err := json.MarshalIndent(envelope.AsMap(), "", "  ")
	if err != nil {
		return
	}
	fmt.Println("  Resolved envelope:")
	for _, line := range strings.Split(string(data), "\n") {
		fmt.Printf("    %s\n", line)
	}
}
