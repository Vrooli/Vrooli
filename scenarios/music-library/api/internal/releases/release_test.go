package releases_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
	"music-library/internal/catalog"
	"music-library/internal/releases"
)

type artistResolver struct{ resolved bool }

func (r artistResolver) ResolveArtist(context.Context, string, string, string) (bool, error) {
	return r.resolved, nil
}

type assetVerifier struct{ verified bool }

func (v assetVerifier) VerifyAsset(context.Context, releases.Asset) (bool, error) {
	return v.verified, nil
}

type derivationVerifier struct{ verified bool }

func (v derivationVerifier) VerifyDerivation(_ context.Context, item releases.Derivation) (bool, error) {
	return v.verified && item.OperationOwner == "music-tools" && item.OperationID == "sample-job-1" && item.EvidenceRef == "music-tools:sample-job-1", nil
}

func TestVersionedEditionKeepsTrackOrderRightsLineageAndOperatorReview(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(catalog.Schema),
		apidb.SchemaProviderFunc(releases.Schema),
	))
	ctx := context.Background()
	catalogStore := catalog.NewStore(db)
	edition := fixtureEdition()
	for i, track := range edition.Tracks {
		_, err := catalogStore.Put(ctx, catalog.Track{SHA256: track.TrackSHA256, Title: "Track", SourcePath: "/fixture/track.wav", TakeID: "take-" + string(rune('a'+i)), ReceiptPath: "receipt-" + track.TrackSHA256, Approval: "liked"})
		require.NoError(t, err, "fixture %d", i)
	}
	// A caller's provenance fields are untrusted; SaveDraft must derive these
	// values from the exact catalog hash while inserting the immutable revision.
	edition.Recordings = []releases.RecordingProvenance{{TrackSHA256: edition.Tracks[0].TrackSHA256, TakeID: "forged", ReceiptRef: "forged", Approval: "liked"}}
	store := releases.NewStoreWithLineageAdapter(db, artistResolver{resolved: true}, assetVerifier{verified: true}, derivationVerifier{verified: true})
	created, err := store.SaveDraft(ctx, edition)
	require.NoError(t, err)
	require.Equal(t, int64(1), created.Revision)
	require.Equal(t, "draft", created.Status)
	require.Empty(t, created.VerifiedDerivationIDs, "client-supplied lineage verification IDs are discarded")
	for _, rights := range created.TrackRights {
		require.Equal(t, "review_required", rights.Decision, "a draft cannot forge an operator rights attestation")
	}
	wrongReceipt := created.TrackRights[0]
	_, err = store.RecordRightsDecision(ctx, releases.RightsDecision{ID: "rights-wrong-receipt", ReleaseID: edition.ReleaseID, Revision: 1, TrackSHA256: wrongReceipt.TrackSHA256, SourceLicense: wrongReceipt.SourceLicense, UsageScope: wrongReceipt.UsageScope, Decision: "confirmed", EvidenceRef: "different-receipt"})
	require.ErrorIs(t, err, releases.ErrNotReady, "a decision must pin the catalog receipt for this exact hash")
	confirmFixtureRights(t, store, created)

	loaded, err := store.Get(ctx, edition.ReleaseID, 1)
	require.NoError(t, err)
	require.Equal(t, edition.Tracks, loaded.Tracks)
	require.Equal(t, edition.Assets, loaded.Assets)
	require.Len(t, loaded.TrackRights, len(edition.TrackRights))
	require.Len(t, loaded.RightsDecisions, len(edition.TrackRights))
	for i, rights := range loaded.TrackRights {
		require.Equal(t, edition.TrackRights[i].TrackSHA256, rights.TrackSHA256)
		require.Equal(t, "operator_confirmed", rights.Decision)
		require.NotEmpty(t, rights.DecisionID)
	}
	require.Equal(t, edition.Derivations, loaded.Derivations)
	require.Equal(t, edition.LocalizedMetadata, loaded.LocalizedMetadata)
	require.Len(t, loaded.Recordings, len(edition.Tracks))
	require.Equal(t, "take-a", loaded.Recordings[0].TakeID)
	require.Equal(t, "receipt-"+edition.Tracks[0].TrackSHA256, loaded.Recordings[0].ReceiptRef)

	_, _, assessment, err := store.Assess(ctx, edition.ReleaseID, 1)
	require.NoError(t, err)
	require.True(t, assessment.ReadyForOperatorJudgment, "verified asset, artist, rights and lineage evidence should qualify a 4-track edition for human review")
	require.False(t, assessment.ShowcaseReady, "a technical readiness result cannot stand in for the operator's EP judgment")

	_, err = store.RecordReview(ctx, releases.Review{ID: "operator-review-missing-listen", ReleaseID: edition.ReleaseID, Revision: 1, Decision: "approved"})
	require.ErrorIs(t, err, releases.ErrNotReady, "approval cannot be recorded without the explicit one-batch listening attestation")
	review, err := store.RecordReview(ctx, releases.Review{ID: "operator-review-1", ReleaseID: edition.ReleaseID, Revision: 1, Decision: "approved", HeardAllTracks: true, ReviewedTrackSHA256s: trackHashes(edition.Tracks), Note: "fixture operator review"})
	require.NoError(t, err)
	require.Equal(t, "operator", review.DecisionSource)
	_, reviews, assessment, err := store.Assess(ctx, edition.ReleaseID, 1)
	require.NoError(t, err)
	require.Len(t, reviews, 1)
	require.True(t, assessment.ShowcaseReady)
	require.Len(t, reviews, 1)
	require.Len(t, reviews[0].EditionSHA256, 64)
	require.True(t, reviews[0].HeardAllTracks)
	require.Equal(t, trackHashes(edition.Tracks), reviews[0].ReviewedTrackSHA256s)
	_, err = db.ExecContext(ctx, `UPDATE music_release_reviews SET note='rewritten' WHERE review_id=?`, review.ID)
	require.Error(t, err, "operator judgments must be append-only")
	_, err = db.ExecContext(ctx, `UPDATE music_release_recording_provenance SET receipt_ref='rewritten' WHERE release_id=? AND revision=1`, edition.ReleaseID)
	require.Error(t, err, "recording provenance cannot be changed after review")
	_, err = db.ExecContext(ctx, `UPDATE music_release_rights_decisions SET note='rewritten' WHERE release_id=? AND revision=1`, edition.ReleaseID)
	require.Error(t, err, "rights decisions are immutable audit records")
	_, err = db.ExecContext(ctx, `UPDATE music_release_versions SET title='rewritten' WHERE release_id=? AND revision=1`, edition.ReleaseID)
	require.Error(t, err, "an approved revision cannot be rewritten behind its review digest")

	changed := edition
	changed.Revision = 2
	changed.Title = "EP revision two"
	changed.LocalizedMetadata = map[string]releases.LocalizedText{
		"en": {Title: "EP revision two"},
		"ja": {Title: "EP revision two ja"},
	}
	updated, err := store.SaveDraft(ctx, changed)
	require.NoError(t, err)
	require.Equal(t, int64(2), updated.Revision)
	_, _, revisionTwo, err := store.Assess(ctx, edition.ReleaseID, 2)
	require.NoError(t, err)
	require.False(t, revisionTwo.ShowcaseReady, "approval of revision one must not transfer to a different edition digest")
	old, err := store.Get(ctx, edition.ReleaseID, 1)
	require.NoError(t, err)
	require.Equal(t, edition.Title, old.Title, "an edition edit appends a version and preserves the reviewed snapshot")
	_, err = store.SaveDraft(ctx, changed)
	require.ErrorIs(t, err, releases.ErrConflict)
}

