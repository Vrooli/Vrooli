package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	"scenario-to-cloud/sshidentity"
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
	dep, manifest, canonicalIdentity, err := o.loadOperationManifest(ctx, op)
	if err != nil {
		return err
	}
	progress := 0.0
	emitError := func(step, stepTitle, errMsg string) {
		o.progressHub.Broadcast(dep.ID, Event{Type: "deployment_error", Step: step, StepTitle: stepTitle, Progress: progress, Error: errMsg, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	}
	o.logOperationStarted(dep, op, plan, ec)
	if err := o.prepareOperationManifest(ctx, dep, &manifest, ec.Options.ProvidedSecrets, emitError); err != nil {
		return err
	}
	bundlePath, bundleSize, err := o.buildOperationBundle(ctx, dep, manifest, plan, ec.Options, emitError)
	if err != nil {
		return err
	}
	if ec.Options.RunPreflight && !ec.Resumed {
		if err := o.runPreflightStage(ctx, dep.ID, manifest, ec.Options.ProvidedSecrets, bundleSize, &progress, emitError); err != nil {
			return err
		}
	}
	trace, execErr, stageStart := o.executeOperationPlan(ctx, ec, dep, manifest, bundlePath, &progress)
	return o.persistOperationResult(ctx, dep, plan, manifest, canonicalIdentity, trace, execErr, stageStart)
}

func (o *Orchestrator) loadOperationManifest(ctx context.Context, op *domain.CloudOperation) (*domain.Deployment, domain.CloudManifest, sshidentity.DeploymentSSHIdentity, error) {
	dep, err := o.repo.GetDeployment(ctx, op.DeploymentID)
	if err != nil {
		return nil, domain.CloudManifest{}, sshidentity.DeploymentSSHIdentity{}, fmt.Errorf("load deployment: %w", err)
	}
	if dep == nil {
		return nil, domain.CloudManifest{}, sshidentity.DeploymentSSHIdentity{}, apierrors.New(apierrors.CodeDeploymentNotFound, "Deployment of the operation no longer exists").WithDetail("deployment_id", op.DeploymentID)
	}
	var manifest domain.CloudManifest
	if err := json.Unmarshal(dep.Manifest, &manifest); err != nil {
		return nil, domain.CloudManifest{}, sshidentity.DeploymentSSHIdentity{}, apierrors.Internal("Deployment manifest is not decodable", err)
	}
	canonicalIdentity, manifest := o.resolveAndPersistIdentity(ctx, dep.ID, manifest)
	return dep, manifest, canonicalIdentity, nil
}

func (o *Orchestrator) logOperationStarted(dep *domain.Deployment, op *domain.CloudOperation, plan *execplan.Plan, ec *operations.ExecutionContext) {
	o.log("operation attempt started", map[string]interface{}{
		"deployment_id": dep.ID, "operation_id": op.ID, "fence": ec.Fence, "scope": plan.Scope, "resumed": ec.Resumed, "plan_digest": op.PlanDigest,
	})
}

func (o *Orchestrator) prepareOperationManifest(ctx context.Context, dep *domain.Deployment, manifest *domain.CloudManifest, provided map[string]string, emitError func(step, stepTitle, errMsg string)) error {
	if err := o.ensureSecretsAvailable(ctx, manifest, provided, dep.ID, emitError); err != nil {
		return missingOperationSecrets(*manifest, provided, dep.ID, err)
	}
	if manifest.Secrets != nil {
		if hydrated, marshalErr := json.Marshal(manifest); marshalErr == nil {
			if persistErr := o.repo.UpdateDeploymentManifest(ctx, dep.ID, hydrated); persistErr != nil {
				o.log("failed to persist hydrated deployment manifest", map[string]interface{}{"deployment_id": dep.ID, "error": persistErr.Error()})
			}
		}
	}
	return nil
}

func missingOperationSecrets(manifest domain.CloudManifest, provided map[string]string, deploymentID string, original error) error {
	missing, err := vps.ValidateUserPromptSecrets(manifest, provided)
	if err != nil && len(missing) > 0 {
		names := make([]string, 0, len(missing))
		for _, secret := range missing {
			names = append(names, secret.Key)
		}
		typed := apierrors.New(apierrors.CodeNeedsInput, "Operator-supplied secrets are required to continue").
			WithDetail("missing", names).
			WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "resume", Reference: "/api/v1/deployments/" + deploymentID + "/execute", Label: "Resubmit with the missing secrets under the same request key"})
		return operations.WaitingInputError(typed)
	}
	return original
}

