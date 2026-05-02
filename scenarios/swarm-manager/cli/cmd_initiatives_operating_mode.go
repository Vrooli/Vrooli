package main

import (
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type operatingModeWorkspaceResponse struct {
	InitiativeName string                  `json:"initiative_name"`
	Mode           string                  `json:"mode"`
	Definition     operatingModeDef        `json:"definition"`
	Rounds         []operatingModeRound    `json:"rounds"`
	Artifacts      []operatingModeArtifact `json:"artifacts"`
}

type operatingModeCatalogResponse struct {
	Modes []operatingModeCatalogEntry `json:"modes"`
}

type operatingModeCatalogEntry struct {
	Mode           string                      `json:"mode"`
	Label          string                      `json:"label"`
	ScopeKind      string                      `json:"scope_kind"`
	RunStrategy    string                      `json:"run_strategy"`
	WorkspaceTabID string                      `json:"workspace_tab_id"`
	Capabilities   operatingModeCapabilities   `json:"capabilities"`
	Default        bool                        `json:"default"`
	Switchable     bool                        `json:"switchable"`
	SupportsPhases bool                        `json:"supports_phases"`
	Phases         []operatingModeCatalogPhase `json:"phases,omitempty"`
}

type operatingModeCapabilities struct {
	SupportsPhases               bool `json:"supports_phases"`
	CanStartPhases               bool `json:"can_start_phases"`
	CanCompleteItems             bool `json:"can_complete_items"`
	CanApplyBacklogSyncProposals bool `json:"can_apply_backlog_sync_proposals"`
	RequiresAcceptanceCriteria   bool `json:"requires_acceptance_criteria"`
	SupportsArtifacts            bool `json:"supports_artifacts"`
	SupportsHandoffs             bool `json:"supports_handoffs"`
	UsesItemExecutionFlow        bool `json:"uses_item_execution_flow"`
}

// operatingModeCatalogPhase mirrors api/internal/operatingmode.ModeCatalogPhase.
// Used by both cmd_operating_mode.go (top-level operating-mode commands) and
// cmd_initiatives_operating_mode.go (per-initiative mode commands).
type operatingModeCatalogPhase struct {
	Phase                 string                            `json:"phase"`
	Title                 string                            `json:"title"`
	Purpose               string                            `json:"purpose"`
	Trigger               string                            `json:"trigger"`
	ProfileKey            string                            `json:"profile_key"`
	WritesRepo            bool                              `json:"writes_repo"`
	RequiresCriteria      bool                              `json:"requires_criteria,omitempty"`
	IsStart               bool                              `json:"is_start,omitempty"`
	IsTerminal            bool                              `json:"is_terminal,omitempty"`
	OutputArtifacts       []operatingModeArtifactDef        `json:"output_artifacts,omitempty"`
	OutputContract        operatingModePhaseContractSummary `json:"output_contract"`
	CatalogID             string                            `json:"catalog_id"`
	SkillID               string                            `json:"skill_id"`
	ActivityPurpose       string                            `json:"activity_purpose"`
	LockPurpose           string                            `json:"lock_purpose"`
	ResultBindings        []operatingModeResultBinding      `json:"result_bindings,omitempty"`
	SamplesReplanRate     bool                              `json:"samples_replan_rate,omitempty"`
	SamplesAcceptanceRate bool                              `json:"samples_acceptance_rate,omitempty"`
}

type operatingModeDef struct {
	Label        string                        `json:"label"`
	ScopeKind    string                        `json:"scope_kind"`
	RunStrategy  string                        `json:"run_strategy"`
	Capabilities operatingModeCapabilities     `json:"capabilities"`
	Phases       []operatingModeWorkspacePhase `json:"phases"`
	Transitions  map[string][]string           `json:"transitions,omitempty"`
}

type operatingModeWorkspacePhase struct {
	Phase      string `json:"phase"`
	ProfileKey string `json:"profile_key"`
	WritesRepo bool   `json:"writes_repo"`
}

type operatingModeArtifact struct {
	Path      string `json:"path"`
	Required  bool   `json:"required,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type operatingModeRound struct {
	Round           int                 `json:"round"`
	Mode            string              `json:"mode"`
	Phase           string              `json:"phase"`
	Status          string              `json:"status"`
	RunID           string              `json:"run_id,omitempty"`
	AgentProfileKey string              `json:"agent_profile_key"`
	GeneratedAt     string              `json:"generated_at"`
	Items           []operatingModeItem `json:"items,omitempty"`
	Payload         map[string]any      `json:"payload,omitempty"`
	Error           string              `json:"error,omitempty"`
}

type operatingModeItem struct {
	Ref    string `json:"ref"`
	Title  string `json:"title,omitempty"`
	Status string `json:"status,omitempty"`
}

type operatingModeSwitchResponse struct {
	InitiativeName         string                   `json:"initiative_name"`
	FromMode               string                   `json:"from_mode"`
	ToMode                 string                   `json:"to_mode"`
	RequiresCancellation   bool                     `json:"requires_cancellation,omitempty"`
	ActiveItemExecutions   []activeItemExecutionCLI `json:"active_item_executions,omitempty"`
	CanceledItemExecutions []activeItemExecutionCLI `json:"canceled_item_executions,omitempty"`
}

type activeItemExecutionCLI struct {
	ItemRef     string `json:"item_ref"`
	ExecutionID string `json:"execution_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	Status      string `json:"status,omitempty"`
}

type operatingModeBacklogSyncResponse struct {
	InitiativeName string                       `json:"initiative_name"`
	Mode           string                       `json:"mode"`
	Phase          string                       `json:"phase"`
	Round          int                          `json:"round"`
	RunID          string                       `json:"run_id,omitempty"`
	CompletedItems []operatingModeCompletedItem `json:"completed_items,omitempty"`
	ProposalResult *operatingModeProposalResult `json:"proposal_result,omitempty"`
	Noop           bool                         `json:"noop,omitempty"`
}

type operatingModeCompletedItem struct {
	ItemRef    string `json:"item_ref"`
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
}

type operatingModeProposalResult struct {
	Applied  int                            `json:"applied"`
	Failed   int                            `json:"failed"`
	Skipped  int                            `json:"skipped"`
	Created  int                            `json:"created,omitempty"`
	Updated  int                            `json:"updated,omitempty"`
	Outcomes []operatingModeProposalOutcome `json:"outcomes,omitempty"`
}

type operatingModeProposalOutcome struct {
	MutationID string `json:"mutation_id"`
	Op         string `json:"op"`
	Target     string `json:"target,omitempty"`
	Applied    bool   `json:"applied"`
	Skipped    bool   `json:"skipped,omitempty"`
	Error      string `json:"error,omitempty"`
}

func printOperatingModeCapabilities(header, prefix string, capabilities operatingModeCapabilities) {
	labels := make([]string, 0, 8)
	if capabilities.SupportsPhases {
		labels = append(labels, "phases")
	}
	if capabilities.CanStartPhases {
		labels = append(labels, "start phases")
	}
	if capabilities.CanCompleteItems {
		labels = append(labels, "complete items")
	}
	if capabilities.CanApplyBacklogSyncProposals {
		labels = append(labels, "apply backlog proposals")
	}
	if capabilities.RequiresAcceptanceCriteria {
		labels = append(labels, "acceptance criteria")
	}
	if capabilities.SupportsArtifacts {
		labels = append(labels, "artifacts")
	}
	if capabilities.SupportsHandoffs {
		labels = append(labels, "handoffs")
	}
	if capabilities.UsesItemExecutionFlow {
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
	body, err := a.core.Get("/operating-modes", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[operatingModeCatalogResponse](body)
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
		fmt.Printf("  - %s%s (%s)\n", mode.Mode, defaultMark, mode.Label)
		if mode.ScopeKind != "" {
			fmt.Printf("    scope: %s\n", mode.ScopeKind)
		}
		if mode.RunStrategy != "" {
			fmt.Printf("    strategy: %s\n", mode.RunStrategy)
		}
		printOperatingModeCapabilities("    capabilities:", "      - ", mode.Capabilities)
		if len(mode.Phases) > 0 {
			fmt.Println("    phases:")
			for _, phase := range mode.Phases {
				writeAccess := "read-only"
				if phase.WritesRepo {
					writeAccess = "writes repo"
				}
				fmt.Printf("      - %s (%s, %s)\n", phase.Phase, phase.ProfileKey, writeAccess)
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

	body, err := a.core.Get("/initiatives/"+name+"/operating-mode/workspace", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[operatingModeWorkspaceResponse](body)
	if err != nil {
		return err
	}

	printSection("Operating Mode")
	fmt.Printf("  Initiative: %s\n", resp.InitiativeName)
	fmt.Printf("  Mode:       %s\n", resp.Mode)
	if resp.Definition.Label != "" {
		fmt.Printf("  Label:      %s\n", resp.Definition.Label)
	}
	if resp.Definition.RunStrategy != "" {
		fmt.Printf("  Strategy:   %s\n", resp.Definition.RunStrategy)
	}
	printOperatingModeCapabilities("  Capabilities:", "    - ", resp.Definition.Capabilities)
	printSection("Phases")
	if len(resp.Definition.Phases) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, phase := range resp.Definition.Phases {
			writeAccess := "read-only"
			if phase.WritesRepo {
				writeAccess = "writes repo"
			}
			fmt.Printf("  - %s (%s, %s)\n", phase.Phase, phase.ProfileKey, writeAccess)
		}
	}
	printSection("Artifacts")
	if len(resp.Artifacts) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, artifact := range resp.Artifacts {
			status := "optional"
			if artifact.Required {
				status = "required"
			}
			if artifact.SizeBytes > 0 {
				status = fmt.Sprintf("%d bytes", artifact.SizeBytes)
			}
			fmt.Printf("  - %s (%s)\n", artifact.Path, status)
		}
	}
	printSection("Rounds")
	if len(resp.Rounds) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, round := range resp.Rounds {
			fmt.Printf("  - round %d: %s/%s", round.Round, round.Phase, round.Status)
			if round.RunID != "" {
				fmt.Printf(" run=%s", round.RunID)
			}
			fmt.Println()
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
	payload := map[string]any{
		"mode":                          strings.TrimSpace(*modeFlag),
		"cancel_active_item_executions": *cancelFlag,
		"requested_by":                  defaultString(strings.TrimSpace(*requestedByFlag), "swarm-manager-cli"),
	}
	body, err := a.core.Request("POST", "/initiatives/"+name+"/operating-mode/switch", nil, mustJSON(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[operatingModeSwitchResponse](body)
	if err != nil {
		return err
	}
	printSection("Mode Switch")
	fmt.Printf("  Initiative: %s\n", resp.InitiativeName)
	fmt.Printf("  Mode:       %s -> %s\n", resp.FromMode, resp.ToMode)
	if len(resp.CanceledItemExecutions) > 0 {
		fmt.Printf("  Canceled item executions: %d\n", len(resp.CanceledItemExecutions))
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "mode-workspace", "--name", name),
	})
	return nil
}

func (a *App) cmdInitiativesModeStart(args []string) error {
	fs := flag.NewFlagSet("initiatives mode-start", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	phaseFlag := fs.String("phase", "", "Phase name")
	noteFlag := fs.String("note", "", "Operator note")
	overrideFlag := fs.Bool("override", false, "Acquire the initiative lock even if it is held")
	requestedByFlag := fs.String("requested-by", "", "Actor recorded for the phase start")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("name", *nameFlag, "phase", *phaseFlag); err != nil {
		return fmt.Errorf("usage: initiatives mode-start --name NAME --phase PHASE [--note MSG] [--override] [--requested-by WHO] [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)
	phase := strings.TrimSpace(*phaseFlag)
	payload := map[string]any{
		"note":         strings.TrimSpace(*noteFlag),
		"override":     *overrideFlag,
		"requested_by": defaultString(strings.TrimSpace(*requestedByFlag), "swarm-manager-cli"),
	}
	body, err := a.core.Request("POST", fmt.Sprintf("/initiatives/%s/operating-mode/phases/%s/start", name, phase), nil, mustJSON(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	round, err := decodeResponse[operatingModeRound](body)
	if err != nil {
		return err
	}
	printModeRound("Started Round", round)
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
	name := strings.TrimSpace(*nameFlag)
	mode := strings.TrimSpace(*modeFlag)
	query := url.Values{"mode": []string{mode}}
	body, err := a.core.Request("POST", fmt.Sprintf("/initiatives/%s/operating-mode/rounds/%d/%s", name, *roundFlag, action), query, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	round, err := decodeResponse[operatingModeRound](body)
	if err != nil {
		return err
	}
	printModeRound(modeRoundActionTitle(action), round)
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
	name := strings.TrimSpace(*nameFlag)
	mode := strings.TrimSpace(*modeFlag)
	query := url.Values{"mode": []string{mode}}
	payload := map[string]any{
		"mode":         mode,
		"run_id":       strings.TrimSpace(*runIDFlag),
		"item_refs":    cliutil.ParseCSV(*itemsFlag),
		"requested_by": defaultString(strings.TrimSpace(*requestedByFlag), "swarm-manager-cli"),
	}
	body, err := a.core.Request("POST", fmt.Sprintf("/initiatives/%s/operating-mode/rounds/%d/complete-items", name, *roundFlag), query, mustJSON(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[operatingModeBacklogSyncResponse](body)
	if err != nil {
		return err
	}
	printBacklogSyncResponse(resp)
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
	name := strings.TrimSpace(*nameFlag)
	mode := strings.TrimSpace(*modeFlag)
	query := url.Values{"mode": []string{mode}}
	payload := map[string]any{
		"mode":                  mode,
		"run_id":                strings.TrimSpace(*runIDFlag),
		"accepted_mutation_ids": cliutil.ParseCSV(*mutationsFlag),
		"requested_by":          defaultString(strings.TrimSpace(*requestedByFlag), "swarm-manager-cli"),
	}
	body, err := a.core.Request("POST", fmt.Sprintf("/initiatives/%s/operating-mode/rounds/%d/apply-backlog-sync", name, *roundFlag), query, mustJSON(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[operatingModeBacklogSyncResponse](body)
	if err != nil {
		return err
	}
	printBacklogSyncResponse(resp)
	return nil
}

func printBacklogSyncResponse(resp operatingModeBacklogSyncResponse) {
	printSection("Backlog Sync")
	fmt.Printf("  Initiative: %s\n", resp.InitiativeName)
	fmt.Printf("  Mode:       %s\n", resp.Mode)
	fmt.Printf("  Round:      %d\n", resp.Round)
	if len(resp.CompletedItems) > 0 {
		fmt.Printf("  Completed:  %d item(s)\n", len(resp.CompletedItems))
		for _, item := range resp.CompletedItems {
			fmt.Printf("  - %s: %s -> %s\n", item.ItemRef, item.FromStatus, item.ToStatus)
		}
	} else if resp.ProposalResult == nil {
		fmt.Println("  Completed:  0 item(s)")
	}
	if resp.ProposalResult != nil {
		fmt.Printf("  Proposal:   %d applied, %d skipped, %d failed\n", resp.ProposalResult.Applied, resp.ProposalResult.Skipped, resp.ProposalResult.Failed)
		if resp.ProposalResult.Created > 0 || resp.ProposalResult.Updated > 0 {
			fmt.Printf("  Mutations:  %d created, %d updated\n", resp.ProposalResult.Created, resp.ProposalResult.Updated)
		}
		for _, outcome := range resp.ProposalResult.Outcomes {
			status := "applied"
			if outcome.Skipped {
				status = "skipped"
			}
			if outcome.Error != "" {
				status = "failed: " + outcome.Error
			}
			target := outcome.Target
			if target == "" {
				target = outcome.Op
			}
			fmt.Printf("  - %s (%s): %s\n", outcome.MutationID, target, status)
		}
	}
}

func printModeRound(title string, round operatingModeRound) {
	printSection(title)
	fmt.Printf("  Round:   %d\n", round.Round)
	fmt.Printf("  Mode:    %s\n", round.Mode)
	fmt.Printf("  Phase:   %s\n", round.Phase)
	fmt.Printf("  Status:  %s\n", round.Status)
	if round.RunID != "" {
		fmt.Printf("  Run ID:  %s\n", round.RunID)
	}
	if round.AgentProfileKey != "" {
		fmt.Printf("  Profile: %s\n", round.AgentProfileKey)
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
