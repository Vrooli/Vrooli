package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Top-level `swarm-manager operating-mode {list,get,set}` commands. Lives
// outside the per-initiative `initiatives mode-*` family because it operates
// on the mode catalog itself, not on a specific initiative's workspace.

type operatingModeListResponse struct {
	Modes []operatingModeListEntry `json:"modes"`
}

type operatingModeListEntry struct {
	Mode           string                          `json:"mode"`
	Label          string                          `json:"label"`
	Description    string                          `json:"description,omitempty"`
	UsageCount     int                             `json:"usage_count"`
	ScopeKind      string                          `json:"scope_kind"`
	RunStrategy    string                          `json:"run_strategy"`
	Default        bool                            `json:"default"`
	SupportsPhases bool                            `json:"supports_phases"`
	Phases         []operatingModeCatalogPhase     `json:"phases,omitempty"`
	PhaseGraph     *operatingModeCatalogPhaseGraph `json:"phase_graph,omitempty"`
}

type operatingModeDetailResponse struct {
	Entry             operatingModeListEntry       `json:"entry"`
	LinkedInitiatives []operatingModeLinkedInitRef `json:"linked_initiatives"`
}

type operatingModeLinkedInitRef struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Status  string `json:"status,omitempty"`
	Updated string `json:"updated,omitempty"`
}

// operatingModeCatalogPhaseGraph mirrors api/internal/operatingmode.ModeCatalogPhaseGraph.
// keep in sync with cmd_initiatives_operating_mode.go.
type operatingModeCatalogPhaseGraph struct {
	StartPhase       string                     `json:"start_phase"`
	Terminal         []string                   `json:"terminal"`
	Transitions      []operatingModeCatalogEdge `json:"transitions"`
	AcceptedVerdicts []string                   `json:"accepted_verdicts,omitempty"`
}

type operatingModeCatalogEdge struct {
	From             string `json:"from"`
	To               string `json:"to"`
	ConditionKind    string `json:"condition_kind"`
	Label            string `json:"label"`
	PayloadKey       string `json:"payload_key,omitempty"`
	ProgressDecision string `json:"progress_decision,omitempty"`
}

