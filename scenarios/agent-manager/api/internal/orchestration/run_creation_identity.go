// Responsibility: verify caller identity, exact replay and dependent admission before creation effects.
package orchestration

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	"slices"
	"strings"
)

// buildRunAdmission captures the immutable requested-versus-effective
// configuration record for a newly admitted run. Requested values come from the
// caller request before resolution; effective values mirror the resolved config
// and its pinned policy snapshot. Unrequested fields stay empty rather than
// being backfilled with defaults, so a reader can distinguish "not requested"
// from "requested and resolved".
// resolveCreateRunIdentity authenticates before any reservation or dispatch.
// Narrowing is never accepted as a substitute for verified authority.
func (o *Orchestrator) resolveCreateRunIdentity(ctx context.Context, req *CreateRunRequest) error {
	if strings.TrimSpace(req.OwnerToken) != "" {
		if o.ownerIdentity == nil {
			return domain.NewConfigMissingError("owner_identity", "verifier not configured", nil)
		}
		owner, err := o.ownerIdentity.Verify(ctx, req.OwnerToken)
		if err != nil || !owner.Verified || owner.Kind != coreidentity.ActorHuman || strings.TrimSpace(owner.Subject) == "" || owner.ExpiresAt.IsZero() || !owner.ExpiresAt.After(o.now()) {
			return domain.NewValidationErrorWithCode("authorization", "owner credential is invalid, expired or unavailable", domain.ErrCodePolicyScope)
		}
		req.OwnerSubject = strings.TrimSpace(owner.Subject)
		req.OwnerScopes = append([]string{}, owner.Scopes...)
		expires := owner.ExpiresAt
		req.OwnerExpiresAt = &expires
	}
	return narrowCreateRunScopes(req)
}

func narrowCreateRunScopes(req *CreateRunRequest) error {
	if req.ExpectedOwnerSubject != "" && req.ExpectedOwnerSubject != req.OwnerSubject {
		return domain.NewValidationErrorWithCode("authorization", "verified owner does not match the requested owner", domain.ErrCodePolicyScope)
	}
	if req.RequestedScopes == nil {
		return nil
	}
	if (req.OwnerSubject == "" && (req.caller == nil || req.caller.Kind != "run")) || req.OwnerScopes == nil {
		return domain.NewValidationErrorWithCode("authorization", "scope narrowing requires verified owner authority", domain.ErrCodePolicyScope)
	}
	if len(req.RequestedScopes) > 100 {
		return domain.NewValidationError("requested_scopes", "at most 100 scopes are allowed")
	}
	for _, scope := range req.RequestedScopes {
		if scope != strings.TrimSpace(scope) || scope == "" || len(scope) > 256 || scopecatalog.IsWildcard(scope) || !scopecatalog.MatchCapability(req.OwnerScopes, scope) {
			return domain.NewValidationErrorWithCode("requested_scopes", "requested scope is not an exact capability held by the owner", domain.ErrCodePolicyScope)
		}
	}
	req.RequestedScopes = identity.IntersectScopes(req.OwnerScopes, nil, req.RequestedScopes)
	return nil
}

