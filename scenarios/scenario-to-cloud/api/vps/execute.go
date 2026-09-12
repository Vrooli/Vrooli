package vps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/faultinject"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/secrets"
)

// ErrUndeclaredEffect is returned when the executor would reach the target
// outside any plan action. Every reach call must belong to an action.
var ErrUndeclaredEffect = errors.New("undeclared effect: target reached outside a plan action")

// AttributedCall is one transport call recorded against the action that
// caused it. Command is the verb and argv joined for inspection; stdin
// payloads are never recorded.
type AttributedCall struct {
	ActionID string
	Kind     string // "exec" | "deliver" | "negotiate"
	Verb     string
	Args     []string
	Command  string
}

// attributedReach wraps the reach seam so every call carries the id of the
// action being executed. A call with no current action is refused.
type attributedReach struct {
	inner reach.Reach

	mu      sync.Mutex
	current string
	calls   []AttributedCall
}

func (a *attributedReach) enter(actionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.current = actionID
}

func (a *attributedReach) leave() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.current = ""
}

func (a *attributedReach) record(kind string, cmd reach.Command) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.current == "" {
		return ErrUndeclaredEffect
	}
	a.calls = append(a.calls, AttributedCall{ActionID: a.current, Kind: kind, Verb: cmd.Verb, Args: append([]string(nil), cmd.Args...), Command: strings.Join(cmd.Argv(), " ")})
	return nil
}

// Calls returns every recorded call in order.
func (a *attributedReach) Calls() []AttributedCall {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]AttributedCall(nil), a.calls...)
}

func (a *attributedReach) Exec(ctx context.Context, target identity.TargetRef, cmd reach.Command) (reach.Result, error) {
	if err := a.record("exec", cmd); err != nil {
		return reach.Result{}, err
	}
	if a.inner == nil {
		return reach.Result{}, &reach.Error{Kind: reach.KindUnavailable, Detail: "reach not configured"}
	}
	return a.inner.Exec(ctx, target, cmd)
}

func (a *attributedReach) Deliver(ctx context.Context, target identity.TargetRef, delivery reach.Delivery) (reach.DeliveryReceipt, error) {
	paths := make([]string, 0, len(delivery.Files))
	for _, f := range delivery.Files {
		paths = append(paths, f.LocalPath+" -> "+f.RemotePath)
	}
	if err := a.record("deliver", reach.Command{Verb: "deliver", Args: paths}); err != nil {
		return reach.DeliveryReceipt{}, err
	}
	if a.inner == nil {
		return reach.DeliveryReceipt{}, &reach.Error{Kind: reach.KindUnavailable, Detail: "reach not configured"}
	}
	return a.inner.Deliver(ctx, target, delivery)
}

func (a *attributedReach) Negotiate(ctx context.Context, target identity.TargetRef) (reach.Capabilities, error) {
	if err := a.record("negotiate", reach.Command{Verb: "negotiate"}); err != nil {
		return reach.Capabilities{}, err
	}
	if a.inner == nil {
		return reach.Capabilities{}, &reach.Error{Kind: reach.KindUnavailable, Detail: "reach not configured"}
	}
	return a.inner.Negotiate(ctx, target)
}

// CredentialProvisioner materialises the manifest's declared credentials on
// the target through the credential authority and revokes them at
// retirement. Values never pass through the executor's inputs or receipts.
type CredentialProvisioner interface {
	Provision(ctx context.Context, req CredentialProvisionRequest) (CredentialProvisionResult, error)
	Revoke(ctx context.Context, req CredentialProvisionRequest) (CredentialProvisionResult, error)
}

// CredentialProvisionRequest is what the executor hands the provisioner.
type CredentialProvisionRequest struct {
	DeploymentID string
	Target       identity.TargetRef
	Identity     Identity
	Manifest     domain.CloudManifest
	// GeneratedValues and OperatorValues are keyed by secret id; they are
	// consumed by the provisioner and never recorded.
	GeneratedValues map[string]string
	OperatorValues  map[string]string
}

// CredentialProvisionResult is metadata only.
type CredentialProvisionResult struct {
	Preserved    []string
	Materialized []string
	Skipped      []string
	Revoked      []string
}

// RecoveryPointRecorder records the recovery point the target owner
// captured (its manifest) on the cloud side so retention, rollback admission
// and restore can reference it.
type RecoveryPointRecorder interface {
	Record(ctx context.Context, deploymentID string, identity Identity, manifest json.RawMessage, releaseDigest string) (string, error)
}

