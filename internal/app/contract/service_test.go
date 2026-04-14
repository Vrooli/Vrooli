package contractapp

import (
	"testing"
)

func TestServiceValidateUsesResolvedRoot(t *testing.T) {
	var resolved string
	svc := Service{
		ResolveRootFn: func() (string, error) { return "/repo", nil },
		ValidateFn: func(root string) (ValidationOutput, error) {
			resolved = root
			return ValidationOutput{Success: true, Root: root}, nil
		},
	}

	output, err := svc.Validate()
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if resolved != "/repo" || output.Root != "/repo" {
		t.Fatalf("output = %#v resolved = %q", output, resolved)
	}
}

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
