package requirements

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"test-genie/internal/orchestrator/phases"

	reqsvc "test-genie/internal/requirements"
	reqtypes "test-genie/internal/requirements/types"
)

type Syncer interface {
	// Sync writes requirement updates from this run's evidence and returns a
	// structured outcome (counts + changes). A nil outcome means there is no
	// requirements directory.
	Sync(ctx context.Context, input SyncInput) (*SyncOutcome, error)
	// Snapshot reads the last persisted requirement state without writing. It
	// powers the execute report when sync is gated/skipped so cached counts are
	// still surfaced. A nil outcome means there is no requirements directory.
	Snapshot(ctx context.Context, input SyncInput) (*SyncOutcome, error)
}

type SyncInput struct {
	ScenarioName     string
	ScenarioDir      string
	PhaseDefinitions []phases.Definition
	PhaseResults     []phases.ExecutionResult
	CommandHistory   []string
}

// OTCount is the complete/total tally for one operational-target priority band.
type OTCount struct {
	Complete int `json:"complete"`
	Total    int `json:"total"`
}

// RequirementChange is one requirement-level status transition surfaced in the
// execute report. Kind is computed server-side so the CLI never re-derives
// status semantics: "promotion" (… → complete), "regression" (complete → …),
// or "other".
type RequirementChange struct {
	ID     string `json:"id"`
	PRDRef string `json:"prdRef,omitempty"`
	From   string `json:"from"`
	To     string `json:"to"`
	Kind   string `json:"kind"`
}

// Change kinds.
const (
	ChangeKindPromotion  = "promotion"
	ChangeKindRegression = "regression"
	ChangeKindOther      = "other"
)

// SyncOutcome is the JSON-serializable requirements summary attached to a suite
// execution result. It is rendered by the CLI execute report regardless of
// which phases ran. On a gated/skipped run, Synced is false, SkipReason is set,
// Changes is empty, and the counts reflect the last persisted state.
type SyncOutcome struct {
	Synced       bool                `json:"synced"`
	SkipReason   string              `json:"skipReason,omitempty"`
	LastSyncedAt *time.Time          `json:"lastSyncedAt,omitempty"`
	OTComplete   int                 `json:"otComplete"`
	OTTotal      int                 `json:"otTotal"`
	OTByPriority map[string]OTCount  `json:"otByPriority,omitempty"`
	ReqComplete  int                 `json:"reqComplete"`
	ReqTotal     int                 `json:"reqTotal"`
	ReqByStatus  map[string]int      `json:"reqByStatus,omitempty"`
	Changes      []RequirementChange `json:"changes,omitempty"`
}

// RegressionCount returns the number of changes classified as regressions.
func (o *SyncOutcome) RegressionCount() int {
	if o == nil {
		return 0
	}
	n := 0
	for _, c := range o.Changes {
		if c.Kind == ChangeKindRegression {
			n++
		}
	}
	return n
}

// newOutcomeFromReport translates a reqsvc.SyncReport into the JSON-facing
// SyncOutcome, classifying each change and flattening the count maps.
func newOutcomeFromReport(report *reqsvc.SyncReport, skipReason string) *SyncOutcome {
	if report == nil {
		return nil
	}
	out := &SyncOutcome{
		Synced:       report.Synced,
		SkipReason:   skipReason,
		LastSyncedAt: report.LastSyncedAt,
		OTComplete:   report.OT.Complete,
		OTTotal:      report.OT.Total,
		ReqTotal:     report.Summary.Total,
	}
	if len(report.OT.ByPriority) > 0 {
		out.OTByPriority = make(map[string]OTCount, len(report.OT.ByPriority))
		for band, c := range report.OT.ByPriority {
			out.OTByPriority[band] = OTCount{Complete: c.Complete, Total: c.Total}
		}
	}
	if len(report.Summary.ByDeclaredStatus) > 0 {
		out.ReqByStatus = make(map[string]int, len(report.Summary.ByDeclaredStatus))
		for status, n := range report.Summary.ByDeclaredStatus {
			out.ReqByStatus[string(status)] = n
		}
		out.ReqComplete = report.Summary.ByDeclaredStatus[reqtypes.StatusComplete]
	}
	for _, ch := range report.Changes {
		out.Changes = append(out.Changes, RequirementChange{
			ID:     ch.ID,
			PRDRef: ch.PRDRef,
			From:   ch.From,
			To:     ch.To,
			Kind:   classifyChange(ch.From, ch.To),
		})
	}
	return out
}

