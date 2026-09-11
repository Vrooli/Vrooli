package capabilities

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/capabilities"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/capabilities/capabilitiesv1connect"
	internalcapabilities "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/capabilities"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type connectHandler struct{ service internalcapabilities.Service }

func NewConnectHandler(service internalcapabilities.Service) capabilitiesconnect.CapabilitiesServiceHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) ListCapabilities(ctx context.Context, _ *connect.Request[capabilitiesv1.ListCapabilitiesRequest]) (*connect.Response[capabilitiesv1.ListCapabilitiesResponse], error) {
	statuses, err := h.service.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("discover capabilities: %w", err))
	}
	return connect.NewResponse(&capabilitiesv1.ListCapabilitiesResponse{Capabilities: statusesToProto(statuses), Count: int32(len(statuses))}), nil
}

func (h *connectHandler) GetCapabilityStatus(ctx context.Context, _ *connect.Request[capabilitiesv1.GetCapabilityStatusRequest]) (*connect.Response[capabilitiesv1.GetCapabilityStatusResponse], error) {
	statuses, err := h.service.Status(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read capability status: %w", err))
	}
	return connect.NewResponse(&capabilitiesv1.GetCapabilityStatusResponse{Statuses: statusesToProto(statuses), Count: int32(len(statuses))}), nil
}

func (h *connectHandler) PreviewCapability(ctx context.Context, req *connect.Request[capabilitiesv1.PreviewCapabilityRequest]) (*connect.Response[capabilitiesv1.PreviewCapabilityResponse], error) {
	action, err := actionRequest(req.Msg.GetTarget(), req.Msg.GetAction())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	preview, err := h.service.Preview(ctx, action)
	if err != nil {
		return nil, actionError(err)
	}
	return connect.NewResponse(previewToProto(preview)), nil
}

func (h *connectHandler) ApplyCapability(ctx context.Context, req *connect.Request[capabilitiesv1.ApplyCapabilityRequest]) (*connect.Response[capabilitiesv1.ApplyCapabilityResponse], error) {
	action, err := actionRequest(req.Msg.GetTarget(), req.Msg.GetAction())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !action.Confirm {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("confirm=true is required for capability apply"))
	}
	result, err := h.service.Apply(ctx, action)
	if err != nil {
		return nil, actionError(err)
	}
	return connect.NewResponse(resultToProto(result)), nil
}

func (h *connectHandler) VerifyCapability(ctx context.Context, req *connect.Request[capabilitiesv1.VerifyCapabilityRequest]) (*connect.Response[capabilitiesv1.VerifyCapabilityResponse], error) {
	verification, err := verificationRequest(req.Msg.GetTarget(), req.Msg.GetVerification())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	evidence, err := h.service.Verify(ctx, verification)
	if err != nil {
		var verificationErr *operatorcapability.VerificationError
		if errors.As(err, &verificationErr) {
			return connect.NewResponse(&capabilitiesv1.VerifyCapabilityResponse{
				CapabilityId:      verification.CapabilityID,
				Remediation:       verificationErr.NextAction,
				ErrorCode:         verificationErr.Code,
				Retryable:         verificationErr.Retryable,
				RetryAfterSeconds: durationSeconds(verificationErr.RetryAfter),
				NextAction:        verificationErr.NextAction,
				Outcome:           "verification_failed",
			}), nil
		}
		return nil, actionError(err)
	}
	return connect.NewResponse(&capabilitiesv1.VerifyCapabilityResponse{CapabilityId: verification.CapabilityID, Evidence: evidenceToProto(evidence), Outcome: "verified"}), nil
}

func durationSeconds(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	return int64((duration + time.Second - 1) / time.Second)
}

