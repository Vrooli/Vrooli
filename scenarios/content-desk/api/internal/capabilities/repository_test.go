package capabilities_test

import (
	"context"
	"testing"

	"content-desk/internal/capabilities"

	db "github.com/vrooli/api-core/databasetest"

	localdb "content-desk/internal/database"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
)

func newRepository(t *testing.T) capabilities.Repository {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d,
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(capabilities.Schema)))
	return capabilities.NewRepository(d)
}

func sampleCapability() capabilities.Capability {
	return capabilities.Capability{
		Name:                     "Product screenshots",
		Medium:                   capabilities.MediumImage,
		Aliases:                  []string{"screenshots", "screen-captures"},
		Channels:                 []string{"blog", "x-twitter"},
		AudienceApplicability:    "developers",
		DeliveryApplicability:    "web",
		ProducingOperation:       "browser-automation-studio capture-surface",
		Prerequisites:            []string{"browser-automation-studio running"},
		Priority:                 1,
		PriorityReason:           "launch evidence",
		PriorityScope:            "launch-2026-09-17",
		DefinitionStatus:         capabilities.DefinitionDocumented,
		ImplementationStatus:     capabilities.ImplementationImplemented,
		OperationalReadiness:     capabilities.ReadinessQualified,
		OutputQuality:            capabilities.QualityUnassessed,
		DistributionConnectivity: capabilities.ConnectivityNotApplicable,
		Owner:                    "browser-automation-studio",
		SourceRefs:               []string{"manifest-2026-09-12"},
		NextAction:               "capture final set",
	}
}

func TestUpsertPreservesSeparateReadinessDimensionsAndJSONLists(t *testing.T) {
	repo := newRepository(t)
	created, err := repo.Upsert(context.Background(), sampleCapability())
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	fetched, err := repo.Get(context.Background(), created.ID, "")
	require.NoError(t, err)
	require.Equal(t, []string{"screenshots", "screen-captures"}, fetched.Aliases)
	require.Equal(t, []string{"blog", "x-twitter"}, fetched.Channels)
	require.Equal(t, []string{"browser-automation-studio running"}, fetched.Prerequisites)
	require.Equal(t, []string{"manifest-2026-09-12"}, fetched.SourceRefs)
	require.Equal(t, capabilities.ImplementationImplemented, fetched.ImplementationStatus)
	require.Equal(t, capabilities.QualityUnassessed, fetched.OutputQuality)
	require.Equal(t, capabilities.ConnectivityNotApplicable, fetched.DistributionConnectivity)
}

func TestGetResolvesAliasAndNotFound(t *testing.T) {
	repo := newRepository(t)
	created, err := repo.Upsert(context.Background(), sampleCapability())
	require.NoError(t, err)
	byAlias, err := repo.Get(context.Background(), "", "screenshots")
	require.NoError(t, err)
	require.Equal(t, created.ID, byAlias.ID)
	_, err = repo.Get(context.Background(), "", "missing")
	require.ErrorIs(t, err, capabilities.ErrNotFound)
	_, err = repo.Get(context.Background(), "", "")
	require.ErrorIs(t, err, capabilities.ErrReferenceRequired)
}

func TestAbsentQualificationReadsAsExplicitUnknown(t *testing.T) {
	repo := newRepository(t)
	created, err := repo.Upsert(context.Background(), sampleCapability())
	require.NoError(t, err)
	fetched, err := repo.Get(context.Background(), created.ID, "")
	require.NoError(t, err)
	require.False(t, fetched.HasQualification())
	require.NotNil(t, fetched.LatestQualification)
	require.True(t, fetched.LatestQualification.IsUnknown())
	require.Equal(t, capabilities.Unknown, fetched.LatestQualification.Environment)
	require.Equal(t, capabilities.Unknown, fetched.LatestQualification.ObservedAt)
	require.Equal(t, capabilities.UnknownMaxAgeSeconds, fetched.LatestQualification.MaxAgeSeconds)

	qualification, err := repo.LatestQualification(context.Background(), created.ID)
	require.NoError(t, err)
	require.True(t, qualification.IsUnknown())
}

