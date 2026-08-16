// Package targetmodel owns the provider-neutral target and transport model
// shared by local probes, bridge inventory, and cross-OS selectors.
package targetmodel

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// TransportKind identifies how a caller reaches a target.
type TransportKind string

const (
	TransportLocal  TransportKind = "local"
	TransportBridge TransportKind = "bridge"
)

// Transport carries reachability metadata. Endpoint is intentionally omitted
// from JSON because credentials and concrete endpoints are not evidence.
type Transport struct {
	Kind      TransportKind `json:"kind"`
	ID        string        `json:"id"`
	Trust     string        `json:"trust,omitempty"`
	Endpoint  string        `json:"-"`
	Available bool          `json:"available"`
	Reason    string        `json:"reason,omitempty"`
}

type TargetHealth struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type BridgeTrust struct {
	Registered         bool   `json:"registered"`
	Online             bool   `json:"online"`
	DispatchAuthorized bool   `json:"dispatch_authorized"`
	Reason             string `json:"reason,omitempty"`
}

// Target is one observable execution destination. It contains identity,
// platform, capability, transport, trust, health, and explicit recovery
// information so an unavailable selection is actionable rather than guessed.
type Target struct {
	ID                string       `json:"id"`
	Ramp              string       `json:"ramp,omitempty"`
	Label             string       `json:"label"`
	Platform          string       `json:"platform"`
	OS                string       `json:"os"`
	Architecture      string       `json:"architecture"`
	DeviceKind        string       `json:"device_kind"`
	Transport         Transport    `json:"transport"`
	NodeID            string       `json:"node_id,omitempty"`
	Mode              string       `json:"mode,omitempty"`
	Capabilities      []string     `json:"capabilities,omitempty"`
	Scopes            []string     `json:"scopes,omitempty"`
	Available         bool         `json:"available"`
	Reason            string       `json:"reason,omitempty"`
	MissingCapability string       `json:"missing_capability,omitempty"`
	NextAction        string       `json:"next_action,omitempty"`
	Health            TargetHealth `json:"health"`
	BridgeTrust       *BridgeTrust `json:"bridge_trust,omitempty"`
	Revoked           bool         `json:"revoked,omitempty"`
}

// Capability vocabulary is provider-neutral. Providers may advertise more
// names; these stable names are the ones the shared spine understands.
const (
	CapabilityCDP             = "cdp"
	CapabilityNativeWindow    = "native-window"
	CapabilityProcessMetrics  = "process-metrics"
	CapabilityOfflineNetwork  = "offline-network"
	CapabilityAndroidSDK      = "android-sdk"
	CapabilityAndroidEmulator = "android-emulator"
	CapabilityAndroidWebView  = "android-webview"
	CapabilityScreenRecording = "screen-recording"
	CapabilityDeviceControl   = "device-control"
)

// Bridge availability reasons are stable cross-consumer vocabulary. Keeping
// them here prevents the gate and delivery ramp from classifying the same
// bridge observation differently merely because they have different callers.
const (
	ReasonBridgeOffline           = "bridge node is offline or not dispatchable"
	ReasonBridgeNoCapability      = "bridge node is online but declares no supported capability"
	ReasonBridgeNoDispatchScope   = "bridge node is online but lacks an authorized scenario-test dispatch scope"
	ReasonBridgeAuthorizedAndroid = "bridge node is online and authorized; Android evidence remains target-owned"
	ReasonBridgeAuthorizedDesktop = "bridge node is online and authorized; desktop evidence remains target-owned"
	ReasonBridgeRevoked           = "bridge node is revoked"
)

func (t Target) Validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return fmt.Errorf("target id is required")
	}
	if strings.TrimSpace(t.Platform) == "" {
		return fmt.Errorf("target %q platform is required", t.ID)
	}
	if t.Transport.Kind != TransportLocal && t.Transport.Kind != TransportBridge {
		return fmt.Errorf("target %q has invalid transport %q", t.ID, t.Transport.Kind)
	}
	if !t.Available {
		if strings.TrimSpace(t.MissingCapability) == "" {
			return fmt.Errorf("target %q is unavailable without a missing capability", t.ID)
		}
		if strings.TrimSpace(t.NextAction) == "" {
			return fmt.Errorf("target %q is unavailable without a next action", t.ID)
		}
	}
	return nil
}

func (t Target) Supports(capability string) bool {
	wanted := strings.TrimSpace(capability)
	if wanted == "" {
		return false
	}
	for _, observed := range t.Capabilities {
		if strings.EqualFold(strings.TrimSpace(observed), wanted) {
			return true
		}
	}
	return false
}

