package rehearsals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	CreateAndRun(context.Context, CreateInput) (Rehearsal, error)
	CreateAndRunTaskScoped(context.Context, CreateInput, WorkspaceSandboxTaskRequest) (TaskScopedRun, error)
	List(context.Context, int) ([]Rehearsal, error)
	Get(context.Context, string) (Rehearsal, error)
	Review(context.Context, ReviewInput) (Rehearsal, error)
}

// seam: rehearsals.TimestampSource substitutes wall-clock reads for service timestamps.
type TimestampSource func() time.Time

type service struct {
	repo            Repository
	engine          Engine
	now             TimestampSource
	sandbox         *WorkspaceSandboxProfileClient
	n8n             N8NWorkflowRuntime
	sandboxHTTPDoer sandboxHTTPDoer
}

func NewService(repo Repository, engine Engine, now TimestampSource) Service {
	if now == nil {
		panic("rehearsals.NewService requires a clock")
	}
	return &service{repo: repo, engine: engine, now: now}
}

// NewServiceWithWorkspaceSandbox enables the explicit task-scoped rehearsal
// path. Ordinary CreateAndRun remains in-process and never allocates a Sandbox.
func NewServiceWithWorkspaceSandbox(repo Repository, engine Engine, now TimestampSource, sandbox *WorkspaceSandboxProfileClient) Service {
	if now == nil {
		panic("rehearsals.NewServiceWithWorkspaceSandbox requires a clock")
	}
	return &service{repo: repo, engine: engine, now: now, sandbox: sandbox}
}

