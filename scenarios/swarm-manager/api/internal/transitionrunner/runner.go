// Package transitionrunner executes declared workflow transitions without
// importing any subject domain package.
package transitionrunner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitions"
	"swarm-manager/internal/workflowcontract"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	InputBuilder func(context.Context, string) (Snapshot, error)
	ApplyFunc    func(context.Context, string, Outcome) error
	StartGuard   func(context.Context, transitions.Definition, string) error
	StartFunc    func(context.Context, StartInvocation) (agentmanager.WorkflowStart, error)
)

// Snapshot is the immutable subject projection a domain builds for one
// transition. The runner rebuilds it at apply time to detect edits made while
// the workflow was running, so a builder must be a pure function of durable
// subject state: anything an operator supplies at start has to be persisted on
// the subject first, or the rebuild cannot reproduce it.
type Snapshot struct {
	Input          *structpb.Value
	EntityVersion  string
	FrontierDigest string
	ApprovalDigest string
	Grant          *workflowcontract.Grant
}

// StartInvocation is the composition-owned seam for a transition whose
// subject must reserve and bind authority before the owner transport is called.
// The ordinary path still calls Agent Manager directly; specialized starters
// receive the same immutable snapshot and declared locator without taking over
// correlation persistence.
type StartInvocation struct {
	Definition transitions.Definition
	SubjectRef string
	Snapshot   Snapshot
	Invocation agentmanager.Invocation
}

// Outcome is the transport-neutral terminal result delivered to a subject's
// registered mutation function.
type Outcome struct {
	ExecutionID    string
	TransitionKey  string
	SubjectRef     string
	EntityVersion  string
	FrontierDigest string
	Name           string
	TerminalCode   string
	BudgetName     string
	WorkflowDigest string
	ApprovalDigest string
	GrantDigest    string
	Usage          *workflowcontract.Usage
	Result         json.RawMessage
	Attempts       []transitionrun.Attempt
}

// PreparedInput is an immutable subject snapshot prepared by a domain when
// operator-supplied inputs are part of a transition. Ordinary transitions use
// their registered InputBuilder through Start; prepared starts preserve the
// same lifecycle, journal, guards, and idempotency semantics.
type PreparedInput struct {
	Input               *structpb.Value
	EntityVersion       string
	FrontierDigest      string
	FirstRunNodeID      string
	Activity            *Activity
	WorkflowKeyOverride string
	ApprovalDigest      string
	Grant               *workflowcontract.Grant
}

// Activity is domain-neutral launch attribution. The runner converts it at
// the Agent Manager boundary so subject packages do not import that client.
type Activity struct{ OwnerType, OwnerKind, OwnerName, OwnerTitle, Purpose string }

// Registrar is the composition-time surface for subject adapters.
type Registrar interface {
	RegisterApply(string, ApplyFunc)
	RegisterInput(string, InputBuilder)
}

type Runner struct {
	registry   transitions.Registry
	workflows  agentmanager.WorkflowInvoker
	store      transitionrun.Store
	guard      StartGuard
	apply      map[string]ApplyFunc
	input      map[string]InputBuilder
	start      map[string]StartFunc
	applyLocks sync.Map // map[string]*sync.Mutex, keyed by execution id
}

func New(registry transitions.Registry, workflows agentmanager.WorkflowInvoker, store transitionrun.Store, guard StartGuard) *Runner {
	return &Runner{registry: registry, workflows: workflows, store: store, guard: guard, apply: map[string]ApplyFunc{}, input: map[string]InputBuilder{}, start: map[string]StartFunc{}}
}
func (r *Runner) RegisterApply(action string, fn ApplyFunc) { r.apply[strings.TrimSpace(action)] = fn }
func (r *Runner) RegisterInput(key string, builder InputBuilder) {
	r.input[strings.TrimSpace(key)] = builder
}

func (r *Runner) RegisterStart(key string, starter StartFunc) {
	r.start[strings.TrimSpace(key)] = starter
}
func (r *Runner) Counts() (int, int) { return len(r.apply), len(r.input) }

