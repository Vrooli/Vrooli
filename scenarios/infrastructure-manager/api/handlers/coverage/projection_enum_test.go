package coverage

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/spacedoc"
	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/coverage"
)

// Every projection the model declares must carry its own label on the typed
// surface. A projection that resolves to UNSPECIFIED is not a missing label —
// it silently merges with every other unlabelled projection in any consumer
// that groups by this field, which is how `substrate` spent its first day on
// the board reporting as `unspecified` with otherwise correct counts.
func TestEveryDeclaredProjectionHasItsOwnProtoLabel(t *testing.T) {
	declared := []spacedoc.Projection{
		spacedoc.ProjectionSupervision, spacedoc.ProjectionAvailability, spacedoc.ProjectionRecovery,
		spacedoc.ProjectionSubstrate, spacedoc.ProjectionCapacity, spacedoc.ProjectionHeadroom,
		spacedoc.ProjectionDurability, spacedoc.ProjectionAttribution, spacedoc.ProjectionValidationCost,
		spacedoc.ProjectionAgentThroughput, spacedoc.ProjectionCommissioning,
	}

	seen := map[coveragev1.Projection]spacedoc.Projection{}
	for _, projection := range declared {
		got := protoProjection(projection)
		require.NotEqual(t, coveragev1.Projection_PROJECTION_UNSPECIFIED, got,
			"projection %q resolves to UNSPECIFIED on the typed surface", projection)
		require.NotContains(t, seen, got,
			"projections %q and %q both resolve to %s", projection, seen[got], got)
		seen[got] = projection
	}
	require.Len(t, seen, len(declared))
}

// An identifier the proto does not know must stay UNSPECIFIED rather than
// borrowing a neighbour's number.
func TestUnknownProjectionResolvesToUnspecified(t *testing.T) {
	require.Equal(t, coveragev1.Projection_PROJECTION_UNSPECIFIED, protoProjection(spacedoc.Projection("not-a-projection")))
}
