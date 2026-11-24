package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/nodereach"
	"github.com/vrooli/api-core/operatorsession"
	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/api-core/targetmodel"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/bridgepermissions"
	hostdomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/host"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/safeguardpermissions"
	"google.golang.org/protobuf/proto"
)

const onboardingTargetProbeProcedure = "/vrooli.vrooli_onboarding.v1.session.SessionService/GetSession"

func hostService(server *Server) hostdomain.Service {
	return hostdomain.Service{
		ListRequirementsFn: listHostRequirements,
		FactsFn:            getHostFacts,
		TargetsFn:          func(ctx context.Context) ([]targetmodel.Target, string, error) { return listHostTargets(ctx, server) },
		PatchSafeguardConfigFn: func(ctx context.Context, target, name, key string, value any) error {
			if target != "local" {
				return errors.New("host safeguards are available only on the local host control plane")
			}
			return patchHostSafeguardConfig(ctx, name, key, value)
		},
		SetNotificationRecipient: func(ctx context.Context, target string, subject string) error {
			if target != "local" {
				return errors.New("notification recipient can only be changed on the local operator-state owner")
			}
			return patchNotificationRecipient(ctx, subject)
		},
	}
}

func hostSafeguardPermissionInventory(ctx context.Context) ([]safeguardpermissions.Safeguard, error) {
	_, safeguards, err := listHostRequirements(ctx, "local")
	if err != nil {
		return nil, err
	}
	result := make([]safeguardpermissions.Safeguard, 0, len(safeguards))
	for _, item := range safeguards {
		schema, err := json.Marshal(item.ConfigSchema)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(schema)
		secretFields := hostSafeguardSecretFields(item.ConfigSchema)
		configFields := hostSafeguardConfigFields(item.ConfigSchema)
		result = append(result, safeguardpermissions.Safeguard{Name: item.Name, SchemaDigest: hex.EncodeToString(digest[:]), ConfigSchema: item.ConfigSchema, ConfigFields: configFields, SecretFields: secretFields})
	}
	return result, nil
}

func hostSafeguardConfigFields(schema map[string]any) []string {
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil
	}
	fields := make([]string, 0, len(properties))
	for name := range properties {
		fields = append(fields, name)
	}
	sort.Strings(fields)
	return fields
}

func hostSafeguardSecretFields(schema map[string]any) []string {
	secretFields := make([]string, 0)
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return secretFields
	}
	for name, raw := range properties {
		if schemaContainsSecret(name, raw) {
			secretFields = append(secretFields, name)
		}
	}
	sort.Strings(secretFields)
	return secretFields
}

func schemaContainsSecret(name string, schema any) bool {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "secret") || strings.Contains(lower, "token") || strings.Contains(lower, "password") || strings.Contains(lower, "credential") {
		return true
	}
	switch node := schema.(type) {
	case map[string]any:
		if node["secret"] == true || node["sensitive"] == true || strings.EqualFold(fmt.Sprint(node["format"]), "password") {
			return true
		}
		if properties, ok := node["properties"].(map[string]any); ok {
			for childName, child := range properties {
				if schemaContainsSecret(childName, child) {
					return true
				}
			}
		}
		if items, ok := node["items"]; ok && schemaContainsSecret("", items) {
			return true
		}
		if additional, ok := node["additionalProperties"]; ok {
			if child, isSchema := additional.(map[string]any); isSchema && schemaContainsSecret("", child) {
				return true
			}
		}
	case []any:
		for _, child := range node {
			if schemaContainsSecret("", child) {
				return true
			}
		}
	}
	return false
}

