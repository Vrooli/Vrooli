package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/operatorsession"
	credentialgrant "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant"
	credentialgrantconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant/credentialgrant_v1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/bridgepermissions"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/credentialpermissions"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// credentialGrantTargetReader reuses the Bridge scope provider's strict
// machine/current-node/generation projection, so credential metadata cannot
// drift onto a stale or revoked node identity.
type credentialGrantTargetReader struct{ scopes *bridgepermissions.Provider }

func (r credentialGrantTargetReader) CurrentTargets(ctx context.Context, _ string) ([]credentialpermissions.Target, error) {
	descriptors, err := r.scopes.List(ctx)
	if err != nil {
		return nil, err
	}
	targets := make([]credentialpermissions.Target, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if !descriptor.TargetBound || descriptor.Target.MachineID == "" || descriptor.Target.NodeID == "" || descriptor.Target.EnrollmentGeneration == 0 {
			return nil, connect.NewError(connect.CodeDataLoss, errInvalidCredentialTarget)
		}
		targets = append(targets, credentialpermissions.Target{Binding: descriptor.Target})
	}
	return targets, nil
}

func (r credentialGrantTargetReader) CurrentTarget(ctx context.Context, token string, requested permissions.TargetBinding) (credentialpermissions.Target, error) {
	targets, err := r.CurrentTargets(ctx, token)
	if err != nil {
		return credentialpermissions.Target{}, err
	}
	for _, target := range targets {
		if target.Binding == requested {
			return target, nil
		}
	}
	return credentialpermissions.Target{}, connect.NewError(connect.CodeFailedPrecondition, errInvalidCredentialTarget)
}

var errInvalidCredentialTarget = errors.New("Bridge machine, current node, or enrollment generation is unavailable")

type credentialGrantMetadataClient struct{}

func (credentialGrantMetadataClient) ListGrantMetadata(ctx context.Context, token string, target permissions.TargetBinding) ([]credentialpermissions.GrantMetadata, error) {
	client, err := credentialGrantClient(ctx)
	if err != nil {
		return nil, err
	}
	request := connect.NewRequest(&credentialgrant.ListGrantMetadataRequest{NodeId: target.NodeID, EnrollmentGeneration: int64(target.EnrollmentGeneration)})
	request.Header().Set("Authorization", token)
	response, err := client.ListGrantMetadata(ctx, request)
	if err != nil {
		return nil, err
	}
	if response == nil || response.Msg == nil || response.Msg.GetNodeId() != target.NodeID || response.Msg.GetEnrollmentGeneration() != int64(target.EnrollmentGeneration) {
		return nil, connect.NewError(connect.CodeDataLoss, errInvalidCredentialTarget)
	}
	grants := make([]credentialpermissions.GrantMetadata, 0, len(response.Msg.GetGrants()))
	for _, grant := range response.Msg.GetGrants() {
		if grant == nil {
			return nil, connect.NewError(connect.CodeDataLoss, errInvalidCredentialTarget)
		}
		grants = append(grants, credentialMetadata(grant))
	}
	return grants, nil
}

func (credentialGrantMetadataClient) GetGrantMetadata(ctx context.Context, token string, target permissions.TargetBinding, id string) (credentialpermissions.GrantMetadata, error) {
	client, err := credentialGrantClient(ctx)
	if err != nil {
		return credentialpermissions.GrantMetadata{}, err
	}
	request := connect.NewRequest(&credentialgrant.GetGrantMetadataRequest{Id: id, NodeId: target.NodeID, EnrollmentGeneration: int64(target.EnrollmentGeneration)})
	request.Header().Set("Authorization", token)
	response, err := client.GetGrantMetadata(ctx, request)
	if err != nil {
		return credentialpermissions.GrantMetadata{}, err
	}
	if response == nil || response.Msg == nil {
		return credentialpermissions.GrantMetadata{}, connect.NewError(connect.CodeDataLoss, errInvalidCredentialTarget)
	}
	return credentialMetadata(response.Msg), nil
}

