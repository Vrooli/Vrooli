package opsrunner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opscatalog"
)

// RunOwnerIndex attributes an operation execution to its owning
// (target, operation, workflow, execution, mode revision) so the evidence ledger
// can map any Agent Manager activity spawned through the runner back to the
// operation that caused it. Production wires this to the operating-mode run-owner
// index / evidence service; tests use an in-memory recorder.
type RunOwnerIndex interface {
	IndexRunOwner(ctx context.Context, ref EvidenceRef) error
}

// Runner is the single operation-runner chokepoint. Construct it with New and
// call Invoke. It holds only generic collaborators — a catalog, a binding
// resolver, a mode preparer, an execution driver, durable stores, a dispatcher —
// and contains NO branch keyed to any named mode, phase, or target kind.
type Runner struct {
	catalog    *opscatalog.Catalog
	resolver   *BindingResolver
	preparer   ModePreparer
	driver     ExecutionDriver
	starter    RunStarter
	repo       *WorkflowRepo
	executions *ExecutionStore
	dispatcher *Dispatcher
	runOwners  RunOwnerIndex
	now        func() time.Time
	newID      func() string
}

// Config configures a Runner. The zero values for now/newID default to time.Now
// and a random hex id generator.
type Config struct {
	Catalog  *opscatalog.Catalog
	Resolver *BindingResolver
	Preparer ModePreparer
	Driver   ExecutionDriver
	// Starter, when non-nil, is the live dispatch seam: Invoke of a non-simulated
	// operation starts the run through it (spawning the agent via the operating-
	// mode engine's agentactivity chokepoint) and returns immediately, leaving the
	// operation running until CommitResult delivers its outcome. When nil, Invoke
	// always drives synchronously through Driver (simulation and tests).
	Starter    RunStarter
	Repo       *WorkflowRepo
	Executions *ExecutionStore
	Dispatcher *Dispatcher
	RunOwners  RunOwnerIndex
	Now        func() time.Time
	NewID      func() string
}

// New constructs a Runner, failing if a required collaborator is missing.
func New(cfg Config) (*Runner, error) {
	if cfg.Catalog == nil || cfg.Resolver == nil || cfg.Preparer == nil || cfg.Driver == nil || cfg.Repo == nil || cfg.Executions == nil || cfg.Dispatcher == nil {
		return nil, fmt.Errorf("opsrunner.New: catalog, resolver, preparer, driver, repo, executions, and dispatcher are all required")
	}
	r := &Runner{
		catalog: cfg.Catalog, resolver: cfg.Resolver, preparer: cfg.Preparer,
		driver: cfg.Driver, starter: cfg.Starter, repo: cfg.Repo, executions: cfg.Executions,
		dispatcher: cfg.Dispatcher, runOwners: cfg.RunOwners,
		now: cfg.Now, newID: cfg.NewID,
	}
	if r.now == nil {
		r.now = time.Now
	}
	if r.newID == nil {
		r.newID = randomID
	}
	return r, nil
}