type operatingModeArtifactDef struct {
	Path        string `json:"path"`
	ContentType string `json:"content_type,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type operatingModePhaseContractSummary struct {
	RequiresStructuredResult bool `json:"requires_structured_result"`
	RequiresProgress         bool `json:"requires_progress"`
	RequiresVerdict          bool `json:"requires_verdict"`
	RequiresHandoff          bool `json:"requires_handoff"`
	RequiredArtifactCount    int  `json:"required_artifact_count"`
}

type operatingModeResultBinding struct {
	Kind     string                   `json:"kind"`
	Artifact operatingModeArtifactDef `json:"artifact"`
}

func (a *App) cmdOperatingModeList(args []string) error {
	fs := flag.NewFlagSet("operating-mode list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.core.Get("/operating-modes", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[operatingModeListResponse](body)
	if err != nil {
		return err
	}
	printSection("Operating Modes")
	if len(resp.Modes) == 0 {
		fmt.Println("  (none)")
		return nil
	}
	for _, mode := range resp.Modes {
		defaultMark := ""
		if mode.Default {
			defaultMark = " [default]"
		}
		fmt.Printf("  - %s%s — %s\n", mode.Mode, defaultMark, mode.Label)
		if mode.Description != "" {
			fmt.Printf("    %s\n", mode.Description)
		}
		fmt.Printf("    scope=%s strategy=%s usage=%d initiative(s)\n", mode.ScopeKind, mode.RunStrategy, mode.UsageCount)
	}
	return nil
}

func (a *App) cmdOperatingModeGet(args []string) error {
	fs := flag.NewFlagSet("operating-mode get", flag.ContinueOnError)
	modeFlag := fs.String("mode", "", "Operating mode ID (e.g., holistic-loop)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("mode", *modeFlag); err != nil {
		return fmt.Errorf("usage: operating-mode get --mode MODE [--json]\n\n%s", err)
	}
	mode := strings.TrimSpace(*modeFlag)
	body, err := a.core.Get("/operating-modes/"+mode, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[operatingModeDetailResponse](body)
	if err != nil {
		return err
	}
	printOperatingModeDetail(resp)
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
	mode := strings.TrimSpace(*modeFlag)
	patch := map[string]any{}
	if strings.TrimSpace(*labelFlag) != "" {
		patch["label"] = strings.TrimSpace(*labelFlag)
	}
	if *clearDesc {
		patch["description"] = ""
	} else if strings.TrimSpace(*descFlag) != "" {
		patch["description"] = strings.TrimSpace(*descFlag)
	}
	if len(patch) == 0 {
		return fmt.Errorf("at least one of --label, --description, or --clear-description is required")
	}
	payload, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	body, err := a.core.Request("PATCH", "/operating-modes/"+mode, nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[operatingModeDetailResponse](body)
	if err != nil {
		return err
	}
	printOperatingModeDetail(resp)
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

func contractChips(c operatingModePhaseContractSummary) []string {
	chips := make([]string, 0, 4)
	if c.RequiresStructuredResult {
		chips = append(chips, "structured")
	}
	if c.RequiresVerdict {
		chips = append(chips, "verdict")
	}
	if c.RequiresHandoff {
		chips = append(chips, "handoff")
	}
	if c.RequiresProgress {
		chips = append(chips, "progress")
	}
	return chips
}

func printOperatingModeDetail(resp operatingModeDetailResponse) {
	printSection("Operating Mode")
	fmt.Printf("  Mode:        %s\n", resp.Entry.Mode)
	fmt.Printf("  Label:       %s\n", resp.Entry.Label)
	if resp.Entry.Description != "" {
		fmt.Printf("  Description: %s\n", resp.Entry.Description)
	}
	fmt.Printf("  Scope:       %s\n", humanizeOperatingModeEnum(resp.Entry.ScopeKind))
	fmt.Printf("  Strategy:    %s\n", humanizeOperatingModeEnum(resp.Entry.RunStrategy))
	fmt.Printf("  Usage:       %d initiative(s)\n", resp.Entry.UsageCount)
	if len(resp.Entry.Phases) > 0 {
		fmt.Println("  Phases:")
		for _, phase := range resp.Entry.Phases {
			printOperatingModePhase(phase)
		}
	}
	if resp.Entry.PhaseGraph != nil {
		printPhaseGraph(resp.Entry.PhaseGraph)
	}
	printSection("Linked Initiatives")
	if len(resp.LinkedInitiatives) == 0 {
		fmt.Println("  (none)")
		return
	}
	for _, init := range resp.LinkedInitiatives {
		title := init.Title
		if title == "" {
			title = "(untitled)"
		}
		statusSuffix := ""
		if init.Status != "" {
			statusSuffix = " [" + init.Status + "]"
		}
		fmt.Printf("  - %s — %s%s\n", init.Name, title, statusSuffix)
	}
}

// printOperatingModePhase renders a single operating-mode phase block,
// including its profile/contract summary and optional artifacts/internals.
func printOperatingModePhase(phase operatingModeCatalogPhase) {
	markers := ""
	if phase.IsStart {
		markers += " [start]"
	}
	if phase.IsTerminal {
		markers += " [terminal]"
	}
	title := phase.Title
	if title == "" {
		title = phase.Phase
	}
	fmt.Printf("    - %s  (%s)%s\n", title, phase.Phase, markers)
	if phase.Purpose != "" {
		fmt.Printf("      Purpose: %s\n", phase.Purpose)
	}
	writeAccess := "no"
	if phase.WritesRepo {
		writeAccess = "yes"
	}
	fmt.Printf("      Profile: %s    Writes repo: %s", phase.ProfileKey, writeAccess)
	if phase.RequiresCriteria {
		fmt.Print("    Requires criteria: yes")
	}
	fmt.Println()
	if chips := contractChips(phase.OutputContract); len(chips) > 0 {
		fmt.Printf("      Contract: %s\n", strings.Join(chips, "  "))
	}
	printOperatingModePhaseArtifacts(phase.OutputArtifacts)
	printOperatingModePhaseInternals(phase)
}

// printOperatingModePhaseArtifacts renders a phase's output artifacts list.
func printOperatingModePhaseArtifacts(artifacts []operatingModeArtifactDef) {
	if len(artifacts) == 0 {
		return
	}
	fmt.Println("      Output artifacts:")
	for _, artifact := range artifacts {
		marker := " "
		if artifact.Required {
			marker = "*"
		}
		fmt.Printf("        %s %s", marker, artifact.Path)
		if artifact.ContentType != "" {
			fmt.Printf("  (%s)", artifact.ContentType)
		}
		if artifact.Required {
			fmt.Print("  required")
		}
		fmt.Println()
	}
}

// printOperatingModePhaseInternals renders the optional "Internals" block for a
// phase when any internal identifier is populated.
func printOperatingModePhaseInternals(phase operatingModeCatalogPhase) {
	if phase.CatalogID == "" && phase.SkillID == "" && phase.ActivityPurpose == "" && phase.LockPurpose == "" {
		return
	}
	fmt.Println("      Internals:")
	if phase.CatalogID != "" {
		fmt.Printf("        Catalog ID:       %s\n", phase.CatalogID)
	}
	if phase.SkillID != "" {
		fmt.Printf("        Skill ID:         %s\n", phase.SkillID)
	}
	if phase.ActivityPurpose != "" {
		fmt.Printf("        Activity purpose: %s\n", phase.ActivityPurpose)
	}
	if phase.LockPurpose != "" {
		fmt.Printf("        Lock purpose:     %s\n", phase.LockPurpose)
	}
	if phase.Trigger != "" {
		fmt.Printf("        Trigger:          %s\n", phase.Trigger)
	}
}

func printPhaseGraph(graph *operatingModeCatalogPhaseGraph) {
	fmt.Println("  Phase graph:")
	if graph.StartPhase != "" {
		fmt.Printf("    Start:    %s\n", graph.StartPhase)
	}
	if len(graph.Terminal) > 0 {
		fmt.Printf("    Terminal: %s\n", strings.Join(graph.Terminal, ", "))
	}
	if len(graph.Transitions) > 0 {
		fmt.Println("    Transitions:")
		// Sort transitions by from/to/label for deterministic output.
		edges := make([]operatingModeCatalogEdge, len(graph.Transitions))
		copy(edges, graph.Transitions)
		sort.SliceStable(edges, func(i, j int) bool {
			if edges[i].From != edges[j].From {
				return edges[i].From < edges[j].From
			}
			if edges[i].To != edges[j].To {
				return edges[i].To < edges[j].To
			}
			return edges[i].Label < edges[j].Label
		})
		for _, edge := range edges {
			fmt.Printf("      %s -> %s (%s)\n", edge.From, edge.To, edge.Label)
		}
	}
	if len(graph.AcceptedVerdicts) > 0 {
		fmt.Printf("    Accepted verdicts: %s\n", strings.Join(graph.AcceptedVerdicts, ", "))
	}
}