func TestRecordQualificationBecomesLatestAndExplicit(t *testing.T) {
	repo := newRepository(t)
	created, err := repo.Upsert(context.Background(), sampleCapability())
	require.NoError(t, err)
	recorded, err := repo.RecordQualification(context.Background(), capabilities.CapabilityQualification{
		CapabilityID:   created.ID,
		Environment:    "browser-automation-studio local",
		ObservedAt:     "2026-09-12T00:00:00Z",
		ValidatedAt:    "2026-09-12T00:00:00Z",
		FreshnessBasis: capabilities.FreshnessCandidateIdentity,
		Limitation:     "pre-publication redaction required",
	})
	require.NoError(t, err)
	require.NotEmpty(t, recorded.ID)

	fetched, err := repo.Get(context.Background(), created.ID, "")
	require.NoError(t, err)
	require.True(t, fetched.HasQualification())
	require.False(t, fetched.LatestQualification.IsUnknown())
	require.Equal(t, "browser-automation-studio local", fetched.LatestQualification.Environment)
	require.Equal(t, capabilities.FreshnessCandidateIdentity, fetched.LatestQualification.FreshnessBasis)

	_, err = repo.RecordQualification(context.Background(), capabilities.CapabilityQualification{
		CapabilityID:   created.ID,
		ObservedAt:     "2026-09-12T00:00:00Z",
		FreshnessBasis: "bogus",
	})
	require.ErrorIs(t, err, capabilities.ErrInvalidFreshness)
}

func TestLinkRejectsInvalidRelationAndTarget(t *testing.T) {
	repo := newRepository(t)
	created, err := repo.Upsert(context.Background(), sampleCapability())
	require.NoError(t, err)
	_, err = repo.Link(context.Background(), capabilities.CapabilityLink{CapabilityID: created.ID, Relation: "", TargetID: "x"})
	require.ErrorIs(t, err, capabilities.ErrInvalidRelation)
	_, err = repo.Link(context.Background(), capabilities.CapabilityLink{CapabilityID: created.ID, Relation: "bogus", TargetID: "x"})
	require.ErrorIs(t, err, capabilities.ErrInvalidRelation)
	_, err = repo.Link(context.Background(), capabilities.CapabilityLink{CapabilityID: created.ID, Relation: capabilities.RelationProduces, TargetID: ""})
	require.ErrorIs(t, err, capabilities.ErrInvalidTarget)

	link, err := repo.Link(context.Background(), capabilities.CapabilityLink{CapabilityID: created.ID, Relation: capabilities.RelationEvidence, TargetID: "draft-1"})
	require.NoError(t, err)
	require.NotEmpty(t, link.ID)
	links, err := repo.LinksForCapability(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, []capabilities.CapabilityLink{link}, links)
	all, err := repo.ListLinks(context.Background())
	require.NoError(t, err)
	require.Len(t, all, 1)
}

func TestUpsertRejectsUnknownEnumeratedValues(t *testing.T) {
	repo := newRepository(t)
	bad := sampleCapability()
	bad.Medium = "hologram"
	_, err := repo.Upsert(context.Background(), bad)
	require.ErrorIs(t, err, capabilities.ErrInvalidMedium)

	bad = sampleCapability()
	bad.OperationalReadiness = "healthy"
	_, err = repo.Upsert(context.Background(), bad)
	require.ErrorIs(t, err, capabilities.ErrInvalidStatus)

	bad = sampleCapability()
	bad.Name = ""
	_, err = repo.Upsert(context.Background(), bad)
	require.ErrorIs(t, err, capabilities.ErrInvalidName)
}

func TestListFiltersByMedium(t *testing.T) {
	repo := newRepository(t)
	image := sampleCapability()
	_, err := repo.Upsert(context.Background(), image)
	require.NoError(t, err)
	video := sampleCapability()
	video.Name = "Short form video"
	video.Medium = capabilities.MediumVideo
	_, err = repo.Upsert(context.Background(), video)
	require.NoError(t, err)

	all, err := repo.List(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, all, 2)
	images, err := repo.List(context.Background(), capabilities.MediumImage)
	require.NoError(t, err)
	require.Len(t, images, 1)
	require.Equal(t, capabilities.MediumImage, images[0].Medium)
	_, err = repo.List(context.Background(), "hologram")
	require.ErrorIs(t, err, capabilities.ErrInvalidMedium)
}

// TestRepositoryDedupesRepeatedAliasesOnRead proves a capability whose stored
// alias list repeats a value (the catalog prepends the slug, and some entries
// already list it) reads back as a set. The report must not present the same
// alias twice.
func TestRepositoryDedupesRepeatedAliasesOnRead(t *testing.T) {
	repo := newRepository(t)
	capability := sampleCapability()
	capability.Aliases = []string{"written-marketing", "written-marketing", "long-form"}
	created, err := repo.Upsert(context.Background(), capability)
	require.NoError(t, err)

	fetched, err := repo.Get(context.Background(), created.ID, "")
	require.NoError(t, err)
	require.Equal(t, []string{"written-marketing", "long-form"}, fetched.Aliases)

	byAlias, err := repo.Get(context.Background(), "", "long-form")
	require.NoError(t, err)
	require.Equal(t, created.ID, byAlias.ID)
}
