package deliveryramp

import (
	"errors"
	"strings"
	"testing"
)

func TestRampRegistryProvidesStableCapabilityViewAndTypedRefusal(t *testing.T) {
	registry, err := NewRampRegistry([]RampDefinition{
		{
			ID: "mobile", Description: "mobile delivery", Targets: []string{"android"}, Formats: []DeliveryFormat{"apk"},
			Capabilities: []CapabilityDefinition{{ID: "device-install", Description: "install on a device"}},
			Operations: []OperationCapability{
				{Operation: OperationProbe, Supported: true},
				{Operation: OperationDistribute, Reason: "store credentials are unavailable", NextAction: "configure the mobile store owner"},
			},
		},
		{ID: "desktop", Description: "desktop delivery", Targets: []string{"linux"}, Formats: []DeliveryFormat{"appimage"}, Operations: []OperationCapability{{Operation: OperationDistribute, Supported: true}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	view := registry.List()
	if len(view) != 2 || view[0].ID != "desktop" || view[1].ID != "mobile" {
		t.Fatalf("unstable registry view: %+v", view)
	}
	view[0].Targets[0] = "mutated"
	resolved, ok := registry.Resolve("desktop")
	if !ok || resolved.Targets[0] != "linux" {
		t.Fatalf("registry returned mutable state: %+v", resolved)
	}
	if err := registry.RequireOperation("desktop", OperationDistribute); err != nil {
		t.Fatalf("supported operation refused: %v", err)
	}
	err = registry.RequireOperation("mobile", OperationDistribute)
	var unsupported *UnsupportedOperationError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "configure the mobile store owner") {
		t.Fatalf("typed refusal = %v", err)
	}
}

func TestRampDefinitionRejectsUnsupportedOperationWithoutRepairPath(t *testing.T) {
	_, err := NewRampRegistry([]RampDefinition{{ID: "desktop", Description: "desktop", Targets: []string{"linux"}, Operations: []OperationCapability{{Operation: OperationRecover}}}})
	if err == nil || !strings.Contains(err.Error(), "reason and next_action") {
		t.Fatalf("missing refusal path error = %v", err)
	}
}