// HasInput reports whether a transition has a registered input builder.
func (r *Runner) HasInput(transitionKey string) bool {
	return r.input[strings.TrimSpace(transitionKey)] != nil
}

// HasStart reports whether a transition has a registered domain admission
// starter in addition to its input projection.
func (r *Runner) HasStart(transitionKey string) bool {
	return r.start[strings.TrimSpace(transitionKey)] != nil
}

// DeclaredOutcomes returns the current registry contract for a transition.
// Reconciliation code uses this for explicit migrations of durable
// correlations created under an older registry definition.
func (r *Runner) DeclaredOutcomes(transitionKey string) ([]string, error) {
	definition, err := r.workflowDefinition(transitionKey)
	if err != nil {
		return nil, err
	}
	return append([]string(nil), definition.TerminalOutcomes...), nil
}

// BuildInput reprojects a subject through its registered builder without
// starting anything. It exists so callers can check that a transition's
// snapshot still satisfies its declared input contract — the property that
// distinguishes a real builder from one that only returns a stored version.
func (r *Runner) BuildInput(ctx context.Context, transitionKey, subjectRef string) (Snapshot, error) {
	builder := r.input[strings.TrimSpace(transitionKey)]
	if builder == nil {
		return Snapshot{}, fmt.Errorf("transition %q has no input builder", transitionKey)
	}
	return builder(ctx, subjectRef)
}

// ListUnapplied exposes durable crash-recovery candidates to the composition
// owned sweeper. It deliberately returns correlations, never domain records.
func (r *Runner) ListUnapplied() ([]transitionrun.Correlation, error) {
	return r.store.ListUnapplied()
}

// GetCorrelation exposes a durable correlation for deprecated subject routes
// that need to report replay semantics without owning lifecycle state.
func (r *Runner) GetCorrelation(executionID string) (transitionrun.Correlation, error) {
	return r.store.Get(executionID)
}

// UpdateCorrelation applies a narrow, serialized mutation to the shared
// lifecycle journal. Subject domains use this only for operator facts that the
// correlation explicitly owns (for example approval attribution); they must
// never copy those facts back onto their own records.
func (r *Runner) UpdateCorrelation(executionID string, mutate func(*transitionrun.Correlation) error) (transitionrun.Correlation, error) {
	if mutate == nil {
		return transitionrun.Correlation{}, errors.New("transition correlation mutation is required")
	}
	lock := r.applyLock(executionID)
	lock.Lock()
	defer lock.Unlock()
	correlation, err := r.store.Get(executionID)
	if err != nil {
		return transitionrun.Correlation{}, err
	}
	if err := mutate(&correlation); err != nil {
		return transitionrun.Correlation{}, err
	}
	if err := r.store.Put(correlation); err != nil {
		return transitionrun.Correlation{}, err
	}
	return correlation, nil
}

// FindCorrelation projects lifecycle state by the transition subject, so
// domains need not persist a second copy of the correlation execution ID.
func (r *Runner) FindCorrelation(transitionKey, subjectRef string) (transitionrun.Correlation, error) {
	return r.store.FindBySubject(strings.TrimSpace(transitionKey), strings.TrimSpace(subjectRef))
}

func (r *Runner) GetDispatchIntent(transitionKey, subjectRef string) (transitionrun.DispatchIntent, error) {
	store, ok := r.store.(transitionrun.DispatchStore)
	if !ok {
		return transitionrun.DispatchIntent{}, fmt.Errorf("durable dispatch inspection is unavailable")
	}
	return store.GetDispatch(transitionKey, subjectRef)
}

