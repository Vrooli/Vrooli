package coreset

import (
	"reflect"
	"testing"
)

func TestCoreSeedScenariosShape(t *testing.T) {
	seed := CoreSeedScenarios()
	if len(seed) != 9 {
		t.Fatalf("expected 9 seed scenarios, got %d: %v", len(seed), seed)
	}

	want := map[string]bool{
		"agent-manager":                true,
		"data-backup-manager":          true,
		"git-control-tower":            true,
		"prompt-manager":               true,
		"scenario-dependency-analyzer": true,
		"swarm-manager":                true,
		"test-genie":                   true,
		"vrooli-events":                true,
		"workspace-sandbox":            true,
	}
	for _, name := range seed {
		if !want[name] {
			t.Errorf("unexpected seed member %q", name)
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Errorf("missing expected seed members: %v", want)
	}
}

func TestSeedIsSortedAndStable(t *testing.T) {
	first := CoreSeedScenarios()
	second := CoreSeedScenarios()
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("accessor not stable: %v vs %v", first, second)
	}
	for i := 1; i < len(first); i++ {
		if first[i-1] > first[i] {
			t.Fatalf("seed not sorted at %d: %v", i, first)
		}
	}
}

func TestDefaultFallbackEqualsSeed(t *testing.T) {
	if !reflect.DeepEqual(DefaultFallbackCoreSet(), CoreSeedScenarios()) {
		t.Fatalf("DefaultFallbackCoreSet must equal the 9-seed")
	}
}

func TestAccessorsReturnMutationSafeCopies(t *testing.T) {
	seed := CoreSeedScenarios()
	if len(seed) == 0 {
		t.Fatal("seed empty")
	}
	seed[0] = "MUTATED"
	if CoreSeedScenarios()[0] == "MUTATED" {
		t.Fatal("mutating an accessor result leaked into the shared seed")
	}

	tb := TrustedBaseScenarios()
	tb[0] = "MUTATED"
	if TrustedBaseScenarios()[0] == "MUTATED" {
		t.Fatal("mutating an accessor result leaked into the shared trusted base")
	}
}

func TestTrustedBaseIsSubsetOfSeed(t *testing.T) {
	tb := TrustedBaseScenarios()
	if len(tb) == 0 {
		t.Fatal("trusted base must not be empty")
	}
	for _, name := range tb {
		if !IsCoreSeed(name) {
			t.Errorf("trusted-base member %q is not in the core seed", name)
		}
	}
	// The three documented members must be present.
	for _, want := range []string{"git-control-tower", "test-genie", "data-backup-manager"} {
		if !IsTrustedBase(want) {
			t.Errorf("expected %q in trusted base", want)
		}
	}
}

func TestIsCoreSeedAndTrustedBasePredicates(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantSeed    bool
		wantTrusted bool
	}{
		{"canonical seed", "agent-manager", true, false},
		{"trusted member", "test-genie", true, true},
		{"case-insensitive", "Test-Genie", true, true},
		{"whitespace-trimmed", "  git-control-tower  ", true, true},
		{"non-member", "landing-page-business-suite", false, false},
		{"empty", "", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsCoreSeed(tc.input); got != tc.wantSeed {
				t.Errorf("IsCoreSeed(%q) = %v, want %v", tc.input, got, tc.wantSeed)
			}
			if got := IsTrustedBase(tc.input); got != tc.wantTrusted {
				t.Errorf("IsTrustedBase(%q) = %v, want %v", tc.input, got, tc.wantTrusted)
			}
		})
	}
}