// classifyChange labels a declared-status transition. Reaching complete is a
// promotion; leaving complete is a regression; everything else is neutral.
func classifyChange(from, to string) string {
	fromComplete := reqtypes.NormalizeDeclaredStatus(from) == reqtypes.StatusComplete
	toComplete := reqtypes.NormalizeDeclaredStatus(to) == reqtypes.StatusComplete
	switch {
	case !fromComplete && toComplete:
		return ChangeKindPromotion
	case fromComplete && !toComplete:
		return ChangeKindRegression
	default:
		return ChangeKindOther
	}
}

// nativeRequirementsSyncer uses the native Go implementation.
type nativeRequirementsSyncer struct {
	service *reqsvc.Service
}

// NewNativeSyncer creates a native Go requirements syncer.
// This is the preferred syncer that does not require Node.js.
func NewNativeSyncer() Syncer {
	return &nativeRequirementsSyncer{
		service: reqsvc.NewService(),
	}
}

func (s *nativeRequirementsSyncer) toServiceInput(input SyncInput) reqsvc.SyncInput {
	return reqsvc.SyncInput{
		ScenarioName:     input.ScenarioName,
		ScenarioDir:      input.ScenarioDir,
		PhaseDefinitions: input.PhaseDefinitions,
		PhaseResults:     input.PhaseResults,
		CommandHistory:   input.CommandHistory,
	}
}

func (s *nativeRequirementsSyncer) Sync(ctx context.Context, input SyncInput) (*SyncOutcome, error) {
	report, err := s.service.Sync(ctx, s.toServiceInput(input))
	if err != nil {
		return nil, err
	}
	return newOutcomeFromReport(report, ""), nil
}

func (s *nativeRequirementsSyncer) Snapshot(ctx context.Context, input SyncInput) (*SyncOutcome, error) {
	report, err := s.service.Snapshot(ctx, s.toServiceInput(input))
	if err != nil {
		return nil, err
	}
	return newOutcomeFromReport(report, ""), nil
}

// NewSyncer creates a requirements syncer, preferring native Go over Node.js.
// Environment variable REQUIREMENTS_SYNC_NATIVE=true forces native Go.
// Environment variable REQUIREMENTS_SYNC_NODE=true forces Node.js (if available).
func NewSyncer(projectRoot string) Syncer {
	// Check for explicit preference
	if os.Getenv("REQUIREMENTS_SYNC_NODE") == "true" {
		if syncer := NewNodeSyncer(projectRoot); syncer != nil {
			return syncer
		}
	}

	// Default to native Go
	return NewNativeSyncer()
}

type nodeRequirementsSyncer struct {
	projectRoot string
	scriptPath  string
	// reader is used to read back persisted counts after the Node script runs
	// and for read-only snapshots, reusing the native status model.
	reader *reqsvc.Service
}

var execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

// NewNodeSyncer creates a Node.js-based syncer (legacy).
// Returns nil if the Node.js script is not available.
func NewNodeSyncer(projectRoot string) Syncer {
	scriptPath := filepath.Join(projectRoot, "scripts", "requirements", "report.js")
	if _, err := os.Stat(scriptPath); err != nil {
		return nil
	}
	return &nodeRequirementsSyncer{
		projectRoot: projectRoot,
		scriptPath:  scriptPath,
		reader:      reqsvc.NewService(),
	}
}

