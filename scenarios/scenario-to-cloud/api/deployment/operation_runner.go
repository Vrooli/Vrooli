package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/faultinject"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/vps"
	"scenario-to-cloud/vps/preflight"
)

// The orchestrator is the operations.Runner: one attempt executes the
// admitted plan JSON through vps.ExecutePlanWithHooks, with the durable
// worker deciding per action (through the StepSink hooks) whether the
// action runs, is skipped because a receipt proves it, or stops. Cloud-local
// prerequisites (secrets, bundle, preflight) run before the first target
// action and never mutate the target.
var _ operations.Runner = (*Orchestrator)(nil)

// Execute implements operations.Runner.
func (o *Orchestrator) Execute(ctx context.Context, ec *operations.ExecutionContext) error {
	op, plan := ec.Operation, ec.Plan
	dep, err := o.repo.GetDeployment(ctx, op.DeploymentID)
	if err != nil {
		return fmt.Errorf("load deployment: %w", err)
	}
	if dep == nil {
		return apierrors.New(apierrors.CodeDeploymentNotFound, "Deployment of the operation no longer exists").WithDetail("deployment_id", op.DeploymentID)
	}
	var manifest domain.CloudManifest
	if err := json.Unmarshal(dep.Manifest, &manifest); err != nil {
		return apierrors.Internal("Deployment manifest is not decodable", err)
	}
	canonicalIdentity, manifest := o.resolveAndPersistIdentity(ctx, dep.ID, manifest)
	progress := 0.0
	emitError := func(step, stepTitle, errMsg string) {
		o.progressHub.Broadcast(dep.ID, Event{Type: "deployment_error", Step: step, StepTitle: stepTitle, Progress: progress, Error: errMsg, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	}

	o.log("operation attempt started", map[string]interface{}{
		"deployment_id": dep.ID, "operation_id": op.ID, "fence": ec.Fence, "scope": plan.Scope, "resumed": ec.Resumed, "plan_digest": op.PlanDigest,
	})

	// Cloud-local prerequisites. Secrets are the one non-durable input: when
	// the owner restarted, provided operator secrets are gone, and a plan
	// that needs them waits for input instead of pretending.
	if err := o.ensureSecretsAvailable(ctx, &manifest, ec.Options.ProvidedSecrets, dep.ID, emitError); err != nil {
		var missing []domain.MissingSecretInfo
		if m, verr := vps.ValidateUserPromptSecrets(manifest, ec.Options.ProvidedSecrets); verr != nil {
			missing = m
		}
		if len(missing) > 0 {
			names := make([]string, 0, len(missing))
			for _, m := range missing {
				names = append(names, m.Key)
			}
			typed := apierrors.New(apierrors.CodeNeedsInput, "Operator-supplied secrets are required to continue").
				WithDetail("missing", names).
				WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "resume", Reference: "/api/v1/deployments/" + dep.ID + "/execute", Label: "Resubmit with the missing secrets under the same request key"})
			return operations.WaitingInputError(typed)
		}
		return err
	}
	bundlePath := ""
	if plan.Scope == execplan.ScopeFull || plan.Scope == execplan.ScopeInstall {
		path, err := o.ensureBundleBuilt(ctx, manifest, dep.BundlePath, ec.Options.ForceBundleBuild, dep.ID, emitError)
		if err != nil {
			return err
		}
		bundlePath = path
	}
	if ec.Options.RunPreflight && !ec.Resumed {
		if err := o.runPreflightStage(ctx, dep.ID, manifest, ec.Options.ProvidedSecrets, &progress, emitError); err != nil {
			return err
		}
	}

	hubAdapter := &progressHubAdapter{hub: o.progressHub}
	repoAdapter := &progressRepoAdapter{repo: o.repo}
	hooks := vps.Hooks{
		Before: func(ctx context.Context, action execplan.Action) (bool, error) {
			decision, err := ec.Steps.Begin(ctx, action)
			if err != nil {
				return false, err
			}
			return decision == operations.DecisionSkip, nil
		},
		After: func(ctx context.Context, action execplan.Action, result domain.VPSActionResult, execErr error) error {
			outcome := operations.StepSucceeded
			switch {
			case execErr == nil:
			case OutcomeUnknown(execErr):
				outcome = operations.StepUnknown
			default:
				outcome = operations.StepFailed
			}
			err := ec.Steps.Commit(ctx, action, outcome, result.Detail, execErr)
			if errors.Is(err, operations.ErrReplayStep) {
				return vps.ErrReplayAction
			}
			return err
		},
	}
	stageStart := time.Now()
	o.appendHistoryEvent(ctx, dep.ID, domain.HistoryEvent{Type: domain.EventDeployStarted, Timestamp: stageStart.UTC(), Message: fmt.Sprintf("Operation %s started (%s)", op.ID, plan.Scope)})
	var weights map[string]float64
	if plan.Scope == execplan.ScopeStart {
		weights = vps.CalculateWeightsForSteps(vps.StartSteps)
	}
	runtime := vps.Runtime{
		Reach:           o.reach,
		Target:          targetFor(dep, manifest),
		Identity:        vps.Identity{OperationID: op.ID, Fence: ec.Fence},
		Backups:         o.backups,
		SecretsGen:      o.secretsGenerator,
		ProvidedSecrets: ec.Options.ProvidedSecrets,
	}
	if o.credentials != nil {
		runtime.Credentials = &credentialProvisioner{bind: o.credentials, store: o.repo}
	}
	trace, execErr := vps.ExecutePlan(ctx, vps.ExecuteRequest{Plan: plan, Manifest: manifest, BundlePath: bundlePath, Runtime: runtime, Hub: hubAdapter, Repo: repoAdapter, DeploymentID: dep.ID, Progress: &progress, Weights: weights, Hooks: hooks})

	digest, _ := plan.SemanticDigest()
	result := domain.VPSDeployResult{OK: execErr == nil, PlanDigest: digest, Actions: trace.Actions, DurationMs: time.Since(stageStart).Milliseconds(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	if execErr != nil {
		result.Error = execErr.Error()
		result.ErrorInfo = execErr.Info
		result.FailedStep = execErr.ActionID
	}
	resultJSON, _ := json.Marshal(result)
	if err := o.repo.UpdateDeploymentDeployResult(ctx, dep.ID, resultJSON, false); err != nil {
		o.log("failed to save operation result", map[string]interface{}{"error": err.Error()})
	}
	if execErr == nil {
		o.enforceVPSBundleRetentionBestEffort(ctx, dep.ID, manifest)
		o.verifyAndPersistIdentity(ctx, dep.ID, manifest, canonicalIdentity)
		return nil
	}
	// Control errors (cancel, unknown effect, recovery) pass through to the
	// worker untouched; everything else is a failed change.
	return execErr.Err
}

// targetFor is the reach target of a deployment: the bound identity with
// the manifest locator filled in when the binding carries none.
func targetFor(dep *domain.Deployment, manifest domain.CloudManifest) identity.TargetRef {
	target := dep.Target
	if target.Transport == "" {
		target.Transport = identity.TransportSSH
	}
	if target.Locator.Host == "" && manifest.Target.VPS != nil {
		target.Locator = identity.TargetLocator{Host: manifest.Target.VPS.Host, Port: manifest.Target.VPS.Port, User: manifest.Target.VPS.User, Workdir: manifest.Target.VPS.Workdir}
	}
	if target.Locator.Workdir == "" && manifest.Target.VPS != nil {
		target.Locator.Workdir = manifest.Target.VPS.Workdir
	}
	return target
}

// OutcomeUnknown classifies an execution error whose effect may have
// happened on the target although no reply arrived: a dropped reply, a
// transport timeout, a transport failure after dispatch or an unreachable
// host. Such a step is never committed as failed; the owner reads the
// target receipt first. A typed target refusal (fence stale, not staged) and
// a reach refusal before dispatch (invalid argument, revoked enrollment,
// missing scope) are known outcomes.
func OutcomeUnknown(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, faultinject.ErrReplyDropped) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if reach.IsKind(err, reach.KindTransport) || reach.IsKind(err, reach.KindTargetOffline) {
		return true
	}
	return false
}

func (o *Orchestrator) runPreflightStage(ctx context.Context, id string, manifest domain.CloudManifest, providedSecrets map[string]string, progress *float64, emitError func(step, stepTitle, errMsg string)) error {
	preflightStart := time.Now()
	preflightTarget := o.targetFor(ctx, id, manifest)
	o.appendHistoryEvent(ctx, id, domain.HistoryEvent{Type: domain.EventPreflightStarted, Timestamp: preflightStart.UTC(), Message: "Preflight checks started"})
	o.progressHub.Broadcast(id, Event{Type: "step_started", Step: "preflight", StepTitle: "Running preflight checks", Progress: *progress, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	resp := preflight.Run(ctx, manifest, o.dnsService, o.reach, preflightTarget, preflight.RunOptions{ProvidedSecrets: providedSecrets})
	if !resp.OK && hasFailingPreflightCheck(resp, domain.PreflightDiskFreeID) {
		if o.tryAutoVPSBundleGC(ctx, id, manifest) {
			resp = preflight.Run(ctx, manifest, o.dnsService, o.reach, preflightTarget, preflight.RunOptions{ProvidedSecrets: providedSecrets})
		}
	}
	preflightJSON, _ := json.Marshal(resp)
	if err := o.repo.UpdateDeploymentPreflightResult(ctx, id, preflightJSON); err != nil {
		o.log("failed to save preflight result", map[string]interface{}{"error": err.Error()})
	}
	o.progressHub.Broadcast(id, Event{Type: "preflight_result", Step: "preflight", StepTitle: "Running preflight checks", Progress: *progress, PreflightResult: &resp, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	if !resp.OK {
		failCount := 0
		for _, check := range resp.Checks {
			if check.Status == domain.PreflightFail {
				failCount++
			}
		}
		errMsg := "Preflight checks failed"
		if failCount > 0 {
			errMsg = fmt.Sprintf("Preflight checks failed (%d issue%s)", failCount, pluralize(failCount))
		}
		o.appendHistoryEvent(ctx, id, domain.HistoryEvent{Type: domain.EventPreflightCompleted, Timestamp: time.Now().UTC(), Message: errMsg, Details: FormatPreflightFailureDetails(resp), DurationMs: time.Since(preflightStart).Milliseconds(), Success: boolPtr(false)})
		emitError("preflight", "Running preflight checks", errMsg)
		return apierrors.New(apierrors.CodeInvalidRequest, errMsg).WithDetail("step", "preflight")
	}
	o.appendHistoryEvent(ctx, id, domain.HistoryEvent{Type: domain.EventPreflightCompleted, Timestamp: time.Now().UTC(), Message: "Preflight checks passed", DurationMs: time.Since(preflightStart).Milliseconds(), Success: boolPtr(true)})
	*progress += vps.StepWeights["preflight"]
	o.progressHub.Broadcast(id, Event{Type: "step_completed", Step: "preflight", StepTitle: "Running preflight checks", Progress: *progress, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	return nil
}

// ---- Projection -----------------------------------------------------------

var _ operations.Projection = (*Orchestrator)(nil)

// OperationStarted projects "executing" onto the deployment record.
func (o *Orchestrator) OperationStarted(ctx context.Context, op *domain.CloudOperation, plan *execplan.Plan) {
	status := domain.StatusSetupRunning
	if plan != nil && (plan.Scope == execplan.ScopeRuntime || plan.Scope == execplan.ScopeStart) {
		status = domain.StatusDeploying
	}
	if err := o.repo.BeginDeploymentRun(ctx, op.DeploymentID, status); err != nil {
		o.log("failed to project operation start", map[string]interface{}{"deployment_id": op.DeploymentID, "error": err.Error()})
	}
}

// OperationFinished projects the terminal operation state onto the
// deployment status, history and the SSE hub. The projection is derived
// from the operation record and never the other way round.
func (o *Orchestrator) OperationFinished(ctx context.Context, op *domain.CloudOperation, plan *execplan.Plan, to operations.State, result *operations.Result, failure *apierrors.Error) {
	now := time.Now().UTC()
	step := ""
	if marker := op.ActiveStepMarker(); marker != nil {
		step = marker.Step
	}
	if receipts, _ := op.Receipts(); step == "" && len(receipts) > 0 {
		last := receipts[len(receipts)-1]
		if last.Outcome == operations.StepFailed {
			step = last.Step
		}
	}
	message := ""
	if result != nil {
		message = result.Message
	}
	switch to {
	case operations.Succeeded:
		dep, _ := o.repo.GetDeployment(ctx, op.DeploymentID)
		if dep != nil && dep.DeployResult.Valid {
			_ = o.repo.UpdateDeploymentDeployResult(ctx, op.DeploymentID, dep.DeployResult.Data, true)
		} else {
			_ = o.repo.UpdateDeploymentStatus(ctx, op.DeploymentID, domain.StatusDeployed, nil, nil)
		}
		o.appendHistoryEvent(ctx, op.DeploymentID, domain.HistoryEvent{Type: domain.EventDeployCompleted, Timestamp: now, Message: "Operation " + op.ID + " succeeded", Success: boolPtr(true)})
		o.progressHub.Broadcast(op.DeploymentID, Event{Type: "completed", Progress: 100, Message: "Deployment successful", Timestamp: now.Format(time.RFC3339)})
	case operations.Cancelled:
		msg := "operation cancelled"
		if message != "" {
			msg = message
		}
		_ = o.repo.UpdateDeploymentStatus(ctx, op.DeploymentID, domain.StatusFailed, &msg, stringPtrOrNil(step))
		o.appendHistoryEvent(ctx, op.DeploymentID, domain.HistoryEvent{Type: domain.EventDeployFailed, Timestamp: now, Message: "Operation " + op.ID + " cancelled", Details: msg, Success: boolPtr(false), StepName: step})
		o.progressHub.Broadcast(op.DeploymentID, Event{Type: "deployment_error", Step: step, Error: msg, Timestamp: now.Format(time.RFC3339)})
	default: // failed, failed_recovery
		msg := "operation failed"
		if failure != nil {
			msg = failure.Message
		} else if message != "" {
			msg = message
		}
		if to == operations.FailedRecovery {
			msg = "recovery failed: " + msg
		}
		_ = o.repo.UpdateDeploymentStatus(ctx, op.DeploymentID, domain.StatusFailed, &msg, stringPtrOrNil(step))
		o.appendHistoryEvent(ctx, op.DeploymentID, domain.HistoryEvent{Type: domain.EventDeployFailed, Timestamp: now, Message: "Operation " + op.ID + " " + string(to), Details: msg, Success: boolPtr(false), StepName: step})
		o.progressHub.Broadcast(op.DeploymentID, Event{Type: "deployment_error", Step: step, Error: msg, Timestamp: now.Format(time.RFC3339)})
	}
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ParseTargetReceipt decodes the JSON the cloud-target verb prints.
func ParseTargetReceipt(stdout string, exitCode int, runErr error) (operations.TargetReceipt, error) {
	var payload struct {
		SchemaVersion int            `json:"schema_version"`
		Outcome       string         `json:"outcome"`
		Fence         uint64         `json:"fence"`
		Details       map[string]any `json:"details"`
		Error         *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		if runErr != nil {
			return operations.TargetReceipt{}, runErr
		}
		return operations.TargetReceipt{}, fmt.Errorf("empty receipt reply")
	}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return operations.TargetReceipt{}, fmt.Errorf("receipt reply is not JSON: %w", err)
	}
	if payload.SchemaVersion == 0 {
		if payload.Error != nil && payload.Error.Code == "receipt_not_found" {
			return operations.TargetReceipt{Found: false}, nil
		}
		if payload.Error != nil {
			return operations.TargetReceipt{}, fmt.Errorf("target refused receipt read: %s: %s", payload.Error.Code, payload.Error.Message)
		}
		if runErr != nil {
			return operations.TargetReceipt{}, runErr
		}
		return operations.TargetReceipt{Found: false}, nil
	}
	out := operations.TargetReceipt{Found: true, Fence: payload.Fence}
	switch payload.Outcome {
	case "succeeded":
		out.Outcome = operations.StepSucceeded
	case "unchanged":
		out.Outcome = operations.StepUnchanged
	case "failed":
		out.Outcome = operations.StepFailed
	default:
		out.Outcome = operations.StepUnknown
	}
	if payload.Error != nil {
		out.Error = payload.Error.Code + ": " + payload.Error.Message
	}
	if payload.Details != nil {
		if b, err := json.Marshal(payload.Details); err == nil {
			out.Detail = string(b)
		}
	}
	_ = exitCode
	return out, nil
}

func safeIdentifier(v string) bool {
	if v == "" || len(v) > 128 {
		return false
	}
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

// FenceArgs renders the identity flags every fenced target verb carries.
func FenceArgs(op *domain.CloudOperation, step string, fence uint64) []string {
	return []string{"--operation", op.ID, "--step", step, "--fence", strconv.FormatUint(fence, 10)}
}
