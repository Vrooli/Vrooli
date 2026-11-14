package capacity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

const fitGiB = int64(1024 * 1024 * 1024)

func fitResource(name string, enabled bool, required bool, steps ...DegradeStep) DeclaredResource {
	requirement := "preferred"
	backends := []string{"cuda", "cpu"}
	if required {
		requirement = "required"
		backends = []string{"cuda"}
	}
	return DeclaredResource{
		Name: name, Enabled: enabled, Backends: backends, Require: requirement,
		Claim: &ResourceClaimSpec{ResourceKind: ResourceKindVRAM, PreferredBytes: steps[0].AmountBytes, FloorBytes: steps[len(steps)-1].AmountBytes, Priority: "service", Profile: &DegradeProfile{Steps: steps}},
	}
}

func TestFitCountsReserveAndProposesLowerRung(t *testing.T) {
	resources := []DeclaredResource{fitResource("model", true, false,
		DegradeStep{Label: "large", AmountBytes: 6 * fitGiB},
		DegradeStep{Label: "cpu", AmountBytes: 0},
	)}
	got := Fit(FitMachine{VRAMBytes: 8 * fitGiB, UsableBytes: 8 * fitGiB, Backends: []string{"cuda", "cpu"}}, FitState{Posture: "balanced", ReserveBytes: 4 * fitGiB}, resources, nil)
	if got.Verdict != FitVerdictOverSubscribed || got.StaticBytes != 6*fitGiB || got.ReserveBytes != 4*fitGiB {
		t.Fatalf("Fit() = %#v", got)
	}
	if len(got.Proposal) != 1 || got.Proposal[0].To != "cpu" || got.ProposedTotalBytes != 4*fitGiB {
		t.Fatalf("proposal = %#v, total=%d", got.Proposal, got.ProposedTotalBytes)
	}
}

func TestFitUsesOperatorGPUIndexForMeasuredFootprint(t *testing.T) {
	resources := []DeclaredResource{fitResource("model", true, false,
		DegradeStep{Label: "gpu", AmountBytes: fitGiB},
		DegradeStep{Label: "cpu", AmountBytes: 0},
	)}
	gpuOne := 1
	state := FitState{Posture: "balanced", Resources: map[string]FitResourceChoice{
		"model": {GPUIndex: &gpuOne},
	}}
	footprints := []Footprint{
		{Resource: "model", Rung: "gpu", GPUIndex: 0, PeakBytes: 2 * fitGiB, Source: FootprintSourceMeasured},
		{Resource: "model", Rung: "gpu", GPUIndex: 1, PeakBytes: 3 * fitGiB, Source: FootprintSourceMeasured},
	}

	got := Fit(FitMachine{VRAMBytes: 8 * fitGiB, UsableBytes: 8 * fitGiB, Backends: []string{"cuda", "cpu"}}, state, resources, footprints)
	if got.StaticBytes != 3*fitGiB {
		t.Fatalf("static bytes = %d, want GPU 1 measured footprint %d", got.StaticBytes, 3*fitGiB)
	}
}

func TestFitPrefersMeasuredFootprint(t *testing.T) {
	resources := []DeclaredResource{fitResource("model", true, false,
		DegradeStep{Label: "gpu", AmountBytes: 2 * fitGiB},
		DegradeStep{Label: "cpu", AmountBytes: 0},
	)}
	rows := []Footprint{{Resource: "model", Rung: "gpu", GPUIndex: 0, PeakBytes: 3 * fitGiB, Source: FootprintSourceMeasured}}
	got := Fit(FitMachine{VRAMBytes: 2 * fitGiB, UsableBytes: 2 * fitGiB, Backends: []string{"cuda", "cpu"}}, FitState{}, resources, rows)
	if got.StaticBytes != 3*fitGiB || len(got.Offenders) != 1 || got.Offenders[0].FootprintSource != FootprintSourceMeasured {
		t.Fatalf("Fit() did not use measured footprint: %#v", got)
	}
}

func TestFitSeparatesRequiredResourceOnZeroVRAM(t *testing.T) {
	resources := []DeclaredResource{fitResource("reranker", true, true, DegradeStep{Label: "model", AmountBytes: fitGiB})}
	got := Fit(FitMachine{Backends: []string{"cuda", "cpu"}, Compute: "12.0"}, FitState{}, resources, nil)
	if got.Verdict != FitVerdictOverSubscribed || len(got.Unrunnable) != 1 || got.Unrunnable[0].Resource != "reranker" || len(got.Offenders) != 0 {
		t.Fatalf("Fit() = %#v", got)
	}
}

func TestRepositoryDeclaredSetDoesNotFitSimulatedHost(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	resources, err := LoadDeclaredResources(root)
	if err != nil {
		t.Fatal(err)
	}
	got := Fit(FitMachine{VRAMBytes: 16709025792, UsableBytes: 16709025792, Backends: []string{"cuda", "cpu"}, Compute: "12.0"}, FitState{Posture: "balanced", ReserveBytes: 4 * fitGiB}, resources, nil)
	if got.Verdict != FitVerdictOverSubscribed {
		t.Fatalf("repository fit verdict = %s, static=%d reserve=%d usable=%d", got.Verdict, got.StaticBytes, got.ReserveBytes, got.AvailableBytes)
	}
}

func TestCapacityPackageHasOneResourceManifestWalker(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, file := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "ReadDir" {
				return true
			}
			ident, ok := selector.X.(*ast.Ident)
			if ok && ident.Name == "os" {
				count++
			}
			return true
		})
	}
	if count != 1 {
		t.Fatalf("capacity resource manifest walker count = %d, want 1", count)
	}
}