func redactHostConfig(config, schema map[string]any) map[string]any {
	result := make(map[string]any, len(config))
	for key, value := range config {
		result[key] = value
	}
	for _, key := range hostSafeguardSecretFields(schema) {
		delete(result, key)
	}
	return result
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
				Platforms: item.Platforms, Commands: item.Commands, ConfigSchema: item.ConfigSchema, Config: redactHostConfig(item.Config, item.ConfigSchema),
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
		server.bridge = nodereach.New(bridgeClientConfig())
	}
	nodes, err := server.bridge.List(ctx, 5*time.Second)
	if err != nil {
		return targets, "bridge unavailable", nil
	}
	requiredScopes, scopeCatalogErr := requiredOnboardingTargetScopes()
	scopeRegistry, scopeDescriptors, scopeInventoryErr := bridgeScopePermissionInventory(ctx)
	for _, node := range nodes {
		if node == nil || node.GetId() == "" {
			continue
		}
		var scopes []string
		scopeErr := scopeInventoryErr
		if scopeErr == nil {
			scopes, scopeErr = readBridgeNodeScopes(ctx, scopeRegistry, scopeDescriptors, node)
		}
		target := nodeToTarget(node, scopes, scopeErr, requiredScopes, scopeCatalogErr)
		if target.Available {
			if err := probeOnboardingTarget(ctx, server.bridge, target.ID); err != nil {
				target.Available = false
				target.Reason = "vrooli-onboarding is not reachable on this target"
				target.NextAction = "refresh or redeploy vrooli-onboarding on the target, then refresh"
				target.Transport.Available = false
				target.Transport.Reason = target.Reason
				target.Health.Reason = target.Reason
			}
		}
		targets = append(targets, target)
	}
	return targets, "", nil
}

type onboardingScenarioReacher interface {
	CallScenario(context.Context, nodereach.ScenarioRequest) ([]byte, error)
}

func probeOnboardingTarget(ctx context.Context, reacher onboardingScenarioReacher, nodeID string) error {
	body, err := proto.Marshal(&sessionv1.GetSessionRequest{Target: "local"})
	if err != nil {
		return err
	}
	probeCtx, cancel := context.WithTimeout(ctx, 1200*time.Millisecond)
	defer cancel()
	_, err = reacher.CallScenario(probeCtx, nodereach.ScenarioRequest{
		NodeID: nodeID, Scenario: "vrooli-onboarding", Procedure: onboardingTargetProbeProcedure,
		Body: body, Timeout: 1200 * time.Millisecond, MaxResponse: 64 << 10,
	})
	return err
}

