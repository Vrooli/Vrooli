package deliveryramp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeProber struct {
	inventory Inventory
	err       error
}

func (f fakeProber) Probe(context.Context, ProbeRequest) (Inventory, error) {
	return f.inventory, f.err
}

func TestDiscoverAcceptsAvailableTarget(t *testing.T) {
	inventory, err := Discover(context.Background(), fakeProber{inventory: Inventory{Targets: []Target{{
		ID: "local-linux-amd64", Platform: "desktop", Transport: Transport{Kind: TransportLocal}, Available: true,
	}}}})
	if err != nil || len(inventory.Targets) != 1 {
		t.Fatalf("discover = %#v, err=%v", inventory, err)
	}
}

func TestUnavailableTargetRequiresRecoveryDetails(t *testing.T) {
	target := Target{
		ID: "local-linux-amd64", Platform: "desktop", Transport: Transport{Kind: TransportLocal},
		Available: false, MissingCapability: CapabilityCDP,
	}
	if err := target.Validate(); err == nil {
		t.Fatal("expected unavailable target without next action to be rejected")
	}
	target.NextAction = "start a display runtime and probe again"
	if err := target.Validate(); err != nil {
		t.Fatalf("valid unavailable target rejected: %v", err)
	}
}

func TestInventorySupportsCaseInsensitiveCapabilityLookup(t *testing.T) {
	target := Target{ID: "local", Platform: "desktop", Transport: Transport{Kind: TransportLocal}, Available: true, Capabilities: []string{CapabilityNativeWindow}}
	if !target.Supports("NATIVE-WINDOW") {
		t.Fatal("expected capability lookup to be case insensitive")
	}
	if target.Supports("missing") {
		t.Fatal("unexpected capability match")
	}
}

func TestRampSelectorUsesSharedTargetModel(t *testing.T) {
	inventory := Inventory{Targets: []Target{
		{ID: "node-b", Platform: "desktop", OS: "linux", Available: true, Transport: Transport{Kind: TransportBridge}},
		{ID: "node-a", Platform: "desktop", OS: "linux", Available: true, Transport: Transport{Kind: TransportBridge}},
	}}
	selection := SelectTarget(inventory, SelectionRequest{OS: "linux"})
	if !selection.Found || !selection.Available || selection.Target.ID != "node-a" {
		t.Fatalf("ramp selection = %+v, want node-a from the shared deterministic selector", selection)
	}
}

func TestDiscoverRejectsUnavailableTargetWithoutRecoveryDetails(t *testing.T) {
	_, err := Discover(context.Background(), fakeProber{inventory: Inventory{Targets: []Target{{
		ID: "local-linux-amd64", Platform: "desktop", Transport: Transport{Kind: TransportLocal}, Available: false,
	}}}})
	if err == nil {
		t.Fatal("expected unavailable target without recovery details to be rejected")
	}
}

func TestTargetInventoryHandlerPreservesOracleCapabilityNumbers(t *testing.T) {
	handler := NewTargetInventoryHandler(fakeProber{inventory: Inventory{Targets: []Target{{
		ID: "local-linux-amd64", Label: "Local host", Platform: "desktop", OS: "linux", Architecture: "amd64", Mode: "native",
		Transport: Transport{Kind: TransportLocal}, Available: true,
		Capabilities: []string{CapabilityProcessMetrics, CapabilityCDP, CapabilityNativeWindow},
		Health:       TargetHealth{Status: "healthy"},
	}}}})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/validation/targets", nil))
	if recorder.Code != http.StatusOK || !containsAll(recorder.Body.String(), `"target_id":"local-linux-amd64"`, `"capabilities":[6,1,2]`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestTargetInventoryHandlerReturnsServiceUnavailableOnProbeError(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewTargetInventoryHandler(fakeProber{err: errors.New("probe failed")}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/validation/targets", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func containsAll(value string, wanted ...string) bool {
	for _, fragment := range wanted {
		if !strings.Contains(value, fragment) {
			return false
		}
	}
	return true
}

// failingBridgeSource reports a specific transport failure.
type failingBridgeSource struct{ err error }

func (f failingBridgeSource) Discover(context.Context) ([]Target, error) { return nil, f.err }

// An unavailable target must name what is missing and what to do next.
// Collapsing every bridge failure into one generic string hid an unreachable
// endpoint, a rejected credential, and a malformed response behind identical
// text, which made a live auth rejection undiagnosable from the inventory.
func TestDiscoverCarriesBridgeFailureCause(t *testing.T) {
	inventory, err := Discover(context.Background(), stubProber{},
		failingBridgeSource{err: errors.New("unauthenticated")})
	if err != nil {
		t.Fatal(err)
	}
	var bridgeTarget *Target
	for i := range inventory.Targets {
		if inventory.Targets[i].MissingCapability == "bridge inventory" {
			bridgeTarget = &inventory.Targets[i]
		}
	}
	if bridgeTarget == nil {
		t.Fatalf("expected an unavailable bridge target, got %#v", inventory.Targets)
	}
	if !strings.Contains(bridgeTarget.Reason, "unauthenticated") {
		t.Fatalf("reason must carry the cause, got %q", bridgeTarget.Reason)
	}
}

type stubProber struct{}

func (stubProber) Probe(context.Context, ProbeRequest) (Inventory, error) {
	return Inventory{Targets: []Target{{
		ID: "local", Platform: "ios", Transport: Transport{Kind: TransportLocal, ID: "local"},
		Available: false, MissingCapability: "apple toolchain", NextAction: "use a macOS node",
		Health: TargetHealth{Status: "unsupported"},
	}}}, nil
}