func (o *Orchestrator) buildOperationBundle(ctx context.Context, dep *domain.Deployment, manifest domain.CloudManifest, plan *execplan.Plan, opts operations.ExecuteOptions, emitError func(step, stepTitle, errMsg string)) (string, int64, error) {
	if plan.Scope != execplan.ScopeFull && plan.Scope != execplan.ScopeInstall {
		return "", 0, nil
	}
	bundlePath, err := o.ensureBundleBuilt(ctx, manifest, dep.BundlePath, opts.ForceBundleBuild, dep.ID, emitError)
	if err != nil {
		return "", 0, err
	}
	var bundleSize int64
	if info, statErr := os.Stat(bundlePath); statErr == nil {
		bundleSize = info.Size()
	}
	return bundlePath, bundleSize, nil
}

func (o *Orchestrator) executeOperationPlan(ctx context.Context, ec *operations.ExecutionContext, dep *domain.Deployment, manifest domain.CloudManifest, bundlePath string, progress *float64) (vps.Trace, *vps.ExecutionError, time.Time) {
	hubAdapter := &progressHubAdapter{hub: o.progressHub}
	repoAdapter := &progressRepoAdapter{repo: o.repo}
	hooks := executionHooks(ec)
	stageStart := time.Now()
	o.appendHistoryEvent(ctx, dep.ID, domain.HistoryEvent{Type: domain.EventDeployStarted, Timestamp: stageStart.UTC(), Message: fmt.Sprintf("Operation %s started (%s)", ec.Operation.ID, ec.Plan.Scope)})
	var weights map[string]float64
	if ec.Plan.Scope == execplan.ScopeStart {
		weights = vps.CalculateWeightsForSteps(vps.StartSteps)
	}
	runtime := o.operationRuntime(dep, manifest, ec)
	trace, err := vps.ExecutePlan(ctx, vps.ExecuteRequest{
		Plan: ec.Plan, Manifest: manifest, BundlePath: bundlePath, Runtime: runtime,
		Hub: hubAdapter, Repo: repoAdapter, DeploymentID: dep.ID,
		Progress: progress, Weights: weights, Hooks: hooks,
	})
	return trace, err, stageStart
}

