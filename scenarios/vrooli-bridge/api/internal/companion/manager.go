package companion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	companionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/companion"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var ErrTargetUnavailable = errors.New("companion target is unavailable")

// CommandPusher is the authenticated Bridge-channel handoff to the node
// agent. It carries lifecycle metadata only; the agent owns the user-context
// LaunchAgent and returns the observed result through Deliver.
type CommandPusher interface {
	PushCompanionCommand(context.Context, string, *companionv1.CompanionCommand) error
}

type ArtifactRequest struct {
	NodeID          string
	Name            string
	SourceRef       string
	DestinationPath string
}

type ArtifactDecision struct {
	DistributionID string
	Pending        bool
	Failed         bool
	Reason         string
}

// ArtifactDistributor is deliberately narrow: the artifacts domain remains
// the authority for byte movement and durable distribution receipts.
type ArtifactDistributor interface {
	Distribute(context.Context, ArtifactRequest) (ArtifactDecision, error)
}

type pendingArtifact struct {
	nodeID  string
	command *companionv1.CompanionCommand
}

type Manager struct {
	mu        sync.Mutex
	nodes     map[string]*companionv1.CompanionOperation
	allowed   func(string) bool
	pusher    CommandPusher
	artifacts ArtifactDistributor
	pending   map[string]pendingArtifact
}

func NewManager(allowed func(string) bool, pushers ...CommandPusher) *Manager {
	var pusher CommandPusher
	if len(pushers) > 0 {
		pusher = pushers[0]
	}
	return &Manager{nodes: make(map[string]*companionv1.CompanionOperation), allowed: allowed, pusher: pusher, pending: make(map[string]pendingArtifact)}
}

func (m *Manager) SetArtifactDistributor(distributor ArtifactDistributor) { m.artifacts = distributor }

func (m *Manager) Install(node, version, display string) (*companionv1.CompanionOperation, error) {
	return m.dispatch(context.Background(), node, version, display, companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_INSTALL, "install_pending", nil)
}

func (m *Manager) InstallWithArtifact(ctx context.Context, node, version, display, sourceRef, name, destinationPath string) (*companionv1.CompanionOperation, error) {
	return m.dispatch(ctx, node, version, display, companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_INSTALL, "install_pending", &ArtifactRequest{NodeID: node, SourceRef: sourceRef, Name: name, DestinationPath: destinationPath})
}

func (m *Manager) Inspect(node string) (*companionv1.CompanionOperation, error) {
	if err := m.check(node); err != nil {
		return nil, err
	}
	m.mu.Lock()
	current, ok := m.nodes[node]
	m.mu.Unlock()
	if ok {
		return clone(current), nil
	}
	return m.dispatch(context.Background(), node, "", "", companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_INSPECT, "probe_pending", nil)
}

func (m *Manager) Upgrade(node, version string) (*companionv1.CompanionOperation, error) {
	return m.dispatch(context.Background(), node, version, "", companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_UPGRADE, "upgrade_pending", nil)
}

func (m *Manager) UpgradeWithArtifact(ctx context.Context, node, version, sourceRef, name, destinationPath string) (*companionv1.CompanionOperation, error) {
	return m.dispatch(ctx, node, version, "", companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_UPGRADE, "upgrade_pending", &ArtifactRequest{NodeID: node, SourceRef: sourceRef, Name: name, DestinationPath: destinationPath})
}

func (m *Manager) Revoke(node, reason string) (*companionv1.CompanionOperation, error) {
	return m.dispatch(context.Background(), node, "", "", companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_REVOKE, "revoke_pending:"+strings.TrimSpace(reason), nil)
}

func (m *Manager) Remove(node string) (*companionv1.CompanionOperation, error) {
	return m.dispatch(context.Background(), node, "", "", companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_REMOVE, "remove_pending", nil)
}

// DeliverArtifactReceipt advances an install/upgrade only after the signed
// node receipt confirms that the executable reached its target path.
func (m *Manager) DeliverArtifactReceipt(ctx context.Context, distributionID, nodeID string, accepted bool, reason string) bool {
	m.mu.Lock()
	pending, ok := m.pending[distributionID]
	if ok {
		delete(m.pending, distributionID)
	}
	m.mu.Unlock()
	if !ok || pending.nodeID != nodeID {
		return false
	}
	if !accepted {
		m.setFailure(nodeID, pending.command.GetOperationId(), "artifact_placement_failed", strings.TrimSpace(reason))
		return true
	}
	if m.pusher == nil || m.pusher.PushCompanionCommand(ctx, nodeID, pending.command) != nil {
		m.setFailure(nodeID, pending.command.GetOperationId(), "companion_delivery_failed", "restore_the_node_channel")
		return true
	}
	m.mu.Lock()
	if op := m.nodes[nodeID]; op != nil && op.GetOperationId() == pending.command.GetOperationId() {
		op.ReasonCode = "install_pending"
		op.Recovery = "await_node_lifecycle_receipt"
		op.ArtifactDistributionId = distributionID
		op.ObservedAt = timestamppb.Now()
	}
	m.mu.Unlock()
	return true
}