func workflowInputDigest(input *structpb.Value) string {
	if input == nil {
		return ""
	}
	data, err := json.Marshal(input.AsInterface())
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ResolveDispatch performs owner reads only. Even a proven current absence
// leaves the pre-RPC intent unresolved: the original submit may still race.
func (r *Runner) ResolveDispatch(ctx context.Context, transitionKey, subjectRef string) (transitionrun.Correlation, error) {
	intent, err := r.GetDispatchIntent(transitionKey, subjectRef)
	if err != nil {
		return transitionrun.Correlation{}, err
	}
	reader, ok := r.workflows.(interface {
		InspectWorkflowStart(context.Context, string, string) (*domainpb.WorkflowExecution, error)
	})
	if !ok {
		return transitionrun.Correlation{}, fmt.Errorf("read-only original workflow lookup is unavailable")
	}
	execution, err := reader.InspectWorkflowStart(ctx, intent.Correlation.WorkflowKey, intent.IdempotencyKey)
	if err != nil {
		return transitionrun.Correlation{}, fmt.Errorf("original workflow submission remains unresolved: %w", err)
	}
	if execution == nil || execution.Id == "" || execution.DefinitionDigest == "" || execution.Owner != intent.Owner || execution.WorkflowKey != intent.Correlation.WorkflowKey || execution.IdempotencyKey != intent.IdempotencyKey || workflowInputDigest(execution.Input) != intent.InputDigest || execution.ApprovalDigest != intent.ApprovalDigest || execution.GrantDigest != intent.GrantDigest {
		return transitionrun.Correlation{}, fmt.Errorf("recovered workflow does not match the retained owner, consumer input, approval, or grant binding")
	}
	grant, err := toAgentGrant(intent.Grant)
	if err != nil || !proto.Equal(grant, execution.EngagementGrant) {
		return transitionrun.Correlation{}, fmt.Errorf("recovered workflow allowance differs from the retained grant")
	}
	correlation := intent.Correlation
	correlation.ExecutionID, correlation.DefinitionDigest = execution.Id, execution.DefinitionDigest
	return r.bindDispatch(intent, correlation)
}

func (r *Runner) bindDispatch(intent transitionrun.DispatchIntent, correlation transitionrun.Correlation) (transitionrun.Correlation, error) {
	if existing, err := r.store.Get(correlation.ExecutionID); err == nil {
		if existing.TransitionKey != correlation.TransitionKey || existing.SubjectRef != correlation.SubjectRef || existing.EntityVersion != correlation.EntityVersion || existing.FrontierDigest != correlation.FrontierDigest || existing.WorkflowKey != correlation.WorkflowKey || existing.DefinitionDigest != correlation.DefinitionDigest {
			return transitionrun.Correlation{}, fmt.Errorf("recovered owner conflicts with its retained transition correlation")
		}
		correlation = existing
	} else if !errors.Is(err, os.ErrNotExist) {
		return transitionrun.Correlation{}, err
	} else if err := r.store.Put(correlation); err != nil {
		return transitionrun.Correlation{}, err
	}
	intent.Correlation = correlation
	intent.State, intent.LastError = "acknowledged", ""
	if err := r.store.(transitionrun.DispatchStore).PutDispatch(intent); err != nil {
		return transitionrun.Correlation{}, err
	}
	return correlation, nil
}

// RecoverDispatches joins lost acknowledgements to their original owner. It
// never retries creation and leaves absent or unreadable intents for recovery.
func (r *Runner) RecoverDispatches(ctx context.Context) error {
	store, ok := r.store.(transitionrun.DispatchStore)
	if !ok {
		return nil
	}
	intents, err := store.ListUnresolvedDispatches()
	if err != nil {
		return err
	}
	for _, intent := range intents {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if _, err := r.ResolveDispatch(ctx, intent.Correlation.TransitionKey, intent.Correlation.SubjectRef); err != nil {
			intent.State, intent.LastError = "unknown", err.Error()
			if err := store.PutDispatch(intent); err != nil {
				return err
			}
		}
	}
	return nil
}

// Cancel stops a runner-owned workflow execution. Subjects route cancellation
// here rather than holding their own workflow client, so the transport stays in
// one place and a subject cannot end up with a nil client that silently reports
// "cancel is not supported".
func (r *Runner) Cancel(ctx context.Context, executionID, idempotencyKey, reason string) error {
	canceler, ok := r.workflows.(interface {
		CancelWorkflow(context.Context, string, string, string) error
	})
	if !ok {
		return fmt.Errorf("workflow cancellation is not supported by the configured transport")
	}
	return canceler.CancelWorkflow(ctx, executionID, idempotencyKey, reason)
}

// CloseUnapplied marks a correlation terminal without applying a workflow
// result. Cancellation ends the engagement before any terminal outcome exists,
// and the correlation has to stop being a recovery candidate — otherwise the
// sweeper retries a cancelled execution forever.
func (r *Runner) CloseUnapplied(executionID, outcome string) error {
	lock := r.applyLock(executionID)
	lock.Lock()
	defer lock.Unlock()

	correlation, err := r.store.Get(executionID)
	if err != nil {
		return err
	}
	if correlation.ApplyState == transitionrun.ApplyStateComplete {
		return nil
	}
	correlation.ApplyState = transitionrun.ApplyStateComplete
	correlation.Outcome = outcome
	correlation.AppliedTime = time.Now().UTC().Format(time.RFC3339Nano)
	return r.store.Put(correlation)
}

// Signal forwards a runner-owned workflow signal without exposing the
// Agent-Manager client to subject packages.
func (r *Runner) Signal(ctx context.Context, executionID, signal string, payload *structpb.Value, idempotencyKey string) error {
	signaler, ok := r.workflows.(interface {
		SignalWorkflow(context.Context, string, string, *structpb.Value, string) error
	})
	if !ok {
		return fmt.Errorf("workflow signaling is not supported")
	}
	return signaler.SignalWorkflow(ctx, executionID, signal, payload, idempotencyKey)
}

func (r *Runner) Start(ctx context.Context, transitionKey, subjectRef string) (transitionrun.Correlation, error) {
	return r.StartWith(ctx, transitionKey, subjectRef, PreparedInput{})
}

// StartWith starts a declared transition while allowing a caller to attach
// transport metadata such as activity attribution. Registry selection,
// snapshots, idempotency, and correlation persistence remain runner-owned.
func (r *Runner) StartWith(ctx context.Context, transitionKey, subjectRef string, options PreparedInput) (transitionrun.Correlation, error) {
	definition, err := r.workflowDefinition(transitionKey)
	if err != nil {
		return transitionrun.Correlation{}, err
	}
	builder := r.input[definition.Key]
	if builder == nil {
		return transitionrun.Correlation{}, fmt.Errorf("transition %q has no input builder", definition.Key)
	}
	snapshot, err := builder(ctx, subjectRef)
	if err != nil {
		return transitionrun.Correlation{}, fmt.Errorf("build transition input: %w", err)
	}
	options.Input, options.EntityVersion, options.FrontierDigest = snapshot.Input, snapshot.EntityVersion, snapshot.FrontierDigest
	options.ApprovalDigest, options.Grant = snapshot.ApprovalDigest, snapshot.Grant
	return r.startPrepared(ctx, definition, subjectRef, options)
}

// StartPrepared starts a workflow using a domain-prepared immutable snapshot.
// It is intentionally narrow: callers cannot override registry-owned workflow
// selection or idempotency construction.
//
// Prefer StartWith. This exists only for a transition whose start carries an
// operator input that is not persisted on the subject — today that is
// plan.repair's maxRepairAttempts. A caller MUST derive EntityVersion exactly
// as the registered builder does, because Apply rebuilds through the builder
// and rejects the completion when the two disagree. A builder that instead
// echoes the stored version would hide that disagreement and silently disable
// the staleness guard, which is the defect this contract exists to prevent.
func (r *Runner) StartPrepared(ctx context.Context, transitionKey, subjectRef string, prepared PreparedInput) (transitionrun.Correlation, error) {
	definition, err := r.workflowDefinition(transitionKey)
	if err != nil {
		return transitionrun.Correlation{}, err
	}
	return r.startPrepared(ctx, definition, subjectRef, prepared)
}

func (r *Runner) startPrepared(ctx context.Context, definition transitions.Definition, subjectRef string, prepared PreparedInput) (transitionrun.Correlation, error) {
	if r.guard != nil {
		if err := r.guard(ctx, definition, subjectRef); err != nil {
			return transitionrun.Correlation{}, err
		}
	}
	if prepared.Input == nil || strings.TrimSpace(prepared.EntityVersion) == "" {
		return transitionrun.Correlation{}, fmt.Errorf("transition %q input builder returned incomplete snapshot", definition.Key)
	}
	var activity *agentmanager.WorkflowActivity
	if prepared.Activity != nil {
		activity = &agentmanager.WorkflowActivity{OwnerType: prepared.Activity.OwnerType, OwnerKind: prepared.Activity.OwnerKind, OwnerName: prepared.Activity.OwnerName, OwnerTitle: prepared.Activity.OwnerTitle, Purpose: prepared.Activity.Purpose}
	}
	workflow := *definition.Workflow
	if strings.TrimSpace(prepared.WorkflowKeyOverride) != "" {
		workflow.Key = strings.TrimSpace(prepared.WorkflowKeyOverride)
	}
	idempotencyKey := definition.Key + "/" + subjectRef + "/" + prepared.EntityVersion
	if strings.TrimSpace(prepared.WorkflowKeyOverride) != "" {
		idempotencyKey += "/" + workflow.Key
	}
	invocation := agentmanager.Invocation{Owner: workflow.Owner, WorkflowKey: workflow.Key, Input: prepared.Input, IdempotencyKey: idempotencyKey, FirstRunNodeID: prepared.FirstRunNodeID, Activity: activity}
	if prepared.Grant != nil {
		grant, err := toAgentGrant(prepared.Grant)
		if err != nil {
			return transitionrun.Correlation{}, err
		}
		invocation.EngagementGrant, invocation.ApprovalDigest, invocation.GrantDigest = grant, prepared.ApprovalDigest, workflowcontract.GrantDigest(prepared.Grant)
	}
	var dispatch *transitionrun.DispatchIntent
	if definition.Key == "plan.execute" && r.start[definition.Key] == nil {
		lock := r.applyLock("dispatch/" + definition.Key + "/" + subjectRef)
		lock.Lock()
		defer lock.Unlock()
		store, ok := r.store.(transitionrun.DispatchStore)
		if !ok {
			return transitionrun.Correlation{}, fmt.Errorf("ordinary execution requires durable dispatch intent storage")
		}
		intent := transitionrun.DispatchIntent{State: "submitting", Correlation: transitionrun.Correlation{TransitionKey: definition.Key, SubjectKind: definition.Subject, SubjectRef: subjectRef, WorkflowKey: workflow.Key, EntityVersion: prepared.EntityVersion, FrontierDigest: prepared.FrontierDigest, ApplyState: transitionrun.ApplyStateClaimed, DeclaredOutcomes: append([]string(nil), definition.TerminalOutcomes...)}, Owner: workflow.Owner, IdempotencyKey: idempotencyKey, InputDigest: workflowInputDigest(prepared.Input), ApprovalDigest: invocation.ApprovalDigest, GrantDigest: invocation.GrantDigest, Grant: prepared.Grant, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		if existing, err := store.GetDispatch(definition.Key, subjectRef); err == nil {
			if existing.IdempotencyKey != intent.IdempotencyKey || existing.InputDigest != intent.InputDigest || existing.Owner != intent.Owner || existing.Correlation.WorkflowKey != intent.Correlation.WorkflowKey || existing.Correlation.EntityVersion != intent.Correlation.EntityVersion || existing.Correlation.FrontierDigest != intent.Correlation.FrontierDigest || existing.ApprovalDigest != intent.ApprovalDigest || existing.GrantDigest != intent.GrantDigest {
				return transitionrun.Correlation{}, fmt.Errorf("execution already has a different immutable dispatch intent")
			}
			if existing.Correlation.ExecutionID != "" {
				return r.store.Get(existing.Correlation.ExecutionID)
			}
			return r.ResolveDispatch(ctx, definition.Key, subjectRef)
		} else if !errors.Is(err, os.ErrNotExist) {
			return transitionrun.Correlation{}, err
		}
		if err := store.PutDispatch(intent); err != nil {
			return transitionrun.Correlation{}, fmt.Errorf("persist workflow submission before dispatch: %w", err)
		}
		dispatch = &intent
	}
	var started agentmanager.WorkflowStart
	var err error
	if starter := r.start[definition.Key]; starter != nil {
		started, err = starter(ctx, StartInvocation{Definition: definition, SubjectRef: subjectRef, Snapshot: Snapshot{Input: prepared.Input, EntityVersion: prepared.EntityVersion, FrontierDigest: prepared.FrontierDigest}, Invocation: invocation})
	} else {
		started, err = r.workflows.StartWorkflow(ctx, invocation)
	}
	if err != nil {
		if dispatch != nil {
			dispatch.State, dispatch.LastError = "unknown", err.Error()
			if saveErr := r.store.(transitionrun.DispatchStore).PutDispatch(*dispatch); saveErr != nil {
				return transitionrun.Correlation{}, errors.Join(err, saveErr)
			}
		}
		return transitionrun.Correlation{}, err
	}
	if prepared.Grant != nil && (started.ApprovalDigest != invocation.ApprovalDigest || started.GrantDigest != invocation.GrantDigest) {
		return transitionrun.Correlation{}, fmt.Errorf("workflow owner did not retain the approved execution grant")
	}
	// Agent Manager may return the same execution for an unchanged
	// idempotency key. Preserve its durable journal state: rewriting a complete
	// correlation as claimed would make a delivered mutation eligible to run
	// again after a harmless repeated start request.
	if existing, getErr := r.store.Get(started.ExecutionID); getErr == nil {
		if existing.TransitionKey != definition.Key || existing.SubjectRef != subjectRef || existing.EntityVersion != prepared.EntityVersion {
			return transitionrun.Correlation{}, fmt.Errorf("idempotent execution %q conflicts with its stored transition correlation", started.ExecutionID)
		}
		return existing, nil
	} else if !errors.Is(getErr, os.ErrNotExist) {
		return transitionrun.Correlation{}, fmt.Errorf("read existing transition correlation: %w", getErr)
	}
	correlation := transitionrun.Correlation{TransitionKey: definition.Key, SubjectKind: definition.Subject, SubjectRef: subjectRef, ExecutionID: started.ExecutionID, WorkflowKey: workflow.Key, DefinitionDigest: started.DefinitionDigest, EntityVersion: prepared.EntityVersion, FrontierDigest: prepared.FrontierDigest, ApplyState: transitionrun.ApplyStateClaimed, DeclaredOutcomes: append([]string(nil), definition.TerminalOutcomes...)}
	if strings.TrimSpace(started.RunID) != "" {
		correlation.Attempts = []transitionrun.Attempt{{RunID: started.RunID}}
	}
	if err := r.store.Put(correlation); err != nil {
		return transitionrun.Correlation{}, fmt.Errorf("persist transition correlation: %w", err)
	}
	if dispatch != nil {
		return r.bindDispatch(*dispatch, correlation)
	}
	return correlation, nil
}

func (r *Runner) Apply(ctx context.Context, transitionKey, executionID string) (transitionrun.Correlation, error) {
	lock := r.applyLock(executionID)
	lock.Lock()
	defer lock.Unlock()

	correlation, err := r.store.Get(executionID)
	if err != nil {
		return transitionrun.Correlation{}, err
	}
	if correlation.TransitionKey != transitionKey {
		return transitionrun.Correlation{}, fmt.Errorf("execution %q belongs to transition %q", executionID, correlation.TransitionKey)
	}
	if correlation.ApplyState == transitionrun.ApplyStateComplete {
		return correlation, nil
	}
	correlation.ApplyAttemptCount++
	correlation.LastApplyAttemptTime = time.Now().UTC().Format(time.RFC3339Nano)
	if err := r.store.Put(correlation); err != nil {
		return transitionrun.Correlation{}, fmt.Errorf("record transition apply attempt: %w", err)
	}
	fail := func(applyErr error) (transitionrun.Correlation, error) {
		correlation.LastApplyError = applyErr.Error()
		if persistErr := r.store.Put(correlation); persistErr != nil {
			return transitionrun.Correlation{}, fmt.Errorf("record transition apply failure %v: %w", applyErr, persistErr)
		}
		return correlation, applyErr
	}
	definition, err := r.workflowDefinition(transitionKey)
	if err != nil {
		return fail(err)
	}
	builder := r.input[definition.Key]
	if builder == nil {
		return fail(fmt.Errorf("transition %q has no input builder", definition.Key))
	}
	fn := r.apply[definition.ApplyAction]
	if fn == nil {
		return fail(fmt.Errorf("transition %q apply action %q is not registered", definition.Key, definition.ApplyAction))
	}
	completion, err := r.workflows.CollectWorkflow(ctx, executionID)
	if err != nil {
		return fail(err)
	}
	// Rebuild the subject snapshot at apply time. A completion is only safe to
	// consume when it still belongs to the immutable input the workflow saw;
	// comparing the correlation version to itself would never detect edits made
	// while the workflow was running. A builder that echoes the stored version
	// instead of recomputing it silently disables this guard, so the rebuild
	// must derive both digests from current subject state.
	current, err := builder(ctx, correlation.SubjectRef)
	if err != nil {
		return fail(fmt.Errorf("build current transition input: %w", err))
	}
	if strings.TrimSpace(current.EntityVersion) == "" {
		return fail(fmt.Errorf("transition %q input builder returned an empty current version", definition.Key))
	}
	outcome, completionForGuard, err := completionOutcome(completion, correlation, current)
	if err != nil {
		return fail(err)
	}
	if err := transitionrun.CanApply(correlation, completionForGuard); err != nil {
		return fail(err)
	}
	if err := fn(ctx, correlation.SubjectRef, outcome); err != nil {
		return fail(err)
	}
	correlation.Outcome, correlation.TerminalCode, correlation.BudgetName, correlation.Result, correlation.Attempts = outcome.Name, outcome.TerminalCode, outcome.BudgetName, outcome.Result, outcome.Attempts
	correlation.ApplyState = transitionrun.ApplyStateComplete
	correlation.LastApplyError = ""
	correlation.AppliedTime = time.Now().UTC().Format(time.RFC3339Nano)
	if err := r.store.Put(correlation); err != nil {
		return transitionrun.Correlation{}, err
	}
	return correlation, nil
}

// ApplyExecution resolves the transition from its durable correlation before
// applying it. Deprecated subject routes use this when their historical URL
// carries only an execution id; generic callers should continue to supply the
// transition key explicitly.
func (r *Runner) ApplyExecution(ctx context.Context, executionID string) (transitionrun.Correlation, error) {
	correlation, err := r.store.Get(executionID)
	if err != nil {
		return transitionrun.Correlation{}, err
	}
	return r.Apply(ctx, correlation.TransitionKey, executionID)
}

func (r *Runner) applyLock(executionID string) *sync.Mutex {
	lock, _ := r.applyLocks.LoadOrStore(executionID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func (r *Runner) workflowDefinition(key string) (transitions.Definition, error) {
	definition, ok := r.registry.Get(key)
	if !ok {
		return transitions.Definition{}, fmt.Errorf("transition %q is not registered", key)
	}
	if definition.Kind != transitions.KindWorkflow || definition.Workflow == nil {
		return transitions.Definition{}, fmt.Errorf("transition %q is not a workflow transition", key)
	}
	return definition, nil
}

// VerifyDispatchTable reports every registry action that has no concrete
// subject mutation registered at composition time.
func (r *Runner) VerifyDispatchTable() error { return VerifyDispatchTable(r.registry, r.apply) }

func VerifyDispatchTable(registry transitions.Registry, apply map[string]ApplyFunc) error {
	missing := map[string]struct{}{}
	for _, definition := range registry.Definitions() {
		if definition.Kind != transitions.KindWorkflow {
			continue
		}
		if definition.ApplyAction != "" && apply[definition.ApplyAction] == nil {
			missing[definition.ApplyAction] = struct{}{}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	actions := make([]string, 0, len(missing))
	for action := range missing {
		actions = append(actions, action)
	}
	sort.Strings(actions)
	return fmt.Errorf("transition dispatch table missing apply actions: %s", strings.Join(actions, ", "))
}

func completionOutcome(completion agentmanager.InvocationCompletion, correlation transitionrun.Correlation, current Snapshot) (Outcome, transitionrun.Completion, error) {
	if current.Grant != nil && (completion.ApprovalDigest != current.ApprovalDigest || completion.GrantDigest != workflowcontract.GrantDigest(current.Grant)) {
		return Outcome{}, transitionrun.Completion{}, fmt.Errorf("workflow completion does not match the retained execution grant")
	}
	status := ""
	if completion.Status == domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		status = transitionrun.CompletionSucceeded
	} else {
		status = strings.ToLower(strings.TrimPrefix(completion.Status.String(), "WORKFLOW_EXECUTION_STATUS_"))
	}
	outcome := Outcome{ExecutionID: completion.ExecutionID, TransitionKey: correlation.TransitionKey, SubjectRef: correlation.SubjectRef, EntityVersion: correlation.EntityVersion, FrontierDigest: correlation.FrontierDigest, TerminalCode: completion.TerminalCode, BudgetName: completion.BudgetName, WorkflowDigest: completion.WorkflowDigest, ApprovalDigest: completion.ApprovalDigest, GrantDigest: completion.GrantDigest}
	outcome.Usage = workflowUsage(completion)
	if completion.Output != nil {
		raw, err := json.Marshal(completion.Output.AsInterface())
		if err != nil {
			return Outcome{}, transitionrun.Completion{}, err
		}
		outcome.Result = raw
		if object, ok := completion.Output.AsInterface().(map[string]any); ok {
			if result, ok := object["result"]; ok {
				outcome.Result, _ = json.Marshal(result)
				if fields, ok := result.(map[string]any); ok {
					outcome.Name, _ = fields["outcome"].(string)
				}
			}
		}
	}
	if outcome.Name == "" && status == transitionrun.CompletionSucceeded {
		return Outcome{}, transitionrun.Completion{}, fmt.Errorf("workflow succeeded without a terminal outcome")
	}
	for _, attempt := range completion.Attempts {
		if attempt != nil {
			outcome.Attempts = append(outcome.Attempts, transitionrun.Attempt{NodeID: attempt.NodeId, Ordinal: attempt.Ordinal, Strategy: attempt.Strategy, RunID: attempt.RunId, ConversationID: attempt.ConversationId, SourceAttemptID: attempt.SourceAttemptId, ProfileIdentity: attempt.ProfileIdentity})
		}
	}
	// The rebuilt snapshot detects edits to the durable subject. When the
	// transport also returns the workflow's original input, retain its digest
	// when it disagrees with the pinned correlation so a malformed or replayed
	// completion cannot pass the immutable-input guard.
	entityVersion := current.EntityVersion
	frontierDigest := current.FrontierDigest
	if completion.Input != nil {
		if value, ok := nestedStringField(completion.Input.AsInterface(), "frontierDigest"); ok && value != correlation.FrontierDigest {
			frontierDigest = value
		}
	}
	return outcome, transitionrun.Completion{ExecutionID: completion.ExecutionID, DefinitionDigest: completion.DefinitionDigest, EntityVersion: entityVersion, FrontierDigest: frontierDigest, Status: status, Outcome: outcome.Name}, nil
}

func nestedStringField(value any, field string) (string, bool) {
	switch typed := value.(type) {
	case map[string]any:
		if found, ok := typed[field].(string); ok && strings.TrimSpace(found) != "" {
			return found, true
		}
		for _, child := range typed {
			if found, ok := nestedStringField(child, field); ok {
				return found, true
			}
		}
	case []any:
		for _, child := range typed {
			if found, ok := nestedStringField(child, field); ok {
				return found, true
			}
		}
	}
	return "", false
}
