package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/supervision"
	"agent-manager/internal/workflowcatalog"
	"agent-manager/internal/workflowruntime"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
	watchpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	executionpb "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
	executionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution/execution_v1connect"
	familypb "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families"
	familyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families/families_v1connect"
	"google.golang.org/protobuf/proto"
)

const planFamilyWorkflowKey = "plan-manager/plan-family-drain"

// Family operations are owner calls around the existing durable interpreter.
// No alternate scheduler, cursor store or child-launch mechanism is introduced.
func defaultFamilyClients(ctx context.Context) (familyconnect.FamiliesServiceClient, executionconnect.ExecutionServiceClient, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "plan-manager")
	if err != nil {
		return nil, nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	return familyconnect.NewFamiliesServiceClient(client, base), executionconnect.NewExecutionServiceClient(client, base), nil
}

type familyOwnerClients struct {
	families   familyconnect.FamiliesServiceClient
	executions executionconnect.ExecutionServiceClient
}

func (o *Orchestrator) resolveFamilyClients(ctx context.Context) (familyconnect.FamiliesServiceClient, executionconnect.ExecutionServiceClient, error) {
	if o.familyOwners != nil {
		return o.familyOwners.families, o.familyOwners.executions, nil
	}
	return defaultFamilyClients(ctx)
}
func familyRequestDigest(raw json.RawMessage) string {
	var decoded any
	if json.Unmarshal(raw, &decoded) != nil {
		return ""
	}
	canonical, _ := json.Marshal(decoded)
	return fmt.Sprintf("%x", sha256.Sum256(canonical))
}
func (o *Orchestrator) SetFamilySupervision(service *supervision.Service) {
	o.familySupervision = service
}

func (o *Orchestrator) preparePlanFamilyWorkflow(ctx context.Context, req *StartWorkflowExecutionRequest) (*domain.WorkflowRevision, error) {
	var input struct {
		FamilyID      string `json:"family_id"`
		PolicyVersion string `json:"policy_version"`
		ParentRunID   string `json:"parent_run_id"`
		RoleRef       string `json:"role_ref"`
	}
	if err := json.Unmarshal(req.Input, &input); err != nil {
		return nil, err
	}
	if input.FamilyID == "" || input.PolicyVersion == "" || len(req.Input) > 2048 || len(input.RoleRef) > 128 || len(input.PolicyVersion) > 128 || len(input.FamilyID) > 128 {
		return nil, fmt.Errorf("family_id and immutable policy_version are required")
	}
	if input.RoleRef == "" {
		input.RoleRef = "code.default"
	}
	families, _, err := o.resolveFamilyClients(ctx)
	if err != nil {
		return nil, err
	}
	got, err := families.GetFamily(ctx, connect.NewRequest(&familypb.GetFamilyRequest{FamilyId: input.FamilyID}))
	if err != nil {
		return nil, err
	}
	frontier, err := families.GetFrontier(ctx, connect.NewRequest(&familypb.GetFrontierRequest{FamilyId: input.FamilyID}))
	if err != nil {
		return nil, err
	}
	definition, executionInput, err := compilePlanFamilyWorkflow(got.Msg.GetFamily(), frontier.Msg, input.PolicyVersion, input.ParentRunID, input.RoleRef)
	if err != nil {
		return nil, err
	}
	definition["metadata"].(map[string]string)["plan-family-request"] = familyRequestDigest(req.Input)
	encoded, err := json.Marshal(definition)
	if err != nil {
		return nil, err
	}
	encoded, err = o.resolveWorkflowPromptRefs(ctx, encoded)
	if err != nil {
		return nil, err
	}
	parsed, err := workflowcatalog.Parse(encoded, nil)
	if err != nil {
		return nil, err
	}
	if domain.HasBlockingDiagnostic(parsed.Diagnostics) {
		return nil, fmt.Errorf("family workflow contract: %v", parsed.Diagnostics)
	}
	if diagnostics := o.validateWorkflowTargets(ctx, &parsed.Definition, nil); domain.HasBlockingDiagnostic(diagnostics) {
		return nil, fmt.Errorf("family workflow targets: %v", diagnostics)
	}
	now := o.now()
	revision := &domain.WorkflowRevision{ID: uuid.New(), Owner: "plan-manager", Key: parsed.Definition.Key, SemanticVersion: parsed.Definition.Version, Digest: parsed.Digest, Definition: parsed.Definition, SourcePath: "plan-family:" + input.FamilyID, SourceHash: parsed.Digest, SourceUpdatedAt: now, Active: true, CreatedAt: now}
	if err = o.enforceWorkflowTrigger(ctx, revision, *req); err != nil {
		return nil, err
	}
	if err = o.workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		return nil, err
	}
	req.Input = executionInput
	return revision, nil
}