// Invoke runs one operation against one target end to end. The sequence is
// fail-closed at every gate: an operation the catalog does not declare, a target
// the operation is incompatible with, or a binding that does not resolve all
// abort before any state is written or any agent is spawned.
func (r *Runner) Invoke(ctx context.Context, req InvokeRequest) (OperationResult, error) {
	// 1. The operation must be declared, and compatible with the target kind.
	lc, ok := r.catalog.Contract(req.Operation, req.OperationVersion)
	if !ok {
		return OperationResult{}, fmt.Errorf("%w: %s@%s", ErrUnknownOperation, req.Operation, req.OperationVersion)
	}
	contract := lc.Contract
	if err := agentops.CheckOperationTargetCompatibility(req.Operation, contract.TargetRequirements.Capabilities, req.Target.Kind); err != nil {
		return OperationResult{}, fmt.Errorf("%w: %v", ErrIncompatibleTarget, err)
	}

	// 2. Resolve the binding (fail-closed precedence + provenance).
	res, err := r.resolver.Resolve(ctx, req)
	if err != nil {
		return OperationResult{}, err
	}

	// 3. Idempotent short-circuit: a prior Invoke with the same key returns its
	// recorded result without starting a second run or firing a second action.
	workflow, err := r.repo.CreateOrLoad(req.Target.Kind, req.Target.ID)
	if err != nil {
		return OperationResult{}, err
	}
	if req.IdempotencyKey != "" {
		if prior, replayed := FindOperationByIdempotencyKey(workflow, req.IdempotencyKey); replayed {
			return r.replayResult(req, workflow, prior)
		}
	}

	// 4. Compile + pin the bound mode and validate caller inputs.
	prepared, err := r.preparer.Prepare(ctx, PrepareRequest{
		Target: req.Target, Operation: req.Operation,
		Mode: res.Binding.Mode, ModeRevision: res.Binding.ModeRevision,
		CallerInputs: req.CallerInputs,
	})
	if err != nil {
		return OperationResult{}, fmt.Errorf("prepare execution: %w", err)
	}

	// 5. Pin the immutable provenance with canonical digests.
	executionID := r.newID()
	idempotencyKey := req.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = "op-" + executionID
	}
	prov, err := r.buildProvenance(req, contract, res, prepared, workflow.InstanceID)
	if err != nil {
		return OperationResult{}, err
	}
	provDigest, err := agentops.DigestOf(prov)
	if err != nil {
		return OperationResult{}, err
	}

	// 6. Persist the self-contained execution snapshot (reproducible after source
	// edits). Written before the run so a crash mid-run leaves a resolvable
	// record.
	snap := ExecutionSnapshot{
		Provenance: prov, CompiledMode: prepared.CompiledMode,
		PromptCatalog: prepared.PromptCatalog, EffectiveInputs: prepared.EffectiveInputs,
		PolicyID:   res.PolicyID,
		RecordedAt: r.now().UTC().Format(time.RFC3339Nano),
	}
	if err := r.executions.SaveExecution(req.Target.Kind, req.Target.ID, executionID, snap); err != nil {
		return OperationResult{}, fmt.Errorf("persist execution snapshot: %w", err)
	}

	// 7. Correlate a running operation record under the workflow (CAS).
	workflow, err = r.appendRunningOperation(workflow, req.Operation, executionID, idempotencyKey, provDigest)
	if err != nil {
		return OperationResult{}, err
	}

	// 8. Attribute the run to its owner for the evidence ledger.
	evidenceRef := EvidenceRef{
		RunID: executionID, WorkflowID: workflow.InstanceID, ExecutionID: executionID,
		Operation: req.Operation, Mode: res.Binding.Mode, ModeRevision: res.Binding.ModeRevision,
	}
	if r.runOwners != nil {
		if err := r.runOwners.IndexRunOwner(ctx, evidenceRef); err != nil {
			return OperationResult{}, fmt.Errorf("index run owner: %w", err)
		}
	}

	// 9. Live path: START the run (non-blocking) and return a running handle. The
	// terminal outcome and transition are deferred to CommitResult when the round
	// is delivered. Simulation/test path: DRIVE synchronously to a terminal
	// outcome and fire the transition inline.
	baseResult := OperationResult{
		WorkflowInstanceID: workflow.InstanceID, ExecutionID: executionID,
		Provenance: prov, ProvenanceDigest: provDigest,
		WorkflowState: workflow.State, EvidenceRefs: []EvidenceRef{evidenceRef},
	}
	if r.starter != nil && !req.Simulate {
		start, err := r.starter.Start(ctx, prepared, RunHandle{
			ExecutionID: executionID, WorkflowID: workflow.InstanceID, Target: req.Target,
			Operation: req.Operation, Preset: req.SimulationPreset, Simulate: false,
		})
		if err != nil {
			// The live start failed after the operation was recorded running. Reap it
			// so it does not linger "running" (which the refresh driver would keep
			// polling for a run that was never created), then propagate the error.
			r.reapFailedStart(workflow, executionID)
			return OperationResult{}, fmt.Errorf("start execution: %w", err)
		}
		if start.RunID == "" {
			// A live start that yields no run id started nothing trackable — the
			// operation could never be delivered to CommitResult. Fail closed and reap
			// rather than returning a phantom running operation.
			r.reapFailedStart(workflow, executionID)
			return OperationResult{}, fmt.Errorf("%w: %s@%s", ErrNoRunID, req.Operation, req.OperationVersion)
		}
		{
			// Persist the live run id on the operation record so a delivered round
			// (which carries only its agent run id) can be correlated back to this
			// execution when CommitResult finalizes it.
			workflow, err = r.recordRunID(workflow, executionID, start.RunID)
			if err != nil {
				return OperationResult{}, err
			}
			baseResult.WorkflowState = workflow.State
			if r.runOwners != nil {
				evidenceRef.RunID = start.RunID
				if err := r.runOwners.IndexRunOwner(ctx, evidenceRef); err != nil {
					return OperationResult{}, fmt.Errorf("index run owner: %w", err)
				}
				baseResult.EvidenceRefs = []EvidenceRef{evidenceRef}
			}
		}
		baseResult.StartHandle = &start
		return baseResult, nil
	}

	outcome, err := r.driver.Drive(ctx, prepared, RunHandle{
		ExecutionID: executionID, WorkflowID: workflow.InstanceID, Target: req.Target,
		Operation: req.Operation, Preset: req.SimulationPreset, Simulate: req.Simulate,
	})
	if err != nil {
		return OperationResult{}, fmt.Errorf("drive execution: %w", err)
	}
	if outcome.RunID != "" {
		evidenceRef.RunID = outcome.RunID
		if r.runOwners != nil {
			if err := r.runOwners.IndexRunOwner(ctx, evidenceRef); err != nil {
				return OperationResult{}, fmt.Errorf("index run owner: %w", err)
			}
		}
	}

	// 10. Record the terminal outcome on the operation record + snapshot (CAS).
	workflow, err = r.recordOutcome(workflow, executionID, outcome)
	if err != nil {
		return OperationResult{}, err
	}
	snap.Outcome = outcome.Outcome
	snap.Result = outcome.Result
	if err := r.executions.SaveExecution(req.Target.Kind, req.Target.ID, executionID, snap); err != nil {
		return OperationResult{}, fmt.Errorf("update execution snapshot: %w", err)
	}

	// 11. Evaluate + dispatch the domain transition (closed action registry).
	result := baseResult
	result.Outcome = outcome.Outcome
	result.Disposition = outcome.Disposition
	result.Result = outcome.Result
	result.WorkflowState = workflow.State
	result.EvidenceRefs = []EvidenceRef{evidenceRef}
	if res.PolicyID != "" {
		acted, err := r.applyTransition(ctx, req.Target, workflow, res.PolicyID, outcome.Outcome, req.Operation, executionID, outcome.Result)
		if err != nil {
			return OperationResult{}, err
		}
		result.Action = acted.action
		result.WorkflowState = acted.state
	}
	return result, nil
}

