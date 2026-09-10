package credentials

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
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
	if err := h.service.Provision(ctx, logicalID, field, value); err != nil {
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "missing") {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("native credential authority could not provision this credential"))
	}
	return connect.NewResponse(&credentialsv1.ProvisionCredentialResponse{Status: "provisioned", LogicalId: logicalID, Field: field}), nil
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
		result = append(result, &credentialsv1.Credential{Resource: item.Resource, LogicalId: item.LogicalID, Field: item.Field, Label: item.Label, Description: item.Description, ObtainUrl: item.ObtainURL, Provisioning: item.Provisioning, DerivedFrom: item.DerivedFrom, Required: item.Required, Status: item.Status, Detail: item.Detail})
	}
	return result
}

func refsToProto(items []credentialclient.CredentialRef) []*credentialsv1.Credential {
	result := make([]*credentialsv1.Credential, 0, len(items))
	for _, item := range items {
		result = append(result, &credentialsv1.Credential{Resource: item.Resource, LogicalId: item.LogicalID, Field: item.Field, Label: item.Label, Description: item.Description, ObtainUrl: item.ObtainURL, Provisioning: item.Provisioning, DerivedFrom: item.DerivedFrom, Required: item.Required})
	}
	return result
}

func providerToProto(value credentialclient.ProviderDiagnosis) *credentialsv1.ProviderDiagnosis {
	return &credentialsv1.ProviderDiagnosis{Platform: value.Platform, Adapter: value.Adapter, Backend: value.Backend, Condition: value.Condition, Available: value.Available, Writable: value.Writable, Explanation: value.Explanation, Fix: value.Fix}
}

func recoveryToProto(value credentialclient.RecoveryStatus) *credentialsv1.RecoveryStatus {
	return &credentialsv1.RecoveryStatus{ReceiptExists: value.ReceiptExists, ExportedAt: value.ExportedAt, EntryCount: int32(value.EntryCount), Uncovered: value.Uncovered, RequiredAbsent: value.RequiredAbsent, Basis: value.Basis, ManagedInstancesIncluded: value.ManagedInstancesIncluded, Status: value.Status, AgeSeconds: value.AgeSeconds, FreshnessReason: value.FreshnessReason}
}
