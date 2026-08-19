package main

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/connectx"
	"google.golang.org/protobuf/types/known/timestamppb"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shared"
	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/targets"
	targetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/targets/targets_v1connect"
)

// targetCatalogRPC is the safe, typed projection of the server-side Bridge
// adapter. It deliberately does not expose the adapter's credential-bearing
// fields or allow clients to choose an arbitrary Bridge endpoint.
type targetCatalogRPC struct {
	server *Server
}

func (h *targetCatalogRPC) List(_ context.Context, _ *connect.Request[targetsv1.ListRequest]) (*connect.Response[targetsv1.ListResponse], error) {
	remote := h.server.remoteTargets()
	state, message, action := remoteCatalogState(remote)
	all := append([]remoteTerminalTarget{localTerminalTarget()}, remote...)
	targets := make([]*sharedv1.Target, 0, len(all))
	for _, target := range all {
		targets = append(targets, targetToProto(target))
	}
	return connect.NewResponse(&targetsv1.ListResponse{
		State:          state,
		Targets:        targets,
		Message:        message,
		RecoveryAction: action,
	}), nil
}

func (h *targetCatalogRPC) Get(_ context.Context, req *connect.Request[targetsv1.GetRequest]) (*connect.Response[targetsv1.GetResponse], error) {
	target, ok := h.server.targetByID(strings.TrimSpace(req.Msg.GetId()))
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("target %q not found", req.Msg.GetId()))
	}
	return connect.NewResponse(&targetsv1.GetResponse{Target: targetToProto(target)}), nil
}

func (h *targetCatalogRPC) Doctor(ctx context.Context, req *connect.Request[targetsv1.DoctorRequest]) (*connect.Response[targetsv1.DoctorResponse], error) {
	response, err := h.Get(ctx, connect.NewRequest(&targetsv1.GetRequest{Id: req.Msg.GetId()}))
	if err != nil {
		return nil, err
	}
	target := response.Msg.GetTarget()
	summary := "target is dispatchable"
	if !target.GetDispatchable() {
		summary = target.GetFailureRung()
		if summary == "" {
			summary = "target is not dispatchable"
		}
	}
	return connect.NewResponse(&targetsv1.DoctorResponse{Target: target, Summary: summary}), nil
}

func (s *Server) mountTargetCatalog() {
	path, handler := targetsconnect.NewTargetCatalogServiceHandler(&targetCatalogRPC{server: s})
	connectx.RegisterServices(s.router, connectx.ServiceMount{Path: path, Handler: handler})
}

func (s *Server) targetByID(id string) (remoteTerminalTarget, bool) {
	if id == "local" {
		return localTerminalTarget(), true
	}
	for _, target := range s.remoteTargets() {
		if target.ID == id {
			return target, true
		}
	}
	return remoteTerminalTarget{}, false
}

func localTerminalTarget() remoteTerminalTarget {
	return remoteTerminalTarget{
		ID:              "local",
		Kind:            "local",
		Label:           "This machine",
		OS:              runtime.GOOS,
		Arch:            runtime.GOARCH,
		Status:          "LOCAL",
		Online:          true,
		Available:       true,
		State:           "dispatchable",
		SurvivesRestart: true,
		Readiness:       []string{"local Web Console process available"},
		ReadinessFacts: []remoteReadinessFact{{
			Key: "local_process", Label: "Web Console process", Passed: true, Detail: "This machine is available to the Web Console",
		}},
	}
}

func readinessFactsForNode(node *registryv1.Node) []remoteReadinessFact {
	return []remoteReadinessFact{
		{Key: "registry_record", Label: "Registered", Passed: node.GetRegistryRecordPresent(), Detail: "Bridge registry record is present"},
		{Key: "heartbeat", Label: "Heartbeat fresh", Passed: node.GetHeartbeatFresh(), Detail: "Node heartbeat is within the freshness window"},
		{Key: "channel", Label: "Live channel", Passed: node.GetChannelHeld(), Detail: "Bridge has a live channel to this node"},
		{Key: "protocol", Label: "Protocol compatible", Passed: node.GetProtocolCompatible(), Detail: "Node and Bridge can speak the same session protocol"},
		{Key: "dispatch", Label: "Dispatchable", Passed: node.GetDispatchable(), Detail: "Web Console may start a session on this node"},
	}
}