func (s *nodeRequirementsSyncer) Sync(ctx context.Context, input SyncInput) (*SyncOutcome, error) {
	if input.ScenarioName == "" {
		return nil, nil
	}
	requirementsDir := filepath.Join(input.ScenarioDir, "requirements")
	if _, err := os.Stat(requirementsDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("requirements directory validation failed: %w", err)
	}

	if err := phases.EnsureCommandAvailable("node"); err != nil {
		return nil, fmt.Errorf("node command not available: %w", err)
	}

	phasePayload := buildPhaseStatusPayload(input.PhaseDefinitions, input.PhaseResults)
	if len(phasePayload) == 0 {
		return nil, fmt.Errorf("phase execution metadata missing")
	}
	phaseJSON, err := json.Marshal(phasePayload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode phase execution metadata: %w", err)
	}

	commandHistory := input.CommandHistory
	if len(commandHistory) == 0 {
		commandHistory = []string{fmt.Sprintf("suite %s", input.ScenarioName)}
	}
	commandJSON, err := json.Marshal(commandHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to encode command history: %w", err)
	}

	cmd := execCommandContext(ctx, "node", s.scriptPath, "--scenario", input.ScenarioName, "--mode", "sync")
	cmd.Dir = s.projectRoot
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("REQUIREMENTS_SYNC_PHASE_STATUS=%s", string(phaseJSON)),
		fmt.Sprintf("REQUIREMENTS_SYNC_TEST_COMMANDS=%s", string(commandJSON)),
	)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("requirements sync command failed: %w: %s", err, strings.TrimSpace(output.String()))
	}

	// The Node script writes the requirement files; read the persisted counts
	// back through the native status model so the report is engine-agnostic.
	// The Node path cannot supply a per-requirement change list, so changes are
	// omitted but counts are accurate and Synced is true.
	report, err := s.reader.Snapshot(ctx, s.toServiceInput(input))
	if err != nil {
		return nil, nil
	}
	if report != nil {
		report.Synced = true
	}
	return newOutcomeFromReport(report, ""), nil
}

func (s *nodeRequirementsSyncer) Snapshot(ctx context.Context, input SyncInput) (*SyncOutcome, error) {
	report, err := s.reader.Snapshot(ctx, s.toServiceInput(input))
	if err != nil {
		return nil, err
	}
	return newOutcomeFromReport(report, ""), nil
}

func (s *nodeRequirementsSyncer) toServiceInput(input SyncInput) reqsvc.SyncInput {
	return reqsvc.SyncInput{
		ScenarioName:     input.ScenarioName,
		ScenarioDir:      input.ScenarioDir,
		PhaseDefinitions: input.PhaseDefinitions,
		PhaseResults:     input.PhaseResults,
		CommandHistory:   input.CommandHistory,
	}
}

type phaseStatusEntry struct {
	Phase    string `json:"phase"`
	Status   string `json:"status"`
	Optional bool   `json:"optional"`
	Recorded bool   `json:"recorded"`
}

func buildPhaseStatusPayload(defs []phases.Definition, results []phases.ExecutionResult) []phaseStatusEntry {
	if len(defs) == 0 {
		return nil
	}
	payload := make([]phaseStatusEntry, 0, len(defs))
	resultLookup := make(map[string]phases.ExecutionResult, len(results))
	for _, result := range results {
		key := phases.NormalizeKey(result.Name)
		if key == "" {
			continue
		}
		resultLookup[key] = result
	}
	for _, def := range defs {
		key := def.Name.Key()
		if key == "" {
			continue
		}
		entry := phaseStatusEntry{
			Phase:    def.Name.String(),
			Optional: def.Optional,
		}
		if result, ok := resultLookup[key]; ok {
			status := strings.ToLower(strings.TrimSpace(result.Status))
			if status == "" {
				status = "unknown"
			}
			entry.Status = status
			entry.Recorded = true
		} else {
			entry.Status = "not_run"
			entry.Recorded = false
		}
		payload = append(payload, entry)
	}
	return payload
}
