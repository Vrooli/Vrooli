package main

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/api-core/targetmodel"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	all := append([]targetConnection{{Target: localTerminalTarget()}}, remote...)
	targets := make([]*sharedv1.Target, 0, len(all))
	for _, target := range all {
		targets = append(targets, targetToProto(target))
	}
	projected := &targetsv1.ListResponse{
		State:   state,
		Targets: targets,
		Message: message,
	}
	setCatalogText(projected, "recovery_action", action)
	return connect.NewResponse(projected), nil
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
		summary = targetText(target, "failure_rung")
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

func (s *Server) targetByID(id string) (targetConnection, bool) {
	if id == "local" {
		return targetConnection{Target: localTerminalTarget()}, true
	}
	for _, target := range s.remoteTargets() {
		if target.ID == id {
			return target, true
		}
	}
	return targetConnection{}, false
}

func localTerminalTarget() targetmodel.Target {
	return targetmodel.Target{
		ID: "local", Ramp: "local", Label: "This machine", Platform: "local",
		OS: runtime.GOOS, Architecture: runtime.GOARCH, DeviceKind: "local",
		Available: true, Mode: "dispatchable", SurvivesRestart: true,
		Transport: targetmodel.Transport{Kind: targetmodel.TransportLocal, ID: "local", Available: true},
		Health:    targetmodel.TargetHealth{Status: "LOCAL"},
		Readiness: []targetmodel.ReadinessCheck{{Identity: "local_process", Label: "Web Console process", Passed: true, Detail: "This machine is available to the Web Console"}},
	}
}

func readinessFactsForNode(node *registryv1.Node) []targetmodel.ReadinessCheck {
	hasSessionGrant := false
	for _, scope := range node.GetScopes() {
		transportScope, ok := scopecatalog.TransportScope("interactive-session:write")
		if ok && (scope == transportScope || scope == "vrooli-bridge:*" || scope == "*:write" || scope == "*") {
			hasSessionGrant = true
			break
		}
	}
	return []targetmodel.ReadinessCheck{
		targetmodel.ReadinessCheckFor(targetmodel.ReadinessRegistry, node.GetRegistryRecordPresent(), "Bridge registry record is present"),
		targetmodel.ReadinessCheckFor(targetmodel.ReadinessHeartbeat, node.GetHeartbeatFresh(), "Node heartbeat is within the freshness window"),
		targetmodel.ReadinessCheckFor(targetmodel.ReadinessChannel, node.GetChannelHeld(), "Bridge has a live channel to this node"),
		targetmodel.ReadinessCheckFor(targetmodel.ReadinessProtocol, node.GetProtocolCompatible(), "Node and Bridge can speak the same session protocol"),
		targetmodel.ReadinessCheckFor(targetmodel.ReadinessDispatch, node.GetDispatchable(), "Web Console may start a session on this node"),
		targetmodel.ReadinessCheckFor(targetmodel.ReadinessBridgeScope, hasSessionGrant, grantDetailForScopes(node.GetScopes())),
		targetmodel.ReadinessCheckFor(targetmodel.ReadinessSessionSupport, hasSessionGrant && node.GetKind() == registryv1.NodeKind_NODE_KIND_AGENT, "Interactive sessions require a Bridge agent and the vrooli-bridge:write grant"),
	}
}

func grantSummaryForScopes(scopes []string) string {
	return scopecatalog.SummarizeScopes(scopes)
}

func grantDetailForScopes(scopes []string) string {
	summary := grantSummaryForScopes(scopes)
	clean := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		clean = append(clean, scope)
	}
	sort.Strings(clean)
	if len(clean) == 0 {
		return summary
	}
	return summary + ". Granted scopes: " + strings.Join(clean, ", ")
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

func remoteCatalogState(targets []targetConnection) (targetsv1.CatalogState, string, string) {
	if len(targets) == 0 {
		return targetsv1.CatalogState_CATALOG_STATE_CONFIGURED_EMPTY,
			"Bridge is connected, but no remote nodes are registered.",
			"Register a node with vrooli-bridge, then refresh this catalog."
	}
	if len(targets) == 1 && !targets[0].Available {
		failure := strings.ToLower(targets[0].Reason)
		switch {
		case strings.Contains(failure, "credential") || strings.Contains(failure, "configured"):
			return targetsv1.CatalogState_CATALOG_STATE_UNCONFIGURED,
				"Remote nodes are not configured for this Web Console.",
				targets[0].NextAction
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

func targetToProto(target targetConnection) *sharedv1.Target {
	facts := make([]*sharedv1.ReadinessFact, 0, len(target.Readiness))
	for _, fact := range target.Readiness {
		facts = append(facts, &sharedv1.ReadinessFact{Key: fact.Identity, Label: fact.Label, Passed: fact.Passed, Detail: fact.Detail})
	}
	var lastSeen *timestamppb.Timestamp
	if !target.LastSeenAt.IsZero() {
		lastSeen = timestamppb.New(target.LastSeenAt)
	}
	projected := &sharedv1.Target{
		Id:              target.ID,
		Kind:            target.DeviceKind,
		Label:           target.Label,
		Os:              target.OS,
		Arch:            target.Architecture,
		NodeId:          target.NodeID,
		Revision:        target.Revision,
		Status:          target.Health.Status,
		Online:          target.BridgeTrust != nil && target.BridgeTrust.Online,
		LastSeenAt:      lastSeen,
		Readiness:       facts,
		Dispatchable:    target.Available,
		State:           targetProtoState(targetAvailability(target)),
		SurvivesRestart: target.SurvivesRestart,
	}
	setTargetText(projected, "failure_rung", target.Reason)
	setTargetText(projected, "recovery_action", target.NextAction)
	return projected
}

func targetAvailability(target targetConnection) string {
	if strings.TrimSpace(target.Mode) != "" {
		return target.Mode
	}
	if target.Available {
		return "dispatchable"
	}
	return "unavailable"
}

func setTargetText(target *sharedv1.Target, name, value string) {
	if target == nil || value == "" {
		return
	}
	field := target.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(name))
	if field != nil && field.Kind() == protoreflect.StringKind {
		target.ProtoReflect().Set(field, protoreflect.ValueOfString(value))
	}
}

func targetText(target *sharedv1.Target, name string) string {
	if target == nil {
		return ""
	}
	field := target.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(name))
	if field == nil || field.Kind() != protoreflect.StringKind {
		return ""
	}
	return target.ProtoReflect().Get(field).String()
}

func setCatalogText(response *targetsv1.ListResponse, name, value string) {
	if response == nil || value == "" {
		return
	}
	field := response.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(name))
	if field != nil && field.Kind() == protoreflect.StringKind {
		response.ProtoReflect().Set(field, protoreflect.ValueOfString(value))
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