// Runtime is the target-facing seam set one execution runs with.
type Runtime struct {
	Reach       reach.Reach
	Target      identity.TargetRef
	Identity    Identity
	Credentials CredentialProvisioner
	Backups     RecoveryPointRecorder
	// SecretsGen mints per-install generated values; nil selects the
	// default generator.
	SecretsGen      secrets.GeneratorFunc
	ProvidedSecrets map[string]string
}

// ExecuteRequest is everything one plan execution needs.
type ExecuteRequest struct {
	Plan         *execplan.Plan
	Manifest     domain.CloudManifest
	BundlePath   string
	Runtime      Runtime
	Hub          ProgressBroadcaster
	Repo         ProgressRepo
	DeploymentID string
	Progress     *float64
	Weights      map[string]float64
	Hooks        Hooks
}

// executor runs one compiled plan against its target through reach. The
// durable operation worker owns execution; the plan owns ordering, ids and
// inputs; this type owns only the mapping from an action to its target
// verbs and the interpretation of their replies.
type executor struct {
	plan         *execplan.Plan
	manifest     domain.CloudManifest
	bundlePath   string
	rt           Runtime
	reach        *attributedReach
	hub          ProgressBroadcaster
	repo         ProgressRepo
	deploymentID string
	progress     *float64
	weights      map[string]float64
	hooks        Hooks

	results []domain.VPSActionResult
}

// Hooks let a durable owner (operations worker) decide per action whether
// it runs and record what happened. Before returning skip=true means the
// action is not executed and counts as completed (a receipt already proves
// it). After receives the observed outcome and returns the outcome the
// executor continues with: nil continues (even when execErr was non-nil and
// the owner proved success through a target receipt), ErrReplayAction
// re-runs the action once (its replay contract allows it), any other error
// stops execution attributed to the action.
type Hooks struct {
	Before func(ctx context.Context, action execplan.Action) (skip bool, err error)
	After  func(ctx context.Context, action execplan.Action, result domain.VPSActionResult, execErr error) error
}

// ErrReplayAction is returned by Hooks.After to re-run the action once.
var ErrReplayAction = errors.New("replay action")

// Trace is what an execution recorded: per-action results and every
// attributed transport call.
type Trace struct {
	Actions []domain.VPSActionResult
	Calls   []AttributedCall
}

// ExecutePlan runs the plan against the target and returns the trace. It
// refuses plans whose outcome is not apply. The result error carries the
// failed action id in FailedAction.
func ExecutePlan(ctx context.Context, req ExecuteRequest) (Trace, *ExecutionError) {
	if req.Hub == nil {
		req.Hub = NoopProgressHub{}
	}
	if req.Repo == nil {
		req.Repo = NoopProgressRepo{}
	}
	if req.Progress == nil {
		zero := 0.0
		req.Progress = &zero
	}
	if req.Runtime.SecretsGen == nil {
		req.Runtime.SecretsGen = secrets.NewGenerator()
	}
	if req.Runtime.Target.Transport == "" {
		req.Runtime.Target = domain.TargetRefFromManifest(req.Manifest)
	}
	if req.Runtime.Target.Locator.Workdir == "" && req.Manifest.Target.VPS != nil {
		req.Runtime.Target.Locator.Workdir = req.Manifest.Target.VPS.Workdir
	}
	if req.Runtime.Identity.OperationID == "" {
		req.Runtime.Identity.OperationID = "adhoc-" + time.Now().UTC().Format("20060102T150405")
	}
	e := &executor{
		plan:         req.Plan,
		manifest:     req.Manifest,
		bundlePath:   req.BundlePath,
		rt:           req.Runtime,
		reach:        &attributedReach{inner: req.Runtime.Reach},
		hub:          req.Hub,
		repo:         req.Repo,
		deploymentID: req.DeploymentID,
		progress:     req.Progress,
		weights:      req.Weights,
		hooks:        req.Hooks,
	}
	execErr := e.run(ctx)
	return Trace{Actions: e.results, Calls: e.reach.Calls()}, execErr
}

// ExecutionError names the action that failed.
type ExecutionError struct {
	ActionID string
	Err      error
	Info     *domain.ErrorInfo
}

func (e *ExecutionError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExecutionError) Unwrap() error { return e.Err }

