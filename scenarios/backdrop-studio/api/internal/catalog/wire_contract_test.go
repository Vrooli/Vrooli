package catalog_test

import (
	"database/sql"
	"testing"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/imageengine"

	"github.com/stretchr/testify/require"
	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	"google.golang.org/protobuf/encoding/protojson"
	_ "modernc.org/sqlite"
)

// TestSeededCatalogParamsParseAsImageToolsOpParams is the cross-scenario
// contract gate.
//
// On 2026-08-12 the catalog was seeded with styles requesting "normalize" and
// brand inks on the Tier-2 screens, neither of which existed on image-tools'
// proto messages. protojson rejects unknown fields, so eleven of sixteen styles
// would have failed their render with a 400 — and every unit suite stayed green,
// because backdrop-studio tests against a fake executor that never touches the
// REST edge and image-tools tests its treatments below the wire.
//
// This asserts what neither side could see alone: the exact bytes backdrop-studio
// would send are bytes image-tools will accept.
func TestSeededCatalogParamsParseAsImageToolsOpParams(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(catalog.Schema())
	require.NoError(t, err)
	store := catalog.NewStore(db)
	require.NoError(t, store.Seed(t.Context()))
	styles, err := store.ListStyles(t.Context(), "", "", "", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, styles)

	// A representative brand, so $brand.* slots resolve exactly as they would
	// at render time.
	palette := map[string]string{
		"$brand.primary":    "#1B3FD8",
		"$brand.background": "#EDE6D2",
	}

	checked := 0
	for _, style := range styles {
		style := style
		t.Run(style.ID, func(t *testing.T) {
			for _, op := range style.Treatments {
				raw := imageengine.ResolveParams(op, style.TreatmentParams[op], palette)
				pb := &opsv1.OpParams{}
				require.NoErrorf(t, protojson.Unmarshal([]byte(raw), pb),
					"style %q op %q emits params image-tools will reject:\n%s", style.ID, op, raw)

				// An unresolved slot means the ink never bound and the render
				// would carry a literal "$brand.primary" downstream.
				require.NotContains(t, raw, "$brand.",
					"style %q op %q left an unresolved brand slot: %s", style.ID, op, raw)
				checked++
			}
		})
	}
	require.Greater(t, checked, 20, "expected the seeded catalog to exercise a real spread of operations")
}
