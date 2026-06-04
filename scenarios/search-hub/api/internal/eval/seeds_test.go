package eval_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"search-hub/internal/eval"
	"search-hub/internal/providers"
)

// TestEvalSeedsAreValidAndReferenceKnownProviders guards every embedded eval
// suite: it must parse (Seeds() panics otherwise), pass the same Validate gate
// RegisterSuite applies, and reference a provider_id that a shipped provider
// seed actually registers — so a freshly-booted hub can run any shipped suite
// against a known provider, with no drift between the suite and the registry.
func TestEvalSeedsAreValidAndReferenceKnownProviders(t *testing.T) {
	suites := eval.Seeds()
	require.NotEmpty(t, suites, "at least the cli-health.commands.primary suite must ship")

	knownProviders := make(map[string]struct{})
	for _, id := range providers.SeedIDs() {
		knownProviders[id] = struct{}{}
	}

	for id, s := range suites {
		require.Equal(t, id, s.GetSuiteId(), "seed map key must equal the suite's suite_id")
		eval.Normalize(s)
		require.NoErrorf(t, eval.Validate(s), "embedded suite %q must be a valid registerable suite", id)
		require.Containsf(t, knownProviders, s.GetProviderId(),
			"suite %q references provider %q which no provider seed registers", id, s.GetProviderId())
	}
}

// TestRegisterSeedsIsIdempotent confirms shipped suites register at boot and
// re-registering (every boot) updates rather than duplicating.
func TestRegisterSeedsIsIdempotent(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	require.NoError(t, eval.RegisterSeeds(ctx, store))
	require.NoError(t, eval.RegisterSeeds(ctx, store), "second boot must be idempotent")

	got, err := store.ListSuites(ctx, eval.ListSuitesFilter{})
	require.NoError(t, err)
	require.Len(t, got, len(eval.SeedIDs()), "register-at-boot must not duplicate suites")
}