func TestReadinessBlocksRightsWideningAndUnavailableExternalProof(t *testing.T) {
	edition := fixtureEdition()
	blocked := releases.AssessShowcase(edition, nil, false, false)
	require.False(t, blocked.ReadyForOperatorJudgment)
	for _, check := range blocked.Checks {
		if check.Property == "fictional_artist_continuity" {
			require.Equal(t, releases.CheckBlocked, check.Status, "an opaque subject pointer without an owner response is not continuity proof")
		}
		if check.Property == "cover_asset_provenance" {
			require.Equal(t, releases.CheckMissing, check.Status, "a URI/hash in draft metadata is not proof the owning asset still exists")
		}
	}

	for i := range edition.TrackRights {
		edition.TrackRights[i].Decision = "operator_confirmed"
		edition.TrackRights[i].DecisionID = "operator-rights-" + strings.Repeat("x", i+1)
		edition.TrackRights[i].UsageScope = "personal_only"
		edition.TrackRights[i].EvidenceRef = "operator-rights-decision"
	}
	edition.TrackRights[1].UsageScope = "commercial"
	blocked = releases.AssessShowcase(edition, nil, true, true)
	for _, check := range blocked.Checks {
		if check.Property == "remix_sample_lineage" {
			require.Equal(t, releases.CheckBlocked, check.Status, "a derivative may not broaden personal-only source rights to commercial")
		}
	}
}

