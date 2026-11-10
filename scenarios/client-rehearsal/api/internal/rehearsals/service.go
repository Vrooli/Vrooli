package rehearsals

import (
	"context"
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
	now := s.now().UTC()
	triggerNodeID := strings.TrimSpace(in.TriggerNodeID)
	if triggerNodeID == "" && len(workflow.EntryNodeIDs) == 1 {
		triggerNodeID = workflow.EntryNodeIDs[0]
	}
	if triggerNodeID == "" {
		return Rehearsal{}, ErrInvalid{Field: "trigger_node_id", Reason: "required when the imported workflow has more than one trigger"}
	}
	trace, runErr := s.engine.Execute(ctx, workflow, triggerNodeID, simulator, in.InjectSeededFault, now)
	state := StateNeedsReview
	fault, repair := "No fault injected.", "No repair required."
	if in.InjectSeededFault {
		fault = "Seeded transient synthetic CRM 503."
		if runErr == nil {
			repair = "The engine retried the CRM operation once and the simulator assertion passed."
		}
	}
	if runErr != nil {
		state = StateFailed
		repair = "The run did not recover; inspect the retained trace before a repair attempt."
		fault = runErr.Error()
	}
	record := Rehearsal{ID: uuid.NewString(), Title: title, SourceFormat: "n8n-workflow-json/v1", SourceSHA256: sourceHash, CredentialReplacements: replacements, SimulatorProfile: profile, SimulatorSourceSHA256: simulatorHash, State: state, Trace: trace, FaultSummary: fault, RepairSummary: repair, CreatedAt: now}
	record.HandoffMarkdown = handoff(record)
	created, err := s.repo.Create(ctx, record)
	if err != nil {
		return Rehearsal{}, err
	}
	return created, nil
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
