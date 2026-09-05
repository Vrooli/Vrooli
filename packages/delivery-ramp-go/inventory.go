package deliveryramp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/targetmodel"
)

type (
	Target           = targetmodel.Target
	TargetHealth     = targetmodel.TargetHealth
	BridgeTrust      = targetmodel.BridgeTrust
	Inventory        = targetmodel.Inventory
	SelectionRequest = targetmodel.SelectionRequest
	Selection        = targetmodel.Selection
)

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

	ReasonBridgeOffline           = "bridge node is offline or not dispatchable"
	ReasonBridgeNoCapability      = "bridge node is online but declares no supported capability"
	ReasonBridgeNoDispatchScope   = "bridge node is online but lacks an authorized scenario-test dispatch scope"
	ReasonBridgeAuthorizedAndroid = "bridge node is online and authorized; Android evidence remains target-owned"
	ReasonBridgeAuthorizedDesktop = "bridge node is online and authorized; desktop evidence remains target-owned"
	ReasonBridgeAuthorizedIOS     = "bridge node is online and authorized; iOS evidence remains target-owned"
	ReasonBridgeNoHostProbe       = "bridge node is online but its host toolchain could not be probed"
	ReasonBridgeRevoked           = "bridge node is revoked"
)

// SelectTarget is the ramp-facing entry point to the shared target selector.
// Keeping this adapter thin makes the ramp's selection contract explicit while
// ensuring it cannot drift from bridge gate selection semantics.
func SelectTarget(inventory Inventory, request SelectionRequest) Selection {
	return targetmodel.Select(inventory, request)
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
			// Carry the cause. An unavailable target must name what is missing
			// and what to do next; collapsing every failure into one generic
			// string hides an unreachable endpoint, a rejected credential, and a
			// malformed response behind identical text.
			targets = []Target{UnavailableTarget(
				fmt.Sprintf("bridge inventory unavailable: %v", sourceErr),
				"bridge inventory",
			)}
		}
		result.Targets = append(result.Targets, targets...)
	}
	if err := result.Validate(); err != nil {
		return Inventory{}, err
	}
	return result, nil
}

func UnavailableTarget(reason, missingCapability string) Target {
	return targetmodel.UnavailableTarget(reason, missingCapability)
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
