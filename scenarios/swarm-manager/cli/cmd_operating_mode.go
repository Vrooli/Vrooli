package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// Top-level `swarm-manager operating-mode {list,get,set}` commands. Lives
// outside the per-initiative `initiatives mode-*` family because it operates
// on the mode catalog itself, not on a specific initiative's workspace. These
// commands speak the typed OperatingModeService Connect contract via the
// generated client (see cmd_operating_mode_authoring.go for operatingModeClient).

func (a *App) cmdOperatingModeList(args []string) error {
	fs := flag.NewFlagSet("operating-mode list", flag.ContinueOnError)
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
		fmt.Printf("  - %s%s — %s\n", mode.GetMode(), defaultMark, mode.GetLabel())
		if mode.GetDescription() != "" {
			fmt.Printf("    %s\n", mode.GetDescription())
		}
		fmt.Printf("    scope=%s strategy=%s usage=%d initiative(s)\n", mode.GetScopeKind(), mode.GetRunStrategy(), mode.GetUsageCount())
	}
	return nil
}

func (a *App) cmdOperatingModeGet(args []string) error {
	fs := flag.NewFlagSet("operating-mode get", flag.ContinueOnError)
	modeFlag := fs.String("mode", "", "Operating mode ID (e.g., holistic-loop)")
	phaseFlag := fs.String("phase", "", "Phase ID to focus (e.g., investigate)")
	presetFlag := fs.String("preset", "", "Simulation preset used by --show-prompt (default: mode happy path)")
	showPrompt := fs.Bool("show-prompt", false, "Render the selected phase prompt through the operating-mode prompt seam")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *modeFlag == "" && fs.NArg() == 1 {
		*modeFlag = fs.Arg(0)
	}
	if err := requireFlag("mode", *modeFlag); err != nil {
		return fmt.Errorf("usage: operating-mode get --mode MODE [--phase PHASE [--show-prompt] [--preset ID]] [--json]\n\n%s", err)
	}
	if *showPrompt {
		if strings.TrimSpace(*phaseFlag) == "" {
			return fmt.Errorf("usage: operating-mode get --mode MODE --phase PHASE --show-prompt [--preset ID] [--json]\n\n--phase is required with --show-prompt")
		}
		return a.printOperatingModePhasePrompt(strings.TrimSpace(*modeFlag), strings.TrimSpace(*phaseFlag), strings.TrimSpace(*presetFlag), *jsonOut)
	}
	resp, err := a.operatingModeClient().GetMode(context.Background(), connect.NewRequest(&apipb.OperatingModeGetRequest{
		Mode: strings.TrimSpace(*modeFlag),
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printOperatingModeDetail(resp.Msg)
	return nil
}

func (a *App) printOperatingModePhasePrompt(mode, phase, preset string, jsonOut bool) error {
	client := a.operatingModeClient()
	sim, err := client.SimulateMode(context.Background(), connect.NewRequest(&apipb.OperatingModeSimulateRequest{
		Mode:   mode,
		Preset: preset,
	}))
	if err != nil {
		return err
	}
	stepIndex := int32(-1)
	for _, step := range sim.Msg.GetTrace() {
		if step.GetPhase() == phase {
			stepIndex = step.GetIndex()
			break
		}
	}
	if stepIndex < 0 {
		return fmt.Errorf("phase %q was not reached by mode %q preset %q", phase, mode, sim.Msg.GetActivePreset())
	}
	rendered, err := client.RenderSimulationPrompt(context.Background(), connect.NewRequest(&apipb.OperatingModeRenderSimulationRequest{
		Mode:      mode,
		Preset:    sim.Msg.GetActivePreset(),
		StepIndex: stepIndex,
	}))
	if err != nil {
		return err
	}
	if jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, rendered.Msg)
	}
	printOperatingModeRenderedPrompt(rendered.Msg)
	return nil
}

func (a *App) cmdOperatingModeSet(args []string) error {
	fs := flag.NewFlagSet("operating-mode set", flag.ContinueOnError)
	modeFlag := fs.String("mode", "", "Operating mode ID")
	labelFlag := fs.String("label", "", "New display label")
	descFlag := fs.String("description", "", "New description (use --clear-description to remove)")
	clearDesc := fs.Bool("clear-description", false, "Clear the description override (restores registry default)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("mode", *modeFlag); err != nil {
		return fmt.Errorf("usage: operating-mode set --mode MODE [--label LABEL] [--description TEXT | --clear-description] [--json]\n\n%s", err)
	}
	req := &apipb.OperatingModeUpdateRequest{Mode: strings.TrimSpace(*modeFlag)}
	if label := strings.TrimSpace(*labelFlag); label != "" {
		req.Label = &label
	}
	if *clearDesc {
		empty := ""
		req.Description = &empty
	} else if desc := strings.TrimSpace(*descFlag); desc != "" {
		req.Description = &desc
	}
	if req.Label == nil && req.Description == nil {
		return fmt.Errorf("at least one of --label, --description, or --clear-description is required")
	}
	resp, err := a.operatingModeClient().UpdateMode(context.Background(), connect.NewRequest(req))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printOperatingModeDetail(resp.Msg)
	return nil
}

func humanizeOperatingModeEnum(value string) string {
	switch value {
	case "backlog_item":
		return "Backlog item"
	case "initiative":
		return "Initiative"
	case "existing_item_flow":
		return "Existing item flow"
	case "single_phase_run":
		return "Single phase run"
	case "sequential_handoff":
		return "Sequential handoff"
	case "operator_gated_loop":
		return "Operator-gated loop"
	default:
		return value
	}
}

func contractChips(c *apipb.OperatingModePhaseOutputContractSummary) []string {
	chips := make([]string, 0, 4)
	if c.GetRequiresStructuredResult() {
		chips = append(chips, "structured")
	}
	if c.GetRequiresVerdict() {
		chips = append(chips, "verdict")
	}
	if c.GetRequiresHandoff() {
		chips = append(chips, "handoff")
	}
	if c.GetRequiresProgress() {
		chips = append(chips, "progress")
	}
	return chips
}

func printOperatingModeDetail(resp *apipb.OperatingModeDetailResponse) {
	entry := resp.GetEntry()
	printSection("Operating Mode")
	fmt.Printf("  Mode:        %s\n", entry.GetMode())
	fmt.Printf("  Label:       %s\n", entry.GetLabel())
	if entry.GetDescription() != "" {
		fmt.Printf("  Description: %s\n", entry.GetDescription())
	}
	fmt.Printf("  Scope:       %s\n", humanizeOperatingModeEnum(entry.GetScopeKind()))
	fmt.Printf("  Strategy:    %s\n", humanizeOperatingModeEnum(entry.GetRunStrategy()))
	fmt.Printf("  Usage:       %d initiative(s)\n", entry.GetUsageCount())
	if len(entry.GetPhases()) > 0 {
		fmt.Println("  Phases:")
		for _, phase := range entry.GetPhases() {
			printOperatingModePhase(phase)
		}
	}
	if entry.GetPhaseGraph() != nil {
		printPhaseGraph(entry.GetPhaseGraph())
	}
	printSection("Linked Initiatives")
	if len(resp.GetLinkedInitiatives()) == 0 {
		fmt.Println("  (none)")
		return
	}
	for _, init := range resp.GetLinkedInitiatives() {
		title := init.GetTitle()
		if title == "" {
			title = "(untitled)"
		}
		statusSuffix := ""
		if init.GetStatus() != "" {
			statusSuffix = " [" + init.GetStatus() + "]"
		}
		fmt.Printf("  - %s — %s%s\n", init.GetName(), title, statusSuffix)
	}
}

// printOperatingModePhase renders a single operating-mode phase block,
// including its profile/contract summary and optional artifacts/internals.
func printOperatingModePhase(phase *apipb.OperatingModeCatalogPhase) {
	markers := ""
	if phase.GetIsStart() {
		markers += " [start]"
	}
	if phase.GetIsTerminal() {
		markers += " [terminal]"
	}
	title := phase.GetTitle()
	if title == "" {
		title = phase.GetPhase()
	}
	fmt.Printf("    - %s  (%s)%s\n", title, phase.GetPhase(), markers)
	if phase.GetPurpose() != "" {
		fmt.Printf("      Purpose: %s\n", phase.GetPurpose())
	}
	writeAccess := "no"
	if phase.GetWritesRepo() {
		writeAccess = "yes"
	}
	fmt.Printf("      Profile: %s    Writes repo: %s", phase.GetProfileKey(), writeAccess)
	if phase.GetRequiresCriteria() {
		fmt.Print("    Requires criteria: yes")
	}
	fmt.Println()
	if chips := contractChips(phase.GetOutputContract()); len(chips) > 0 {
		fmt.Printf("      Contract: %s\n", strings.Join(chips, "  "))
	}
	printOperatingModePhaseArtifacts(phase.GetOutputArtifacts())
	printOperatingModePhaseInternals(phase)
}

func printOperatingModeRenderedPrompt(resp *apipb.OperatingModeRenderPromptResponse) {
	printSection("Operating Mode Phase Prompt")
	fmt.Printf("  Mode:    %s\n", resp.GetMode())
	fmt.Printf("  Phase:   %s\n", resp.GetPhase())
	if resp.GetPreset() != "" {
		fmt.Printf("  Preset:  %s\n", resp.GetPreset())
	}
	fmt.Printf("  SkillID: %s\n", resp.GetSkillId())
	if resp.GetProfileKey() != "" {
		fmt.Printf("  Profile: %s\n", resp.GetProfileKey())
	}
	if resp.GetDegraded() {
		fmt.Printf("  Status:  degraded - %s\n", resp.GetDegradedReason())
		fmt.Println("  Variables:")
		printOperatingModePromptVariables(resp.GetVariables())
		return
	}
	fmt.Println()
	fmt.Println(resp.GetPrompt())
}

func printOperatingModePromptVariables(variables map[string]string) {
	if len(variables) == 0 {
		fmt.Println("    (none)")
		return
	}
	keys := make([]string, 0, len(variables))
	for key := range variables {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := strings.TrimSpace(variables[key])
		if value == "" {
			continue
		}
		fmt.Printf("    %s: %s\n", key, value)
	}
}

// printOperatingModePhaseArtifacts renders a phase's output artifacts list.
func printOperatingModePhaseArtifacts(artifacts []*apipb.OperatingModeArtifactDefinition) {
	if len(artifacts) == 0 {
		return
	}
	fmt.Println("      Output artifacts:")
	for _, artifact := range artifacts {
		marker := " "
		if artifact.GetRequired() {
			marker = "*"
		}
		fmt.Printf("        %s %s", marker, artifact.GetPath())
		if artifact.GetContentType() != "" {
			fmt.Printf("  (%s)", artifact.GetContentType())
		}
		if artifact.GetRequired() {
			fmt.Print("  required")
		}
		fmt.Println()
	}
}

// printOperatingModePhaseInternals renders the optional "Internals" block for a
// phase when any internal identifier is populated.
func printOperatingModePhaseInternals(phase *apipb.OperatingModeCatalogPhase) {
	if phase.GetCatalogId() == "" && phase.GetSkillId() == "" && phase.GetActivityPurpose() == "" && phase.GetLockPurpose() == "" {
		return
	}
	fmt.Println("      Internals:")
	if phase.GetCatalogId() != "" {
		fmt.Printf("        Catalog ID:       %s\n", phase.GetCatalogId())
	}
	if phase.GetSkillId() != "" {
		fmt.Printf("        Skill ID:         %s\n", phase.GetSkillId())
	}
	if phase.GetActivityPurpose() != "" {
		fmt.Printf("        Activity purpose: %s\n", phase.GetActivityPurpose())
	}
	if phase.GetLockPurpose() != "" {
		fmt.Printf("        Lock purpose:     %s\n", phase.GetLockPurpose())
	}
	if phase.GetTrigger() != "" {
		fmt.Printf("        Trigger:          %s\n", phase.GetTrigger())
	}
}

func printPhaseGraph(graph *apipb.OperatingModeCatalogPhaseGraph) {
	fmt.Println("  Phase graph:")
	if graph.GetStartPhase() != "" {
		fmt.Printf("    Start:    %s\n", graph.GetStartPhase())
	}
	if len(graph.GetTerminal()) > 0 {
		fmt.Printf("    Terminal: %s\n", strings.Join(graph.GetTerminal(), ", "))
	}
	if len(graph.GetTransitions()) > 0 {
		fmt.Println("    Transitions:")
		// Sort transitions by from/to/label for deterministic output.
		edges := make([]*apipb.OperatingModeCatalogTransition, len(graph.GetTransitions()))
		copy(edges, graph.GetTransitions())
		sort.SliceStable(edges, func(i, j int) bool {
			if edges[i].GetFrom() != edges[j].GetFrom() {
				return edges[i].GetFrom() < edges[j].GetFrom()
			}
			if edges[i].GetTo() != edges[j].GetTo() {
				return edges[i].GetTo() < edges[j].GetTo()
			}
			return edges[i].GetLabel() < edges[j].GetLabel()
		})
		for _, edge := range edges {
			fmt.Printf("      %s -> %s (%s)\n", edge.GetFrom(), edge.GetTo(), edge.GetLabel())
		}
	}
	if len(graph.GetAcceptedVerdicts()) > 0 {
		fmt.Printf("    Accepted verdicts: %s\n", strings.Join(graph.GetAcceptedVerdicts(), ", "))
	}
}
