package rehearsals

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	CreateAndRun(context.Context, CreateInput) (Rehearsal, error)
	List(context.Context, int) ([]Rehearsal, error)
	Get(context.Context, string) (Rehearsal, error)
	Review(context.Context, ReviewInput) (Rehearsal, error)
}
type service struct {
	repo   Repository
	engine Engine
	now    func() time.Time
}

func NewService(repo Repository, engine Engine, now func() time.Time) Service {
	if now == nil {
		now = time.Now
	}
	return &service{repo: repo, engine: engine, now: now}
}

func (s *service) CreateAndRun(ctx context.Context, in CreateInput) (Rehearsal, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" || len(title) > 120 {
		return Rehearsal{}, ErrInvalid{Field: "title", Reason: "must contain 1 to 120 characters"}
	}
	profile := strings.TrimSpace(in.SimulatorProfile)
	if profile == "" {
		profile = simulatorSourceKind(in.SimulatorSourceJSON)
	}
	workflow, sourceHash, replacements, err := normalizeWorkflow(in.WorkflowJSON)
	if err != nil {
		return Rehearsal{}, err
	}
	simulator, err := NewDescriptionSimulator(in.SimulatorSourceJSON, in.InjectSeededFault)
	if err != nil {
		return Rehearsal{}, err
	}
	simulatorHash, err := simulatorSourceSHA256(in.SimulatorSourceJSON)
	if err != nil {
		return Rehearsal{}, err
	}
	requiredPaths, err := normalizeRequiredOutputPaths(in.RequiredOutputPaths)
	if err != nil {
		return Rehearsal{}, err
	}
	var repairedRun *Rehearsal
	if repairOfID := strings.TrimSpace(in.RepairOfID); repairOfID != "" {
		prior, getErr := s.repo.Get(ctx, repairOfID)
		if getErr != nil {
			return Rehearsal{}, getErr
		}
		if prior.State != StateFailed || prior.SimulatorSourceSHA256 != simulatorHash || prior.SourceSHA256 == sourceHash {
			return Rehearsal{}, ErrInvalid{Field: "repair_of_id", Reason: "must reference a failed run with the same simulator profile and a different workflow revision"}
		}
		priorPaths, ok := missingPathsFromFault(prior.FaultSummary)
		if !ok || !sameStrings(priorPaths, requiredPaths) {
			return Rehearsal{}, ErrInvalid{Field: "required_output_paths", Reason: "a repair must repeat the exact missing-output checks from its failed run"}
		}
		repairedRun = &prior
	}
	now := s.now().UTC()
	triggerNodeID := strings.TrimSpace(in.TriggerNodeID)
	if triggerNodeID == "" && len(workflow.EntryNodeIDs) == 1 {
		triggerNodeID = workflow.EntryNodeIDs[0]
	}
	if triggerNodeID == "" {
		return Rehearsal{}, ErrInvalid{Field: "trigger_node_id", Reason: "required when the imported workflow has more than one trigger"}
	}
	var execution ExecutionResult
	var runErr error
	if detailed, ok := s.engine.(DetailedEngine); ok {
		execution, runErr = detailed.ExecuteDetailed(ctx, workflow, triggerNodeID, simulator, in.InjectSeededFault, now)
	} else {
		if len(requiredPaths) > 0 {
			return Rehearsal{}, ErrInvalid{Field: "required_output_paths", Reason: "the selected engine does not return internal assertion output"}
		}
		execution.Trace, runErr = s.engine.Execute(ctx, workflow, triggerNodeID, simulator, in.InjectSeededFault, now)
	}
	missingPaths := missingRequiredOutputs(execution.Output, requiredPaths)
	for _, path := range requiredPaths {
		status, summary := "succeeded", "required output path present"
		if contains(missingPaths, path) {
			status, summary = "failed", "required output path missing"
		}
		execution.Trace = append(execution.Trace, TraceEvent{Sequence: uint32(len(execution.Trace) + 1), Operation: "assert output path exists", RequestSummary: path, ResponseSummary: summary, Status: status, OccurredAt: now})
	}
	state := StateNeedsReview
	fault, repair := "No fault injected.", "No repair required."
	if in.InjectSeededFault {
		fault = "Seeded transient synthetic service HTTP 503."
		if runErr == nil {
			repair = "The engine retried the failed synthetic service operation once and the simulator assertion passed."
		}
	}
	if runErr != nil {
		state = StateFailed
		repair = "The run did not recover; inspect the retained trace before a repair attempt."
		fault = runErr.Error()
	}
	if len(missingPaths) > 0 {
		state = StateFailed
		fault = "Required output paths missing: " + strings.Join(missingPaths, ",")
		repair = "Correct the workflow mapping and submit a changed revision against the same simulator profile."
	}
	if repairedRun != nil {
		priorFault := repairedRun.FaultSummary
		if len(missingPaths) == 0 && runErr == nil {
			fault = "Prior run " + repairedRun.ID + " failed output checks: " + priorFault
			repair = fmt.Sprintf("Changed workflow revision passed %d required output checks and repairs run %s.", len(requiredPaths), repairedRun.ID)
		} else {
			fault = "Repair of run " + repairedRun.ID + " still failed output checks: " + strings.Join(missingPaths, ",")
			repair = "The changed revision still fails the required output checks."
		}
		execution.Trace = append(execution.Trace, TraceEvent{Sequence: uint32(len(execution.Trace) + 1), Operation: "repair lineage", RequestSummary: "prior run " + repairedRun.ID, ResponseSummary: "changed workflow revision linked to failed evidence", Status: "succeeded", OccurredAt: now})
	}
	record := Rehearsal{ID: uuid.NewString(), Title: title, SourceFormat: "n8n-workflow-json/v1", SourceSHA256: sourceHash, CredentialReplacements: replacements, SimulatorProfile: profile, SimulatorSourceSHA256: simulatorHash, State: state, Trace: execution.Trace, FaultSummary: fault, RepairSummary: repair, CreatedAt: now}
	record.HandoffMarkdown = handoff(record)
	created, err := s.repo.Create(ctx, record)
	if err != nil {
		return Rehearsal{}, err
	}
	return created, nil
}

