package main

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/api-core/nodereach"
	"github.com/vrooli/api-core/targetmodel"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	hostdomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/host"
)

func hostService(server *Server) hostdomain.Service {
	return hostdomain.Service{
		ListRequirementsFn: listHostRequirements,
		FactsFn:            getHostFacts,
		TargetsFn:          func(ctx context.Context) ([]targetmodel.Target, string, error) { return listHostTargets(ctx, server) },
		PatchSafeguardConfigFn: func(ctx context.Context, _ string, name, key string, value any) error {
			return patchHostSafeguardConfig(ctx, name, key, value)
		},
		SetNotificationRecipient: func(ctx context.Context, _ string, subject string) error {
			return patchNotificationRecipient(ctx, subject)
		},
	}
}

func listHostRequirements(ctx context.Context, _ string) ([]hostdomain.Requirement, []hostdomain.Requirement, error) {
	root, err := manifestRoot()
	if err != nil {
		return nil, nil, err
	}
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return nil, nil, err
	}
	models, err := loadScenarioReadModels()
	if err != nil {
		return nil, nil, err
	}
	hostModels, err := hostRequirementScenarioModels(root, models, state)
	if err != nil {
		return nil, nil, err
	}
	response, err := deriveV2HostRequirements(root, state, hostModels)
	if err != nil {
		return nil, nil, err
	}
	tools, safeguards, err := hostRequirementsToDomain(root, response)
	return tools, safeguards, err
}

func hostRequirementsToDomain(root string, response hostRequirementsResponse) ([]hostdomain.Requirement, []hostdomain.Requirement, error) {
	convert := func(kind string, items []hostItem) []hostdomain.Requirement {
		result := make([]hostdomain.Requirement, 0, len(items))
		for _, item := range items {
			observed := inspectToolReadiness(item)
			if kind == "safeguard" {
				observed = inspectSafeguardReadiness(root, item)
			}
			result = append(result, hostdomain.Requirement{
				Name: item.Name, Required: item.Required, Reason: item.Reason, Notes: item.Notes,
				Description: item.Description, Risk: item.Risk, Privilege: item.Privilege, Bundling: item.Bundling,
				Platforms: item.Platforms, Commands: item.Commands, ConfigSchema: item.ConfigSchema, Config: item.Config,
				Status: item.Status, Disposition: hostDisposition(observed.Status, observed.Detail), Detail: observed.Detail,
				Remediation: observed.Remediation,
			})
		}
		return result
	}
	return convert("tool", response.Tools), convert("safeguard", response.Safeguards), nil
}

func hostDisposition(status, detail string) string {
	lower := strings.ToLower(detail)
	if strings.Contains(lower, "permission denied") || strings.Contains(lower, "operation not permitted") {
		return "permission_denied"
	}
	if strings.Contains(lower, "content mismatch") || strings.Contains(lower, "does not match") || strings.Contains(lower, "differs") {
		return "content_mismatch"
	}
	switch status {
	case "ready":
		return "ready"
	case "missing":
		return "missing"
	case "deferred":
		return "deferred"
	case "unsupported":
		return "unsupported"
	case "not_applicable":
		return "not_applicable"
	default:
		return ""
	}
}

func getHostFacts(ctx context.Context, _ string) (hostdomain.Facts, error) {
	hostFactsMu.Lock()
	if time.Since(cachedHostFactsAt) < time.Minute {
		response := cachedHostFacts
		hostFactsMu.Unlock()
		return hostFactsToDomain(response), nil
	}
	hostFactsMu.Unlock()
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	result := make(chan struct {
		response hostFactsResponse
		err      error
	}, 1)
	go func() {
		response, err := hostFactsProbe(probeCtx)
		result <- struct {
			response hostFactsResponse
			err      error
		}{response, err}
	}()
	select {
	case outcome := <-result:
		response := outcome.response
		if outcome.err != nil && !response.Available {
			response.Reason = outcome.err.Error()
		}
		if response.Available {
			hostFactsMu.Lock()
			cachedHostFacts, cachedHostFactsAt = response, operatorStateNow()
			hostFactsMu.Unlock()
		}
		return hostFactsToDomain(response), nil
	case <-probeCtx.Done():
		return hostdomain.Facts{Available: false, Reason: "host inventory probe exceeded the 2 second budget"}, nil
	}
}