type transitionResult struct {
	action agentops.ActionName
	state  agentops.WorkflowState
}

// applyTransition evaluates the pinned policy against the workflow state + the
// execution outcome and dispatches the selected domain action, if any.
func (r *Runner) applyTransition(ctx context.Context, target TargetRef, w agentops.WorkflowInstance, policyID, outcome string, op agentops.OperationID, executionID string, result json.RawMessage) (transitionResult, error) {
	lp, ok := r.catalog.Policy(policyID)
	if !ok {
		return transitionResult{state: w.State}, nil
	}
	sel, err := EvaluateTransition(lp.Policy, w.State, op, outcome)
	if err != nil {
		if err == ErrNoTransition {
			return transitionResult{state: w.State}, nil
		}
		return transitionResult{}, err
	}
	actionKey := "act-" + executionID
	dr, err := r.dispatcher.Dispatch(ctx, target, w, sel, outcome, op, actionKey, DispatchDelivery{ExecutionID: executionID, Result: result})
	if err != nil {
		return transitionResult{}, err
	}
	return transitionResult{action: sel.Action, state: dr.State}, nil
}

// buildProvenance assembles and validates the immutable provenance record.
func (r *Runner) buildProvenance(req InvokeRequest, contract agentops.OperationContract, res Resolution, prep Prepared, workflowID string) (agentops.ExecutionProvenance, error) {
	compiledDigest, err := agentops.CanonicalDigest(prep.CompiledMode)
	if err != nil {
		return agentops.ExecutionProvenance{}, fmt.Errorf("digest compiled mode: %w", err)
	}
	promptDigest, err := agentops.CanonicalDigest(prep.PromptCatalog)
	if err != nil {
		return agentops.ExecutionProvenance{}, fmt.Errorf("digest prompt catalog: %w", err)
	}
	inputDigest, err := agentops.CanonicalDigest(prep.EffectiveInputs)
	if err != nil {
		return agentops.ExecutionProvenance{}, fmt.Errorf("digest caller inputs: %w", err)
	}
	prov := agentops.ExecutionProvenance{
		Kind:                  "agentops-execution-provenance",
		Operation:             req.Operation,
		OperationVersion:      contract.Version,
		Binding:               provenanceBinding(res.Binding),
		Mode:                  res.Binding.Mode,
		ModeRevision:          res.Binding.ModeRevision,
		CompiledModeDigest:    compiledDigest,
		PromptCatalogRevision: nonEmpty(prep.PromptCatalogRevision, "none"),
		PromptCatalogDigest:   promptDigest,
		Target:                agentops.ProvenanceTarget{Kind: req.Target.Kind, ID: req.Target.ID},
		CallerInputDigest:     inputDigest,
		PolicyRevision:        nonEmpty(res.PolicyRevision, "no-policy"),
		WorkflowInstanceID:    workflowID,
	}
	raw, err := json.Marshal(prov)
	if err != nil {
		return agentops.ExecutionProvenance{}, err
	}
	if err := agentops.ValidateProvenance(raw); err != nil {
		return agentops.ExecutionProvenance{}, fmt.Errorf("provenance incomplete: %w", err)
	}
	return prov, nil
}