func TestReadinessRequiresEPIdentityBoundCreditsCoverOperationAndIncludedDerivedTrack(t *testing.T) {
	edition := fixtureEdition()
	_, _, ready := assessFixture(t, edition)
	require.True(t, ready.ReadyForOperatorJudgment)
	withoutOwnerReceipt := edition
	withoutOwnerReceipt.VerifiedDerivationIDs = nil
	_, _, blocked := assessFixture(t, withoutOwnerReceipt)
	require.False(t, blocked.ReadyForOperatorJudgment, "a claimed operation id and evidence string do not verify Music Tools lineage")

	edition.Kind = "album"
	_, _, notReady := assessFixture(t, edition)
	require.False(t, notReady.ReadyForOperatorJudgment, "the named showcase is an EP, not any 4-6 track release")
	edition = fixtureEdition()
	edition.Credits[0].SubjectID = "different-subject"
	_, _, notReady = assessFixture(t, edition)
	require.False(t, notReady.ReadyForOperatorJudgment, "release credit identity must match the pinned artist subject")
	edition = fixtureEdition()
	edition.Assets[0].OperationID = ""
	_, _, notReady = assessFixture(t, edition)
	require.False(t, notReady.ReadyForOperatorJudgment, "a hash without its producing operation is not cover provenance")
	edition = fixtureEdition()
	edition.Derivations[0].DerivedTrackSHA256 = strings.Repeat("e", 64)
	_, _, notReady = assessFixture(t, edition)
	require.False(t, notReady.ReadyForOperatorJudgment, "lineage must terminate at a track actually included on the EP")
}

func assessFixture(t *testing.T, edition releases.Edition) (releases.Edition, []releases.Review, releases.Eligibility) {
	t.Helper()
	return edition, nil, releases.AssessShowcase(edition, nil, true, true)
}

func TestSaveDraftRejectsMissingExactCatalogHashes(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(catalog.Schema),
		apidb.SchemaProviderFunc(releases.Schema),
	))
	edition := fixtureEdition()
	store := releases.NewStore(db)
	_, err := store.SaveDraft(context.Background(), edition)
	require.Error(t, err, "foreign key should reject a track occurrence without its exact catalog hash")
}