// bridgeClientConfig gives onboarding the same owner-session fallback used by
// other local apps such as System Monitor. Without the provider, an enrolled
// desktop has no Authorization header for the Bridge registry call and the
// target picker silently degrades to the local option.
func bridgeClientConfig() nodereach.Config {
	return nodereach.Config{
		Token:         firstNonEmpty(os.Getenv("VROOLI_BRIDGE_API_TOKEN"), os.Getenv("VROOLI_API_TOKEN")),
		TokenProvider: operatorsession.LocalOwnerTokenProvider(),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func nodeToTarget(node *registryv1.Node, scopes []string, scopeErr error, requiredScopes []string, requirementErr error) targetmodel.Target {
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
	nextAction := ""
	if !available {
		nextAction = "bring the target online and refresh"
	} else if requirementErr != nil {
		available = false
		reason = "onboarding permission scope catalog is unavailable"
		nextAction = "repair the onboarding scope catalog, then refresh"
	} else if scopeErr != nil {
		available = false
		reason = "Bridge node-scope permission is unavailable"
		nextAction = "refresh the target permission inventory, then retry"
	} else if missing := missingRequiredScopes(scopes, requiredScopes); len(missing) > 0 {
		available = false
		reason = "target is connected but does not grant " + strings.Join(missing, " or ")
		nextAction = "grant the required onboarding scope on the target, then refresh"
	}
	return targetmodel.Target{ID: node.GetId(), Label: node.GetName(), Platform: node.GetOs(), OS: node.GetOs(), Architecture: node.GetArch(), DeviceKind: node.GetKind().String(), Available: available, Reason: reason, NextAction: nextAction, Capabilities: node.GetCapabilities(), Scopes: append([]string(nil), scopes...), Transport: targetmodel.Transport{Kind: targetmodel.TransportBridge, ID: node.GetId(), Available: node.GetOnline(), Reason: reason}, Health: targetmodel.TargetHealth{Status: status, Reason: reason}, Readiness: checks}
}

func requiredOnboardingTargetScopes() ([]string, error) {
	root, err := manifestRoot()
	if err != nil {
		return nil, err
	}
	catalog, err := scopecatalog.Build(root)
	if err != nil {
		return nil, fmt.Errorf("build onboarding authorization catalog: %w", err)
	}
	return requiredOnboardingScopesFromCatalog(catalog)
}

func requiredOnboardingScopesFromCatalog(catalog scopecatalog.Catalog) ([]string, error) {
	var profile *scopecatalog.AuthenticationProfile
	for i := range catalog.AuthenticationProfiles {
		if catalog.AuthenticationProfiles[i].Scenario == "vrooli-onboarding" {
			profile = &catalog.AuthenticationProfiles[i]
			break
		}
	}
	if profile == nil || !profile.Declared || len(profile.MissingDeclarations) > 0 {
		return nil, errors.New("onboarding authentication profile is missing or invalid")
	}
	byEffect := map[scopecatalog.Effect]string{}
	for _, capability := range profile.Capabilities {
		if capability.Effect != scopecatalog.EffectRead && capability.Effect != scopecatalog.EffectWrite {
			continue
		}
		if _, exists := byEffect[capability.Effect]; exists || strings.TrimSpace(capability.ID) == "" {
			return nil, errors.New("onboarding read/write capabilities are ambiguous")
		}
		byEffect[capability.Effect] = capability.ID
	}
	read, hasRead := byEffect[scopecatalog.EffectRead]
	write, hasWrite := byEffect[scopecatalog.EffectWrite]
	if !hasRead || !hasWrite {
		return nil, errors.New("onboarding read and write capabilities are required")
	}
	return []string{read, write}, nil
}

func missingRequiredScopes(held, required []string) []string {
	missing := make([]string, 0, len(required))
	for _, scope := range required {
		if !scopecatalog.Resolve(held, scope) {
			missing = append(missing, scope)
		}
	}
	return missing
}

func bridgeScopePermissionInventory(ctx context.Context) (*permissions.Registry, []permissions.Descriptor, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsHuman() || !principal.IsVerified() {
		return nil, nil, errors.New("verified human operator required to read target scopes")
	}
	service := onboardingOperatorStateService()
	registry, err := service.PermissionRegistry()
	if err != nil {
		return nil, nil, err
	}
	descriptors, err := registry.InventoryForProvider(ctx, bridgepermissions.ProviderID)
	if err != nil {
		return nil, nil, err
	}
	return registry, descriptors, nil
}

func readBridgeNodeScopes(ctx context.Context, registry *permissions.Registry, descriptors []permissions.Descriptor, node *registryv1.Node) ([]string, error) {
	if registry == nil || node == nil || node.GetId() == "" || node.GetEnrollmentGeneration() <= 0 {
		return nil, errors.New("Bridge node-scope binding is unavailable")
	}
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsHuman() || !principal.IsVerified() || principal.Subject == "" {
		return nil, errors.New("verified human operator required to read target scopes")
	}
	var descriptor *permissions.Descriptor
	for i := range descriptors {
		candidate := &descriptors[i]
		if candidate.SubjectKind != "node-scopes" || candidate.Target.NodeID != node.GetId() {
			continue
		}
		if candidate.Target.EnrollmentGeneration != uint64(node.GetEnrollmentGeneration()) {
			return nil, errors.New("Bridge node-scope permission belongs to a different enrollment generation")
		}
		if descriptor != nil {
			return nil, errors.New("Bridge node has ambiguous permission descriptors")
		}
		descriptor = candidate
	}
	if descriptor == nil || descriptor.Owner != bridgepermissions.Owner || !descriptor.TargetBound || !descriptor.CanRead {
		return nil, errors.New("Bridge node-scope permission is not registered")
	}
	result, err := registry.Read(ctx, descriptor.ID, permissions.ReadRequest{ActorID: principal.Subject, Owner: bridgepermissions.Owner, Target: descriptor.Target})
	if err != nil {
		return nil, err
	}
	if result.PermissionID != descriptor.ID || result.Owner != bridgepermissions.Owner || result.Target != descriptor.Target || result.Revision == "" {
		return nil, errors.New("Bridge node-scope read returned a mismatched source binding")
	}
	var scopes struct {
		Scopes []string `json:"scopes"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(result.Value)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&scopes); err != nil {
		return nil, fmt.Errorf("decode Bridge node scopes: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return nil, fmt.Errorf("decode Bridge node scopes trailing data: %w", err)
		}
		return nil, errors.New("Bridge node-scope value contains trailing JSON")
	}
	seen := make(map[string]struct{}, len(scopes.Scopes))
	for i, scope := range scopes.Scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			return nil, errors.New("Bridge node-scope value contains an empty entry")
		}
		if _, exists := seen[scope]; exists {
			return nil, errors.New("Bridge node-scope value contains a duplicate entry")
		}
		seen[scope] = struct{}{}
		scopes.Scopes[i] = scope
	}
	return scopes.Scopes, nil
}

func patchHostSafeguardConfig(ctx context.Context, name, key string, value any) error {
	return writeHostSafeguardPermission(ctx, "local", name, map[string]any{"config_key": key, "value": value})
}

func writeHostSafeguardPermission(ctx context.Context, target, name string, value any) error {
	if strings.TrimSpace(target) != "local" {
		return errors.New("host safeguards are available only on the local host control plane")
	}
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsHuman() {
		return errors.New("verified human operator required")
	}
	service := onboardingOperatorStateService()
	registry, err := service.PermissionRegistry()
	if err != nil {
		return err
	}
	id := safeguardpermissions.PermissionID(name)
	read, err := registry.Read(ctx, id, permissions.ReadRequest{ActorID: principal.Subject, Owner: safeguardpermissions.Owner})
	if err != nil {
		return err
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	requestIDBytes := make([]byte, 16)
	if _, err := rand.Read(requestIDBytes); err != nil {
		return err
	}
	requestID := hex.EncodeToString(requestIDBytes)
	_, err = registry.Write(ctx, id, permissions.WriteRequest{ActorID: principal.Subject, Owner: safeguardpermissions.Owner, ExpectedRevision: read.Revision, IdempotencyKey: requestID, Value: payload})
	return err
}

func patchNotificationRecipient(ctx context.Context, subject string) error {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return errors.New("notification recipient subject is required")
	}
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsHuman() || !principal.IsVerified() || principal.Subject == "" {
		return errors.New("verified human operator required to set notification recipient")
	}
	registry, err := onboardingOperatorStateService().PermissionRegistry()
	if err != nil {
		return err
	}
	const permissionID = "onboarding/operator-state"
	read, err := registry.Read(ctx, permissionID, permissions.ReadRequest{ActorID: principal.Subject, Owner: "vrooli-onboarding"})
	if err != nil {
		return err
	}
	if read.PermissionID != permissionID || read.Owner != "vrooli-onboarding" || read.Revision == "" || read.Target != (permissions.TargetBinding{}) {
		return errors.New("operator-state permission returned a mismatched or incomplete source binding")
	}
	value, err := json.Marshal(map[string]any{
		"state":       map[string]any{"notifications": map[string]any{"recipient": subject}},
		"update_mask": []string{"notifications.recipient"},
	})
	if err != nil {
		return err
	}
	requestID := make([]byte, 16)
	if _, err := rand.Read(requestID); err != nil {
		return fmt.Errorf("create notification recipient request id: %w", err)
	}
	idempotencyKey := "host-notification-recipient:" + hex.EncodeToString(requestID)
	receipt, err := registry.Write(ctx, permissionID, permissions.WriteRequest{
		ActorID: principal.Subject, Owner: "vrooli-onboarding", ExpectedRevision: read.Revision,
		IdempotencyKey: idempotencyKey, Value: value,
	})
	if err != nil {
		return err
	}
	if receipt.PermissionID != permissionID || receipt.Owner != "vrooli-onboarding" || receipt.ActorID != principal.Subject ||
		receipt.ExpectedRevision != read.Revision || receipt.ResultRevision == "" || receipt.IdempotencyKey != idempotencyKey || receipt.Outcome == "" {
		return errors.New("notification recipient write returned a mismatched or incomplete owner receipt")
	}
	return nil
}
