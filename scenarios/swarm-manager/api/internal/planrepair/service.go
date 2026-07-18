package planrepair

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/planclient"
)

// WorkflowStarter is the declared-workflow boundary. It intentionally has no
// raw Run operations.
type WorkflowStarter interface {
	StartPlanRepair(context.Context, agentmanager.PlanRepairSnapshot, string) (agentmanager.WorkflowStart, error)
}

type WorkflowCollector interface {
	CollectPlanRepair(context.Context, string) (agentmanager.PlanRepairWorkflowCompletion, error)
}

type TerminalResult struct {
	Outcome       string   `json:"outcome"`
	CandidatePlan string   `json:"candidatePlan,omitempty"`
	Summary       string   `json:"summary,omitempty"`
	Reason        string   `json:"reason,omitempty"`
	Unresolved    []string `json:"unresolvedFindings,omitempty"`
}

type StartRequest struct {
	ID                 string
	EntityKind         string
	EntityName         string
	EntityVersion      string
	PlanReference      string
	PlanContent        string
	FrontierDigest     string
	ValidationFindings []any
	CheckedAt          string
	MaxRepairAttempts  int
}

type Service struct {
	store    *Store
	workflow WorkflowStarter
}

type Canonicalizer interface {
	ImportPlan(context.Context, planclient.ImportPlanInput) (*sharedv1.Plan, error)
	RenderMarkdown(context.Context, string, bool) (planclient.RenderMarkdownResult, error)
}

// Binder remains domain-owned: it re-reads the current entity and moves its
// canonical plan reference only when the authorized frontier still matches.
type Binder interface {
	BindRepairedPlan(context.Context, Record, *sharedv1.Plan) error
}

func NewService(store *Store, workflow WorkflowStarter) *Service {
	return &Service{store: store, workflow: workflow}
}

// SetStartGuard installs the server-owned registry/preflight policy on the
// default generic workflow client. Test doubles remain policy-free.
func (s *Service) SetStartGuard(guard agentmanager.WorkflowStartGuard) {
	if workflow, ok := s.workflow.(*agentmanager.WorkflowService); ok {
		workflow.SetStartGuard(guard)
	}
}

func (s *Service) Start(ctx context.Context, request StartRequest) (Record, error) {
	if s.store == nil || s.workflow == nil {
		return Record{}, fmt.Errorf("plan repair store and workflow are required")
	}
	if strings.TrimSpace(request.ID) == "" {
		request.ID = repairID(request)
	}
	if existing, err := s.store.Load(request.ID); err == nil {
		return existing, nil
	} else if !isNotExist(err) {
		return Record{}, err
	}
	snapshot := agentmanager.PlanRepairSnapshot{EntityKind: request.EntityKind, EntityName: request.EntityName, EntityVersion: request.EntityVersion, PlanReference: request.PlanReference, PlanContent: request.PlanContent, FrontierDigest: request.FrontierDigest, ValidationFindings: request.ValidationFindings, CheckedAt: request.CheckedAt, MaxRepairAttempts: request.MaxRepairAttempts}
	started, err := s.workflow.StartPlanRepair(ctx, snapshot, "plan-repair/"+request.ID)
	if err != nil {
		return Record{}, err
	}
	record := Record{ID: request.ID, EntityKind: request.EntityKind, EntityName: request.EntityName, EntityVersion: request.EntityVersion, PlanReference: request.PlanReference, FrontierDigest: request.FrontierDigest, WorkflowExecution: started.ExecutionID, WorkflowDigest: started.DefinitionDigest, ApplyState: ApplyPending}
	if err := s.store.Save(record); err != nil {
		return Record{}, err
	}
	return record, nil
}