func TestUnreviewedCatalogHashPinsUnreviewedProvenanceAndCannotReachOperatorReview(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(catalog.Schema),
		apidb.SchemaProviderFunc(releases.Schema),
	))
	ctx := context.Background()
	catalogStore := catalog.NewStore(db)
	edition := fixtureEdition()
	for index, track := range edition.Tracks {
		approval := "liked"
		if index == 1 {
			approval = "unreviewed"
		}
		_, err := catalogStore.Put(ctx, catalog.Track{SHA256: track.TrackSHA256, Title: "Track", SourcePath: "/fixture/track.wav", TakeID: "take-" + string(rune('a'+index)), ReceiptPath: "receipt-" + track.TrackSHA256, Approval: approval})
		require.NoError(t, err)
	}
	store := releases.NewStoreWithLineageAdapter(db, artistResolver{resolved: true}, assetVerifier{verified: true}, derivationVerifier{verified: true})
	_, err := store.SaveDraft(ctx, edition)
	require.NoError(t, err, "a draft can retain provenance gaps without upgrading them to proof")
	loaded, _, assessment, err := store.Assess(ctx, edition.ReleaseID, 1)
	require.NoError(t, err)
	for _, check := range assessment.Checks {
		if check.Property == "track_recording_provenance" {
			require.Equal(t, releases.CheckBlocked, check.Status)
			require.Contains(t, strings.Join(check.Evidence, " "), "unreviewed")
		}
	}
	require.Equal(t, "unreviewed", loaded.Recordings[1].Approval)
	_, err = store.RecordReview(ctx, releases.Review{ID: "review-unliked", ReleaseID: edition.ReleaseID, Revision: 1, Decision: "approved", HeardAllTracks: true, ReviewedTrackSHA256s: trackHashes(edition.Tracks)})
	require.ErrorIs(t, err, releases.ErrNotReady)
}

func confirmFixtureRights(t *testing.T, store *releases.Store, edition releases.Edition) {
	t.Helper()
	for _, rights := range edition.TrackRights {
		_, err := store.RecordRightsDecision(context.Background(), releases.RightsDecision{
			ID: "rights-" + rights.TrackSHA256[:12], ReleaseID: edition.ReleaseID, Revision: edition.Revision,
			TrackSHA256: rights.TrackSHA256, SourceLicense: rights.SourceLicense, UsageScope: rights.UsageScope,
			Decision: "confirmed", EvidenceRef: rights.EvidenceRef, Note: "fixture operator rights confirmation",
		})
		require.NoError(t, err)
	}
}

func trackHashes(tracks []releases.TrackOccurrence) []string {
	hashes := make([]string, 0, len(tracks))
	for _, track := range tracks {
		hashes = append(hashes, track.TrackSHA256)
	}
	return hashes
}

func TestListUsesLatestImmutableRevisionAndStableKeysetCursor(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(catalog.Schema),
		apidb.SchemaProviderFunc(releases.Schema),
	))
	ctx := context.Background()
	catalogStore := catalog.NewStore(db)
	store := releases.NewStore(db)
	ids := []string{strings.Repeat("1", 64), strings.Repeat("2", 64), strings.Repeat("3", 64)}
	for i, hash := range ids {
		_, err := catalogStore.Put(ctx, catalog.Track{SHA256: hash, Title: "Track", SourcePath: "/fixture/track.wav", TakeID: "take", ReceiptPath: "receipt", Approval: "liked"})
		require.NoError(t, err)
		id := "release-" + string(rune('a'+i))
		edition := releases.Edition{ReleaseID: id, Revision: 1, Kind: "ep", Title: id, DefaultLocale: "en", LocalizedMetadata: map[string]releases.LocalizedText{"en": {Title: id}}, Tracks: []releases.TrackOccurrence{{Order: 1, TrackSHA256: hash}}}
		_, err = store.SaveDraft(ctx, edition)
		require.NoError(t, err)
		if i == 0 {
			edition.Revision = 2
			edition.Title = "latest-a"
			edition.LocalizedMetadata = map[string]releases.LocalizedText{"en": {Title: edition.Title}}
			_, err = store.SaveDraft(ctx, edition)
			require.NoError(t, err)
		}
	}
	first, err := store.List(ctx, 2, "")
	require.NoError(t, err)
	require.Len(t, first.Items, 2)
	require.NotEmpty(t, first.NextCursor)
	second, err := store.List(ctx, 2, first.NextCursor)
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	require.Empty(t, second.NextCursor)
	require.NotEqual(t, first.Items[0].ReleaseID, first.Items[1].ReleaseID)
	require.NotEqual(t, first.Items[0].ReleaseID, second.Items[0].ReleaseID)
	require.NotEqual(t, first.Items[1].ReleaseID, second.Items[0].ReleaseID)
	latest, err := store.List(ctx, 50, "")
	require.NoError(t, err)
	for _, item := range latest.Items {
		if item.ReleaseID == "release-a" {
			require.Equal(t, int64(2), item.Revision)
			require.Equal(t, "latest-a", item.Title)
			require.Equal(t, 1, item.TrackCount)
		}
	}
	_, err = store.List(ctx, 2, "not-a-cursor")
	require.ErrorIs(t, err, releases.ErrInvalid)
}

