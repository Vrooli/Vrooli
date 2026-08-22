package reconcile

import (
	"context"
	"errors"
	"testing"
)

func TestCompareSeparatesGhostFromOutOfScope(t *testing.T) {
	// alpha is installed and expected; delta is installed but outside the
	// core set; ghost is registered but no longer installed.
	diff := Compare(
		[]string{"scenario-alpha", "scenario-delta", "scenario-ghost", "infra-network"},
		[]string{"alpha", "beta", "delta"},
		[]string{"alpha", "beta"},
	)
	if len(diff.GhostChecks) != 1 || diff.GhostChecks[0] != "scenario-ghost" {
		t.Fatalf("ghost checks = %#v", diff.GhostChecks)
	}
	if len(diff.OutOfScopeChecks) != 1 || diff.OutOfScopeChecks[0] != "scenario-delta" {
		t.Fatalf("out-of-scope checks = %#v", diff.OutOfScopeChecks)
	}
	if len(diff.UnsupervisedPlant) != 1 || diff.UnsupervisedPlant[0] != "beta" {
		t.Fatalf("unsupervised plant = %#v", diff.UnsupervisedPlant)
	}
	if !diff.GhostDetectionAvailable {
		t.Fatal("ghost detection should be available when the installed set is known")
	}
}

// An installed scenario outside the core set must never be reported as a
// ghost: ghost readings are dropped from every aggregate, so misclassifying
// one silently removes live plant from uptime accounting.
func TestCompareNeverGhostsAnInstalledScenario(t *testing.T) {
	diff := Compare([]string{"scenario-search-hub"}, []string{"search-hub"}, []string{"prompt-manager"})
	if len(diff.GhostChecks) != 0 {
		t.Fatalf("installed scenario was ghosted: %#v", diff.GhostChecks)
	}
	if len(diff.OutOfScopeChecks) != 1 || diff.OutOfScopeChecks[0] != "scenario-search-hub" {
		t.Fatalf("out-of-scope checks = %#v", diff.OutOfScopeChecks)
	}
}

// A scenario whose name begins with a check-family prefix is still a real
// target. Prefix-matching the bare name would drop it from the installed set
// and report its check as a ghost.
func TestCompareHandlesScenarioNamesThatLookLikeCheckFamilies(t *testing.T) {
	diff := Compare([]string{"scenario-vrooli-events"}, []string{"vrooli-events"}, []string{"vrooli-events"})
	if len(diff.GhostChecks) != 0 || len(diff.OutOfScopeChecks) != 0 || len(diff.UnsupervisedPlant) != 0 {
		t.Fatalf("expected a clean diff, got %#v", diff)
	}
}

// When the installed set cannot be read, absence from it is not evidence of
// absence from the plant, so nothing may be classified.
func TestCompareWithoutInstalledSetClassifiesNothing(t *testing.T) {
	diff := Compare([]string{"scenario-alpha", "scenario-ghost"}, nil, []string{"alpha"})
	if len(diff.GhostChecks) != 0 || len(diff.OutOfScopeChecks) != 0 {
		t.Fatalf("expected no classification, got %#v", diff)
	}
	if diff.GhostDetectionAvailable || diff.GhostUnavailableReason == "" {
		t.Fatal("expected ghost detection to be reported unavailable with a reason")
	}
	if len(diff.UnsupervisedPlant) != 0 {
		t.Fatalf("unsupervised plant should still be computable: %#v", diff.UnsupervisedPlant)
	}
}

func TestCoreSetProviderDoesNotAcceptUnavailableSource(t *testing.T) {
	provider := NewCoreSetProviderWithRunner(func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"source":"cached","core_set":["alpha"]}`), nil
	})
	if _, err := provider.Expected(context.Background()); err == nil {
		t.Fatal("expected non-computed source to be unavailable")
	}
	provider = NewCoreSetProviderWithRunner(func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("offline")
	})
	if _, err := provider.Expected(context.Background()); err == nil {
		t.Fatal("expected command error")
	}
}

// Several scenarios are genuinely named `scenario-*`. Their check id is
// therefore `scenario-scenario-authenticator`, and normalizing the name side
// with the same TrimPrefix used on the id side reported each of them as a
// ghost check and as unsupervised plant simultaneously.
func TestCompareHandlesScenariosNamedScenarioSomething(t *testing.T) {
	diff := Compare(
		[]string{"scenario-scenario-authenticator", "scenario-scenario-dependency-analyzer"},
		[]string{"scenario-authenticator", "scenario-dependency-analyzer"},
		[]string{"scenario-authenticator", "scenario-dependency-analyzer"},
	)
	if len(diff.GhostChecks) != 0 {
		t.Fatalf("installed scenario-* scenarios were ghosted: %#v", diff.GhostChecks)
	}
	if len(diff.UnsupervisedPlant) != 0 {
		t.Fatalf("supervised scenario-* scenarios reported unsupervised: %#v", diff.UnsupervisedPlant)
	}
	if len(diff.OutOfScopeChecks) != 0 {
		t.Fatalf("out-of-scope = %#v, want empty", diff.OutOfScopeChecks)
	}
}