func verificationRequest(target string, request *capabilitiesv1.VerificationRequest) (operatorcapability.VerificationRequest, error) {
	if request == nil {
		return operatorcapability.VerificationRequest{}, errors.New("verification is required")
	}
	target = strings.TrimSpace(target)
	requestedTarget := strings.TrimSpace(request.GetTargetId())
	if target != "" && requestedTarget != "" && target != requestedTarget {
		return operatorcapability.VerificationRequest{}, errors.New("target and verification.target_id must match")
	}
	if requestedTarget == "" {
		requestedTarget = target
	}
	result := operatorcapability.VerificationRequest{
		CapabilityID: request.GetCapabilityId(), TargetID: requestedTarget, Environment: request.GetEnvironment(), AccountIdentity: request.GetAccountIdentity(), Operation: request.GetOperation(), Context: request.GetContext(),
		Effect:  operatorcapability.EffectBudget{Class: operatorcapability.EffectClass(request.GetEffectClass()), MaxOperations: int(request.GetMaxOperations()), CleanupPolicy: request.GetCleanupPolicy()},
		Timeout: time.Duration(request.GetTimeoutSeconds()) * time.Second,
	}
	if ref := request.GetCredentialRef(); ref != nil {
		result.CredentialRef = &operatorcapability.CredentialEvidenceRef{LogicalID: ref.GetLogicalId(), Field: ref.GetField(), Version: ref.GetVersion()}
	}
	if result.Effect.Class == "" {
		result.Effect.Class = operatorcapability.EffectReadOnly
	}
	if result.Timeout == 0 {
		result.Timeout = 30 * time.Second
	}
	return result, result.Validate()
}

func actionRequest(target string, action *capabilitiesv1.ActionRequest) (operatorcapability.ActionRequest, error) {
	if action == nil {
		return operatorcapability.ActionRequest{}, errors.New("action is required")
	}
	inputs, err := internalcapabilities.RawInputs(action.GetInputs().AsMap())
	if err != nil {
		return operatorcapability.ActionRequest{}, err
	}
	target = strings.TrimSpace(target)
	actionTarget := strings.TrimSpace(action.GetTargetId())
	if target != "" && actionTarget != "" && target != actionTarget {
		return operatorcapability.ActionRequest{}, errors.New("target and action.target_id must match")
	}
	if actionTarget == "" {
		actionTarget = target
	}
	request := operatorcapability.ActionRequest{CapabilityID: action.GetCapabilityId(), TargetID: actionTarget, IdempotencyKey: action.GetIdempotencyKey(), Confirm: action.GetConfirm(), Inputs: inputs}
	if err := internalcapabilities.NormalizeActionRequest(&request); err != nil {
		return operatorcapability.ActionRequest{}, err
	}
	return request, nil
}

