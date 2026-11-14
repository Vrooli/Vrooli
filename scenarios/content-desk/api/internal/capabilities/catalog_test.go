package capabilities_test

import (
	"context"
	"testing"

	"content-desk/internal/capabilities"

	"github.com/stretchr/testify/require"
)

func TestCanonicalCatalogHasNineteenStableCapabilities(t *testing.T) {
	catalog, err := capabilities.CanonicalCatalog()
	require.NoError(t, err)
	require.Len(t, catalog, 19)

	again, err := capabilities.CanonicalCatalog()
	require.NoError(t, err)
	require.Equal(t, catalog, again, "catalog ids and fields must be deterministic")

	seen := map[string]bool{}
	for _, capability := range catalog {
		require.NotEmpty(t, capability.ID)
		require.False(t, seen[capability.ID], "duplicate capability id %s", capability.ID)
		seen[capability.ID] = true
		require.NotEmpty(t, capability.Name)
		require.True(t, capabilities.IsValidMedium(capability.Medium), "medium %q", capability.Medium)
		require.True(t, capabilities.IsValidDefinitionStatus(capability.DefinitionStatus))
		require.True(t, capabilities.IsValidImplementationStatus(capability.ImplementationStatus))
		require.True(t, capabilities.IsValidOperationalReadiness(capability.OperationalReadiness))
		require.True(t, capabilities.IsValidOutputQuality(capability.OutputQuality))
		require.True(t, capabilities.IsValidDistributionConnectivity(capability.DistributionConnectivity))
		require.NotEmpty(t, capability.NextAction)
		require.NotEmpty(t, capability.SourceRefs)
		// Every catalogued capability is documented by the MARKETING.md catalog.
		require.Equal(t, capabilities.DefinitionDocumented, capability.DefinitionStatus)
		// None has accepted output yet; approval has never been exercised.
		require.Equal(t, capabilities.QualityUnassessed, capability.OutputQuality)
	}
}

func TestCanonicalCatalogAliasesAreUnique(t *testing.T) {
	catalog, err := capabilities.CanonicalCatalog()
	require.NoError(t, err)
	for _, capability := range catalog {
		seen := map[string]bool{}
		for _, alias := range capability.Aliases {
			require.NotEmpty(t, alias, "%s has an empty alias", capability.Name)
			require.False(t, seen[alias], "%s repeats alias %q in %v", capability.Name, alias, capability.Aliases)
			seen[alias] = true
		}
	}
}

func TestSeedInsertsOwnedCapabilitiesAndPreservesUnknownOwnerGaps(t *testing.T) {
	repo := newRepository(t)
	result, err := capabilities.SeedCatalog(context.Background(), repo)
	require.NoError(t, err)
	require.Equal(t, capabilities.SeedResult{Inserted: 19, Existing: 0}, result)

	listed, err := repo.List(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, listed, 19)

	var ownerless, owned int
	for _, capability := range listed {
		if capability.Owner == "" {
			ownerless++
			// An ownerless capability must read as unavailable, never as zeroed
			// success, and its qualification must remain explicitly unknown.
			require.Equal(t, capabilities.ReadinessUnavailable, capability.OperationalReadiness)
		} else {
			owned++
		}
		require.False(t, capability.HasQualification())
		require.NotNil(t, capability.LatestQualification)
		require.True(t, capability.LatestQualification.IsUnknown())
	}
	require.Equal(t, 7, ownerless, "the inventory names seven categories with no usable owner")
	require.Equal(t, 12, owned)
}

func TestSeedIsIdempotentAndPreservesOperatorEdits(t *testing.T) {
	repo := newRepository(t)
	_, err := capabilities.SeedCatalog(context.Background(), repo)
	require.NoError(t, err)

	catalog, err := capabilities.CanonicalCatalog()
	require.NoError(t, err)
	edited := catalog[0]
	edited.NextAction = "operator override"

	updated, err := repo.Upsert(context.Background(), edited)
	require.NoError(t, err)
	require.Equal(t, "operator override", updated.NextAction)

	second, err := capabilities.SeedCatalog(context.Background(), repo)
	require.NoError(t, err)
	require.Equal(t, capabilities.SeedResult{Inserted: 0, Existing: 19}, second)

	fetched, err := repo.Get(context.Background(), edited.ID, "")
	require.NoError(t, err)
	require.Equal(t, "operator override", fetched.NextAction, "seed must not overwrite an existing capability")
}