// ScopeAllows applies the bridge command-scope grammar without importing a
// bridge domain package. The grammar is intentionally small: exact grants,
// trailing-prefix grants, and the universal grant. Keeping this primitive in
// the shared target package lets discovery, selectors, and bridge adapters
// classify one node observation the same way.
func ScopeAllows(scopes []string, command string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		switch {
		case scope == "*":
			return true
		case strings.HasSuffix(scope, "*") && strings.HasPrefix(command, strings.TrimSuffix(scope, "*")):
			return true
		case scope == command:
			return true
		}
	}
	return false
}

// Inventory is a point-in-time target snapshot.
type Inventory struct {
	Targets  []Target  `json:"targets"`
	Observed time.Time `json:"observed_at"`
}

func (i Inventory) Validate() error {
	if len(i.Targets) == 0 {
		return fmt.Errorf("inventory contains no targets")
	}
	for index, target := range i.Targets {
		if err := target.Validate(); err != nil {
			return fmt.Errorf("target %d: %w", index, err)
		}
	}
	return nil
}

// SelectionRequest describes the shared target constraints used by both
// bridge gates and delivery validation matrices.
type SelectionRequest struct {
	OS                   string          `json:"os"`
	RequiredCapabilities []string        `json:"required_capabilities,omitempty"`
	TransportKinds       []TransportKind `json:"transport_kinds,omitempty"`
}

// Selection is an explicit result, including the reason and next action when
// the requested target cannot currently be used.
type Selection struct {
	Target     Target `json:"target"`
	Found      bool   `json:"found"`
	Available  bool   `json:"available"`
	Reason     string `json:"reason"`
	NextAction string `json:"next_action"`
}

// Select deterministically chooses the lowest-ID target matching the request.
// It considers every matching target before declaring the request unavailable,
// so a degraded target does not mask another healthy target on the same OS.
func Select(inventory Inventory, request SelectionRequest) Selection {
	os := strings.ToLower(strings.TrimSpace(request.OS))
	if os == "" {
		return Selection{Reason: "target OS is required", NextAction: "provide a target OS and select again"}
	}

	candidates := make([]Target, 0, len(inventory.Targets))
	for _, target := range inventory.Targets {
		if strings.ToLower(strings.TrimSpace(target.OS)) != os || !transportAllowed(target.Transport.Kind, request.TransportKinds) {
			continue
		}
		candidates = append(candidates, target)
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	if len(candidates) == 0 {
		return Selection{Reason: fmt.Sprintf("no target matches OS %q", os), NextAction: "register or probe a target for this OS"}
	}

	var unavailable *Selection
	for _, target := range candidates {
		if !target.Available {
			if unavailable == nil {
				candidate := Selection{Target: target, Found: true, Reason: target.Reason, NextAction: target.NextAction}
				if candidate.Reason == "" {
					candidate.Reason = fmt.Sprintf("target %q is unavailable", target.ID)
				}
				if candidate.NextAction == "" {
					candidate.NextAction = "restore target availability and select again"
				}
				unavailable = &candidate
			}
			continue
		}
		missing := missingCapabilities(target, request.RequiredCapabilities)
		if len(missing) > 0 {
			if unavailable == nil {
				unavailable = &Selection{
					Target:     target,
					Found:      true,
					Reason:     fmt.Sprintf("target %q is missing capabilities: %s", target.ID, strings.Join(missing, ", ")),
					NextAction: "provide the missing capabilities and select again",
				}
			}
			continue
		}
		return Selection{Target: target, Found: true, Available: true, Reason: target.Reason, NextAction: target.NextAction}
	}
	return *unavailable
}

// SelectByOS applies the same deterministic selector to each requested OS.
func SelectByOS(inventory Inventory, oses []string, request SelectionRequest) map[string]Selection {
	result := make(map[string]Selection, len(oses))
	for _, raw := range oses {
		os := strings.ToLower(strings.TrimSpace(raw))
		if os == "" {
			continue
		}
		request.OS = os
		if _, exists := result[os]; !exists {
			result[os] = Select(inventory, request)
		}
	}
	return result
}

func transportAllowed(kind TransportKind, allowed []TransportKind) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, candidate := range allowed {
		if kind == candidate {
			return true
		}
	}
	return false
}

func missingCapabilities(target Target, required []string) []string {
	missing := make([]string, 0, len(required))
	for _, capability := range required {
		if !target.Supports(capability) {
			missing = append(missing, strings.TrimSpace(capability))
		}
	}
	return missing
}

func UnavailableTarget(reason, missingCapability string) Target {
	return Target{
		ID: "bridge:unavailable", Label: "Bridge fleet", Platform: "desktop", DeviceKind: "desktop",
		Mode: "remote", Transport: Transport{Kind: TransportBridge, ID: "bridge", Available: false, Reason: reason},
		Available: false, Reason: reason, MissingCapability: missingCapability,
		NextAction:  "restore bridge inventory and probe again",
		Health:      TargetHealth{Status: "unavailable", Reason: reason},
		BridgeTrust: &BridgeTrust{Reason: "bridge node identity was not verified"},
	}
}