func (e *executor) commandContext() CommandContext {
	return CommandContext{DeploymentID: e.deploymentID, ScenarioID: e.manifest.Scenario.ID, Identity: e.rt.Identity, Manifest: e.manifest}
}

func (e *executor) run(ctx context.Context) *ExecutionError {
	if e.plan == nil {
		return &ExecutionError{Err: fmt.Errorf("plan is required")}
	}
	switch e.plan.Outcome {
	case execplan.OutcomeApply:
	case execplan.OutcomeNoOp:
		return nil
	case execplan.OutcomeNeedsInput:
		return &ExecutionError{ActionID: execplan.OpInputResumeHandoff, Err: execplan.NeedsInputError(e.plan.Handoff)}
	default:
		return &ExecutionError{Err: apierrors.Newf(apierrors.CodeInvalidRequest, "Unknown plan outcome %q", e.plan.Outcome)}
	}
	if e.deploymentID == "" {
		e.deploymentID = e.plan.DeploymentID
	}
	completed := map[string]bool{}
	for _, action := range e.plan.Actions {
		for _, dep := range action.DependsOn {
			if !completed[dep] {
				return &ExecutionError{ActionID: action.ID, Err: fmt.Errorf("action %s depends on %s which has not completed", action.ID, dep)}
			}
		}
		handler, ok := actionHandlers[action.OwnerOperation]
		if !ok {
			return e.fail(action, apierrors.Newf(apierrors.CodeUnsupportedCapability, "No executor for owner operation %q", action.OwnerOperation))
		}
		if err := ctx.Err(); err != nil {
			return e.fail(action, err)
		}
		title := execplan.Render(&execplan.Plan{Actions: []execplan.Action{action}}).Changes[0].Summary
		if e.hooks.Before != nil {
			skip, err := e.hooks.Before(ctx, action)
			if err != nil {
				return e.fail(action, err)
			}
			if skip {
				e.results = append(e.results, domain.VPSActionResult{ID: action.ID, OwnerOperation: action.OwnerOperation, Status: "skipped", Detail: "committed by a previous attempt"})
				completed[action.ID] = true
				*e.progress += e.weight(action.ID)
				e.emit("step_completed", action.ID, title)
				continue
			}
		}
		e.emit("step_started", action.ID, title)
		var (
			result domain.VPSActionResult
			err    error
		)
		for attempt := 0; attempt < 2; attempt++ {
			started := time.Now()
			e.reach.enter(action.ID)
			detail, runErr := handler(ctx, e, action)
			e.reach.leave()
			duration := time.Since(started).Milliseconds()
			if runErr == nil {
				runErr = faultinject.Hit(ctx, faultinject.WorkerAfterEffect)
			}
			if runErr != nil {
				result = domain.VPSActionResult{ID: action.ID, OwnerOperation: action.OwnerOperation, Status: "failed", DurationMs: duration, Detail: detail, Error: runErr.Error()}
			} else {
				result = domain.VPSActionResult{ID: action.ID, OwnerOperation: action.OwnerOperation, Status: "succeeded", DurationMs: duration, Detail: detail}
			}
			err = runErr
			if e.hooks.After != nil {
				err = e.hooks.After(ctx, action, result, runErr)
				if errors.Is(err, ErrReplayAction) && attempt == 0 {
					continue
				}
				if err == nil && runErr != nil {
					result.Status = "succeeded"
					result.Error = ""
					result.Detail = "confirmed by target receipt"
				}
			}
			break
		}
		e.results = append(e.results, result)
		if err != nil {
			return e.fail(action, err)
		}
		completed[action.ID] = true
		*e.progress += e.weight(action.ID)
		e.emit("step_completed", action.ID, title)
	}
	return nil
}

func (e *executor) weight(actionID string) float64 {
	if e.weights != nil {
		if w, ok := e.weights[actionID]; ok {
			return w
		}
	}
	return StepWeights[actionID]
}

func (e *executor) emit(eventType, stepID, stepTitle string) {
	event := NewProgressEvent(eventType, stepID, stepTitle, *e.progress)
	e.hub.Broadcast(e.deploymentID, event)
	if err := e.repo.UpdateDeploymentProgress(context.Background(), e.deploymentID, stepID, *e.progress); err != nil {
		log.Printf("progress update failed (step=%s): %v", stepID, err)
	}
}