// Plan Manager supplies all dependency/conflict ordering. This compiler only
// maps its reviewed waves onto standard run/parallel/join/end nodes.
func compilePlanFamilyWorkflow(f *familypb.PlanFamily, frontier *familypb.GetFrontierResponse, policy, parent, role string) (map[string]any, json.RawMessage, error) {
	if f == nil || !frontier.GetLaunchable() || frontier.GetGraphRevision() != f.GetGraph().GetRevision() {
		return nil, nil, fmt.Errorf("a current reviewed launchable family is required")
	}
	members := map[string]*familypb.FamilyMember{}
	for _, m := range f.GetMembers() {
		if m.GetState() == familypb.MemberState_MEMBER_STATE_RUNNING {
			return nil, nil, fmt.Errorf("attach to the existing family execution before starting another")
		}
		members[m.GetPlanId()] = m
	}
	batches := frontier.GetProjectedBatches()
	if len(batches) == 0 {
		return nil, nil, fmt.Errorf("owner supplied no projected schedule")
	}
	nodes := []any{}
	edges := []any{}
	data := map[string]any{}
	metadata := map[string]string{"plan-family-id": f.GetFamilyId(), "plan-family-graph": strconv.FormatUint(frontier.GetGraphRevision(), 10), "plan-family-policy": policy, "plan-family-parent": parent}
	entry, previous := "", ""
	childCount := 0
	addEdge := func(a, b string) { edges = append(edges, map[string]any{"from": a, "to": b}) }
	for wave, batch := range batches {
		if len(batch.GetPlanIds()) == 0 {
			continue
		}
		start, end := fmt.Sprintf("wave%d", wave), fmt.Sprintf("join%d", wave)
		if entry == "" {
			entry = start
		}
		if previous != "" {
			addEdge(previous, start)
		}
		nodes = append(nodes, map[string]any{"id": start, "kind": "branch", "branch": map[string]any{"parallel": len(batch.GetPlanIds()) > 1}}, map[string]any{"id": end, "kind": "join", "join": map[string]any{"strategy": "all"}})
		for _, id := range batch.GetPlanIds() {
			member := members[id]
			if member == nil || member.GetState() != familypb.MemberState_MEMBER_STATE_PENDING || member.GetExecutionId() == "" {
				return nil, nil, fmt.Errorf("pending member %s requires its Plan Manager execution identity", id)
			}
			childCount++
			if childCount > 32 {
				return nil, nil, fmt.Errorf("family executor accepts at most 32 pending children")
			}
			node := fmt.Sprintf("child%d", childCount)
			data[node] = map[string]any{"plan_id": id, "execution_id": member.GetExecutionId(), "family_id": f.GetFamilyId()}
			metadata["plan-node:"+node] = id
			metadata["plan-execution:"+node] = member.GetExecutionId()
			metadata["plan-wave:"+node] = strconv.Itoa(wave)
			nodes = append(nodes, map[string]any{"id": node, "kind": "run", "run": map[string]any{"roleRef": role, "promptRef": map[string]any{"skillId": "plan-manager-family-child"}, "maxTurns": 80, "timeoutSeconds": 3600, "bindings": []any{map[string]any{"name": "child", "source": "workflow_input", "selector": "$.children." + node, "limit": 1, "maxBytes": 2048, "renderAs": "json", "missingPolicy": "error"}}, "resultSpec": map[string]any{"version": "result-spec/v1", "kind": "json_schema", "extractionMode": "deterministic_only", "schema": map[string]any{"type": "object", "properties": map[string]any{"outcome": map[string]any{"enum": []string{"completed", "blocked", "abstained"}}, "evidence": map[string]any{"type": "array", "maxItems": 20, "items": map[string]any{"type": "string", "maxLength": 256}}}, "required": []string{"outcome", "evidence"}, "additionalProperties": false}}}})
			addEdge(start, node)
			addEdge(node, end)
		}
		previous = end
	}
	nodes = append(nodes, map[string]any{"id": "complete", "kind": "end", "end": map[string]any{"status": "succeeded"}})
	addEdge(previous, "complete")
	definition := map[string]any{"schemaVersion": "agent-workflow/v1", "owner": "plan-manager", "key": planFamilyWorkflowKey + "-" + f.GetFamilyId(), "version": fmt.Sprintf("1.0.%d+%x", frontier.GetGraphRevision(), sha256.Sum256([]byte(policy+"\x00"+role+"\x00"+parent))), "description": "Execute one reviewed plan family through owner admission and durable child receipts.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"children": map[string]any{"type": "object"}}, "required": []string{"children"}, "additionalProperties": false}, "outputSchema": map[string]any{"type": "object"}, "entryNode": entry, "nodes": nodes, "edges": edges, "metadata": metadata, "budgets": map[string]any{"wallTimeSeconds": 14400, "maxTurns": 80 * childCount, "maxTokens": 200000 * childCount, "maxChargeMicroUsd": 20000000, "maxNodeAttempts": childCount + 3*len(batches) + 1, "maxChildren": childCount, "maxConcurrency": f.GetPolicy().GetMaximumParallelPlans(), "maxRecursion": 1, "maxRetries": 1, "maxWaitSeconds": 14400}}
	raw, _ := json.Marshal(map[string]any{"children": data})
	return definition, raw, nil
}

func (o *Orchestrator) admitPlanFamilyChild(ctx context.Context, req workflowruntime.ChildRequest) error {
	if o.workflowExecutions == nil || o.workflows == nil {
		return nil
	}
	execution, err := o.workflowExecutions.Get(ctx, req.ExecutionID)
	if err != nil || execution == nil {
		return err
	}
	revision, err := o.workflows.GetByDigest(ctx, execution.DefinitionDigest)
	if err != nil || revision == nil {
		return err
	}
	meta := revision.Definition.Metadata
	familyID := meta["plan-family-id"]
	if familyID == "" {
		return nil
	}
	families, _, err := o.resolveFamilyClients(ctx)
	if err != nil {
		return err
	}
	response, err := families.GetFamily(ctx, connect.NewRequest(&familypb.GetFamilyRequest{FamilyId: familyID}))
	if err != nil {
		return err
	}
	family := response.Msg.GetFamily()
	if strconv.FormatUint(family.GetGraph().GetRevision(), 10) != meta["plan-family-graph"] {
		return fmt.Errorf("family topology changed; new admission requires review")
	}
	for _, member := range family.GetMembers() {
		if member.GetPlanId() != meta["plan-node:"+req.NodeID] {
			continue
		}
		if member.GetExecutionId() != meta["plan-execution:"+req.NodeID] {
			return fmt.Errorf("family member execution identity changed")
		}
		if member.GetState() == familypb.MemberState_MEMBER_STATE_RUNNING {
			if member.GetAdmissionKey() != req.IdempotencyKey {
				return fmt.Errorf("child admission belongs to another workflow attempt")
			}
			return nil
		}
		changed := proto.Clone(member).(*familypb.FamilyMember)
		changed.State = familypb.MemberState_MEMBER_STATE_RUNNING
		changed.AdmissionKey = req.IdempotencyKey
		_, err = families.PutMember(ctx, connect.NewRequest(&familypb.PutMemberRequest{FamilyId: familyID, ExpectedRevision: family.GetRevision(), Member: changed}))
		return err
	}
	return fmt.Errorf("family member no longer exists")
}

func (o *Orchestrator) reconcilePlanFamilyExecution(ctx context.Context, x *domain.WorkflowExecution) error {
	revision, err := o.workflows.GetByDigest(ctx, x.DefinitionDigest)
	if err != nil || revision == nil {
		return err
	}
	meta := revision.Definition.Metadata
	if meta["plan-family-id"] == "" {
		return nil
	}
	families, executions, err := o.resolveFamilyClients(ctx)
	if err != nil {
		return err
	}
	response, err := families.GetFamily(ctx, connect.NewRequest(&familypb.GetFamilyRequest{FamilyId: meta["plan-family-id"]}))
	if err != nil {
		return err
	}
	family := response.Msg.GetFamily()
	attempts, err := o.workflowExecutions.ListAttempts(ctx, x.ID)
	if err != nil {
		return err
	}
	waves := map[string][]*watchpb.WatchSubject{}
	expected := map[string]int{}
	neverStarted := map[string]int{}
	for k, wave := range meta {
		if strings.HasPrefix(k, "plan-wave:") {
			expected[wave]++
		}
	}
	for _, attempt := range attempts {
		planID := meta["plan-node:"+attempt.NodeID]
		if planID == "" {
			continue
		}
		if attempt.RunID == nil && attempt.Status == domain.WorkflowAttemptFailed {
			neverStarted[meta["plan-wave:"+attempt.NodeID]]++
		}
		if attempt.RunID != nil {
			wave := meta["plan-wave:"+attempt.NodeID]
			waves[wave] = append(waves[wave], &watchpb.WatchSubject{FamilyExecutionId: x.ID.String(), PlanId: planID, RunId: attempt.RunID.String()})
		}
		if attempt.Status != domain.WorkflowAttemptCompleted && attempt.Status != domain.WorkflowAttemptFailed {
			continue
		}
		for _, member := range family.GetMembers() {
			if member.GetPlanId() != planID || member.GetState() != familypb.MemberState_MEMBER_STATE_RUNNING || member.GetAdmissionKey() != attempt.IdempotencyKey {
				continue
			}
			status, err := executions.GetStatus(ctx, connect.NewRequest(&executionpb.GetStatusRequest{ExecutionId: meta["plan-execution:"+attempt.NodeID]}))
			if err != nil {
				return err
			}
			changed := proto.Clone(member).(*familypb.FamilyMember)
			if status.Msg.GetExecution().GetComplete() && status.Msg.GetExecution().GetId() == meta["plan-execution:"+attempt.NodeID] && status.Msg.GetExecution().GetPlanId() == planID && attempt.Status == domain.WorkflowAttemptCompleted {
				changed.State = familypb.MemberState_MEMBER_STATE_SUCCEEDED
			} else {
				changed.State = familypb.MemberState_MEMBER_STATE_BLOCKED
				changed.Detail = "child ended without authoritative plan completion"
			}
			updated, err := families.PutMember(ctx, connect.NewRequest(&familypb.PutMemberRequest{FamilyId: family.GetFamilyId(), ExpectedRevision: family.GetRevision(), Member: changed}))
			if err != nil {
				return err
			}
			family = updated.Msg.GetFamily()

		}
	}
	blocked, active := false, false
	for _, member := range family.GetMembers() {
		blocked = blocked || member.GetState() == familypb.MemberState_MEMBER_STATE_BLOCKED || member.GetState() == familypb.MemberState_MEMBER_STATE_FAILED
	}
	for _, attempt := range attempts {
		active = active || attempt.Status != domain.WorkflowAttemptCompleted && attempt.Status != domain.WorkflowAttemptFailed
	}
	if blocked && !active {
		_, err = o.workflowEngine.Fail(ctx, x.ID, "plan_acceptance_missing", "a child ended without Plan Manager acceptance")
		return err
	}
	if o.familySupervision == nil {
		return fmt.Errorf("family supervision owner is unavailable")
	}
	for wave, subjects := range waves {
		if len(subjects)+neverStarted[wave] != expected[wave] {
			continue
		}
		_, _, err = o.familySupervision.Create(ctx, &watchpb.CreateCohortWatchRequest{IdempotencyKey: "family-workflow:" + x.ID.String() + ":" + wave, Spec: &watchpb.WatchSpec{FamilyExecutionId: x.ID.String(), ParentRunId: meta["plan-family-parent"], PolicyVersion: meta["plan-family-policy"], Subjects: subjects, Triggers: &watchpb.WatchTriggers{Terminal: true}}})
		if err != nil {
			return err
		}
	}
	return nil
}

// Terminal workflows release only their own claims after child termination is
// observable. An ambiguous dispatch retains the claim for durable recovery.
func (o *Orchestrator) cleanupPlanFamilyClaims(ctx context.Context, x *domain.WorkflowExecution) error {
	if x == nil || o.workflows == nil {
		return nil
	}
	revision, err := o.workflows.GetByDigest(ctx, x.DefinitionDigest)
	if err != nil {
		return err
	}
	if revision == nil {
		return nil
	}
	meta := revision.Definition.Metadata
	if meta["plan-family-id"] == "" {
		return nil
	}
	families, _, err := o.resolveFamilyClients(ctx)
	if err != nil {
		return err
	}
	response, err := families.GetFamily(ctx, connect.NewRequest(&familypb.GetFamilyRequest{FamilyId: meta["plan-family-id"]}))
	if err != nil {
		return err
	}
	family := response.Msg.GetFamily()
	attempts, err := o.workflowExecutions.ListAttempts(ctx, x.ID)
	if err != nil {
		return err
	}
	for _, attempt := range attempts {
		for _, member := range family.GetMembers() {
			if member.GetState() != familypb.MemberState_MEMBER_STATE_RUNNING || member.GetPlanId() != meta["plan-node:"+attempt.NodeID] || member.GetAdmissionKey() != attempt.IdempotencyKey {
				continue
			}
			if attempt.RunID != nil {
				state, err := o.workflowEngine.Children.Inspect(ctx, *attempt.RunID)
				if err != nil {
					return err
				}
				if !state.Terminal {
					return fmt.Errorf("family cleanup awaits child %s terminal state", attempt.RunID)
				}
			} else {
				run, err := o.persistedWorkflowChild(ctx, x.ID, attempt.ID)
				if err != nil {
					return err
				}
				if run != nil && !run.Status.IsTerminal() {
					return fmt.Errorf("family cleanup awaits unlinked child %s", run.ID)
				}
				if run == nil && attempt.Status != domain.WorkflowAttemptFailed {
					return fmt.Errorf("family cleanup retains ambiguous dispatch claim %s", attempt.ID)
				}
			}
			changed := proto.Clone(member).(*familypb.FamilyMember)
			changed.State = familypb.MemberState_MEMBER_STATE_BLOCKED
			changed.Detail = "workflow terminated; child termination confirmed; acceptance requires review"
			updated, err := families.PutMember(ctx, connect.NewRequest(&familypb.PutMemberRequest{FamilyId: family.FamilyId, ExpectedRevision: family.Revision, Member: changed}))
			if err != nil {
				return err
			}
			family = updated.Msg.Family
		}
	}
	if o.familySupervision != nil {
		watches, err := o.familySupervision.List(ctx, &watchpb.ListCohortWatchesRequest{FamilyExecutionId: x.ID.String(), PageSize: 100})
		if err != nil {
			return err
		}
		for _, watch := range watches.GetWatches() {
			if watch.GetStatus() == watchpb.WatchStatus_WATCH_STATUS_ACTIVE {
				if _, err := o.familySupervision.Cancel(ctx, &watchpb.CancelCohortWatchRequest{WatchId: watch.WatchId, ExpectedRevision: watch.Revision}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
