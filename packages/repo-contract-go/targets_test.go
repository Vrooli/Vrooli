package repocontract

import (
	"testing"
)

func TestEnumerateCanonicalTargets(t *testing.T) {
	contract := mustLoadDefault(t, "/home/matthalloran8/Vrooli")
	targets, err := contract.EnumerateTargets("/home/matthalloran8/Vrooli")
	if err != nil {
		t.Fatalf("EnumerateTargets: %v", err)
	}
	counts := map[TargetKind]int{}
	for _, target := range targets {
		counts[target.Kind]++
	}
	t.Logf("target counts: %#v total=%d", counts, len(targets))
	if len(targets) == 0 {
		t.Fatal("expected canonical repository targets")
	}
	if len(targets) != 240 {
		t.Fatalf("canonical target count = %d, want 240", len(targets))
	}
	for kind, want := range map[TargetKind]int{
		TargetKindScenario: 115, TargetKindTool: 48, TargetKindResource: 25,
		TargetKindPackage: 23, TargetKindSafeguard: 20, TargetKindTeam: 6,
		TargetKindControlPlane: 2, TargetKindDocs: 1,
	} {
		if counts[kind] != want {
			t.Fatalf("target count for %s = %d, want %d", kind, counts[kind], want)
		}
	}
}
