package deployment

import (
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestBuildTargetDAGPackage(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	response, err := BuildTargetDAG(root, "package:api-core", true, true)
	if err != nil {
		t.Fatalf("BuildTargetDAG(package): %v", err)
	}
	if response.TargetKind != "package" {
		t.Fatalf("target kind = %q, want package", response.TargetKind)
	}
	if len(response.DAG) == 0 {
		t.Fatal("package DAG has no dependency edges")
	}
}

func TestBuildTargetDAGResource(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	response, err := BuildTargetDAG(root, "resource:ollama", true, false)
	if err != nil {
		t.Fatalf("BuildTargetDAG(resource): %v", err)
	}
	if response.TargetKind != "resource" {
		t.Fatalf("target kind = %q, want resource", response.TargetKind)
	}
}

func TestBuildTargetDAGBareScenario(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	response, err := BuildTargetDAG(root, "unit-health", false, false)
	if err != nil {
		t.Fatalf("BuildTargetDAG(scenario): %v", err)
	}
	if response.TargetKind != "scenario" || response.TargetID != "unit-health" {
		t.Fatalf("unexpected scenario target: %+v", response)
	}
}
