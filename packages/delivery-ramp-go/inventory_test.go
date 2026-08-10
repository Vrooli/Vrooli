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
