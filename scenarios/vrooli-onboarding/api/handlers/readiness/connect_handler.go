package readiness

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/internal/credentialspec"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/readiness"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/shared"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	internalreadiness "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/readiness"
)

type connectHandler struct {
	service internalreadiness.Service
}

func NewConnectHandler(service internalreadiness.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) GetReadiness(ctx context.Context, req *connect.Request[readinessv1.GetReadinessRequest]) (*connect.Response[readinessv1.GetReadinessResponse], error) {
	target := "local"
	if req != nil && req.Msg != nil && strings.TrimSpace(req.Msg.GetTarget()) != "" {
		target = strings.TrimSpace(req.Msg.GetTarget())
	}
	result, err := h.service.Get(ctx, target)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("compute readiness: %w", err))
	}
	return connect.NewResponse(toProto(result)), nil
}

func (h *connectHandler) AcknowledgeDegradedReadiness(ctx context.Context, req *connect.Request[readinessv1.AcknowledgeDegradedReadinessRequest]) (*connect.Response[readinessv1.AcknowledgeDegradedReadinessResponse], error) {
	digest := ""
	if req != nil && req.Msg != nil {
		digest = strings.TrimSpace(req.Msg.GetReadinessDigest())
	}
	if digest == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("readiness_digest is required"))
	}
	result, err := h.service.Accept(ctx, digest)
	if err != nil {
		if strings.Contains(err.Error(), "not the current one") {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*sharedv1.CompletionBlocker, 0, len(result.Degraded))
	for _, item := range result.Degraded {
		items = append(items, &sharedv1.CompletionBlocker{Kind: item.Kind, Name: item.Name, Reason: item.Reason, Remediation: item.Remediation})
	}
	return connect.NewResponse(&readinessv1.AcknowledgeDegradedReadinessResponse{Status: result.Status, ReadinessDigest: result.ReadinessDigest, Degraded: items}), nil
}

func toProto(result internalreadiness.Response) *readinessv1.GetReadinessResponse {
	credentials := make([]*readinessv1.Credential, 0, len(result.Credentials))
	for _, item := range result.Credentials {
		provenance := make([]*readinessv1.CredentialProvenance, 0, len(item.Provenance))
		for _, value := range item.Provenance {
			consumers := make([]*readinessv1.CredentialConsumerProvenance, 0, len(value.Consumers))
			for _, consumer := range value.Consumers {
				consumers = append(consumers, &readinessv1.CredentialConsumerProvenance{LogicalId: consumer.LogicalID, AddressPattern: consumer.AddressPattern, Field: consumer.Field, Kind: consumer.Kind, Consumer: consumer.Consumer, SourceRef: consumer.SourceRef, Required: consumer.Required, Reason: consumer.Reason, Tiers: consumer.Tiers})
			}
			provenance = append(provenance, &readinessv1.CredentialProvenance{Version: value.Version, Owner: value.Owner, SourceRef: value.SourceRef, Kind: value.Kind, Provider: value.Provider, AppliesWhen: applicabilityToReadinessProto(value.AppliesWhen), RequirementGroup: value.RequirementGroup, ConsumerRefs: value.ConsumerRefs, CompanionSettings: value.CompanionSettings, CompanionCredentials: value.CompanionCredentials, AcquisitionRef: value.AcquisitionRef, VerificationRef: value.VerificationRef, RecoveryRef: value.RecoveryRef, HelpRef: value.HelpRef, EvidencePolicy: value.EvidencePolicy, ProviderVersion: value.ProviderVersion, Env: value.Env, Label: value.Label, Description: value.Description, ObtainUrl: value.ObtainURL, Provisioning: value.Provisioning, DerivedFrom: value.DerivedFrom, Required: value.Required, Consumers: consumers})
		}
		credentials = append(credentials, &readinessv1.Credential{Resource: item.Resource, LogicalId: item.LogicalID, Field: item.Field, Label: item.Label, Description: item.Description, ObtainUrl: item.ObtainURL, Provisioning: item.Provisioning, DerivedFrom: item.DerivedFrom, Required: item.Required, Status: state(item.Status), LegacyStatus: item.Status, Detail: item.Detail, Owner: item.Owner, SourceRef: item.SourceRef, Kind: item.Kind, ConsumerRefs: item.ConsumerRefs, Version: item.Version, Provider: item.Provider, AppliesWhen: applicabilityToReadinessProto(item.AppliesWhen), RequirementGroup: item.RequirementGroup, CompanionSettings: item.CompanionSettings, CompanionCredentials: item.CompanionCredentials, AcquisitionRef: item.AcquisitionRef, VerificationRef: item.VerificationRef, RecoveryRef: item.RecoveryRef, HelpRef: item.HelpRef, EvidencePolicy: item.EvidencePolicy, ProviderVersion: item.ProviderVersion, MigrationDiagnostics: migrationDiagnosticsToReadinessProto(item.MigrationDiagnostics), EvidenceStatus: item.EvidenceStatus, EvidenceDetail: item.EvidenceDetail, Provenance: provenance})
	}
	items := make([]*readinessv1.ReadinessItem, 0, len(result.Integrations))
	for _, item := range result.Integrations {
		items = append(items, itemProto(item))
	}
	hosts := make([]*readinessv1.HostReadiness, 0, len(result.Hosts))
	for _, item := range result.Hosts {
		hosts = append(hosts, &readinessv1.HostReadiness{Item: itemProto(item.Item), Kind: item.Kind, Required: item.Required})
	}
	return &readinessv1.GetReadinessResponse{
		Target: result.Target, ConfigurationRevision: result.ConfigurationRevision, ExpiresAt: checkedAt(result.ExpiresAt),
		Status: state(result.Status), Scenarios: result.Scenarios, Resources: result.Resources, Credentials: credentials,
		Hosts: hosts, Integrations: items, CheckedAt: checkedAt(result.CheckedAt), CredentialDiagnosis: jsonStruct(result.CredentialDiagnosis),
		Recovery: recoveryProto(result.Recovery), Blockers: blockersProto(result.Blockers), Degraded: blockersProto(result.Degraded),
		DegradedDigest: result.DegradedDigest, DegradedAcknowledged: result.DegradedAcknowledged,
		ManagedKeyConfigured: result.ManagedKeyConfigured, TrustAnchorMatch: result.TrustAnchorMatch,
	}
}

