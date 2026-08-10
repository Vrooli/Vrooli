package deliveryramp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Capability names are deliberately provider-neutral. A ramp may expose more
// capabilities than this initial set; the spine only assigns meaning to names
// it requires for a journey or validation cell.
const (
	CapabilityCDP            = "cdp"
	CapabilityNativeWindow   = "native-window"
	CapabilityProcessMetrics = "process-metrics"
	CapabilityOfflineNetwork = "offline-network"
)

// Target is a capability observation for one validation destination. It is
// intentionally free of generated transport types so a ramp can implement a
// Prober without importing another scenario's API.
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
	Available         bool         `json:"available"`
	Reason            string       `json:"reason,omitempty"`
	MissingCapability string       `json:"missing_capability,omitempty"`
	NextAction        string       `json:"next_action,omitempty"`
	Health            TargetHealth `json:"health"`
	BridgeTrust       *BridgeTrust `json:"bridge_trust,omitempty"`
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

// Validate enforces the fail-closed inventory contract. An unavailable target
// must tell the operator what is missing and what can recover it.
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

// Inventory is a point-in-time capability snapshot. The timestamp is
// metadata, not evidence of execution on the target.
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

// ProbeRequest scopes an inventory probe to the requirements of one cell.
type ProbeRequest struct {
	ProfileID          string          `json:"profile_id"`
	RequiredCapability []string        `json:"required_capability,omitempty"`
	TransportKinds     []TransportKind `json:"transport_kinds,omitempty"`
}

// Prober is the only seam through which platform capability enters the
// delivery spine. Platform-specific implementations belong to ramps.
type Prober interface {
	Probe(context.Context, ProbeRequest) (Inventory, error)
}

// BridgeSource supplies trusted-node inventory. Reachability and node
// authorization remain bridge responsibilities; this seam only projects the
// resulting target observations into the spine contract.
type BridgeSource interface {
	Discover(context.Context) ([]Target, error)
}

func Discover(ctx context.Context, prober Prober, bridgeSources ...BridgeSource) (Inventory, error) {
	if prober == nil {
		return Inventory{}, fmt.Errorf("target inventory prober is not configured")
	}
	result, err := prober.Probe(ctx, ProbeRequest{})
	if err != nil {
		return Inventory{}, fmt.Errorf("probe local target: %w", err)
	}
	for _, source := range bridgeSources {
		if source == nil {
			continue
		}
		targets, sourceErr := source.Discover(ctx)
		if sourceErr != nil {
			targets = []Target{UnavailableTarget("bridge inventory unavailable", "bridge inventory")}
		}
		result.Targets = append(result.Targets, targets...)
	}
	if err := result.Validate(); err != nil {
		return Inventory{}, err
	}
	return result, nil
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

// TargetInventoryHandler owns serialization of the stable target inventory
// endpoint. The scenario supplies only the Prober and optional BridgeSource;
// route registration stays at the scenario edge.
type TargetInventoryHandler struct {
	Prober        Prober
	BridgeSources []BridgeSource
}

func NewTargetInventoryHandler(prober Prober, bridgeSources ...BridgeSource) *TargetInventoryHandler {
	return &TargetInventoryHandler{Prober: prober, BridgeSources: bridgeSources}
}

func (h *TargetInventoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		http.Error(w, "target inventory is unavailable", http.StatusServiceUnavailable)
		return
	}
	result, err := Discover(r.Context(), h.Prober, h.BridgeSources...)
	if err != nil {
		http.Error(w, "target inventory is unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(inventoryResponse(result))
}

type inventoryHTTPResponse struct {
	Targets []inventoryTarget `json:"targets"`
}

type inventoryTarget struct {
	Descriptor   inventoryDescriptor `json:"descriptor"`
	Kind         TransportKind       `json:"kind"`
	NodeID       string              `json:"node_id,omitempty"`
	OS           string              `json:"os"`
	Architecture string              `json:"architecture"`
	Mode         string              `json:"mode"`
	Reason       string              `json:"reason,omitempty"`
	Health       TargetHealth        `json:"health"`
	BridgeTrust  *BridgeTrust        `json:"bridge_trust,omitempty"`
}

type inventoryDescriptor struct {
	TargetID     string `json:"target_id"`
	DisplayName  string `json:"display_name"`
	Capabilities []int  `json:"capabilities,omitempty"`
	Available    bool   `json:"available"`
	Reason       string `json:"reason,omitempty"`
}

func inventoryResponse(inventory Inventory) inventoryHTTPResponse {
	response := inventoryHTTPResponse{Targets: make([]inventoryTarget, 0, len(inventory.Targets))}
	for _, target := range inventory.Targets {
		response.Targets = append(response.Targets, inventoryTarget{
			Descriptor: inventoryDescriptor{
				TargetID: target.ID, DisplayName: target.Label, Capabilities: capabilityNumbers(target.Capabilities),
				Available: target.Available, Reason: target.Reason,
			},
			Kind: target.Transport.Kind, NodeID: target.NodeID, OS: target.OS, Architecture: target.Architecture,
			Mode: target.Mode, Reason: target.Reason, Health: target.Health, BridgeTrust: target.BridgeTrust,
		})
	}
	return response
}

func capabilityNumbers(capabilities []string) []int {
	result := make([]int, 0, len(capabilities))
	for _, capability := range capabilities {
		switch strings.ToLower(strings.TrimSpace(capability)) {
		case CapabilityCDP:
			result = append(result, 1)
		case CapabilityNativeWindow:
			result = append(result, 2)
		case CapabilityProcessMetrics:
			result = append(result, 6)
		case CapabilityOfflineNetwork:
			result = append(result, 8)
		}
	}
	return result
}