// appendRunningOperation adds a running operation record via compare-and-swap.
func (r *Runner) appendRunningOperation(w agentops.WorkflowInstance, op agentops.OperationID, executionID, idempotencyKey, provDigest string) (agentops.WorkflowInstance, error) {
	prev := w.Version
	next := cloneWorkflow(w)
	next.Operations = append(next.Operations, agentops.OperationExecutionRecord{
		Operation: op, ExecutionID: executionID, IdempotencyKey: idempotencyKey,
		ProvenanceDigest: provDigest, State: "running",
	})
	next.IdempotencyKeys = appendUnique(next.IdempotencyKeys, idempotencyKey)
	if next.State == agentops.WorkflowOpen {
		next.State = agentops.WorkflowRunning
	}
	next.Version = prev + 1
	if err := r.repo.Commit(prev, next); err != nil {
		return agentops.WorkflowInstance{}, err
	}
	return next, nil
}

// recordRunID persists the live agent-run id on a running operation record via
// compare-and-swap, so a round delivered under that run id resolves back to this
// execution in CommitResult.
func (r *Runner) recordRunID(w agentops.WorkflowInstance, executionID, runID string) (agentops.WorkflowInstance, error) {
	prev := w.Version
	next := cloneWorkflow(w)
	for i := range next.Operations {
		if next.Operations[i].ExecutionID == executionID {
			next.Operations[i].RunID = runID
		}
	}
	next.Version = prev + 1
	if err := r.repo.Commit(prev, next); err != nil {
		return agentops.WorkflowInstance{}, err
	}
	return next, nil
}

// recordOutcome marks the operation record terminal via compare-and-swap.
func (r *Runner) recordOutcome(w agentops.WorkflowInstance, executionID string, outcome ExecutionOutcome) (agentops.WorkflowInstance, error) {
	prev := w.Version
	next := cloneWorkflow(w)
	for i := range next.Operations {
		if next.Operations[i].ExecutionID == executionID {
			next.Operations[i].State = recordStateFor(outcome.Disposition)
			next.Operations[i].Outcome = outcome.Outcome
		}
	}
	next.Version = prev + 1
	if err := r.repo.Commit(prev, next); err != nil {
		return agentops.WorkflowInstance{}, err
	}
	return next, nil
}

// reapFailedStart marks a running operation failed after its live start failed
// or yielded no run id, so a start that never produced a trackable run does not
// leave a phantom running record. Best-effort: a compare-and-swap conflict is
// ignored (a concurrent path is already finalizing the operation).
func (r *Runner) reapFailedStart(w agentops.WorkflowInstance, executionID string) {
	_, _ = r.recordOutcome(w, executionID, ExecutionOutcome{Disposition: "failed"})
}

// replayResult reconstructs the typed result of a prior idempotent execution.
func (r *Runner) replayResult(req InvokeRequest, w agentops.WorkflowInstance, prior agentops.OperationExecutionRecord) (OperationResult, error) {
	snap, found, err := r.executions.Load(req.Target.Kind, req.Target.ID, prior.ExecutionID)
	if err != nil {
		return OperationResult{}, err
	}
	res := OperationResult{
		WorkflowInstanceID: w.InstanceID, ExecutionID: prior.ExecutionID,
		Outcome: prior.Outcome, WorkflowState: w.State, Replayed: true,
	}
	if found {
		res.Provenance = snap.Provenance
		res.ProvenanceDigest = prior.ProvenanceDigest
		res.Result = snap.Result
	}
	return res, nil
}

// recordStateFor maps a contract disposition onto the operation-record state
// enum (running|completed|canceled|failed|needs-attention).
func recordStateFor(d Disposition) string {
	switch d {
	case "success", "continue":
		return "completed"
	case "failed":
		return "failed"
	case "canceled":
		return "canceled"
	default:
		return "needs-attention"
	}
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func randomID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("exec-%d", time.Now().UnixNano())
	}
	return "exec-" + hex.EncodeToString(b)
}
