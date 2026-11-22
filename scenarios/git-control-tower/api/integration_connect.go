package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/collaboration"
	integrationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/integration"
	integrationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/integration/integration_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ integrationconnect.IntegrationServiceHandler = integrationConnectServer{}

type integrationConnectServer struct{ server *Server }

func (s integrationConnectServer) GetCapabilities(ctx context.Context, _ *connect.Request[integrationv1.GetCapabilitiesRequest]) (*connect.Response[integrationv1.GetCapabilitiesResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	states := s.server.capabilities.Resolve(ctx)
	response := &integrationv1.GetCapabilitiesResponse{
		Capabilities: make([]*structpb.Struct, 0, len(states)),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
	for _, state := range states {
		encoded, err := json.Marshal(state)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		var value map[string]any
		if err := json.Unmarshal(encoded, &value); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		capability, err := structpb.NewStruct(value)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		response.Capabilities = append(response.Capabilities, capability)
	}
	return connect.NewResponse(response), nil
}

func (s integrationConnectServer) GetCollaborationStatus(_ context.Context, req *connect.Request[integrationv1.GetCollaborationStatusRequest]) (*connect.Response[integrationv1.GetCollaborationStatusResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("collaboration status request is required"))
	}
	identity := collaboration.HostIdentity{
		Kind:           collaboration.HostKind(strings.TrimSpace(req.Msg.GetKind())),
		InstanceURL:    strings.TrimSpace(req.Msg.GetInstanceUrl()),
		InstallationID: strings.TrimSpace(req.Msg.GetInstallationId()),
		AccountID:      strings.TrimSpace(req.Msg.GetAccountId()),
		RepositoryID:   strings.TrimSpace(req.Msg.GetRepositoryId()),
	}
	response := &integrationv1.GetCollaborationStatusResponse{}
	if err := identity.Validate(); err != nil {
		response.Status = "unconfigured"
		response.Reason = err.Error()
		response.Capabilities = collaborationCapabilityStatuses(unavailableCapabilities())
		return connect.NewResponse(response), nil
	}
	response.Status = "unavailable"
	response.Reason = "Integration Hub host adapter is not active"
	response.Host = &integrationv1.CollaborationHostIdentity{
		Kind: string(identity.Kind), InstanceUrl: identity.InstanceURL,
		InstallationId: identity.InstallationID, AccountId: identity.AccountID,
		RepositoryId: identity.RepositoryID,
	}
	response.Capabilities = collaborationCapabilityStatuses(unavailableCapabilities())
	return connect.NewResponse(response), nil
}

func collaborationCapabilityStatuses(capabilities []collaboration.CapabilityStatus) []*integrationv1.CollaborationCapabilityStatus {
	result := make([]*integrationv1.CollaborationCapabilityStatus, 0, len(capabilities))
	for _, capability := range capabilities {
		result = append(result, &integrationv1.CollaborationCapabilityStatus{
			Capability: string(capability.Capability), Standing: string(capability.Standing),
			Scopes: append([]string(nil), capability.Scopes...), Reason: capability.Reason,
		})
	}
	return result
}

func unavailableCapabilities() []collaboration.CapabilityStatus {
	return []collaboration.CapabilityStatus{
		{Capability: collaboration.CapabilityReadChange, Standing: collaboration.StandingUnconfigured},
		{Capability: collaboration.CapabilityComments, Standing: collaboration.StandingUnconfigured},
		{Capability: collaboration.CapabilityReviews, Standing: collaboration.StandingUnconfigured},
		{Capability: collaboration.CapabilityChecks, Standing: collaboration.StandingUnconfigured},
		{Capability: collaboration.CapabilityReleases, Standing: collaboration.StandingUnconfigured},
		{Capability: collaboration.CapabilityPublish, Standing: collaboration.StandingUnconfigured},
	}
}