// Collect validates immutable execution provenance and decodes only the typed
// declared result. It deliberately does not bind a plan or inspect prose.
func (s *Service) Collect(ctx context.Context, id string) (Record, TerminalResult, error) {
	collector, ok := s.workflow.(WorkflowCollector)
	if !ok {
		return Record{}, TerminalResult{}, fmt.Errorf("plan repair workflow cannot collect terminal result")
	}
	record, err := s.store.Load(id)
	if err != nil {
		return Record{}, TerminalResult{}, err
	}
	completion, err := collector.CollectPlanRepair(ctx, record.WorkflowExecution)
	if err != nil {
		return Record{}, TerminalResult{}, err
	}
	if completion.ExecutionID != record.WorkflowExecution || completion.DefinitionDigest != record.WorkflowDigest {
		return Record{}, TerminalResult{}, fmt.Errorf("workflow result does not match authorized repair frontier")
	}
	var result TerminalResult
	if err := json.Unmarshal(completion.Result, &result); err != nil {
		return Record{}, TerminalResult{}, fmt.Errorf("decode typed plan repair result: %w", err)
	}
	switch result.Outcome {
	case "ready":
		if strings.TrimSpace(result.CandidatePlan) == "" {
			return Record{}, TerminalResult{}, fmt.Errorf("ready repair result is missing candidatePlan")
		}
	case "needs_attention", "abstained":
		if strings.TrimSpace(result.Reason) == "" {
			return Record{}, TerminalResult{}, fmt.Errorf("%s repair result is missing reason", result.Outcome)
		}
	default:
		return Record{}, TerminalResult{}, fmt.Errorf("unsupported plan repair outcome %q", result.Outcome)
	}
	return record, result, nil
}

// Canonicalize imports a ready candidate as a superseding Plan Manager plan
// and requires Plan Manager's rendered quality verdict before it can be bound.
func Canonicalize(ctx context.Context, client Canonicalizer, record Record, result TerminalResult) (*sharedv1.Plan, error) {
	if client == nil {
		return nil, fmt.Errorf("plan manager client is required")
	}
	if result.Outcome != "ready" {
		return nil, fmt.Errorf("only ready repair results can be canonicalized")
	}
	plan, err := client.ImportPlan(ctx, planclient.ImportPlanInput{Markdown: result.CandidatePlan, SourcePath: "swarm-manager:repair/" + record.ID, Supersede: record.PlanReference})
	if err != nil {
		return nil, err
	}
	if plan == nil || strings.TrimSpace(plan.GetId()) == "" {
		return nil, fmt.Errorf("plan manager import omitted canonical plan")
	}
	rendered, err := client.RenderMarkdown(ctx, plan.GetId(), true)
	if err != nil {
		return nil, err
	}
	if rendered.QualityStatus != "pass" {
		return nil, fmt.Errorf("canonical repaired plan quality is %q", rendered.QualityStatus)
	}
	return plan, nil
}

func (s *Service) CompleteApply(ctx context.Context, binder Binder, record Record, plan *sharedv1.Plan) (Record, error) {
	if binder == nil || plan == nil || strings.TrimSpace(plan.GetId()) == "" {
		return Record{}, fmt.Errorf("repair binder and canonical plan are required")
	}
	if record.ApplyState == ApplyComplete {
		return record, nil
	}
	if record.ApplyState != ApplyPending && record.ApplyState != ApplyClaimed {
		return Record{}, fmt.Errorf("repair %q is not applicable", record.ID)
	}
	record.ApplyState = ApplyClaimed
	if err := s.store.Save(record); err != nil {
		return Record{}, err
	}
	if err := binder.BindRepairedPlan(ctx, record, plan); err != nil {
		return Record{}, err
	}
	record.ApplyState, record.AppliedPlanID = ApplyComplete, plan.GetId()
	if err := s.store.Save(record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func repairID(request StartRequest) string {
	h := sha256.Sum256([]byte(strings.Join([]string{request.EntityKind, request.EntityName, request.EntityVersion, request.PlanReference, request.FrontierDigest}, "\x00")))
	return "repair-" + hex.EncodeToString(h[:16])
}
func isNotExist(err error) bool { return os.IsNotExist(err) }
