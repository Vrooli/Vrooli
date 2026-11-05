// Responsibility: verify caller identity, exact replay and dependent admission before creation effects.
package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"

	"github.com/google/uuid"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
)

// createRunRequestDigest binds an idempotency key to the complete caller
// request. CreateRunRequest's JSON representation intentionally excludes
// credentials and verified identity projections; those are validated
// independently during replay. The digest is captured before admission adds
// derived values such as inherited parent work references.
func createRunRequestDigest(req CreateRunRequest) (string, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("encode create-run request identity: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// buildRunAdmission captures the immutable requested-versus-effective
// configuration record for a newly admitted run. Requested values come from the
// caller request before resolution; effective values mirror the resolved config
// and its pinned policy snapshot. Unrequested fields stay empty rather than
// being backfilled with defaults, so a reader can distinguish "not requested"
// from "requested and resolved".
// resolveCreateRunIdentity authenticates before any reservation or dispatch.
// Narrowing is never accepted as a substitute for verified authority.
func (o *Orchestrator) resolveCreateRunIdentity(ctx context.Context, req *CreateRunRequest) error {
	if owner, ok := coreidentity.PrincipalFromContext(ctx); ok && owner.IsHuman() {
		if owner.ExpiresAt.IsZero() || !owner.ExpiresAt.After(o.now()) {
			return domain.NewValidationErrorWithCode("authorization", "verified owner proof expired", domain.ErrCodePolicyScope)
		}
		req.OwnerSubject = owner.Subject
		req.OwnerScopes = append([]string(nil), owner.Scopes...)
		expires := owner.ExpiresAt
		req.OwnerExpiresAt = &expires
	} else if strings.TrimSpace(req.OwnerToken) != "" {
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
	return run, nil
}

func validateRunIdentityReplay(run *domain.Run, req CreateRunRequest) error {
	if run == nil {
		return domain.NewValidationError("idempotency_key", "original run unavailable")
	}
	// Rows written before request digests existed retain their historical
	// identity checks below. New rows require the same key to replay the exact
	// original request, not merely the same caller and owner scopes.
	if run.ResolvedConfig != nil && run.ResolvedConfig.Admission != nil && run.ResolvedConfig.Admission.CreateRequestDigest != "" {
		digest := req.requestDigest
		if digest == "" {
			var err error
			digest, err = createRunRequestDigest(req)
			if err != nil {
				return domain.NewIdempotencyReplayConflictError()
			}
		}
		if digest != run.ResolvedConfig.Admission.CreateRequestDigest {
			return domain.NewIdempotencyReplayConflictError()
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
		CreateRequestDigest: req.requestDigest,
		RequestedRunner:     strings.TrimSpace(req.PreferredRunner),
		EffectiveRunner:     string(cfg.RunnerType),
		EffectiveModel:      cfg.Model,
		EffectiveEffort:     string(cfg.Effort),
		EffectiveTimeout:    cfg.Timeout,
		EffectiveMaxTurns:   cfg.MaxTurns,
		EffectiveUntil:      cfg.Until,
	}
	if req.workflowIdentity != nil {
		copy := *req.workflowIdentity
		admission.WorkflowIdentity = &copy
	}
	if req.caller != nil {
		copy := *req.caller
		admission.CreateCaller = &copy
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
	_, err := o.dependentDelegationParent(ctx, req, cfg)
	return err
}

// dependentDelegationParent loads the parent's retained run snapshot and applies
// the existing dependent-execution gate. The qualified snapshot is also the
// source for child effort attribution; caller-supplied effort fields do not
// replace the parent's canonical reference terms.
func (o *Orchestrator) dependentDelegationParent(ctx context.Context, req CreateRunRequest, cfg *domain.RunConfig) (*domain.Run, error) {
	if req.ParentRunID == nil {
		return nil, nil
	}
	if o.runs == nil {
		return nil, domain.NewConfigMissingError("runs", "run repository not configured", nil)
	}
	parent, err := o.runs.Get(ctx, *req.ParentRunID)
	if err != nil {
		return nil, domain.NewValidationErrorWithHint("parentRunId",
			"parent run could not be read for dependent-delegation admission: "+err.Error(),
			"retry after the parent run is durable")
	}
	if err := dependentDelegationAdmissionError(parent, cfg); err != nil {
		return nil, err
	}
	return parent, nil
}

// inheritParentEffortReferences derives a child's effort role only from the
// exact parent snapshot that passed dependent-delegation admission. Other
// producer-neutral work references keep their existing caller declaration.
func inheritParentEffortReferences(parent *domain.Run, requested []*eventpb.WorkReference) ([]*eventpb.WorkReference, error) {
	if parent == nil {
		return requested, nil
	}

	byID := make(map[string]*eventpb.WorkReference)
	var parentOrder []string
	for _, ref := range parent.WorkReferences {
		commissionedWorker := ref != nil && ref.GetRelationship() == "worker" && parent.ResolvedConfig != nil && parent.ResolvedConfig.DeliveryCommissions[ref.GetId()] != ""
		if ref == nil || ref.GetKind() != "effort" || ref.GetId() == "" || (ref.GetRelationship() != "orchestrator" && !commissionedWorker) ||
			!ref.GetVerified() || ref.GetVisibility() != eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC ||
			ref.GetState() != eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE {
			continue
		}
		if existing, ok := byID[ref.GetId()]; ok {
			if !proto.Equal(existing, ref) {
				return nil, domain.NewValidationError("parentRunId", "parent has conflicting orchestrator effort references")
			}
			continue
		}
		byID[ref.GetId()] = proto.Clone(ref).(*eventpb.WorkReference)
		parentOrder = append(parentOrder, ref.GetId())
	}

	result := make([]*eventpb.WorkReference, 0, len(requested)+len(byID))
	emitted := make(map[string]bool, len(byID))
	for _, ref := range requested {
		if ref != nil && ref.GetKind() == "effort" {
			if canonical, ok := byID[ref.GetId()]; ok {
				if !compatibleChildEffortReference(ref, canonical) {
					return nil, domain.NewValidationErrorWithHint("work_references",
						"child effort reference conflicts with the qualified parent's worker attribution",
						"omit the inherited effort reference or use its worker role and matching source terms")
				}
				if !emitted[ref.GetId()] {
					worker := proto.Clone(canonical).(*eventpb.WorkReference)
					worker.Relationship = "worker"
					result = append(result, worker)
					emitted[ref.GetId()] = true
				}
				continue
			}
		}
		result = append(result, ref)
	}
	for _, id := range parentOrder {
		if emitted[id] {
			continue
		}
		worker := proto.Clone(byID[id]).(*eventpb.WorkReference)
		worker.Relationship = "worker"
		result = append(result, worker)
	}
	return result, nil
}

// A child may redundantly name an inherited effort only when it does not
// contradict the parent-derived worker form. Empty optional strings and
// unspecified enums are omitted terms; the persisted parent supplies them.
func compatibleChildEffortReference(candidate, parent *eventpb.WorkReference) bool {
	if candidate.GetRelationship() != "" && candidate.GetRelationship() != "worker" {
		return false
	}
	if candidate.GetRevision() != "" && candidate.GetRevision() != parent.GetRevision() {
		return false
	}
	if candidate.GetSourceEventId() != "" && candidate.GetSourceEventId() != parent.GetSourceEventId() {
		return false
	}
	if candidate.GetSourceRunId() != "" && candidate.GetSourceRunId() != parent.GetSourceRunId() {
		return false
	}
	if candidate.GetEvidenceDigest() != "" && candidate.GetEvidenceDigest() != parent.GetEvidenceDigest() {
		return false
	}
	if candidate.GetVisibility() != eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_UNSPECIFIED && candidate.GetVisibility() != parent.GetVisibility() {
		return false
	}
	if candidate.GetState() != eventpb.WorkReferenceState_WORK_REFERENCE_STATE_UNSPECIFIED && candidate.GetState() != parent.GetState() {
		return false
	}
	if candidate.GetUnavailableReason() != "" && candidate.GetUnavailableReason() != parent.GetUnavailableReason() {
		return false
	}
	return !candidate.GetVerified() || parent.GetVerified()
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
