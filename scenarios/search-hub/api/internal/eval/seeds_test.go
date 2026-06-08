package eval_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"search-hub/internal/eval"
)

// TestEvalSeedsAreValid guards every embedded eval suite: it must parse (Seeds()
// panics otherwise) and pass the same Validate gate RegisterSuite applies, so a
// freshly-booted hub can register and run any shipped suite.
//
// Note: this no longer cross-checks each suite's provider_id against a shipped
// provider catalog — search-hub stopped shipping provider descriptors when
// providers began self-registering from their own .vrooli/search.json (Phase 2).
// The provider set is now dynamic (whatever has registered), so a suite's
// provider is resolved at RUN time; an unknown provider_id fails the run loudly
// rather than at this static gate. We still assert each suite names SOME provider
// so a blank reference is caught early.
func TestEvalSeedsAreValid(t *testing.T) {
	suites := eval.Seeds()
	require.NotEmpty(t, suites, "at least one eval suite must ship")

	for id, s := range suites {
		require.Equal(t, id, s.GetSuiteId(), "seed map key must equal the suite's suite_id")
		eval.Normalize(s)
		require.NoErrorf(t, eval.Validate(s), "embedded suite %q must be a valid registerable suite", id)
		require.NotEmptyf(t, strings.TrimSpace(s.GetProviderId()),
			"suite %q must reference a provider_id (resolved at run time)", id)
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