func actionError(err error) error {
	if strings.Contains(err.Error(), "capability applied but pending operator metadata") {
		return connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewError(connect.CodeFailedPrecondition, err)
}

func statusesToProto(statuses []operatorcapability.Status) []*capabilitiesv1.CapabilityStatus {
	result := make([]*capabilitiesv1.CapabilityStatus, 0, len(statuses))
	for _, status := range statuses {
		result = append(result, statusToProto(status))
	}
	return result
}

func statusToProto(status operatorcapability.Status) *capabilitiesv1.CapabilityStatus {
	return &capabilitiesv1.CapabilityStatus{
		Descriptor_: descriptorToProto(status.Descriptor), State: stateToProto(status.State), Candidates: candidatesToProto(status.Candidates),
		MissingInputs: status.MissingInputs, Evidence: evidenceToProto(status.Evidence), Remediation: status.Remediation,
		UpdatedAt: timestamp(status.UpdatedAt),
	}
}

func descriptorToProto(descriptor operatorcapability.Descriptor) *capabilitiesv1.CapabilityDescriptor {
	inputs := make([]*capabilitiesv1.CapabilityInputDescriptor, 0, len(descriptor.Inputs))
	for _, input := range descriptor.Inputs {
		inputs = append(inputs, &capabilitiesv1.CapabilityInputDescriptor{Id: input.ID, Kind: string(input.Kind), Label: input.Label, Description: input.Description, Required: input.Required, Declinable: input.Declinable, Options: input.Options, DefaultValue: input.Default, Candidates: candidatesToProto(input.Candidates), Validation: input.Validation, Constraints: &capabilitiesv1.CapabilityInputConstraints{MinLength: int32(input.Constraints.MinLength), MaxLength: int32(input.Constraints.MaxLength), MinDuration: input.Constraints.MinDuration, MaxDuration: input.Constraints.MaxDuration}, CredentialLogicalId: input.CredentialLogicalID, CredentialField: input.CredentialField, Provider: input.Provider, RequirementGroup: input.RequirementGroup, ConsumerRefs: input.ConsumerRefs, CompanionSettings: input.CompanionSettings, CompanionCredentials: input.CompanionCredentials, AcquisitionRef: input.AcquisitionRef, VerificationRef: input.VerificationRef, RecoveryRef: input.RecoveryRef, HelpRef: input.HelpRef, EvidencePolicy: input.EvidencePolicy})
	}
	return &capabilitiesv1.CapabilityDescriptor{
		Version: descriptor.Version, Id: descriptor.ID, Owner: descriptor.Owner, Title: descriptor.Title, Description: descriptor.Description, Risk: descriptor.Risk,
		Inputs: inputs, Prerequisites: descriptor.Prerequisites,
		Policy:      &capabilitiesv1.CapabilityPolicy{RequiresConfirmation: descriptor.Policy.RequiresConfirmation, Idempotent: descriptor.Policy.Idempotent, Retryable: descriptor.Policy.Retryable, ProtectedRoots: descriptor.Policy.ProtectedRoots, Remediation: descriptor.Policy.Remediation},
		Evidence:    &capabilitiesv1.CapabilityEvidenceContract{Kinds: descriptor.Evidence.Kinds, Stages: verificationStagesToStrings(descriptor.Evidence.Stages), RequiredFields: descriptor.Evidence.RequiredFields, SecretFree: descriptor.Evidence.SecretFree, Freshness: descriptor.Evidence.Freshness},
		Remediation: descriptor.Remediation, Scope: descriptor.Scope, Purpose: descriptor.Purpose, Sensitivity: string(descriptor.Sensitivity), Disposition: string(descriptor.Disposition), DispositionReason: descriptor.DispositionReason, ReferenceUrl: descriptor.ReferenceURL,
		Applicability: &capabilitiesv1.CapabilityApplicability{Platforms: descriptor.Applicability.Platforms, Environments: descriptor.Applicability.Environments, Targets: descriptor.Applicability.Targets},
		Provenance:    &capabilitiesv1.CapabilityPermissionProvenance{Requester: descriptor.Provenance.Requester, Scope: descriptor.Provenance.Scope, GrantSource: descriptor.Provenance.GrantSource, RevocationLimit: descriptor.Provenance.RevocationLimit},
		Lifecycle:     &capabilitiesv1.CapabilityLifecycle{Preview: descriptor.Lifecycle.Preview, Apply: descriptor.Lifecycle.Apply, Verify: descriptor.Lifecycle.Verify, Revoke: descriptor.Lifecycle.Revoke, Recover: descriptor.Lifecycle.Recover, Recovery: descriptor.Lifecycle.Recovery},
	}
}

func candidatesToProto(candidates []operatorcapability.Candidate) []*capabilitiesv1.CapabilityCandidate {
	result := make([]*capabilitiesv1.CapabilityCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, &capabilitiesv1.CapabilityCandidate{Id: candidate.ID, Kind: candidate.Kind, Label: candidate.Label, Location: candidate.Location, StableIdentity: candidate.StableIdentity, DeviceIdentity: candidate.DeviceIdentity, Writable: candidate.Writable, PhysicalIndependence: candidate.PhysicallyIndependent, Status: candidate.Status, Risk: candidate.Risk, Remediation: candidate.Remediation, Metadata: candidate.Metadata})
	}
	return result
}