// NewServiceWithNativeN8N enables owner-mediated task execution in the n8n
// resource. Callers must only select this constructor after the operator's
// license decision and deployment configuration allow a runtime.
func NewServiceWithNativeN8N(repo Repository, engine Engine, now TimestampSource, sandbox *WorkspaceSandboxProfileClient, runtime N8NWorkflowRuntime) Service {
	if now == nil {
		panic("rehearsals.NewServiceWithNativeN8N requires a clock")
	}
	return &service{repo: repo, engine: engine, now: now, sandbox: sandbox, n8n: runtime}
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
		if ok && !sameStrings(priorPaths, requiredPaths) {
			return Rehearsal{}, ErrInvalid{Field: "required_output_paths", Reason: "a repair must repeat the exact missing-output checks from its failed run"}
		}
		if !ok {
			// Execution failures have no missing-output list to repeat. Permit a
			// corrected revision only when the failed run had no output assertions
			// and retained an explicit failed execution event.
			if len(requiredPaths) != 0 || hasOutputAssertion(prior.Trace) || strings.TrimSpace(prior.FaultSummary) == "" {
				return Rehearsal{}, ErrInvalid{Field: "required_output_paths", Reason: "a repair must repeat the exact missing-output checks or reference an execution failure without output assertions"}
			}
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
	if _, detailed := s.engine.(DetailedEngine); !detailed && len(requiredPaths) > 0 {
		return Rehearsal{}, ErrInvalid{Field: "required_output_paths", Reason: "the selected engine does not return internal assertion output"}
	}
	if in.taskSandbox != nil && s.n8n != nil {
		if !containsTrigger(workflow.TriggerKinds, TriggerManual) {
			return Rehearsal{}, ErrInvalid{Field: "workflow.trigger", Reason: "native n8n execution currently supports only a Manual Trigger"}
		}
		if in.InjectSeededFault {
			return Rehearsal{}, ErrInvalid{Field: "inject_seeded_fault", Reason: "native n8n execution does not yet have a fault-injection profile"}
		}
		if len(requiredPaths) > 0 {
			return Rehearsal{}, ErrInvalid{Field: "required_output_paths", Reason: "native n8n execution retains output inside the resource and cannot return values for assertions"}
		}
	}
	var taskLease WorkspaceSandboxProfileLease
	var lifecycle *WorkspaceSandboxLifecycleEvidence
	if in.taskSandbox != nil {
		if s.sandbox == nil {
			return Rehearsal{}, errors.New("task-scoped rehearsal requires a configured Workspace Sandbox client")
		}
		exported, exportErr := simulator.ExportHTTPMockProfile()
		if exportErr != nil {
			return Rehearsal{}, fmt.Errorf("export synthetic Sandbox profile: %w", exportErr)
		}
		lease, evidence, startErr := s.beginTaskSandbox(ctx, *in.taskSandbox, exported)
		if startErr != nil {
			return Rehearsal{}, fmt.Errorf("start task-scoped Workspace Sandbox profile: %w", startErr)
		}
		taskLease, lifecycle = lease, &evidence
	}
	var execution ExecutionResult
	var runErr error
	engineSimulator := Simulator(simulator)
	if taskLease.SandboxID != "" && taskLease.Runtime.RuntimeID == "" {
		adapter, adapterErr := NewSandboxEngineAdapter(simulator, taskLease.Receipt, s.sandboxHTTPDoer)
		if adapterErr != nil {
			runErr = fmt.Errorf("attach Sandbox operation routes to in-process engine: %w", adapterErr)
		} else {
			engineSimulator = adapter
		}
	}
	if taskLease.Runtime.RuntimeID != "" {
		runtimeWorkflow, rewriteErr := rewriteWorkflowHTTPRequestsForSandbox(in.WorkflowJSON, taskLease.Receipt.OperationAddresses)
		runtimeEvidence := N8NWorkflowExecutionEvidence{}
		runtimeErr := rewriteErr
		if runtimeErr == nil {
			runtimeEvidence, runtimeErr = s.n8n.ImportAndExecuteManual(ctx, taskLease.Runtime, runtimeWorkflow)
		}
		runErr = runtimeErr
		lifecycle.N8NExecution = &runtimeEvidence
		lifecycle.N8NRuntimeStatus = "execution-complete"
		if runtimeErr != nil {
			lifecycle.N8NRuntimeError = runtimeErr.Error()
			lifecycle.N8NRuntimeStatus = "execution-failed"
		}
		summary, _ := json.Marshal(runtimeEvidence)
		status := "succeeded"
		if runtimeErr != nil {
			status = "failed"
		}
		execution.Trace = append(execution.Trace, TraceEvent{Sequence: uint32(len(execution.Trace) + 1), Operation: "n8n owner workflow import and execution", RequestSummary: "manual trigger; credentials replaced by n8n owner", ResponseSummary: string(summary), Status: status, OccurredAt: now})
	} else if runErr == nil {
		if detailed, ok := s.engine.(DetailedEngine); ok {
			execution, runErr = detailed.ExecuteDetailed(ctx, workflow, triggerNodeID, engineSimulator, in.InjectSeededFault, now)
		} else {
			execution.Trace, runErr = s.engine.Execute(ctx, workflow, triggerNodeID, engineSimulator, in.InjectSeededFault, now)
		}
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
			if _, hadOutputChecks := missingPathsFromFault(priorFault); hadOutputChecks {
				fault = "Prior run " + repairedRun.ID + " failed output checks: " + priorFault
				repair = fmt.Sprintf("Changed workflow revision passed %d required output checks and repairs run %s.", len(requiredPaths), repairedRun.ID)
			} else {
				fault = "Prior run " + repairedRun.ID + " failed during execution: " + priorFault
				repair = "Corrected workflow revision completed the synthetic execution and repairs run " + repairedRun.ID + "; no output-path assertions were configured."
			}
		} else {
			fault = "Repair of run " + repairedRun.ID + " still failed output checks: " + strings.Join(missingPaths, ",")
			repair = "The changed revision still fails the required output checks."
		}
		execution.Trace = append(execution.Trace, TraceEvent{Sequence: uint32(len(execution.Trace) + 1), Operation: "repair lineage", RequestSummary: "prior run " + repairedRun.ID, ResponseSummary: "changed workflow revision linked to failed evidence", Status: "succeeded", OccurredAt: now})
	}
	if lifecycle != nil {
		hits, hitsErr := s.sandbox.ReadRouteHits(ctx, taskLease.SandboxID, taskLease.Name)
		if hitsErr != nil {
			lifecycle.RouteHitsStatus = "unavailable"
			lifecycle.RouteHitsError = hitsErr.Error()
		} else {
			lifecycle.RouteHitsStatus = "available"
			lifecycle.RouteHits = &hits
		}
		if taskLease.Runtime.RuntimeID != "" && nativeWorkflowRequiresCompanionRoute(workflow) && (hitsErr != nil || !nativeWorkflowHasCompanionRouteHit(workflow, hits)) {
			gateErr := "native workflow completed without a matching verified request through its Workspace Sandbox companion route"
			if hitsErr != nil {
				gateErr = "native companion route evidence is unavailable: " + hitsErr.Error()
			}
			if runErr == nil {
				state = StateFailed
				fault = gateErr
				repair = "Inspect the retained route receipt and ensure the selected native workflow path reaches a declared companion operation."
			}
			execution.Trace = append(execution.Trace, TraceEvent{Sequence: uint32(len(execution.Trace) + 1), Operation: "native companion route evidence gate", RequestSummary: "task-owned n8n HTTP Request nodes require a route hit", ResponseSummary: gateErr, Status: "failed", OccurredAt: now})
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		var cleanupErr error
		if taskLease.Runtime.RuntimeID != "" {
			if stopErr := s.n8n.Stop(cleanupCtx, taskLease.Runtime); stopErr != nil {
				lifecycle.N8NRuntimeStatus = "stop-failed"
				lifecycle.N8NRuntimeError = stopErr.Error()
				cleanupErr = fmt.Errorf("stop task-owned n8n runtime: %w", stopErr)
			} else {
				lifecycle.N8NRuntimeStatus = "stopped"
			}
		}
		if cleanupErr == nil {
			cleanupErr = s.stopAndDeleteTaskSandbox(cleanupCtx, taskLease, taskLease.Task)
		}
		cancel()
		lifecycle.CleanupConfirmedAt = s.now().UTC()
		if cleanupErr != nil {
			lifecycle.TeardownStatus = "failed"
			lifecycle.TeardownError = cleanupErr.Error()
		} else {
			lifecycle.TeardownStatus = "succeeded"
		}
		if !lifecycle.SyntheticTargets || lifecycle.TeardownStatus != "succeeded" {
			state = StateFailed
			if lifecycle.TeardownStatus != "succeeded" {
				fault = "Workspace Sandbox task teardown failed: " + lifecycle.TeardownError
				repair = "Inspect the Sandbox owner receipt and retry cleanup before accepting this rehearsal."
			} else {
				fault = "Workspace Sandbox profile did not prove synthetic-only operation routes."
			}
		}
		evidenceJSON, _ := json.Marshal(lifecycle)
		execution.Trace = append(execution.Trace, TraceEvent{Sequence: uint32(len(execution.Trace) + 1), Operation: "sandbox task lifecycle receipt", RequestSummary: "sandbox=" + lifecycle.SandboxID + " profile=" + lifecycle.ProfileSHA256, ResponseSummary: string(evidenceJSON), Status: lifecycle.TeardownStatus, OccurredAt: lifecycle.CleanupConfirmedAt})
	}
	record := Rehearsal{ID: uuid.NewString(), Title: title, SourceFormat: "n8n-workflow-json/v1", SourceSHA256: sourceHash, CredentialReplacements: replacements, SimulatorProfile: profile, SimulatorSourceSHA256: simulatorHash, State: state, Trace: execution.Trace, FaultSummary: fault, RepairSummary: repair, CreatedAt: now}
	record.HandoffMarkdown = handoff(record)
	created, err := s.repo.Create(ctx, record)
	if err != nil {
		return Rehearsal{}, err
	}
	return created, nil
}

func nativeWorkflowRequiresCompanionRoute(workflow Workflow) bool {
	for _, node := range workflow.Nodes {
		if node.Type == "n8n-nodes-base.httpRequest" && node.Request != nil {
			return true
		}
	}
	return false
}

func nativeWorkflowHasCompanionRouteHit(workflow Workflow, hits HTTPMockRouteHitsReceipt) bool {
	for _, node := range workflow.Nodes {
		if node.Type != "n8n-nodes-base.httpRequest" || node.Request == nil {
			continue
		}
		for _, hit := range hits.Routes {
			if hit.Count > 0 && strings.EqualFold(hit.Host, node.Request.Host) && strings.EqualFold(hit.Method, node.Request.Method) && hit.Path == node.Request.URL {
				return true
			}
		}
	}
	return false
}

func hasOutputAssertion(trace []TraceEvent) bool {
	for _, event := range trace {
		if event.Operation == "assert output path exists" {
			return true
		}
	}
	return false
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

func containsTrigger(values []TriggerKind, expected TriggerKind) bool {
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
		handoffSandboxEvidence(r.Trace) +
		"- Run state: `" + string(r.State) + "`\n" +
		"- Fault: " + r.FaultSummary + "\n" +
		"- Repair: " + r.RepairSummary + "\n\n" +
		"Human review is required; this rehearsal is synthetic evidence and does not contact a client service."
}

func handoffSandboxEvidence(trace []TraceEvent) string {
	evidence := lifecycleEvidenceFromTrace(trace)
	if evidence.SandboxID == "" {
		return ""
	}
	handoff := "- Workspace Sandbox task: `" + evidence.SandboxID + "` (owner `" + evidence.Owner + "`, scope SHA-256 `" + evidence.ScopeSHA256 + "`)\n" +
		"- Synthetic companion profile SHA-256: `" + evidence.ProfileSHA256 + "` (" + itoa(uint32(evidence.OperationCount)) + " operations; synthetic targets: `" + fmt.Sprint(evidence.SyntheticTargets) + "`)\n" +
		"- Sandbox isolation evidence: " + evidence.IsolationClaim + "\n" +
		"- Sandbox teardown: `" + evidence.TeardownStatus + "` at `" + evidence.CleanupConfirmedAt.UTC().Format(time.RFC3339) + "` " + evidence.TeardownError + "\n"
	if evidence.N8NExecution != nil {
		handoff += "- Native n8n runtime: `" + evidence.N8NRuntimeStatus + "`; execution `" + evidence.N8NExecution.Status + "` (" + itoa(uint32(evidence.N8NExecution.NodeCount)) + " nodes, " + itoa(uint32(evidence.N8NExecution.RunCount)) + " run(s), " + itoa(uint32(evidence.N8NExecution.ItemCount)) + " item(s))\n"
	}
	if evidence.RouteHitsStatus != "" {
		hits, routes := uint64(0), 0
		if evidence.RouteHits != nil {
			hits, routes = evidence.RouteHits.TotalHits, len(evidence.RouteHits.Routes)
		}
		handoff += "- Companion route-hit receipt: `" + evidence.RouteHitsStatus + "` (" + fmt.Sprint(hits) + " hit(s) across " + itoa(uint32(routes)) + " route(s)) " + evidence.RouteHitsError + "\n"
	}
	return handoff
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
