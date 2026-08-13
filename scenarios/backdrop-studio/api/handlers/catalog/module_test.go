package catalog

import (
	"testing"

	"backdrop-studio/internal/catalog"

	"github.com/stretchr/testify/require"
)

// TestStyleRoundTripsItsParametersAndLineage is the regression gate for the
// three fields this boundary used to drop.
//
// The mapping is hand-written in both directions, so a field added to the store
// and to the proto can still be missed here — and the failure is invisible from
// either side: the catalog holds the value, the proto declares it, and only a
// consumer reading the wire ever notices it is empty. That is exactly how the
// studio came to show a treatment chain with no parameters under it.
func TestStyleRoundTripsItsParametersAndLineage(t *testing.T) {
	original := catalog.Style{
		ID: "fork", Name: "Fork", Version: 2, Role: "ambient", Subject: "horizon",
		Lineage: "riso_zine", Strategy: "procedural-treated", Treatments: []string{"halftone"},
		Placements: []string{"full_bleed"}, ContrastThreshold: 4.5,
		TreatmentParams: map[string]string{"halftone": `{"lpi":72}`},
		Inks:            map[string]string{"$brand.primary": "#1b3fbf"},
		ParentID:        "riso-horizon",
	}

	round := fromProto(toProto(original))

	require.Equal(t, original.TreatmentParams, round.TreatmentParams,
		"a style's parameters are what decide how it looks; losing them ships a chain nobody can reproduce")
	require.Equal(t, original.Inks, round.Inks,
		"the ink defaults are what let a style render with no brand bound")
	require.Equal(t, original.ParentID, round.ParentID,
		"lineage is what makes a fork reviewable rather than just another entry")
}