func hostFactsToDomain(response hostFactsResponse) hostdomain.Facts {
	return hostdomain.Facts{Available: response.Available, Reason: response.Reason, CPUCount: response.CPUCount, MemoryTotalBytes: response.MemoryTotalBytes, MemoryAvailableBytes: response.MemoryAvailableBytes, DiskFreeBytes: response.DiskFreeBytes, GPUs: response.GPUs, Platform: response.Platform}
}

func listHostTargets(ctx context.Context, server *Server) ([]targetmodel.Target, string, error) {
	targets := []targetmodel.Target{{ID: "local", Label: "This machine", Platform: runtime.GOOS, OS: runtime.GOOS, Architecture: runtime.GOARCH, DeviceKind: "local", Available: true, Transport: targetmodel.Transport{Kind: targetmodel.TransportLocal, ID: "local", Available: true}, Health: targetmodel.TargetHealth{Status: "local"}}}
	if server.bridge == nil {
		server.bridge = nodereach.New(nodereach.Config{})
	}
	nodes, err := server.bridge.List(ctx, 5*time.Second)
	if err != nil {
		return targets, "bridge unavailable", nil
	}
	for _, node := range nodes {
		if node == nil || node.GetId() == "" {
			continue
		}
		targets = append(targets, nodeToTarget(node))
	}
	return targets, "", nil
}

func nodeToTarget(node *registryv1.Node) targetmodel.Target {
	status := node.GetStatus().String()
	available := node.GetOnline() && node.GetStatus() == registryv1.NodeStatus_NODE_STATUS_ONLINE
	checks := []targetmodel.ReadinessCheck{
		targetmodel.ReadinessCheckFor(targetmodel.ReadinessRegistry, node.GetRegistryRecordPresent(), "registry record"),
		targetmodel.ReadinessCheckFor(targetmodel.ReadinessHeartbeat, node.GetHeartbeatFresh(), "heartbeat freshness"),
		targetmodel.ReadinessCheckFor(targetmodel.ReadinessChannel, node.GetChannelHeld(), "Bridge channel"),
		targetmodel.ReadinessCheckFor(targetmodel.ReadinessProtocol, node.GetProtocolCompatible(), "protocol compatibility"),
		targetmodel.ReadinessCheckFor(targetmodel.ReadinessDispatch, node.GetDispatchable(), "dispatch authorization"),
	}
	reason := ""
	if !available {
		reason = "target is offline or not dispatchable"
	}
	return targetmodel.Target{ID: node.GetId(), Label: node.GetName(), Platform: node.GetOs(), OS: node.GetOs(), Architecture: node.GetArch(), DeviceKind: node.GetKind().String(), Available: available, Reason: reason, NextAction: "bring the target online and refresh", Capabilities: node.GetCapabilities(), Scopes: node.GetScopes(), Transport: targetmodel.Transport{Kind: targetmodel.TransportBridge, ID: node.GetId(), Available: node.GetOnline(), Reason: reason}, Health: targetmodel.TargetHealth{Status: status, Reason: reason}, Readiness: checks}
}

func patchHostSafeguardConfig(ctx context.Context, name, key string, value any) error {
	patch := map[string]any{"host_safeguards": map[string]any{name: map[string]any{"config": map[string]any{key: value}}}}
	data, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = operatorStateService().ApplyValidated(ctx, data, validateOperatorState)
	return err
}

func patchNotificationRecipient(ctx context.Context, subject string) error {
	data, err := json.Marshal(map[string]any{"notifications": map[string]any{"recipient": strings.TrimSpace(subject)}})
	if err != nil {
		return err
	}
	_, err = operatorStateService().ApplyValidated(ctx, data, validateOperatorState)
	return err
}
