package planrepair

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/planclient"
	"swarm-manager/internal/transitions"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type TerminalResult struct {
	Outcome       string          `json:"outcome"`
	CandidatePlan json.RawMessage `json:"candidatePlan,omitempty"`
	Summary       string          `json:"summary,omitempty"`
	Reason        string          `json:"reason,omitempty"`
	Unresolved    []string        `json:"unresolvedFindings,omitempty"`
}

type StartRequest struct {
	ID                 string
	EntityKind         string
	EntityName         string
	EntityVersion      string
	PlanReference      string
	BaseContentHash    string
	PlanContent        string
	FrontierDigest     string
	ValidationFindings []any
	CheckedAt          string
	MaxRepairAttempts  int
}

type Service struct {
	store              *Store
	workflow           agentmanager.WorkflowInvoker
	transitionRegistry transitions.Registry
}

type CandidateCanonicalizer interface {
	planclient.CandidateClient
}

// Binder remains domain-owned: it re-reads the current entity and moves its
// canonical plan reference only when the authorized frontier still matches.
type Binder interface {
	BindRepairedPlan(context.Context, Record, *sharedv1.Plan) error
}

func NewService(store *Store, workflow agentmanager.WorkflowInvoker) *Service {
	svc := &Service{store: store, workflow: workflow}
	if registry, err := transitions.LoadDir(filepath.Join(pathutil.ResolveScenarioRoot("swarm-manager"), ".vrooli", "swarm-transitions")); err == nil {
		svc.transitionRegistry = registry
	}
	return svc
}

// SetStartGuard installs the server-owned registry/preflight policy on the
// default generic workflow client. Test doubles remain policy-free.
func (s *Service) SetStartGuard(guard agentmanager.WorkflowStartGuard) {
	if workflow, ok := s.workflow.(*agentmanager.WorkflowService); ok {
		workflow.SetStartGuard(guard)
	}
}

// SetTransitionRegistry supplies the immutable Swarm transition catalog used
// to locate the declared plan-repair workflow.
func (s *Service) SetTransitionRegistry(registry transitions.Registry) {
	s.transitionRegistry = registry
}

func (s *Service) Start(ctx context.Context, request StartRequest) (Record, error) {
	if s.store == nil || s.workflow == nil {
		return Record{}, fmt.Errorf("plan repair store and workflow are required")
	}
	workflow, err := s.transitionRegistry.ResolveWorkflow("plan.repair")
	if err != nil {
		return Record{}, fmt.Errorf("resolve plan.repair workflow: %w", err)
	}
	if strings.TrimSpace(request.ID) == "" {
		request.ID = repairID(request)
	}
	if existing, err := s.store.Load(request.ID); err == nil {
		return existing, nil
	} else if !isNotExist(err) {
		return Record{}, err
	}
	input, err := structpb.NewValue(map[string]any{
		"entity":      map[string]any{"kind": request.EntityKind, "name": request.EntityName, "version": request.EntityVersion},
		"plan":        map[string]any{"reference": request.PlanReference, "content": request.PlanContent, "frontierDigest": request.FrontierDigest},
		"validation":  map[string]any{"findings": request.ValidationFindings, "checkedAt": request.CheckedAt},
		"constraints": map[string]any{"maxRepairAttempts": request.MaxRepairAttempts},
	})
	if err != nil {
		return Record{}, fmt.Errorf("encode plan repair snapshot: %w", err)
	}
	started, err := s.workflow.StartWorkflow(ctx, agentmanager.Invocation{
		Owner: workflow.Owner, WorkflowKey: workflow.Key, Input: input,
		IdempotencyKey: "plan-repair/" + request.ID, FirstRunNodeID: "repair",
	})
	if err != nil {
		return Record{}, err
	}
	record := Record{ID: request.ID, EntityKind: request.EntityKind, EntityName: request.EntityName, EntityVersion: request.EntityVersion, PlanReference: request.PlanReference, BaseContentHash: request.BaseContentHash, FrontierDigest: request.FrontierDigest, WorkflowExecution: started.ExecutionID, WorkflowDigest: started.DefinitionDigest, ApplyState: ApplyPending}
	if err := s.store.Save(record); err != nil {
		return Record{}, err
	}
	return record, nil
}

// Collect validates immutable execution provenance and decodes only the typed
// declared result. It deliberately does not bind a plan or inspect prose.
func (s *Service) Collect(ctx context.Context, id string) (Record, TerminalResult, error) {
	record, err := s.store.Load(id)
	if err != nil {
		return Record{}, TerminalResult{}, err
	}
	completion, err := s.workflow.CollectWorkflow(ctx, record.WorkflowExecution)
	if err != nil {
		return Record{}, TerminalResult{}, err
	}
	if completion.ExecutionID != record.WorkflowExecution || completion.DefinitionDigest != record.WorkflowDigest {
		return Record{}, TerminalResult{}, fmt.Errorf("workflow result does not match authorized repair frontier")
	}
	if completion.Status != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED || completion.Output == nil {
		return Record{}, TerminalResult{}, fmt.Errorf("plan repair workflow did not produce an applicable terminal result")
	}
	output, ok := completion.Output.AsInterface().(map[string]any)
	if !ok {
		return Record{}, TerminalResult{}, fmt.Errorf("plan repair workflow output is invalid")
	}
	raw, ok := output["result"]
	if !ok {
		return Record{}, TerminalResult{}, fmt.Errorf("plan repair workflow output omitted result")
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return Record{}, TerminalResult{}, fmt.Errorf("encode typed plan repair result: %w", err)
	}
	var result TerminalResult
	if err := json.Unmarshal(data, &result); err != nil {
		return Record{}, TerminalResult{}, fmt.Errorf("decode typed plan repair result: %w", err)
	}
	switch result.Outcome {
	case "ready":
		if len(result.CandidatePlan) == 0 {
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

// Canonicalize persists a whole-plan candidate without changing the canonical
// plan or the backlog item's PlanRef. The operator-facing Plan Workshop owns
// preview, authorization, and the eventual guarded candidate application.
func Canonicalize(ctx context.Context, client CandidateCanonicalizer, record Record, result TerminalResult) (*plansv1.CandidateRevisionPreview, error) {
	if client == nil {
		return nil, fmt.Errorf("plan manager client is required")
	}
	if result.Outcome != "ready" {
		return nil, fmt.Errorf("only ready repair results can be canonicalized")
	}
	candidatePlan := &sharedv1.Plan{}
	if err := protojson.Unmarshal(result.CandidatePlan, candidatePlan); err != nil {
		return nil, fmt.Errorf("decode whole-plan repair candidate: %w", err)
	}
	candidate, err := client.CreateCandidateRevision(ctx, planclient.CandidateRevisionInput{
		PlanID: record.PlanReference, ExpectedBaseContentHash: record.BaseContentHash,
		ProposalProvenance: "swarm-manager:repair/" + record.ID, CandidatePlan: candidatePlan,
	})
	if err != nil {
		return nil, err
	}
	if candidate == nil || strings.TrimSpace(candidate.GetId()) == "" {
		return nil, fmt.Errorf("plan manager candidate creation omitted candidate id")
	}
	return client.PreviewCandidateRevision(ctx, candidate.GetId())
}

func repairID(request StartRequest) string {
	h := sha256.Sum256([]byte(strings.Join([]string{request.EntityKind, request.EntityName, request.EntityVersion, request.PlanReference, request.FrontierDigest}, "\x00")))
	return "repair-" + hex.EncodeToString(h[:16])
}
func isNotExist(err error) bool { return os.IsNotExist(err) }