func (credentialGrantMetadataClient) CreateGrantWithReceipt(ctx context.Context, token string, target permissions.TargetBinding, requestID string, value credentialpermissions.CreateMutation) (credentialpermissions.MutationReceipt, error) {
	client, err := credentialGrantClient(ctx)
	if err != nil {
		return credentialpermissions.MutationReceipt{}, err
	}
	return credentialpermissions.NewConnectGrantClient(client).CreateGrantWithReceipt(ctx, token, target, requestID, value)
}

func (credentialGrantMetadataClient) RevokeGrantWithReceipt(ctx context.Context, token string, target permissions.TargetBinding, id string, revision int64, requestID string) (credentialpermissions.MutationReceipt, error) {
	client, err := credentialGrantClient(ctx)
	if err != nil {
		return credentialpermissions.MutationReceipt{}, err
	}
	return credentialpermissions.NewConnectGrantClient(client).RevokeGrantWithReceipt(ctx, token, target, id, revision, requestID)
}

func credentialGrantClient(ctx context.Context) (credentialgrantconnect.CredentialGrantServiceClient, error) {
	base, err := discoveredBridgeBase(ctx)
	if err != nil {
		return nil, err
	}
	return credentialgrantconnect.NewCredentialGrantServiceClient(http.DefaultClient, strings.TrimRight(base, "/")), nil
}

func credentialMetadata(grant *credentialgrant.CredentialGrantMetadata) credentialpermissions.GrantMetadata {
	return credentialpermissions.GrantMetadata{
		ID: grant.GetId(), NodeID: grant.GetNodeId(), LogicalID: grant.GetLogicalId(), Field: grant.GetField(),
		Class: grant.GetClass(), Retention: grant.GetRetention(), Generation: grant.GetGeneration(),
		AckedGeneration: grant.GetAckedGeneration(), GrantedAt: credentialTime(grant.GetGrantedAt()),
		RevokedAt: credentialTime(grant.GetRevokedAt()), ReceiptAt: credentialTime(grant.GetReceiptAt()),
		ReceiptAccepted: grant.GetReceiptAccepted(), PurgeState: grant.GetPurgeState(),
		PurgeReceiptAt: credentialTime(grant.GetPurgeReceiptAt()), PurgeAccepted: grant.GetPurgeReceiptAccepted(), Version: grant.GetVersion(),
	}
}

func credentialTime(value *timestamppb.Timestamp) time.Time {
	if value == nil || !value.IsValid() {
		return time.Time{}
	}
	return value.AsTime().UTC()
}

func bridgeCredentialGrantPermissionProvider() *credentialpermissions.Provider {
	return credentialpermissions.NewProvider(credentialGrantTargetReader{scopes: bridgeScopePermissionProvider()}, credentialGrantMetadataClient{}, bridgeClientConfig().TokenProvider, credentialpermissions.WithOwnerSessionProvider(localCredentialGrantOwnerSession))
}

func localCredentialGrantOwnerSession(context.Context) (credentialpermissions.OwnerSession, error) {
	store, err := operatorsession.DefaultFileStore()
	if err != nil {
		return credentialpermissions.OwnerSession{}, err
	}
	resolution, err := (operatorsession.LocalResolver{Store: store}).Resolve()
	if err != nil {
		return credentialpermissions.OwnerSession{}, err
	}
	if strings.TrimSpace(resolution.Enrollment.OperatorID) == "" || strings.TrimSpace(resolution.Token) == "" {
		return credentialpermissions.OwnerSession{}, errors.New("Bridge owner session identity is unavailable")
	}
	return credentialpermissions.OwnerSession{Token: operatorsession.LocalSessionScheme + " " + resolution.Token, OwnerID: resolution.Enrollment.OperatorID}, nil
}
