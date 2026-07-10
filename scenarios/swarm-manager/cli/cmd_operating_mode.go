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
		fmt.Printf("    target=%s strategy=%s usage=%d initiative(s)\n", mode.GetTargetKind(), mode.GetRunStrategy(), mode.GetUsageCount())
	}
	return nil
}

// cmdOperatingModeStart is the plan-first entry point: start a round of a
// non-initiative-target mode directly on its target (a plan-manager plan by
// execution id/slug, or a plan-ref path) — no initiative created. Round
// follow-up uses `initiatives mode-refresh/mode-cancel --name <scope-id>
// --mode <mode> --round N`, where the scope id is the resolved target id
// printed on the started round.
func (a *App) cmdOperatingModeStart(args []string) error {
	fs := flag.NewFlagSet("operating-mode start", flag.ContinueOnError)
	modeFlag := fs.String("mode", "", "Operating mode ID (e.g., phased-plan-drain)")
	targetFlag := fs.String("target", "", "Target ref: plan-manager execution id/slug, or plan-ref path")
	phaseFlag := fs.String("phase", "", "Phase name (defaults to the mode's start phase)")
	noteFlag := fs.String("note", "", "Operator note")
	inputsFlag := fs.String("inputs", "", "Caller inputs as a JSON object keyed by logical input ID")
	overrideFlag := fs.Bool("override", false, "Acquire the target lock even if it is held")
	requestedByFlag := fs.String("requested-by", "", "Actor recorded for the phase start")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("mode", *modeFlag, "target", *targetFlag); err != nil {
		return fmt.Errorf("usage: operating-mode start --mode MODE --target REF [--phase PHASE] [--note MSG] [--inputs JSON] [--override] [--requested-by WHO] [--json]\n\n%s", err)
	}
	inputs, err := parseProtoStructJSON(*inputsFlag)
	if err != nil {
		return err
	}
	resp, err := a.operatingModeClient().StartTargetPhase(context.Background(), connect.NewRequest(&apipb.OperatingModeStartTargetPhaseRequest{
		Mode:        strings.TrimSpace(*modeFlag),
		TargetRef:   strings.TrimSpace(*targetFlag),
		Phase:       strings.TrimSpace(*phaseFlag),
		Note:        strings.TrimSpace(*noteFlag),
		Inputs:      inputs,
		Override:    *overrideFlag,
		RequestedBy: defaultString(strings.TrimSpace(*requestedByFlag), "swarm-manager-cli"),
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printModeRound("Started Round", resp.Msg)
	fmt.Printf("  Follow up: swarm-manager initiatives mode-refresh --name %s --mode %s --round %d\n", resp.Msg.GetScopeId(), resp.Msg.GetMode(), resp.Msg.GetRound())
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
	printOperatingModeDetail(resp.Msg, a.resolveDelegatedSubModes(resp.Msg.GetEntry()))
	return nil
}

// resolveDelegatedSubModes fetches the catalog entry of every sub-mode a
// phase delegates to (executed_by), so the detail printer can render the
// composed graph inline. The backend stays the routing SSOT — this only reads
// each sub-mode's own catalog to display its phases under the delegating phase.
// A sub-mode that fails to resolve is simply omitted (the delegating phase
// still renders its "delegates to" marker).
func (a *App) resolveDelegatedSubModes(entry *apipb.OperatingModeCatalogEntry) map[string]*apipb.OperatingModeCatalogEntry {
	if entry == nil {
		return nil
	}
	out := map[string]*apipb.OperatingModeCatalogEntry{}
	for _, phase := range entry.GetPhases() {
		sub := phase.GetExecutedBy()
		if sub == "" {
			continue
		}
		if _, ok := out[sub]; ok {
			continue
		}
		resp, err := a.operatingModeClient().GetMode(context.Background(), connect.NewRequest(&apipb.OperatingModeGetRequest{Mode: sub}))
		if err != nil {
			continue
		}
		out[sub] = resp.Msg.GetEntry()
	}
	return out
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
	printOperatingModeDetail(resp.Msg, a.resolveDelegatedSubModes(resp.Msg.GetEntry()))
	return nil
}

func humanizeOperatingModeEnum(value string) string {
	switch value {
	case "plan-manager-plan":
		return "Plan-manager plan"
	case "plan-ref":
		return "Plan reference"
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

func printOperatingModeDetail(resp *apipb.OperatingModeDetailResponse, subModes map[string]*apipb.OperatingModeCatalogEntry) {
	entry := resp.GetEntry()
	printSection("Operating Mode")
	fmt.Printf("  Mode:        %s\n", entry.GetMode())
	fmt.Printf("  Label:       %s\n", entry.GetLabel())
	if entry.GetDescription() != "" {
		fmt.Printf("  Description: %s\n", entry.GetDescription())
	}
	fmt.Printf("  Target:      %s\n", humanizeOperatingModeEnum(entry.GetTargetKind()))
	fmt.Printf("  Strategy:    %s\n", humanizeOperatingModeEnum(entry.GetRunStrategy()))
	fmt.Printf("  Usage:       %d initiative(s)\n", entry.GetUsageCount())
	if len(entry.GetPhases()) > 0 {
		fmt.Println("  Phases:")
		for _, phase := range entry.GetPhases() {
			printOperatingModePhase(phase, subModes)
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
// including its profile/contract summary and optional artifacts/internals. A
// phase delegated to a sub-mode (executed_by) renders the sub-mode's phases
// inline from subModes; a phase with a classification-on-transition contract
// renders it as a built-in step rather than an agent phase.
func printOperatingModePhase(phase *apipb.OperatingModeCatalogPhase, subModes map[string]*apipb.OperatingModeCatalogEntry) {
	markers := ""
	if phase.GetIsStart() {
		markers += " [start]"
	}
	if phase.GetIsTerminal() {
		markers += " [terminal]"
	}
	if phase.GetExecutedBy() != "" {
		markers += " [delegated]"
	}
	title := phase.GetTitle()
	if title == "" {
		title = phase.GetPhase()
	}
	fmt.Printf("    - %s  (%s)%s\n", title, phase.GetPhase(), markers)
	if phase.GetExecutedBy() != "" {
		printDelegatedSubMode(phase.GetExecutedBy(), subModes)
		return
	}
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
	printOperatingModePhaseReads(phase.GetReads())
	if chips := contractChips(phase.GetOutputContract()); len(chips) > 0 {
		fmt.Printf("      Contract: %s\n", strings.Join(chips, "  "))
	}
	printOperatingModePhaseClassification(phase.GetClassification())
	printOperatingModePhaseArtifacts(phase.GetOutputArtifacts())
	printOperatingModePhaseInternals(phase)
}

// printOperatingModePhaseReads renders a phase's declared input contract
// grouped by supplying provider (generic base vs target adapter), rendered
// from data rather than a fixed category list.
func printOperatingModePhaseReads(reads *apipb.OperatingModePhaseReads) {
	if reads == nil || (len(reads.GetBase()) == 0 && len(reads.GetTarget()) == 0) {
		return
	}
	fmt.Print("      Reads:")
	if len(reads.GetBase()) > 0 {
		fmt.Printf(" base[%s]", strings.Join(reads.GetBase(), ", "))
	}
	if len(reads.GetTarget()) > 0 {
		fmt.Printf(" target[%s]", strings.Join(reads.GetTarget(), ", "))
	}
	fmt.Println()
}

// printOperatingModePhaseClassification renders the classification-on-transition
// contract as a built-in step: the routing field derived at the edge, the
// closed enum, and the handoff field it derives from. Costs no agent round.
func printOperatingModePhaseClassification(c *apipb.OperatingModeTransitionClassification) {
	if c == nil {
		return
	}
	fmt.Printf("      Classification (built-in): derive %s", c.GetField())
	if enum := c.GetEnum(); len(enum) > 0 {
		fmt.Printf(" ∈ {%s}", strings.Join(enum, ", "))
	}
	if from := c.GetFrom(); from != "" {
		fmt.Printf(" from %s", from)
	}
	fmt.Println()
	if desc := c.GetDescription(); desc != "" {
		fmt.Printf("        %s\n", desc)
	}
}

// printDelegatedSubMode renders a delegated phase's composed graph inline: the
// sub-mode's phases and phase graph, marked delegated. The backend remains the
// routing SSOT; this reads the sub-mode's own catalog for display only.
func printDelegatedSubMode(subMode string, subModes map[string]*apipb.OperatingModeCatalogEntry) {
	fmt.Printf("      Executed by sub-mode: %s\n", subMode)
	sub := subModes[subMode]
	if sub == nil {
		return
	}
	fmt.Printf("      Composed graph (target=%s):\n", humanizeOperatingModeEnum(sub.GetTargetKind()))
	for _, phase := range sub.GetPhases() {
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
		fmt.Printf("        · %s  (%s)%s\n", title, phase.GetPhase(), markers)
		if c := phase.GetClassification(); c != nil {
			fmt.Printf("          classification: derive %s", c.GetField())
			if enum := c.GetEnum(); len(enum) > 0 {
				fmt.Printf(" ∈ {%s}", strings.Join(enum, ", "))
			}
			fmt.Println()
		}
	}
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
			suffix := ""
			if edge.GetClassified() {
				suffix = " [classified]"
			}
			fmt.Printf("      %s -> %s (%s)%s\n", edge.GetFrom(), edge.GetTo(), edge.GetLabel(), suffix)
		}
	}
	if len(graph.GetAcceptedVerdicts()) > 0 {
		fmt.Printf("    Accepted verdicts: %s\n", strings.Join(graph.GetAcceptedVerdicts(), ", "))
	}
}