func executionHooks(ec *operations.ExecutionContext) vps.Hooks {
	return vps.Hooks{
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
}

func (o *Orchestrator) operationRuntime(dep *domain.Deployment, manifest domain.CloudManifest, ec *operations.ExecutionContext) vps.Runtime {
	runtime := vps.Runtime{
		Reach: o.reach, Target: targetFor(dep, manifest), Identity: vps.Identity{OperationID: ec.Operation.ID, Fence: ec.Fence},
		Backups: o.backups, ManagedDNS: o.managedDNS, SecretsGen: o.secretsGenerator,
		ProvidedSecrets: ec.Options.ProvidedSecrets, LocalHandoff: o.localCredentialHandoff,
	}
	if o.credentials != nil {
		runtime.Credentials = &credentialProvisioner{bind: o.credentials, store: o.repo}
	}
	return runtime
}

func (o *Orchestrator) persistOperationResult(ctx context.Context, dep *domain.Deployment, plan *execplan.Plan, manifest domain.CloudManifest, canonicalIdentity sshidentity.DeploymentSSHIdentity, trace vps.Trace, execErr *vps.ExecutionError, stageStart time.Time) error {
	digest, _ := plan.SemanticDigest()
	result := domain.VPSDeployResult{OK: execErr == nil, PlanDigest: digest, Actions: trace.Actions, DurationMs: time.Since(stageStart).Milliseconds(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	if execErr != nil {
		result.Error = execErr.Error()
		result.ErrorInfo = execErr.Info
		result.FailedStep = execErr.ActionID
	}
	resultJSON, _ := json.Marshal(result)
	if err := o.repo.UpdateDeploymentDeployResult(ctx, dep.ID, resultJSON); err != nil {
		o.log("failed to save operation result", map[string]interface{}{"error": err.Error()})
	}
	o.enforceVPSBundleRetentionBestEffort(ctx, dep.ID, manifest)
	if execErr == nil {
		o.verifyAndPersistIdentity(ctx, dep.ID, manifest, canonicalIdentity)
		return nil
	}
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

func (o *Orchestrator) runPreflightStage(ctx context.Context, id string, manifest domain.CloudManifest, providedSecrets map[string]string, bundleSizeBytes int64, progress *float64, emitError func(step, stepTitle, errMsg string)) error {
	preflightStart := time.Now()
	preflightTarget := o.targetFor(ctx, id, manifest)
	o.appendHistoryEvent(ctx, id, domain.HistoryEvent{Type: domain.EventPreflightStarted, Timestamp: preflightStart.UTC(), Message: "Preflight checks started"})
	o.progressHub.Broadcast(id, Event{Type: "step_started", Step: "preflight", StepTitle: "Running preflight checks", Progress: *progress, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	preflightOptions := preflight.RunOptions{ProvidedSecrets: providedSecrets, BundleSizeBytes: bundleSizeBytes}
	resp := preflight.Run(ctx, manifest, o.dnsService, o.reach, preflightTarget, preflightOptions)
	resp = o.recoverPreflightDiskPressure(ctx, id, manifest, preflightTarget, preflightOptions, resp)
	preflightJSON, _ := json.Marshal(resp)
	if err := o.repo.UpdateDeploymentPreflightResult(ctx, id, preflightJSON); err != nil {
		o.log("failed to save preflight result", map[string]interface{}{"error": err.Error()})
	}
	o.progressHub.Broadcast(id, Event{Type: "preflight_result", Step: "preflight", StepTitle: "Running preflight checks", Progress: *progress, PreflightResult: &resp, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	if !resp.OK {
		return o.failPreflight(ctx, id, resp, preflightStart, emitError)
	}
	o.appendHistoryEvent(ctx, id, domain.HistoryEvent{Type: domain.EventPreflightCompleted, Timestamp: time.Now().UTC(), Message: "Preflight checks passed", DurationMs: time.Since(preflightStart).Milliseconds(), Success: boolPtr(true)})
	*progress += vps.StepWeights["preflight"]
	o.progressHub.Broadcast(id, Event{Type: "step_completed", Step: "preflight", StepTitle: "Running preflight checks", Progress: *progress, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	return nil
}

func (o *Orchestrator) recoverPreflightDiskPressure(ctx context.Context, id string, manifest domain.CloudManifest, target identity.TargetRef, options preflight.RunOptions, response domain.PreflightResponse) domain.PreflightResponse {
	if response.OK || !hasFailingPreflightCheck(response, domain.PreflightDiskFreeID) {
		return response
	}
	cleanupAttempted, cleanupSucceeded := o.cleanupPreflightDisk(ctx, id, target, response)
	if o.tryAutoVPSBundleGC(ctx, id, manifest) || cleanupAttempted {
		response = preflight.Run(ctx, manifest, o.dnsService, o.reach, target, options)
	}
	if cleanupSucceeded && diskDeficitWithinTolerance(response, 256*1024) {
		response = o.markPreflightDiskWarning(ctx, id, response)
	}
	return response
}

func (o *Orchestrator) cleanupPreflightDisk(ctx context.Context, id string, target identity.TargetRef, response domain.PreflightResponse) (bool, bool) {
	disk := preflightDiskFacts(response)
	if disk.availableKB < 0 {
		return false, false
	}
	cleanup := preflight.CleanupDiskPressure(ctx, o.reach, target, disk.availableKB, disk.requiredKB)
	attempted, succeeded := len(cleanup.Attempted) > 0, len(cleanup.Succeeded) > 0
	if attempted {
		o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
			Type: domain.EventVPSBundleGC, Timestamp: time.Now().UTC(),
			Message: "Adaptive VPS disk cleanup attempted before release GC",
			Details: fmt.Sprintf("attempted=%v succeeded=%v errors=%v", cleanup.Attempted, cleanup.Succeeded, cleanup.Errors),
			Success: boolPtr(succeeded), StepName: "preflight",
		})
	}
	return attempted, succeeded
}

func (o *Orchestrator) markPreflightDiskWarning(ctx context.Context, id string, response domain.PreflightResponse) domain.PreflightResponse {
	for i := range response.Checks {
		if response.Checks[i].ID == domain.PreflightDiskFreeID && response.Checks[i].Status == domain.PreflightFail {
			response.Checks[i].Status = domain.PreflightWarn
			response.Checks[i].Details += "; continuing after bounded cleanup because the remaining deficit is within 256 MB"
		}
	}
	response.OK = true
	o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
		Type: domain.EventVPSBundleGC, Timestamp: time.Now().UTC(),
		Message: "Deployment continued after bounded disk cleanup",
		Details: "remaining disk deficit was within 256 MB and no other preflight check failed",
		Success: boolPtr(true), StepName: "preflight",
	})
	return response
}

func (o *Orchestrator) failPreflight(ctx context.Context, id string, response domain.PreflightResponse, preflightStart time.Time, emitError func(step, stepTitle, errMsg string)) error {
	failCount := preflightFailureCount(response)
	errMsg := "Preflight checks failed"
	if failCount > 0 {
		errMsg = fmt.Sprintf("Preflight checks failed (%d issue%s)", failCount, pluralize(failCount))
	}
	o.appendHistoryEvent(ctx, id, domain.HistoryEvent{Type: domain.EventPreflightCompleted, Timestamp: time.Now().UTC(), Message: errMsg, Details: FormatPreflightFailureDetails(response), DurationMs: time.Since(preflightStart).Milliseconds(), Success: boolPtr(false)})
	emitError("preflight", "Running preflight checks", errMsg)
	return apierrors.New(apierrors.CodeInvalidRequest, errMsg).WithDetail("step", "preflight")
}

func preflightFailureCount(response domain.PreflightResponse) int {
	count := 0
	for _, check := range response.Checks {
		if check.Status == domain.PreflightFail {
			count++
		}
	}
	return count
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
	step := operationTerminalStep(op)
	message := operationResultMessage(result)
	switch to {
	case operations.Succeeded:
		o.projectOperationSuccess(ctx, op, now)
	case operations.Cancelled:
		o.projectOperationCancellation(ctx, op, step, message, now)
	default:
		o.projectOperationFailure(ctx, op, to, step, message, failure, now)
	}
}

func operationTerminalStep(op *domain.CloudOperation) string {
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
	return step
}

func operationResultMessage(result *operations.Result) string {
	if result == nil {
		return ""
	}
	return result.Message
}

func (o *Orchestrator) projectOperationSuccess(ctx context.Context, op *domain.CloudOperation, now time.Time) {
	dep, _ := o.repo.GetDeployment(ctx, op.DeploymentID)
	if dep != nil && dep.DeployResult.Valid {
		_ = o.repo.UpdateDeploymentDeployResult(ctx, op.DeploymentID, dep.DeployResult.Data)
	}
	_ = o.repo.UpdateDeploymentStatus(ctx, op.DeploymentID, domain.StatusDeployed, nil, nil)
	o.appendHistoryEvent(ctx, op.DeploymentID, domain.HistoryEvent{Type: domain.EventDeployCompleted, Timestamp: now, Message: "Operation " + op.ID + " succeeded", Success: boolPtr(true)})
	o.progressHub.Broadcast(op.DeploymentID, Event{Type: "completed", Progress: 100, Message: "Deployment successful", Timestamp: now.Format(time.RFC3339)})
}

func (o *Orchestrator) projectOperationCancellation(ctx context.Context, op *domain.CloudOperation, step, message string, now time.Time) {
	msg := message
	if msg == "" {
		msg = "operation cancelled"
	}
	_ = o.repo.UpdateDeploymentStatus(ctx, op.DeploymentID, domain.StatusFailed, &msg, stringPtrOrNil(step))
	o.appendHistoryEvent(ctx, op.DeploymentID, domain.HistoryEvent{Type: domain.EventDeployFailed, Timestamp: now, Message: "Operation " + op.ID + " cancelled", Details: msg, Success: boolPtr(false), StepName: step})
	o.progressHub.Broadcast(op.DeploymentID, Event{Type: "deployment_error", Step: step, Error: msg, Timestamp: now.Format(time.RFC3339)})
}

func (o *Orchestrator) projectOperationFailure(ctx context.Context, op *domain.CloudOperation, to operations.State, step, message string, failure *apierrors.Error, now time.Time) {
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
		return emptyReceiptReply(runErr)
	}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return operations.TargetReceipt{}, fmt.Errorf("receipt reply is not JSON: %w", err)
	}
	if payload.SchemaVersion == 0 {
		return missingTargetReceipt(payload.Error, runErr)
	}
	out := operations.TargetReceipt{Found: true, Fence: payload.Fence, Outcome: targetReceiptOutcome(payload.Outcome)}
	attachTargetReceiptError(&out, payload.Error)
	attachTargetReceiptDetails(&out, payload.Details)
	_ = exitCode
	return out, nil
}

func emptyReceiptReply(runErr error) (operations.TargetReceipt, error) {
	if runErr != nil {
		return operations.TargetReceipt{}, runErr
	}
	return operations.TargetReceipt{}, fmt.Errorf("empty receipt reply")
}

func missingTargetReceipt(targetErr *struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}, runErr error) (operations.TargetReceipt, error) {
	if targetErr != nil && targetErr.Code == "receipt_not_found" {
		return operations.TargetReceipt{Found: false}, nil
	}
	if targetErr != nil {
		return operations.TargetReceipt{}, fmt.Errorf("target refused receipt read: %s: %s", targetErr.Code, targetErr.Message)
	}
	if runErr != nil {
		return operations.TargetReceipt{}, runErr
	}
	return operations.TargetReceipt{Found: false}, nil
}

func targetReceiptOutcome(value string) operations.StepOutcome {
	switch value {
	case "succeeded":
		return operations.StepSucceeded
	case "unchanged":
		return operations.StepUnchanged
	case "failed":
		return operations.StepFailed
	default:
		return operations.StepUnknown
	}
}

func attachTargetReceiptError(out *operations.TargetReceipt, targetErr *struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}) {
	if targetErr != nil {
		out.Error = targetErr.Code + ": " + targetErr.Message
	}
}

func attachTargetReceiptDetails(out *operations.TargetReceipt, details map[string]any) {
	if details != nil {
		if b, err := json.Marshal(details); err == nil {
			out.Detail = string(b)
		}
	}
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