func (o *Orchestrator) getIdentityBoundRunReplay(ctx context.Context, id uuid.UUID, req CreateRunRequest) (*domain.Run, error) {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := validateRunIdentityReplay(run, req); err != nil {
		return nil, err
	}
	if err := o.acceptEffortRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func validateRunIdentityReplay(run *domain.Run, req CreateRunRequest) error {
	if run == nil {
		return domain.NewValidationError("idempotency_key", "original run unavailable")
	}
	if req.effort != nil {
		if run.ResolvedConfig == nil || run.ResolvedConfig.Admission == nil || run.ResolvedConfig.Admission.Effort == nil || *run.ResolvedConfig.Admission.Effort != req.effort.binding {
			return effortauthority.ErrRefused
		}
		if req.effort.proof != nil && (run.ResolvedConfig.Admission.EffortIntent == nil || effortauthority.Digest(*run.ResolvedConfig.Admission.EffortIntent) != effortauthority.Digest(req.effort.proof.Intent)) {
			return effortauthority.ErrRefused
		}
	}
	if req.caller != nil {
		if run.ResolvedConfig == nil || run.ResolvedConfig.Admission == nil || run.ResolvedConfig.Admission.CreateCaller == nil || *run.ResolvedConfig.Admission.CreateCaller != *req.caller {
			return domain.NewValidationErrorWithCode("idempotency_key", "original admission has no matching verified caller", domain.ErrCodePolicyScope)
		}
		if run.TaskID != req.TaskID || !sameCreateRunScopeCeiling(run.OwnerScopes, req.OwnerScopes) {
			return domain.NewValidationErrorWithCode("idempotency_key", "task or verified scope ceiling differs from original admission", domain.ErrCodePolicyScope)
		}
		if (run.ParentRunID == nil) != (req.ParentRunID == nil) || (run.ParentRunID != nil && *run.ParentRunID != *req.ParentRunID) {
			return domain.NewValidationErrorWithCode("idempotency_key", "parent differs from original admission", domain.ErrCodePolicyScope)
		}
	}
	// Historical unauthenticated requests persisted absent narrowing as [].
	// They carry no owner grant; keep their ordinary recovery compatible.
	if run.OwnerSubject == "" && req.OwnerSubject == "" && req.RequestedScopes == nil {
		return nil
	}
	if run.OwnerSubject != req.OwnerSubject || (run.RequestedScopes == nil) != (req.RequestedScopes == nil) || !slices.Equal(run.RequestedScopes, req.RequestedScopes) {
		return domain.NewValidationErrorWithCode("idempotency_key", "run identity differs from the original dispatch", domain.ErrCodePolicyScope)
	}
	return nil
}

// Scope order/duplicates are immaterial; a renewed credential may not widen or
// reduce the admitted grant through replay. Original expiry is never renewed.
func sameCreateRunScopeCeiling(a, b []string) bool {
	left, right := slices.Clone(a), slices.Clone(b)
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(slices.Compact(left), slices.Compact(right))
}

func buildRunAdmission(req CreateRunRequest, cfg *domain.RunConfig) *domain.RunAdmission {
	if cfg == nil {
		return nil
	}
	admission := &domain.RunAdmission{
		RequestedRunner:   strings.TrimSpace(req.PreferredRunner),
		EffectiveRunner:   string(cfg.RunnerType),
		EffectiveModel:    cfg.Model,
		EffectiveEffort:   string(cfg.Effort),
		EffectiveTimeout:  cfg.Timeout,
		EffectiveMaxTurns: cfg.MaxTurns,
		EffectiveUntil:    cfg.Until,
	}
	if req.caller != nil {
		copy := *req.caller
		admission.CreateCaller = &copy
		if req.effort != nil {
			b := req.effort.binding
			i := req.effort.intent
			admission.Effort = &b
			admission.EffortIntent = &i
		}
	}
	if req.RoleRef != nil {
		admission.RequestedRoleRef = strings.TrimSpace(*req.RoleRef)
	}
	if req.Model != nil {
		admission.RequestedModel = strings.TrimSpace(*req.Model)
	}
	if req.Effort != nil {
		admission.RequestedEffort = string(*req.Effort)
	}
	if req.Timeout != nil {
		admission.RequestedTimeout = *req.Timeout
	}
	if req.MaxTurns != nil {
		admission.RequestedMaxTurns = *req.MaxTurns
	}
	if strings.TrimSpace(req.Until) != "" {
		admission.RequestedGoalMode = "until"
	}
	if cfg.PolicySnapshot != nil {
		admission.CatalogDigest = cfg.PolicySnapshot.CatalogDigest
		admission.PolicyDigest = cfg.PolicySnapshot.SelectedCandidate.PolicyDigest
		admission.PolicyPath = cfg.PolicySnapshot.SelectedCandidate.PolicyPath
		admission.SelectionReason = cfg.PolicySnapshot.SelectionReason
	}
	return admission
}

// admitDependentDelegation enforces the Agent Manager-owned qualification gate
// at the dependent-delegation admission point. A run without a parent is not
// dependent delegation and is admitted unchanged. A child run is admitted only
// when its parent's observed runner/model/effort match and execution
// restrictions are retained. A model receipt never authorizes policy changes.
func (o *Orchestrator) admitDependentDelegation(ctx context.Context, req CreateRunRequest, cfg *domain.RunConfig) error {
	if req.ParentRunID == nil {
		return nil
	}
	if o.runs == nil {
		return domain.NewConfigMissingError("runs", "run repository not configured", nil)
	}
	parent, err := o.runs.Get(ctx, *req.ParentRunID)
	if err != nil {
		return domain.NewValidationErrorWithHint("parentRunId",
			"parent run could not be read for dependent-delegation admission: "+err.Error(),
			"retry after the parent run is durable")
	}
	return dependentDelegationAdmissionError(parent, cfg)
}

// dependentDelegationAdmissionError is the pure gate decision: it reads the
// parent's persisted execution policy and qualification against the child's
// resolved configuration. Admission and focused tests use the same decision.
func dependentDelegationAdmissionError(parent *domain.Run, cfg *domain.RunConfig) error {
	if parent == nil || parent.ResolvedConfig == nil || parent.ResolvedConfig.Admission == nil {
		return domain.NewValidationError("qualification", "dependent delegation is closed: parent run has no admission record")
	}
	if cfg == nil {
		return domain.NewValidationError("qualification", "dependent delegation is closed: resolved configuration is missing")
	}
	if err := dependentExecutionPermissionsError(parent.ResolvedConfig, cfg); err != nil {
		return err
	}
	req := domain.DependentDelegationRequest{
		Runner: string(cfg.RunnerType),
		Model:  cfg.Model,
		Effort: string(cfg.Effort),
	}
	if parent.ResolvedConfig.Admission.Receipt != nil {
		return domain.AdmitDependentDelegation(parent.ResolvedConfig.Admission.Receipt, req)
	}
	if parent.Status != domain.RunStatusStarting && parent.Status != domain.RunStatusRunning {
		return domain.NewValidationError("qualification", "dependent delegation is closed: completed parent has no live qualification receipt")
	}
	// A live coordinator must be able to delegate before its own terminal
	// seam can capture accepted output. Use the weaker launch-observed identity
	// check for that case; terminal parents retain the receipt gate above.
	return domain.AdmitLiveDependentDelegation(parent.ResolvedConfig.Admission, req)
}

// recordPassedInvocation records the runner-native control arguments the
// selected codec emits for a resolved configuration, plus the runner-observed
// runtime version. The control arguments are the "passed" layer between
// owner-resolved effective values and provider acknowledgment; the runtime
// version is live-only runtime identity. Both are evidence, never a launch: a
// missing registry or codec, or a codec that refuses the configuration, records
// a translation diagnostic instead of changing the creation outcome.
// Unsupported settings still fail at the runner boundary before a process
// starts; this method makes that refusal visible in the admission record instead
// of leaving the passed layer silently empty.
func (o *Orchestrator) recordPassedInvocation(ctx context.Context, admission *domain.RunAdmission, cfg *domain.RunConfig) {
	if admission == nil {
		return
	}
	if o == nil || o.runners == nil {
		admission.TranslationDiagnostics = append(admission.TranslationDiagnostics,
			"runner registry unavailable; passed control arguments not captured")
		return
	}
	selected, err := o.runners.Get(cfg.RunnerType)
	if err != nil {
		admission.TranslationDiagnostics = append(admission.TranslationDiagnostics,
			fmt.Sprintf("runner %q unavailable; passed control arguments not captured: %v", cfg.RunnerType, err))
		return
	}
	// The runtime version is live-only identity the qualification receipt
	// requires; capture it whenever the runner resolves, independently of
	// whether control translation succeeds.
	o.recordRuntimeVersion(ctx, admission, selected)
	info, ok := selected.(runner.AgentLaunchInfo)
	if !ok {
		admission.TranslationDiagnostics = append(admission.TranslationDiagnostics,
			fmt.Sprintf("runner %q does not expose control translation; passed control arguments not captured", cfg.RunnerType))
		return
	}
	args, err := info.ControlArgs(cfg)
	if err != nil {
		admission.TranslationDiagnostics = append(admission.TranslationDiagnostics,
			"control translation refused: "+err.Error())
		return
	}
	admission.PassedControlArgs = append(admission.PassedControlArgs, args...)
	if len(args) == 0 {
		admission.TranslationDiagnostics = append(admission.TranslationDiagnostics,
			"no runner-native control arguments emitted")
	}
}

// recordRuntimeVersion captures the concrete CLI runtime version observed for
// the selected runner into the admission record. It is the live-only identity
// the qualification receipt requires, so an unobserved value stays empty and
// is explained by a translation diagnostic rather than backfilled.
func (o *Orchestrator) recordRuntimeVersion(ctx context.Context, admission *domain.RunAdmission, selected runner.Runner) {
	reporter, ok := selected.(runner.RuntimeVersionReporter)
	if !ok {
		admission.TranslationDiagnostics = append(admission.TranslationDiagnostics,
			fmt.Sprintf("runner %q does not report a runtime version", selected.Type()))
		return
	}
	version, err := reporter.RuntimeVersion(ctx)
	if err != nil {
		admission.TranslationDiagnostics = append(admission.TranslationDiagnostics,
			"runtime version not observed: "+err.Error())
		return
	}
	admission.RuntimeVersion = strings.TrimSpace(version)
}
