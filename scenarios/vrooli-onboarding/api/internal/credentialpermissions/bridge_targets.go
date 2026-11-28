package credentialpermissions

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/bridgepermissions"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

// ConnectTargetReader resolves only Bridge's active Machine-to-current-Node
// projection. It never accepts an empty/dynamic target and never accesses
// credential-grant or credential-value APIs.
type ConnectTargetReader struct {
	nodes    bridgepermissions.NodeRegistryClient
	machines bridgepermissions.MachineClient
}

func NewConnectTargetReader(nodes bridgepermissions.NodeRegistryClient, machines bridgepermissions.MachineClient) *ConnectTargetReader {
	return &ConnectTargetReader{nodes: nodes, machines: machines}
}

func (r *ConnectTargetReader) CurrentTargets(ctx context.Context, token string) ([]Target, error) {
	if err := r.available(token); err != nil {
		return nil, err
	}
	nReq := connect.NewRequest(&registryv1.ListNodesRequest{})
	nReq.Header().Set("Authorization", token)
	nResp, err := r.nodes.ListNodes(ctx, nReq)
	if err != nil {
		return nil, err
	}
	mReq := connect.NewRequest(&machinesv1.ListMachinesRequest{})
	mReq.Header().Set("Authorization", token)
	mResp, err := r.machines.ListMachines(ctx, mReq)
	if err != nil {
		return nil, err
	}
	if nResp == nil || nResp.Msg == nil || mResp == nil || mResp.Msg == nil {
		return nil, connect.NewError(connect.CodeDataLoss, errors.New("Bridge returned incomplete target inventory"))
	}
	nodes := map[string]*registryv1.Node{}
	for _, node := range nResp.Msg.GetNodes() {
		if node != nil && node.GetId() != "" {
			nodes[node.GetId()] = node
		}
	}
	out := make([]Target, 0)
	for _, machine := range mResp.Msg.GetMachines() {
		if machine == nil || machine.GetLifecycle() != "active" || machine.GetEnrollmentGeneration() <= 0 {
			continue
		}
		detail, err := r.getMachine(ctx, token, machine.GetId())
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				continue
			}
			return nil, err
		}
		if detail.GetCurrentNode() == nil {
			continue
		}
		node := nodes[detail.GetCurrentNode().GetNodeId()]
		if !currentBindingMatches(detail, machine.GetId(), node) {
			continue
		}
		out = append(out, Target{Binding: permissions.TargetBinding{MachineID: machine.GetId(), NodeID: node.GetId(), EnrollmentGeneration: uint64(detail.GetMachine().GetEnrollmentGeneration())}})
	}
	return out, nil
}

func (r *ConnectTargetReader) CurrentTarget(ctx context.Context, token string, want permissions.TargetBinding) (Target, error) {
	if err := validateTarget(want); err != nil {
		return Target{}, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err := r.available(token); err != nil {
		return Target{}, err
	}
	detail, err := r.getMachine(ctx, token, want.MachineID)
	if err != nil {
		return Target{}, err
	}
	if detail.GetCurrentNode() == nil || detail.GetCurrentNode().GetNodeId() != want.NodeID || detail.GetMachine().GetEnrollmentGeneration() <= 0 || uint64(detail.GetMachine().GetEnrollmentGeneration()) != want.EnrollmentGeneration || !hasCurrentLineage(detail, want.NodeID) {
		return Target{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("Bridge machine current node or enrollment generation changed"))
	}
	nReq := connect.NewRequest(&registryv1.GetNodeRequest{Id: want.NodeID})
	nReq.Header().Set("Authorization", token)
	nResp, err := r.nodes.GetNode(ctx, nReq)
	if err != nil {
		return Target{}, err
	}
	if nResp == nil || nResp.Msg == nil || nResp.Msg.GetNode() == nil {
		return Target{}, connect.NewError(connect.CodeDataLoss, errors.New("Bridge returned no current node"))
	}
	node := nResp.Msg.GetNode()
	if !currentBindingMatches(detail, want.MachineID, node) || node.GetId() != want.NodeID {
		return Target{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("Bridge target node is revoked or no longer current"))
	}
	return Target{Binding: want}, nil
}

func (r *ConnectTargetReader) available(token string) error {
	if r == nil || r.nodes == nil || r.machines == nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("Bridge target reader is unavailable"))
	}
	if strings.TrimSpace(token) == "" {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("Bridge owner authorization is required"))
	}
	return nil
}
func (r *ConnectTargetReader) getMachine(ctx context.Context, token, id string) (*machinesv1.GetMachineResponse, error) {
	if strings.TrimSpace(id) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("machine ID is required"))
	}
	req := connect.NewRequest(&machinesv1.GetMachineRequest{Id: id})
	req.Header().Set("Authorization", token)
	resp, err := r.machines.GetMachine(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetMachine() == nil {
		return nil, connect.NewError(connect.CodeDataLoss, errors.New("Bridge returned no machine"))
	}
	if resp.Msg.GetMachine().GetLifecycle() != "active" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Bridge machine is not active"))
	}
	return resp.Msg, nil
}

func currentBindingMatches(detail *machinesv1.GetMachineResponse, machineID string, node *registryv1.Node) bool {
	if detail == nil || detail.GetMachine() == nil || detail.GetCurrentNode() == nil || node == nil {
		return false
	}
	m := detail.GetMachine()
	lineageOK := hasCurrentLineage(detail, node.GetId())
	return m.GetId() == machineID && m.GetLifecycle() == "active" && m.GetEnrollmentGeneration() > 0 && m.GetEnrollmentGeneration() == node.GetEnrollmentGeneration() && detail.GetCurrentNode().GetNodeId() == node.GetId() && detail.GetCurrentNode().GetEnrollmentGeneration() == m.GetEnrollmentGeneration() && lineageOK && node.GetRevokedAt() == nil && node.GetStatus() != registryv1.NodeStatus_NODE_STATUS_REVOKED
}
func hasCurrentLineage(detail *machinesv1.GetMachineResponse, nodeID string) bool {
	if detail == nil || detail.GetMachine() == nil {
		return false
	}
	count := 0
	matched := false
	for _, lineage := range detail.GetMachine().GetNodeLineage() {
		if lineage != nil && lineage.GetCurrent() {
			count++
			matched = lineage.GetNodeId() == nodeID && lineage.GetEnrollmentGeneration() == detail.GetMachine().GetEnrollmentGeneration()
		}
	}
	return count == 1 && matched
}

var _ TargetReader = (*ConnectTargetReader)(nil)