func evidenceToProto(evidence []operatorcapability.EvidenceReference) []*capabilitiesv1.CapabilityEvidence {
	result := make([]*capabilitiesv1.CapabilityEvidence, 0, len(evidence))
	for _, item := range evidence {
		var credentialRef *capabilitiesv1.CredentialEvidenceRef
		if item.CredentialRef != nil {
			credentialRef = &capabilitiesv1.CredentialEvidenceRef{LogicalId: item.CredentialRef.LogicalID, Field: item.CredentialRef.Field, Version: item.CredentialRef.Version}
		}
		result = append(result, &capabilitiesv1.CapabilityEvidence{Kind: item.Kind, ArtifactIdentity: item.ArtifactIdentity, SourceGeneration: item.SourceGeneration, Checksum: item.Checksum, Coverage: item.Coverage, ObservedAt: timestamp(item.ObservedAt), Verified: item.Verified, Remediation: item.Remediation, SchemaVersion: item.SchemaVersion, CapabilityId: item.CapabilityID, CredentialRef: credentialRef, TargetId: item.TargetID, Environment: item.Environment, AccountIdentity: item.AccountIdentity, Operation: item.Operation, Status: item.Status, ExpiresAt: timestamp(item.ExpiresAt), ArtifactRefs: item.ArtifactRefs, Limitations: item.Limitations, NextAction: item.NextAction, EffectClass: item.EffectClass, EffectsUsed: int32(item.EffectsUsed), CleanupCompleted: item.CleanupCompleted, Stage: string(item.Stage)})
	}
	return result
}

func verificationStagesToStrings(stages []operatorcapability.VerificationStage) []string {
	result := make([]string, 0, len(stages))
	for _, stage := range stages {
		result = append(result, string(stage))
	}
	return result
}

func previewToProto(preview operatorcapability.Preview) *capabilitiesv1.PreviewCapabilityResponse {
	mutations := make([]*capabilitiesv1.CapabilityMutation, 0, len(preview.Mutations))
	for _, item := range preview.Mutations {
		mutations = append(mutations, &capabilitiesv1.CapabilityMutation{Id: item.ID, Summary: item.Summary, Reversible: item.Reversible})
	}
	return &capabilitiesv1.PreviewCapabilityResponse{CapabilityId: preview.CapabilityID, PlanId: preview.PlanID, State: stateToProto(preview.State), Mutations: mutations, Candidates: candidatesToProto(preview.Candidates), Remediation: preview.Remediation, ExpiresAt: timestamp(preview.ExpiresAt)}
}

func resultToProto(result operatorcapability.Result) *capabilitiesv1.ApplyCapabilityResponse {
	mutations := make([]*capabilitiesv1.CapabilityMutation, 0, len(result.Mutations))
	for _, item := range result.Mutations {
		mutations = append(mutations, &capabilitiesv1.CapabilityMutation{Id: item.ID, Summary: item.Summary, Reversible: item.Reversible})
	}
	return &capabilitiesv1.ApplyCapabilityResponse{CapabilityId: result.CapabilityID, State: stateToProto(result.State), Outcome: result.Outcome, Retryable: result.Retryable, ErrorCode: result.ErrorCode, Remediation: result.Remediation, Evidence: evidenceToProto(result.Evidence), Mutations: mutations, CompletedAt: timestamp(result.CompletedAt)}
}

func stateToProto(state operatorcapability.State) capabilitiesv1.CapabilityState {
	return map[operatorcapability.State]capabilitiesv1.CapabilityState{
		operatorcapability.StateDiscovered: capabilitiesv1.CapabilityState_CAPABILITY_STATE_DISCOVERED, operatorcapability.StateNeedsInput: capabilitiesv1.CapabilityState_CAPABILITY_STATE_NEEDS_OPERATOR_INPUT,
		operatorcapability.StateReadyToPreview: capabilitiesv1.CapabilityState_CAPABILITY_STATE_READY_TO_PREVIEW, operatorcapability.StateApplying: capabilitiesv1.CapabilityState_CAPABILITY_STATE_APPLYING,
		operatorcapability.StateVerifying: capabilitiesv1.CapabilityState_CAPABILITY_STATE_VERIFYING, operatorcapability.StateReady: capabilitiesv1.CapabilityState_CAPABILITY_STATE_READY,
		operatorcapability.StateRetryableFailure: capabilitiesv1.CapabilityState_CAPABILITY_STATE_RETRYABLE_FAILURE, operatorcapability.StateDegraded: capabilitiesv1.CapabilityState_CAPABILITY_STATE_DEGRADED,
		operatorcapability.StateUnsupported: capabilitiesv1.CapabilityState_CAPABILITY_STATE_UNSUPPORTED,
	}[state]
}

func timestamp(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}