func fixtureEdition() releases.Edition {
	ids := []string{strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64), strings.Repeat("d", 64)}
	tracks := make([]releases.TrackOccurrence, 0, len(ids))
	rights := make([]releases.TrackRights, 0, len(ids))
	credits := []releases.Credit{{ID: "artist-credit", Scope: "release", Role: "primary_artist", CreditedAs: "Fictional Artist", SubjectOwner: "subjects", SubjectID: "artist-1", SubjectVersion: "v1"}}
	for i, id := range ids {
		order := i + 1
		tracks = append(tracks, releases.TrackOccurrence{Order: order, TrackSHA256: id})
		credits = append(credits,
			releases.Credit{ID: "performer-" + string(rune('0'+order)), Scope: "track", TrackOrder: order, Role: "performer", CreditedAs: "Fictional Artist", SubjectOwner: "subjects", SubjectID: "artist-1", SubjectVersion: "v1"},
			releases.Credit{ID: "composer-" + string(rune('0'+order)), Scope: "track", TrackOrder: order, Role: "composer", CreditedAs: "Fictional Artist", SubjectOwner: "subjects", SubjectID: "artist-1", SubjectVersion: "v1"},
		)
		rights = append(rights, releases.TrackRights{TrackSHA256: id, SourceLicense: "fixture-license", UsageScope: "personal_only", Decision: "operator_confirmed", DecisionID: "fixture-rights-" + string(rune('a'+i)), EvidenceRef: "receipt-" + id})
	}
	recordings := make([]releases.RecordingProvenance, 0, len(ids))
	for index, id := range ids {
		recordings = append(recordings, releases.RecordingProvenance{TrackSHA256: id, TakeID: "take-" + string(rune('a'+index)), ReceiptRef: "receipt-" + id, Approval: "liked"})
	}
	return releases.Edition{
		ReleaseID: "fixture-ep", Revision: 1, Kind: "ep", Status: "draft", Title: "Fixture EP", DefaultLocale: "en",
		ArtistOwner: "subjects", ArtistSubjectID: "artist-1", ArtistSubjectVersion: "v1",
		LocalizedMetadata: map[string]releases.LocalizedText{"en": {Title: "Fixture EP"}, "ja": {Title: "Fixture EP ja"}},
		Tracks:            tracks, Recordings: recordings, Credits: credits,
		VerifiedDerivationIDs: []string{"sample-1"},
		Assets:                []releases.Asset{{ID: "cover-1", Role: "cover_front", OwnerScenario: "image-tools", Ref: "blob:cover-1", SHA256: strings.Repeat("f", 64), OperationID: "img-job-1", ModelID: "cover-model", ModelRevision: "rev1", LicenseLane: "personal-only", RightsStatus: "personal_use", ProvenanceRef: "image-tools:img-job-1"}},
		TrackRights:           rights,
		Derivations:           []releases.Derivation{{ID: "sample-1", Kind: "sample", SourceTrackSHA256: ids[0], DerivedTrackSHA256: ids[1], OperationOwner: "music-tools", OperationID: "sample-job-1", SourceStartSeconds: 1, SourceEndSeconds: 4, DerivedStartSeconds: 8, DerivedEndSeconds: 11, InheritedRights: "personal_only", EvidenceRef: "music-tools:sample-job-1"}},
	}
}
