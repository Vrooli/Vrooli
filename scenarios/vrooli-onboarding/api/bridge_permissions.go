package main

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines/machines_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry/registry_v1connect"
	bridgepermissions "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/bridgepermissions"
)

type discoveredNodeRegistryClient struct{}
type discoveredMachineClient struct{}

func (*discoveredNodeRegistryClient) ListNodes(ctx context.Context, req *connect.Request[registryv1.ListNodesRequest]) (*connect.Response[registryv1.ListNodesResponse], error) {
	client, err := discoveredNodeClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.ListNodes(ctx, req)
}
func (*discoveredNodeRegistryClient) GetNode(ctx context.Context, req *connect.Request[registryv1.GetNodeRequest]) (*connect.Response[registryv1.GetNodeResponse], error) {
	client, err := discoveredNodeClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.GetNode(ctx, req)
}
func (*discoveredNodeRegistryClient) UpdateNode(ctx context.Context, req *connect.Request[registryv1.UpdateNodeRequest]) (*connect.Response[registryv1.UpdateNodeResponse], error) {
	client, err := discoveredNodeClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.UpdateNode(ctx, req)
}
func (*discoveredMachineClient) ListMachines(ctx context.Context, req *connect.Request[machinesv1.ListMachinesRequest]) (*connect.Response[machinesv1.ListMachinesResponse], error) {
	client, err := discoveredMachineServiceClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.ListMachines(ctx, req)
}
func (*discoveredMachineClient) GetMachine(ctx context.Context, req *connect.Request[machinesv1.GetMachineRequest]) (*connect.Response[machinesv1.GetMachineResponse], error) {
	client, err := discoveredMachineServiceClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.GetMachine(ctx, req)
}

func discoveredBridgeBase(ctx context.Context) (string, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "vrooli-bridge")
	if err != nil {
		return "", err
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return "", errors.New("Bridge URL discovery returned an empty endpoint")
	}
	return base, nil
}
func discoveredNodeClient(ctx context.Context) (registryconnect.NodeRegistryServiceClient, error) {
	base, err := discoveredBridgeBase(ctx)
	if err != nil {
		return nil, err
	}
	return registryconnect.NewNodeRegistryServiceClient(http.DefaultClient, base), nil
}
func discoveredMachineServiceClient(ctx context.Context) (machinesconnect.MachineServiceClient, error) {
	base, err := discoveredBridgeBase(ctx)
	if err != nil {
		return nil, err
	}
	return machinesconnect.NewMachineServiceClient(http.DefaultClient, base), nil
}

func bridgeScopePermissionProvider() *bridgepermissions.Provider {
	return bridgepermissions.NewProvider(&discoveredNodeRegistryClient{}, &discoveredMachineClient{}, bridgeClientConfig().TokenProvider)
}
