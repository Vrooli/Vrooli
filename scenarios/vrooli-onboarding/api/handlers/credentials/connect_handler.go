package credentials

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/internal/credentialspec"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	credentialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/credentials"
	credentialsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/credentials/credentialsv1connect"
	internalcredentials "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/credentials"
	readiness "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/readiness"
)

type connectHandler struct{ service internalcredentials.Service }

func NewConnectHandler(service internalcredentials.Service) credentialsconnect.CredentialsServiceHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) ListCredentials(ctx context.Context, _ *connect.Request[credentialsv1.ListCredentialsRequest]) (*connect.Response[credentialsv1.ListCredentialsResponse], error) {
	items, err := h.service.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list credentials: %w", err))
	}
	return connect.NewResponse(&credentialsv1.ListCredentialsResponse{Credentials: credentialsToProto(items), Count: int32(len(items))}), nil
}

func (h *connectHandler) ProvisionCredential(ctx context.Context, req *connect.Request[credentialsv1.ProvisionCredentialRequest]) (*connect.Response[credentialsv1.ProvisionCredentialResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provision request is required"))
	}
	logicalID := strings.TrimSpace(req.Msg.GetLogicalId())
	field := strings.TrimSpace(req.Msg.GetField())
	if field == "" {
		field = "value"
	}
	value := req.Msg.GetValue()
	if logicalID == "" || strings.TrimSpace(value) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("logical_id and a non-empty credential value are required"))
	}
	provisioned, err := h.service.Provision(ctx, logicalID, field, value)
	if err != nil {
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "missing") {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("native credential authority could not provision this credential"))
	}
	return connect.NewResponse(&credentialsv1.ProvisionCredentialResponse{Status: "provisioned", LogicalId: logicalID, Field: field, Version: provisioned.Version}), nil
}

func (h *connectHandler) DiagnoseCredentials(ctx context.Context, _ *connect.Request[credentialsv1.DiagnoseCredentialsRequest]) (*connect.Response[credentialsv1.DiagnoseCredentialsResponse], error) {
	diagnosis, err := h.service.Diagnose(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("credential diagnosis is unavailable"))
	}
	return connect.NewResponse(&credentialsv1.DiagnoseCredentialsResponse{
		Provider: providerToProto(diagnosis.Provider), Credentials: refsToProto(diagnosis.Credentials),
		CredentialCount: int32(diagnosis.CredentialCount), DeclarationSiteCount: int32(diagnosis.DeclarationSiteCount),
		InventoryBasis: diagnosis.InventoryBasis, ManagedInstancesIncluded: diagnosis.ManagedInstancesIncluded,
		Recovery: recoveryToProto(diagnosis.Recovery),
	}), nil
}

func credentialsToProto(items []readiness.Credential) []*credentialsv1.Credential {
	result := make([]*credentialsv1.Credential, 0, len(items))
	for _, item := range items {
		result = append(result, &credentialsv1.Credential{Resource: item.Resource, LogicalId: item.LogicalID, Field: item.Field, Label: item.Label, Description: item.Description, ObtainUrl: item.ObtainURL, Placeholder: item.Placeholder, Provisioning: item.Provisioning, DerivedFrom: item.DerivedFrom, Required: item.Required, Status: item.Status, Detail: item.Detail, Owner: item.Owner, SourceRef: item.SourceRef, Kind: item.Kind, ConsumerRefs: item.ConsumerRefs, Version: item.Version, Provider: item.Provider, AppliesWhen: applicabilityToCredentialsProto(item.AppliesWhen), RequirementGroup: item.RequirementGroup, CompanionSettings: item.CompanionSettings, CompanionCredentials: item.CompanionCredentials, AcquisitionRef: item.AcquisitionRef, VerificationRef: item.VerificationRef, RecoveryRef: item.RecoveryRef, HelpRef: item.HelpRef, EvidencePolicy: item.EvidencePolicy, ProviderVersion: item.ProviderVersion, MigrationDiagnostics: migrationDiagnosticsToCredentialsProto(item.MigrationDiagnostics), Tiers: item.Tiers, Provenance: readinessProvenanceToProto(item.Provenance)})
	}
	return result
}

