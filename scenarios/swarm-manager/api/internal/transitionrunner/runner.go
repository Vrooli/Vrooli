// Package transitionrunner executes declared workflow transitions without
// importing any subject domain package.
package transitionrunner

import (
	"context"
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

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	InputBuilder func(context.Context, string) (Snapshot, error)
	ApplyFunc    func(context.Context, string, Outcome) error
	StartGuard   func(context.Context, transitions.Definition, string) error
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
	applyLocks sync.Map // map[string]*sync.Mutex, keyed by execution id
}

func New(registry transitions.Registry, workflows agentmanager.WorkflowInvoker, store transitionrun.Store, guard StartGuard) *Runner {
	return &Runner{registry: registry, workflows: workflows, store: store, guard: guard, apply: map[string]ApplyFunc{}, input: map[string]InputBuilder{}}
}
func (r *Runner) RegisterApply(action string, fn ApplyFunc) { r.apply[strings.TrimSpace(action)] = fn }
func (r *Runner) RegisterInput(key string, builder InputBuilder) {
	r.input[strings.TrimSpace(key)] = builder
}
func (r *Runner) Counts() (int, int) { return len(r.apply), len(r.input) }

// HasInput reports whether a transition has a registered input builder.
func (r *Runner) HasInput(transitionKey string) bool {
	return r.input[strings.TrimSpace(transitionKey)] != nil
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
	started, err := r.workflows.StartWorkflow(ctx, agentmanager.Invocation{Owner: workflow.Owner, WorkflowKey: workflow.Key, Input: prepared.Input, IdempotencyKey: idempotencyKey, FirstRunNodeID: prepared.FirstRunNodeID, Activity: activity})
	if err != nil {
		return transitionrun.Correlation{}, err
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
	status := ""
	if completion.Status == domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		status = transitionrun.CompletionSucceeded
	} else {
		status = strings.ToLower(strings.TrimPrefix(completion.Status.String(), "WORKFLOW_EXECUTION_STATUS_"))
	}
	outcome := Outcome{ExecutionID: completion.ExecutionID, TransitionKey: correlation.TransitionKey, SubjectRef: correlation.SubjectRef, EntityVersion: correlation.EntityVersion, FrontierDigest: correlation.FrontierDigest, TerminalCode: completion.TerminalCode, BudgetName: completion.BudgetName}
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