func (e *executor) fail(action execplan.Action, err error) *ExecutionError {
	var info *domain.ErrorInfo
	if typed := apierrors.As(err); typed != nil {
		info = &domain.ErrorInfo{Category: typed.Code, Message: typed.Message, Retryable: typed.Retryable}
	} else if re := (*reach.Error)(nil); errors.As(err, &re) {
		info = &domain.ErrorInfo{Category: string(re.Kind), Message: re.Detail, Retryable: re.Kind == reach.KindTargetOffline || re.Kind == reach.KindTransport}
	}
	title := execplan.Render(&execplan.Plan{Actions: []execplan.Action{action}}).Changes[0].Summary
	e.hub.Broadcast(e.deploymentID, NewStructuredErrorEvent(action.ID, title, *e.progress, err.Error(), info))
	return &ExecutionError{ActionID: action.ID, Err: err, Info: info}
}

// ---- target replies -------------------------------------------------------

// verbReply is the JSON every cloud-target verb prints.
type verbReply struct {
	Receipt *struct {
		Outcome string         `json:"outcome"`
		Details map[string]any `json:"details"`
	} `json:"receipt"`
	Replayed bool `json:"replayed"`
	Error    *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Report        json.RawMessage `json:"report"`
	RecoveryPoint json.RawMessage `json:"recovery_point"`
	Result        json.RawMessage `json:"result"`
}

// TargetRefusal is a typed answer of the target owner (non-zero exit with a
// code); it is never a transport failure.
func targetError(code, message string) *apierrors.Error {
	switch code {
	case "fence_stale":
		return apierrors.New(apierrors.CodeFenceStale, message)
	case "rollback_not_eligible":
		return apierrors.New(apierrors.CodeRollbackIncompatible, message)
	case "":
		return apierrors.New(apierrors.CodeInternal, message)
	}
	return apierrors.New(code, message)
}

// invoke runs one target command and decodes the cloud-target reply. A
// transport error is returned untouched so the owner can classify it as an
// unknown outcome; a typed refusal becomes an apierrors value.
func (e *executor) invoke(ctx context.Context, tc TargetCommand) (verbReply, reach.Result, error) {
	res, err := e.reach.Exec(ctx, e.rt.Target, tc.Command)
	if err != nil {
		return verbReply{}, res, err
	}
	var reply verbReply
	trimmed := strings.TrimSpace(res.Stdout)
	if strings.HasPrefix(tc.Command.Verb, "cloud-target") {
		if trimmed == "" {
			if res.ExitCode != 0 {
				return reply, res, fmt.Errorf("%s exited %d without a reply: %s", tc.Command.Verb, res.ExitCode, strings.TrimSpace(res.Stderr))
			}
			return reply, res, fmt.Errorf("%s printed no reply", tc.Command.Verb)
		}
		if uerr := json.Unmarshal([]byte(trimmed), &reply); uerr != nil {
			return reply, res, fmt.Errorf("%s reply is not JSON: %w", tc.Command.Verb, uerr)
		}
		if reply.Error != nil {
			return reply, res, targetError(reply.Error.Code, reply.Error.Message)
		}
		if reply.Receipt != nil && reply.Receipt.Outcome == "failed" {
			return reply, res, targetError("", tc.Command.Verb+" recorded a failed receipt")
		}
		if res.ExitCode != 0 {
			return reply, res, targetError("", fmt.Sprintf("%s exited %d", tc.Command.Verb, res.ExitCode))
		}
		return reply, res, nil
	}
	if res.ExitCode != 0 {
		return reply, res, apierrors.Newf(apierrors.CodeInternal, "vrooli %s exited %d: %s", tc.Command.Verb, res.ExitCode, strings.TrimSpace(firstNonEmpty(res.Stderr, res.Stdout)))
	}
	return reply, res, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// runCommands executes every invocation of an action in order and returns a
// short detail line. Replayed receipts count as success.
func (e *executor) runCommands(ctx context.Context, action execplan.Action) (string, error) {
	commands, err := ActionCommands(action, e.commandContext())
	if err != nil {
		return "", err
	}
	details := make([]string, 0, len(commands))
	for _, tc := range commands {
		reply, _, err := e.invoke(ctx, tc)
		if err != nil {
			return strings.Join(details, "; "), err
		}
		outcome := "ok"
		if reply.Receipt != nil {
			outcome = reply.Receipt.Outcome
			if reply.Replayed {
				outcome += " (replayed)"
			}
		}
		details = append(details, tc.Step+": "+outcome)
	}
	return strings.Join(details, "; "), nil
}