func applicabilityToReadinessProto(value *credentialspec.Applicability) *readinessv1.CredentialApplicability {
	if value == nil {
		return nil
	}
	result := &readinessv1.CredentialApplicability{}
	for _, rule := range value.All {
		result.All = append(result.All, applicabilityRuleToReadinessProto(rule))
	}
	for _, rule := range value.Any {
		result.Any = append(result.Any, applicabilityRuleToReadinessProto(rule))
	}
	if value.Not != nil {
		result.Not = applicabilityRuleToReadinessProto(*value.Not)
	}
	return result
}

func applicabilityRuleToReadinessProto(value credentialspec.ApplicabilityRule) *readinessv1.CredentialApplicabilityRule {
	result := &readinessv1.CredentialApplicabilityRule{}
	if value.Eq != nil {
		result.Comparison = &readinessv1.CredentialApplicabilityRule_Eq{Eq: applicabilityMatchToReadinessProto(*value.Eq)}
	}
	if value.Neq != nil {
		result.Comparison = &readinessv1.CredentialApplicabilityRule_Neq{Neq: applicabilityMatchToReadinessProto(*value.Neq)}
	}
	return result
}

func applicabilityMatchToReadinessProto(value credentialspec.ApplicabilityMatch) *readinessv1.CredentialApplicabilityMatch {
	return &readinessv1.CredentialApplicabilityMatch{Context: value.Context, Setting: value.Setting, Operation: value.Operation, Role: value.Role, Target: value.Target, Environment: value.Environment, Capability: value.Capability, Provider: value.Provider, Value: value.Value}
}

func migrationDiagnosticsToReadinessProto(values []credentialspec.MigrationDiagnostic) []*readinessv1.CredentialMigrationDiagnostic {
	result := make([]*readinessv1.CredentialMigrationDiagnostic, 0, len(values))
	for _, value := range values {
		result = append(result, &readinessv1.CredentialMigrationDiagnostic{Address: value.Address, Code: value.Code, Severity: value.Severity, Message: value.Message})
	}
	return result
}

func itemProto(item internalreadiness.Item) *readinessv1.ReadinessItem {
	return &readinessv1.ReadinessItem{Name: item.Name, Category: item.Category, Status: state(item.Status), LegacyStatus: item.Status, Detail: item.Detail, Remediation: item.Remediation, Required: item.Required}
}

func state(value string) readinessv1.ReadinessState {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ready", "configured":
		return readinessv1.ReadinessState_READINESS_STATE_READY
	case "missing", "unconfigured", "locked", "unresponsive":
		return readinessv1.ReadinessState_READINESS_STATE_MISSING
	case "degraded":
		return readinessv1.ReadinessState_READINESS_STATE_DEGRADED
	case "unsupported":
		return readinessv1.ReadinessState_READINESS_STATE_UNSUPPORTED
	case "deferred":
		return readinessv1.ReadinessState_READINESS_STATE_DEFERRED
	case "not_applicable":
		return readinessv1.ReadinessState_READINESS_STATE_NOT_APPLICABLE
	default:
		return readinessv1.ReadinessState_READINESS_STATE_UNSPECIFIED
	}
}

func checkedAt(value string) *timestamppb.Timestamp {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return timestamppb.New(parsed)
}

func jsonStruct(value []byte) *structpb.Struct {
	if len(value) == 0 || !json.Valid(value) {
		return nil
	}
	var object map[string]any
	if json.Unmarshal(value, &object) != nil {
		return nil
	}
	result, err := structpb.NewStruct(object)
	if err != nil {
		return nil
	}
	return result
}

func recoveryProto(value internalreadiness.Recovery) *readinessv1.Recovery {
	gaps := make([]*readinessv1.RecoveryGap, 0, len(value.RequiredAbsentDetails))
	for _, gap := range value.RequiredAbsentDetails {
		gaps = append(gaps, &readinessv1.RecoveryGap{Address: gap.Address, Description: gap.Description})
	}
	return &readinessv1.Recovery{ReceiptExists: value.ReceiptExists, ExportedAt: value.ExportedAt, EntryCount: int32(value.EntryCount), Uncovered: value.Uncovered, RequiredAbsent: value.RequiredAbsent, RequiredAbsentDetails: gaps, RootCopy: jsonStruct(value.RootCopy), RootCopyIssues: value.RootCopyIssues, Status: value.Status, AgeSeconds: value.AgeSeconds, FreshnessReason: value.FreshnessReason}
}

func blockersProto(values []internalreadiness.CompletionBlocker) []*sharedv1.CompletionBlocker {
	result := make([]*sharedv1.CompletionBlocker, 0, len(values))
	for _, value := range values {
		result = append(result, &sharedv1.CompletionBlocker{Kind: value.Kind, Name: value.Name, Reason: value.Reason, Remediation: value.Remediation})
	}
	return result
}