func readinessProvenanceToProto(items []readiness.CredentialProvenance) []*credentialsv1.CredentialProvenance {
	result := make([]*credentialsv1.CredentialProvenance, 0, len(items))
	for _, item := range items {
		consumers := make([]*credentialsv1.CredentialConsumerProvenance, 0, len(item.Consumers))
		for _, consumer := range item.Consumers {
			consumers = append(consumers, &credentialsv1.CredentialConsumerProvenance{LogicalId: consumer.LogicalID, AddressPattern: consumer.AddressPattern, Field: consumer.Field, Kind: consumer.Kind, Consumer: consumer.Consumer, SourceRef: consumer.SourceRef, Required: consumer.Required, Reason: consumer.Reason, Tiers: consumer.Tiers})
		}
		result = append(result, &credentialsv1.CredentialProvenance{Version: item.Version, Owner: item.Owner, SourceRef: item.SourceRef, Kind: item.Kind, Provider: item.Provider, AppliesWhen: applicabilityToCredentialsProto(item.AppliesWhen), Tiers: item.Tiers, RequirementGroup: item.RequirementGroup, ConsumerRefs: item.ConsumerRefs, CompanionSettings: item.CompanionSettings, CompanionCredentials: item.CompanionCredentials, AcquisitionRef: item.AcquisitionRef, VerificationRef: item.VerificationRef, RecoveryRef: item.RecoveryRef, HelpRef: item.HelpRef, EvidencePolicy: item.EvidencePolicy, ProviderVersion: item.ProviderVersion, Env: item.Env, Label: item.Label, Description: item.Description, Placeholder: item.Placeholder, ObtainUrl: item.ObtainURL, Provisioning: item.Provisioning, DerivedFrom: item.DerivedFrom, Required: item.Required, Consumers: consumers})
	}
	return result
}

func refsToProto(items []credentialclient.CredentialRef) []*credentialsv1.Credential {
	result := make([]*credentialsv1.Credential, 0, len(items))
	for _, item := range items {
		provenance := make([]*credentialsv1.CredentialProvenance, 0, len(item.Provenance))
		for _, value := range item.Provenance {
			consumers := make([]*credentialsv1.CredentialConsumerProvenance, 0, len(value.Consumers))
			for _, consumer := range value.Consumers {
				consumers = append(consumers, &credentialsv1.CredentialConsumerProvenance{LogicalId: consumer.LogicalID, AddressPattern: consumer.AddressPattern, Field: consumer.Field, Kind: consumer.Kind, Consumer: consumer.Consumer, SourceRef: consumer.SourceRef, Required: consumer.Required, Reason: consumer.Reason, Tiers: consumer.Tiers})
			}
			provenance = append(provenance, &credentialsv1.CredentialProvenance{Version: value.Version, Owner: value.Owner, SourceRef: value.SourceRef, Kind: value.Kind, Provider: value.Provider, AppliesWhen: applicabilityToCredentialsProto(value.AppliesWhen), Tiers: value.Tiers, RequirementGroup: value.RequirementGroup, ConsumerRefs: value.ConsumerRefs, CompanionSettings: value.CompanionSettings, CompanionCredentials: value.CompanionCredentials, AcquisitionRef: value.AcquisitionRef, VerificationRef: value.VerificationRef, RecoveryRef: value.RecoveryRef, HelpRef: value.HelpRef, EvidencePolicy: value.EvidencePolicy, ProviderVersion: value.ProviderVersion, Env: value.Env, Label: value.Label, Description: value.Description, Placeholder: value.Placeholder, ObtainUrl: value.ObtainURL, Provisioning: value.Provisioning, DerivedFrom: value.DerivedFrom, Required: value.Required, Consumers: consumers})
		}
		result = append(result, &credentialsv1.Credential{Resource: item.Resource, LogicalId: item.LogicalID, Field: item.Field, Label: item.Label, Description: item.Description, ObtainUrl: item.ObtainURL, Placeholder: item.Placeholder, Provisioning: item.Provisioning, DerivedFrom: item.DerivedFrom, Required: item.Required, Owner: item.Owner, SourceRef: item.SourceRef, Kind: item.Kind, ConsumerRefs: item.ConsumerRefs, Version: item.Version, Provider: item.Provider, AppliesWhen: applicabilityToCredentialsProto(item.AppliesWhen), RequirementGroup: item.RequirementGroup, CompanionSettings: item.CompanionSettings, CompanionCredentials: item.CompanionCredentials, AcquisitionRef: item.AcquisitionRef, VerificationRef: item.VerificationRef, RecoveryRef: item.RecoveryRef, HelpRef: item.HelpRef, EvidencePolicy: item.EvidencePolicy, ProviderVersion: item.ProviderVersion, MigrationDiagnostics: migrationDiagnosticsToCredentialsProto(item.MigrationDiagnostics), Tiers: item.Tiers, Provenance: provenance})
	}
	return result
}