func targetStateForNode(node *registryv1.Node, dispatchable bool) string {
	if dispatchable {
		return "dispatchable"
	}
	switch node.GetStatus() {
	case registryv1.NodeStatus_NODE_STATUS_NEEDS_UPDATE:
		return "needs-update"
	case registryv1.NodeStatus_NODE_STATUS_OFFLINE:
		return "offline"
	default:
		if !node.GetOnline() || !node.GetHeartbeatFresh() {
			return "offline"
		}
		return "unavailable"
	}
}

func recoveryActionForNode(node *registryv1.Node, failure string) string {
	switch failure {
	case "heartbeat freshness", "live channel":
		return "Reconnect the Bridge agent on this node, then refresh the catalog"
	case "protocol compatibility":
		return "Update the Bridge agent on this node, then refresh the catalog"
	case "dispatchability":
		return "Check this node's Bridge session capability and owner grants"
	default:
		if node.GetKind() != registryv1.NodeKind_NODE_KIND_AGENT {
			return "This node is registered, but it does not host Web Console agent sessions"
		}
		return "Refresh the catalog and inspect Bridge readiness"
	}
}

func remoteCatalogState(targets []remoteTerminalTarget) (targetsv1.CatalogState, string, string) {
	if len(targets) == 0 {
		return targetsv1.CatalogState_CATALOG_STATE_CONFIGURED_EMPTY,
			"Bridge is connected, but no remote nodes are registered.",
			"Register a node with vrooli-bridge, then refresh this catalog."
	}
	if len(targets) == 1 && !targets[0].Available {
		failure := strings.ToLower(targets[0].FailureRung)
		switch {
		case strings.Contains(failure, "credential") || strings.Contains(failure, "configured"):
			return targetsv1.CatalogState_CATALOG_STATE_UNCONFIGURED,
				"Remote nodes are not configured for this Web Console.",
				targets[0].RecoveryAction
		case strings.Contains(failure, "registry"):
			return targetsv1.CatalogState_CATALOG_STATE_REGISTRY_ERROR,
				"Bridge is configured, but its node registry could not be read.",
				"Check Bridge health and refresh the catalog."
		case strings.Contains(failure, "no registered"):
			return targetsv1.CatalogState_CATALOG_STATE_CONFIGURED_EMPTY,
				"Bridge is connected, but no remote nodes are registered.",
				"Register a node with vrooli-bridge, then refresh this catalog."
		}
	}
	return targetsv1.CatalogState_CATALOG_STATE_READY,
		"Remote node readiness is shown for every registered node.", ""
}

func targetToProto(target remoteTerminalTarget) *sharedv1.Target {
	facts := make([]*sharedv1.ReadinessFact, 0, len(target.ReadinessFacts))
	for _, fact := range target.ReadinessFacts {
		facts = append(facts, &sharedv1.ReadinessFact{Key: fact.Key, Label: fact.Label, Passed: fact.Passed, Detail: fact.Detail})
	}
	if len(facts) == 0 {
		for _, fact := range target.Readiness {
			facts = append(facts, &sharedv1.ReadinessFact{Key: fact, Label: fact, Passed: target.Available, Detail: fact})
		}
	}
	var lastSeen *timestamppb.Timestamp
	if !target.LastSeenAt.IsZero() {
		lastSeen = timestamppb.New(target.LastSeenAt)
	}
	return &sharedv1.Target{
		Id:              target.ID,
		Kind:            target.Kind,
		Label:           target.Label,
		Os:              target.OS,
		Arch:            target.Arch,
		NodeId:          target.NodeID,
		Revision:        target.Revision,
		Status:          target.Status,
		Online:          target.Online,
		LastSeenAt:      lastSeen,
		Readiness:       facts,
		Dispatchable:    target.Available,
		FailureRung:     target.FailureRung,
		State:           targetProtoState(target.State),
		RecoveryAction:  target.RecoveryAction,
		SurvivesRestart: target.SurvivesRestart,
	}
}

func targetProtoState(state string) sharedv1.TargetState {
	switch state {
	case "dispatchable":
		return sharedv1.TargetState_TARGET_STATE_DISPATCHABLE
	case "offline":
		return sharedv1.TargetState_TARGET_STATE_OFFLINE
	case "needs-update":
		return sharedv1.TargetState_TARGET_STATE_NEEDS_UPDATE
	default:
		return sharedv1.TargetState_TARGET_STATE_UNAVAILABLE
	}
}

var _ targetsconnect.TargetCatalogServiceHandler = (*targetCatalogRPC)(nil)
