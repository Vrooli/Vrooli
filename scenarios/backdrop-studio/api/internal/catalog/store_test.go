package catalog

import (
	"context"
	"testing"

	"backdrop-studio/internal/testutil/db"
	"github.com/stretchr/testify/require"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.Exec(Schema())
	require.NoError(t, err)
	return NewStore(sqlDB)
}

func TestSeedIsIdempotentAndCatalogIsQueryable(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	require.NoError(t, store.Seed(ctx))
	require.NoError(t, store.Seed(ctx))

	surfaces, err := store.ListSurfaces(ctx)
	require.NoError(t, err)
	require.Len(t, surfaces, 9)
	require.Contains(t, []string{"play.feature-graphic", "play.phone-screenshot", "play.tablet-screenshot", "app-store-6.7-screenshot", "app-store-6.5-screenshot", "app-store-12.9-screenshot"}, surfaces[3].ID)

	styles, err := store.ListStyles(ctx, "ambient", "horizon", "duotone", "wpa_poster", "full_bleed")
	require.NoError(t, err)
	require.Len(t, styles, 1)
	require.Equal(t, "horizon-ink", styles[0].ID)
}

func TestCreateStyleRejectsUnknownClosedAxesAndEmptyOpenAxes(t *testing.T) {
	store := testStore(t)
	base := Style{
		ID: "test", Name: "Test", Role: "ambient", Subject: "horizon", Lineage: "wpa_poster",
		Strategy: "procedural", Treatments: []string{"posterize"}, Placements: []string{"full_bleed"},
	}

	badRole := base
	badRole.Role = "decorative"
	require.ErrorContains(t, store.CreateStyle(context.Background(), badRole), "invalid role")

	badStrategy := base
	badStrategy.Strategy = "model-only"
	require.ErrorContains(t, store.CreateStyle(context.Background(), badStrategy), "invalid strategy")

	badSubjectEmpty := base
	badSubjectEmpty.Subject = ""
	require.ErrorContains(t, store.CreateStyle(context.Background(), badSubjectEmpty), "invalid subject")

	badPlacement := base
	badPlacement.Placements = []string{""}
	require.ErrorContains(t, store.CreateStyle(context.Background(), badPlacement), "invalid placement")

	badTreatment := base
	badTreatment.Treatments = []string{"unknown-treatment"}
	require.ErrorContains(t, store.CreateStyle(context.Background(), badTreatment), "invalid treatment")

	badSubject := base
	badSubject.Subject = "unknown-subject"
	require.ErrorContains(t, store.CreateStyle(context.Background(), badSubject), "invalid subject")

	badLineage := base
	badLineage.Lineage = "unknown-lineage"
	require.ErrorContains(t, store.CreateStyle(context.Background(), badLineage), "invalid lineage")

	badGeneration := base
	badGeneration.Generation = &GenerationBlock{Role: "image.generate.default", Profile: "PROFILE_QUALITY_FIRST", PromptTemplate: "bad shape"}
	require.ErrorContains(t, store.CreateStyle(context.Background(), badGeneration), "cannot carry generation")

	badRegion := base
	badRegion.Regions = []Region{{X: .5, Y: .5, Width: .6, Height: .2, Kind: "overlay"}}
	require.ErrorContains(t, store.CreateStyle(context.Background(), badRegion), "invalid region")

	require.NoError(t, store.CreateStyle(context.Background(), base))
}

func TestReleasedStyleCannotBeTouched(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	v := Style{ID: "released", Name: "Released", Role: "ambient", Subject: "horizon", Lineage: "wpa_poster", Strategy: "procedural", Treatments: []string{"grain"}, Placements: []string{"full_bleed"}}
	require.NoError(t, store.CreateStyle(ctx, v))
	_, err := store.db.ExecContext(ctx, "UPDATE backdrop_styles SET released=1 WHERE id=?", v.ID)
	require.NoError(t, err)
	require.ErrorContains(t, store.TouchStyle(ctx, v.ID), "immutable")
}

func TestForkMutatesExactlyOneAxisAndPreservesLineage(t *testing.T) {
	store := testStore(t)
	parent := Style{ID: "parent", Name: "Parent", Version: 1, Role: "ambient", Subject: "horizon", Lineage: "wpa_poster", Strategy: "procedural", Treatments: []string{"posterize"}, Placements: []string{"full_bleed"}, Regions: []Region{{X: .1, Y: .1, Width: .4, Height: .2, Kind: "overlay"}}, ContrastThreshold: 4.5}
	require.NoError(t, store.CreateStyle(context.Background(), parent))
	child, err := store.ForkStyle(context.Background(), "parent", "child", map[string]string{"lineage": "bauhaus"})
	require.NoError(t, err)
	require.Equal(t, "parent", child.ParentID)
	require.Equal(t, "bauhaus", child.Lineage)
	_, err = store.ForkStyle(context.Background(), "parent", "bad", map[string]string{})
	require.ErrorContains(t, err, "exactly one axis")
}

func TestStylePackRoundTrips(t *testing.T) {
	style := Style{ID: "pack-style", Name: "Pack", Version: 1, Role: "ambient", Subject: "horizon", Lineage: "wpa_poster", Strategy: "procedural", Treatments: []string{"posterize"}, Placements: []string{"full_bleed"}, Regions: []Region{{X: .1, Y: .1, Width: .4, Height: .2, Kind: "overlay"}}, ContrastThreshold: 4.5}
	raw, err := ExportStylePack([]Style{style})
	require.NoError(t, err)
	styles, err := ImportStylePack(raw)
	require.NoError(t, err)
	require.Equal(t, []Style{style}, styles)
}