func normalizeRequiredOutputPaths(paths []string) ([]string, error) {
	if len(paths) > 32 {
		return nil, ErrInvalid{Field: "required_output_paths", Reason: "must contain at most 32 paths"}
	}
	result := make([]string, 0, len(paths))
	seen := map[string]bool{}
	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		if len(path) == 0 || len(path) > 256 || !validJSONPath(path) || seen[path] {
			return nil, ErrInvalid{Field: "required_output_paths", Reason: "paths must be unique, bounded static JSON paths"}
		}
		for _, part := range strings.Split(path, ".") {
			if secretKey(part) {
				return nil, ErrInvalid{Field: "required_output_paths", Reason: "paths must not target credential-shaped fields"}
			}
		}
		seen[path] = true
		result = append(result, path)
	}
	sort.Strings(result)
	return result, nil
}

func missingRequiredOutputs(output map[string]any, paths []string) []string {
	missing := make([]string, 0)
	for _, path := range paths {
		if _, ok := jsonPathValue(output, path); !ok {
			missing = append(missing, path)
		}
	}
	return missing
}

func missingPathsFromFault(fault string) ([]string, bool) {
	const prefix = "Required output paths missing: "
	if !strings.HasPrefix(fault, prefix) {
		return nil, false
	}
	paths, err := normalizeRequiredOutputPaths(strings.Split(strings.TrimPrefix(fault, prefix), ","))
	return paths, err == nil
}

func sameStrings(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func (s *service) List(ctx context.Context, limit int) ([]Rehearsal, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.List(ctx, limit)
}

func (s *service) Get(ctx context.Context, id string) (Rehearsal, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Rehearsal{}, ErrInvalid{Field: "id", Reason: "required"}
	}
	return s.repo.Get(ctx, id)
}

func (s *service) Review(ctx context.Context, in ReviewInput) (Rehearsal, error) {
	if in.Decision != DecisionApprove && in.Decision != DecisionRequestChanges {
		return Rehearsal{}, ErrInvalid{Field: "decision", Reason: "must explicitly approve or request changes"}
	}
	if len(in.Note) > 2000 {
		return Rehearsal{}, ErrInvalid{Field: "note", Reason: "must be at most 2000 characters"}
	}
	return s.repo.RecordReview(ctx, strings.TrimSpace(in.ID), in.Decision, strings.TrimSpace(in.Note), s.now().UTC())
}

func handoff(r Rehearsal) string {
	return "# Rehearsal handoff: " + r.Title + "\n\n" +
		"- Source: `" + r.SourceFormat + "` (sanitized SHA-256 `" + r.SourceSHA256 + "`)\n" +
		"- Credentials replaced: " + itoa(r.CredentialReplacements) + "\n" +
		"- Simulator profile: `" + r.SimulatorProfile + "`\n" +
		"- Simulator source fingerprint: `" + r.SimulatorSourceSHA256 + "`\n" +
		"- Run state: `" + string(r.State) + "`\n" +
		"- Fault: " + r.FaultSummary + "\n" +
		"- Repair: " + r.RepairSummary + "\n\n" +
		"Human review is required; this rehearsal is synthetic evidence and does not contact a client service."
}

func itoa(n uint32) string {
	if n == 0 {
		return "0"
	}
	var b [10]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
