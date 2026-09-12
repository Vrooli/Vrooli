package contractapp

import (
	"testing"
)

func TestServiceResolveScenarioUsesResolvedRoot(t *testing.T) {
	svc := Service{
		ResolveRootFn: func() (string, error) { return "/repo", nil },
		ResolveScenarioFn: func(root, name, key string) (ResolveScenarioOutput, error) {
			return ResolveScenarioOutput{Success: true, Root: root, Scenario: name, File: key, Path: "/repo/scenarios/demo"}, nil
		},
	}

	output, err := svc.ResolveScenario(ResolveScenarioRequest{ScenarioName: "demo", FileKey: "service"})
	if err != nil {
		t.Fatalf("ResolveScenario: %v", err)
	}
	if output.Root != "/repo" || output.Scenario != "demo" || output.File != "service" {
		t.Fatalf("output = %#v", output)
	}
}