// Deliver applies a node-authenticated lifecycle observation. Unknown or
// mismatched operation IDs are ignored so late receipts cannot mutate a new
// operation's state.
func (m *Manager) Deliver(response *companionv1.CompanionResponse) bool {
	if response == nil || strings.TrimSpace(response.GetOperationId()) == "" || strings.TrimSpace(response.GetNodeId()) == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.nodes[response.GetNodeId()]
	if !ok || current.GetOperationId() != response.GetOperationId() {
		return false
	}
	current.Version = response.GetVersion()
	current.State = response.GetState()
	current.ReasonCode = response.GetReasonCode()
	current.Recovery = response.GetRecovery()
	current.UserLaunchAgent = response.GetUserLaunchAgent()
	current.ObservedAt = response.GetObservedAt()
	return true
}

func (m *Manager) check(node string) error {
	if strings.TrimSpace(node) == "" || m.allowed == nil || !m.allowed(node) {
		return ErrTargetUnavailable
	}
	return nil
}

func (m *Manager) dispatch(ctx context.Context, node, version, display string, kind companionv1.CompanionOperationKind, reason string, artifact *ArtifactRequest) (*companionv1.CompanionOperation, error) {
	if err := m.check(node); err != nil {
		return nil, err
	}
	op := m.set(node, version, companionv1.CompanionState_COMPANION_STATE_INSTALLING, reason, "await_node_lifecycle_receipt", false)
	if m.pusher == nil {
		return m.setFailure(node, op.GetOperationId(), "companion_transport_unavailable", "connect_the_node_agent"), nil
	}
	command := &companionv1.CompanionCommand{OperationId: op.GetOperationId(), NodeId: node, Kind: kind, Version: version, DisplayId: display}
	if artifact != nil {
		if m.artifacts == nil {
			return m.setFailure(node, op.GetOperationId(), "artifact_transport_unavailable", "configure_companion_artifact_distribution"), nil
		}
		decision, err := m.artifacts.Distribute(ctx, *artifact)
		if err != nil || decision.Failed {
			recovery := strings.TrimSpace(decision.Reason)
			if recovery == "" {
				recovery = "verify_companion_artifact_source"
			}
			return m.setFailure(node, op.GetOperationId(), "artifact_distribution_failed", recovery), nil
		}
		if decision.Pending && strings.TrimSpace(decision.DistributionID) == "" {
			return m.setFailure(node, op.GetOperationId(), "artifact_distribution_failed", "artifact distributor returned no operation id"), nil
		}
		if decision.Pending {
			m.mu.Lock()
			m.pending[decision.DistributionID] = pendingArtifact{nodeID: node, command: command}
			current := m.nodes[node]
			current.ReasonCode = "artifact_pending"
			current.Recovery = "await_artifact_placement_receipt"
			current.ArtifactDistributionId = decision.DistributionID
			current.ObservedAt = timestamppb.Now()
			pendingOp := clone(current)
			m.mu.Unlock()
			return pendingOp, nil
		}
	}
	if err := m.pusher.PushCompanionCommand(ctx, node, command); err != nil {
		return m.setFailure(node, op.GetOperationId(), "companion_delivery_failed", "restore_the_node_channel"), nil
	}
	return op, nil
}

func (m *Manager) set(node, version string, state companionv1.CompanionState, reason, recovery string, launchAgent bool) *companionv1.CompanionOperation {
	m.mu.Lock()
	defer m.mu.Unlock()
	if version == "" && m.nodes[node] != nil {
		version = m.nodes[node].GetVersion()
	}
	out := &companionv1.CompanionOperation{OperationId: fmt.Sprintf("companion-%d", time.Now().UnixNano()), NodeId: node, Version: version, State: state, ReasonCode: reason, Recovery: recovery, UserLaunchAgent: launchAgent, ObservedAt: timestamppb.Now()}
	m.nodes[node] = out
	return clone(out)
}

func (m *Manager) setFailure(node, operationID, reason, recovery string) *companionv1.CompanionOperation {
	m.mu.Lock()
	defer m.mu.Unlock()
	current := m.nodes[node]
	if current == nil || current.GetOperationId() != operationID {
		return nil
	}
	current.State = companionv1.CompanionState_COMPANION_STATE_FAILED
	current.ReasonCode = reason
	current.Recovery = recovery
	current.ObservedAt = timestamppb.Now()
	return clone(current)
}

func clone(in *companionv1.CompanionOperation) *companionv1.CompanionOperation {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