func applicabilityToCredentialsProto(value *credentialspec.Applicability) *credentialsv1.CredentialApplicability {
	if value == nil {
		return nil
	}
	result := &credentialsv1.CredentialApplicability{}
	for _, rule := range value.All {
		result.All = append(result.All, applicabilityRuleToCredentialsProto(rule))
	}
	for _, rule := range value.Any {
		result.Any = append(result.Any, applicabilityRuleToCredentialsProto(rule))
	}
	if value.Not != nil {
		result.Not = applicabilityRuleToCredentialsProto(*value.Not)
	}
	return result
}

func applicabilityRuleToCredentialsProto(value credentialspec.ApplicabilityRule) *credentialsv1.CredentialApplicabilityRule {
	result := &credentialsv1.CredentialApplicabilityRule{}
	if value.Eq != nil {
		result.Comparison = &credentialsv1.CredentialApplicabilityRule_Eq{Eq: applicabilityMatchToCredentialsProto(*value.Eq)}
	}
	if value.Neq != nil {
		result.Comparison = &credentialsv1.CredentialApplicabilityRule_Neq{Neq: applicabilityMatchToCredentialsProto(*value.Neq)}
	}
	return result
}

func applicabilityMatchToCredentialsProto(value credentialspec.ApplicabilityMatch) *credentialsv1.CredentialApplicabilityMatch {
	return &credentialsv1.CredentialApplicabilityMatch{Context: value.Context, Setting: value.Setting, Operation: value.Operation, Role: value.Role, Target: value.Target, Environment: value.Environment, Capability: value.Capability, Provider: value.Provider, Value: value.Value}
}

func migrationDiagnosticsToCredentialsProto(values []credentialspec.MigrationDiagnostic) []*credentialsv1.CredentialMigrationDiagnostic {
	result := make([]*credentialsv1.CredentialMigrationDiagnostic, 0, len(values))
	for _, value := range values {
		result = append(result, &credentialsv1.CredentialMigrationDiagnostic{Address: value.Address, Code: value.Code, Severity: value.Severity, Message: value.Message})
	}
	return result
}

func providerToProto(value credentialclient.ProviderDiagnosis) *credentialsv1.ProviderDiagnosis {
	return &credentialsv1.ProviderDiagnosis{Platform: value.Platform, Adapter: value.Adapter, Backend: value.Backend, Condition: value.Condition, Available: value.Available, Writable: value.Writable, Explanation: value.Explanation, Fix: value.Fix}
}

func recoveryToProto(value credentialclient.RecoveryStatus) *credentialsv1.RecoveryStatus {
	return &credentialsv1.RecoveryStatus{ReceiptExists: value.ReceiptExists, ExportedAt: value.ExportedAt, EntryCount: int32(value.EntryCount), Uncovered: value.Uncovered, RequiredAbsent: value.RequiredAbsent, Basis: value.Basis, ManagedInstancesIncluded: value.ManagedInstancesIncluded, Status: value.Status, AgeSeconds: value.AgeSeconds, FreshnessReason: value.FreshnessReason}
}
