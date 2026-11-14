package healthlifecycle

import "testing"

func TestCheckStopProcessOwnershipRejectsHostWidePattern(t *testing.T) {
	manifest := []byte(`{"lifecycle":{"stop":{"steps":[{"exec":["pkill","-f","node server.js"]}]}}}`)
	violations := CheckStopProcessOwnership(manifest, "/repo/scenarios/demo/.vrooli/service.json")
	if len(violations) != 1 {
		t.Fatalf("expected one ownership violation, got %d", len(violations))
	}
}

func TestCheckStopProcessOwnershipAllowsScenarioScopedOrPIDStops(t *testing.T) {
	manifest := []byte(`{"lifecycle":{"stop":{"steps":[{"exec":["pkill","-f","node server.js --scenario demo"]},{"exec":["kill","$API_PID"]}]}}}`)
	if violations := CheckStopProcessOwnership(manifest, "/repo/scenarios/demo/.vrooli/service.json"); len(violations) != 0 {
		t.Fatalf("expected scoped/PID stop to pass, got %v", violations)
	}
}
