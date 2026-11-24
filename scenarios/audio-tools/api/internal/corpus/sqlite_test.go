//go:build audio_lab

package corpus_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	"audio-tools/internal/corpus"
	localdb "audio-tools/internal/database"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(corpus.Schema),
	))
	return d
}

func TestRepository_ReviewReferenceGroupIsAtomic(t *testing.T) {
	ctx := context.Background()
	db := newSchemaDB(t)
	clk := scheduletest.New(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	repo := corpus.NewSQLiteRepository(db, clk)
	text := "source words"
	digest := corpus.ReferenceDigest(text)
	manifestDigest := strings.Repeat("a", 64)
	members := make([]corpus.ReviewMember, 0, 2)
	for _, id := range []string{"variant-a", "variant-b"} {
		_, err := repo.Create(ctx, corpus.Clip{ID: id, ReferenceText: text, BlobKey: id, CorpusRevision: "v1", SourceID: "source-1", SpeakerPseudonym: "speaker-1", AccentGroup: "uk", ManifestTranscriptSHA256: digest, LineageManifestSHA256: manifestDigest})
		require.NoError(t, err)
		members = append(members, corpus.ReviewMember{ID: id, ExpectedReferenceSHA256: digest})
	}
	_, err := db.Exec(`CREATE TRIGGER fail_second_group_review BEFORE INSERT ON corpus_reference_reviews WHEN NEW.clip_id = 'variant-b' BEGIN SELECT RAISE(ABORT, 'forced review failure'); END`)
	require.NoError(t, err)
	review := corpus.ReferenceReview{Reviewer: "human", Protocol: "listen-each-v1", ReviewedAt: "2026-06-29T12:00:00Z", ReferenceSHA256: corpus.ReferenceDigest("corrected words")}
	_, err = repo.ReviewReferenceGroup(ctx, members, "source-1", "v1", "speaker-1", "uk", digest, manifestDigest, "corrected words", review)
	require.Error(t, err)
	for _, id := range []string{"variant-a", "variant-b"} {
		clip, getErr := repo.Get(ctx, id)
		require.NoError(t, getErr)
		require.Equal(t, text, clip.ReferenceText, "all transcript writes roll back")
		require.Nil(t, clip.ReferenceReview, "all review records roll back")
	}
}

func TestRepository_LegacyRowsRemainUngroupedAfterAdditiveMigration(t *testing.T) {
	ctx := context.Background()
	db := db.NewSQLite(t)
	_, err := db.Exec(`CREATE TABLE corpus_clips (id TEXT PRIMARY KEY, reference_text TEXT NOT NULL DEFAULT '', tags TEXT NOT NULL DEFAULT '[]', duration_ms INTEGER NOT NULL DEFAULT 0, sample_rate_hz INTEGER NOT NULL DEFAULT 0, format TEXT NOT NULL DEFAULT '', blob_key TEXT NOT NULL, source TEXT NOT NULL DEFAULT 'free_form', corpus_revision TEXT NOT NULL DEFAULT '', license_reference TEXT NOT NULL DEFAULT '', speaker_pseudonym TEXT NOT NULL DEFAULT '', split TEXT NOT NULL DEFAULT '', noise_condition TEXT NOT NULL DEFAULT '', snr_db REAL NOT NULL DEFAULT 0, content_sha256 TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO corpus_clips (id, reference_text, blob_key, created_at) VALUES ('legacy', 'old text', 'legacy-audio', '2026-06-29T12:00:00Z')`)
	require.NoError(t, err)
	require.NoError(t, localdb.ApplyMigrations(ctx, db))
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(corpus.Schema)))
	clk := scheduletest.New(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	repo := corpus.NewSQLiteRepository(db, clk)
	legacy, err := repo.Get(ctx, "legacy")
	require.NoError(t, err)
	require.Empty(t, legacy.SourceID)
	require.Empty(t, legacy.AccentGroup)
	require.Empty(t, legacy.ManifestTranscriptSHA256)
	_, err = repo.Create(ctx, corpus.Clip{ID: "new", ReferenceText: "new text", BlobKey: "new-audio"})
	require.NoError(t, err)
	created, err := repo.Get(ctx, "new")
	require.NoError(t, err)
	require.Empty(t, created.SourceID, "clips without manifest-backed import metadata remain ungrouped")
}

func TestRepository_BackfillSourceLineageIsExactDryRunAndIdempotent(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	text := "manifest transcript"
	transcriptDigest := corpus.ReferenceDigest(text)
	manifestDigest := strings.Repeat("b", 64)
	entries := make([]corpus.LineageEntry, 0, 2)
	for i, id := range []string{"import-a", "import-b"} {
		contentDigest := strings.Repeat(string(rune('c'+i)), 64)
		_, err := repo.Create(ctx, corpus.Clip{ID: id, ReferenceText: text, BlobKey: id, CorpusRevision: "v1", ContentSHA256: contentDigest})
		require.NoError(t, err)
		entries = append(entries, corpus.LineageEntry{ClipID: id, CorpusRevision: "v1", ContentSHA256: contentDigest, ManifestTranscriptSHA256: transcriptDigest, SourceID: "source-a", SpeakerPseudonym: "speaker-a", AccentGroup: "uk", Split: "held_out", LicenseReference: "VCTK license", ManifestSHA256: manifestDigest})
	}
	count, err := repo.BackfillSourceLineage(ctx, entries, true)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	before, err := repo.Get(ctx, "import-a")
	require.NoError(t, err)
	require.Empty(t, before.SourceID, "dry-run must not mutate rows")
	count, err = repo.BackfillSourceLineage(ctx, entries, false)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	for _, entry := range entries {
		clip, getErr := repo.Get(ctx, entry.ClipID)
		require.NoError(t, getErr)
		require.Equal(t, entry.SourceID, clip.SourceID)
		require.Equal(t, manifestDigest, clip.LineageManifestSHA256)
		require.Nil(t, clip.ReferenceReview)
	}
	count, err = repo.BackfillSourceLineage(ctx, entries, false)
	require.NoError(t, err)
	require.Zero(t, count, "repeated backfill is idempotent")
}

func TestRepository_BackfillSourceLineageRejectsAnyMismatchAtomically(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	entries := make([]corpus.LineageEntry, 0, 2)
	for i, id := range []string{"exact-a", "exact-b"} {
		_, err := repo.Create(ctx, corpus.Clip{ID: id, ReferenceText: "expected text", BlobKey: id, CorpusRevision: "v1", ContentSHA256: strings.Repeat(string(rune('d'+i)), 64)})
		require.NoError(t, err)
		entries = append(entries, corpus.LineageEntry{ClipID: id, CorpusRevision: "v1", ContentSHA256: strings.Repeat(string(rune('d'+i)), 64), ManifestTranscriptSHA256: corpus.ReferenceDigest("expected text"), SourceID: "source", SpeakerPseudonym: "speaker", AccentGroup: "uk", Split: "held_out", LicenseReference: "license", ManifestSHA256: strings.Repeat("e", 64)})
	}
	bad := append([]corpus.LineageEntry(nil), entries...)
	bad[1].ContentSHA256 = strings.Repeat("f", 64)
	_, err := repo.BackfillSourceLineage(ctx, bad, false)
	require.ErrorContains(t, err, "exact revision/content/transcript check failed")
	for _, entry := range entries {
		clip, getErr := repo.Get(ctx, entry.ClipID)
		require.NoError(t, getErr)
		require.Empty(t, clip.SourceID)
	}
	bad = append([]corpus.LineageEntry(nil), entries...)
	bad[1].ManifestTranscriptSHA256 = corpus.ReferenceDigest("not expected text")
	_, err = repo.BackfillSourceLineage(ctx, bad, false)
	require.ErrorContains(t, err, "exact revision/content/transcript check failed")
	_, err = repo.BackfillSourceLineage(ctx, entries[:1], false)
	require.NoError(t, err)
	conflicting := append([]corpus.LineageEntry(nil), entries...)
	conflicting[0].SourceID = "different-source"
	_, err = repo.BackfillSourceLineage(ctx, conflicting, false)
	require.ErrorContains(t, err, "conflicts with existing metadata")
}

func newRepo(t *testing.T) (corpus.Repository, *scheduletest.FakeClock) {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	return corpus.NewSQLiteRepository(newSchemaDB(t), clk), clk
}

func TestRepository_CreateGetList(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	saved, err := repo.Create(ctx, corpus.Clip{
		ReferenceText:    "the quick brown fox",
		Tags:             []string{"news", "clean"},
		DurationMs:       1500,
		SampleRateHz:     16000,
		Format:           "pcm_s16le",
		BlobKey:          "clips/2026-06/abc.pcm",
		Source:           corpus.SourceScripted,
		CorpusRevision:   "voice-pilot-en-v1",
		LicenseReference: "license-1",
		SpeakerPseudonym: "speaker-01",
		Split:            corpus.SplitDev,
		NoiseCondition:   corpus.NoiseClean,
		SNRDB:            0,
		ContentSHA256:    "abc123",
		TimedReference: &corpus.TimedReference{
			Words: []corpus.TimedWord{
				{Word: "the", StartSeconds: 0, EndSeconds: .2},
				{Word: "quick", StartSeconds: .21, EndSeconds: .5},
				{Word: "brown", StartSeconds: .51, EndSeconds: .8},
				{Word: "fox", StartSeconds: .81, EndSeconds: 1.1},
			},
			Provenance: corpus.TimingProvenance{Status: "human_reviewed", SourceReference: "source-1", AlignmentTool: "aligner", AlignmentModel: "model-1", Reviewer: "reviewer-1", ReviewMethod: "listen-and-correct", ReviewedAt: "2026-06-29T12:00:00Z"},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, saved.ID, "id is generated")
	require.False(t, saved.CreatedAt.IsZero(), "created_at is stamped from the clock")

	got, err := repo.Get(ctx, saved.ID)
	require.NoError(t, err)
	require.Equal(t, "the quick brown fox", got.ReferenceText)
	require.Equal(t, []string{"news", "clean"}, got.Tags)
	require.Equal(t, int64(1500), got.DurationMs)
	require.Equal(t, 16000, got.SampleRateHz)
	require.Equal(t, "pcm_s16le", got.Format)
	require.Equal(t, corpus.SourceScripted, got.Source)
	require.Equal(t, "voice-pilot-en-v1", got.CorpusRevision)
	require.Equal(t, "license-1", got.LicenseReference)
	require.Equal(t, "speaker-01", got.SpeakerPseudonym)
	require.Equal(t, corpus.SplitDev, got.Split)
	require.Equal(t, corpus.NoiseClean, got.NoiseCondition)
	require.Equal(t, "abc123", got.ContentSHA256)
	require.Equal(t, saved.TimedReference, got.TimedReference)

	list, err := repo.List(ctx, corpus.ListFilter{})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, saved.TimedReference, list[0].TimedReference)
}

func TestRepositoryRejectsTimedReferenceThatDoesNotMatchTranscript(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.Create(context.Background(), corpus.Clip{
		ReferenceText: "hello world",
		BlobKey:       "clips/2026-06/mismatch.pcm",
		TimedReference: &corpus.TimedReference{
			Words:      []corpus.TimedWord{{Word: "hello", StartSeconds: 0, EndSeconds: .2}, {Word: "there", StartSeconds: .3, EndSeconds: .5}},
			Provenance: corpus.TimingProvenance{Status: "draft", SourceReference: "test-source"},
		},
	})
	require.ErrorContains(t, err, "timed reference transcript mismatch")
}

func TestRepositoryCreateRollsBackClipWhenTimedReferenceWriteFails(t *testing.T) {
	db := newSchemaDB(t)
	_, err := db.Exec(`CREATE TRIGGER reject_timed_reference BEFORE INSERT ON corpus_timed_references BEGIN SELECT RAISE(ABORT, 'forced annotation failure'); END`)
	require.NoError(t, err)
	repo := corpus.NewSQLiteRepository(db, scheduletest.New(time.Now()))
	_, err = repo.Create(context.Background(), corpus.Clip{
		ReferenceText: "hello",
		BlobKey:       "clips/test/rollback.pcm",
		TimedReference: &corpus.TimedReference{
			Words:      []corpus.TimedWord{{Word: "hello", StartSeconds: 0, EndSeconds: .2}},
			Provenance: corpus.TimingProvenance{Status: "draft", SourceReference: "test-source"},
		},
	})
	require.ErrorContains(t, err, "store timed reference")
	var clips, references int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM corpus_clips`).Scan(&clips))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM corpus_timed_references`).Scan(&references))
	require.Zero(t, clips, "failed annotation write must roll back the clip row")
	require.Zero(t, references)
}

func TestRepositoryDeleteRollsBackTimedReferenceWhenClipDeleteFails(t *testing.T) {
	db := newSchemaDB(t)
	repo := corpus.NewSQLiteRepository(db, scheduletest.New(time.Now()))
	created, err := repo.Create(context.Background(), corpus.Clip{
		ReferenceText: "hello",
		BlobKey:       "clips/test/delete-rollback.pcm",
		TimedReference: &corpus.TimedReference{
			Words:      []corpus.TimedWord{{Word: "hello", StartSeconds: 0, EndSeconds: .2}},
			Provenance: corpus.TimingProvenance{Status: "draft", SourceReference: "test-source"},
		},
	})
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TRIGGER reject_clip_delete BEFORE DELETE ON corpus_clips BEGIN SELECT RAISE(ABORT, 'forced clip delete failure'); END`)
	require.NoError(t, err)
	require.Error(t, repo.Delete(context.Background(), created.ID))
	got, err := repo.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.TimedReference, got.TimedReference, "failed delete must restore the annotation row")
}

func TestRepository_ListTagFilterAndOrder(t *testing.T) {
	ctx := context.Background()
	repo, clk := newRepo(t)

	mk := func(ref, tag string) {
		_, err := repo.Create(ctx, corpus.Clip{ReferenceText: ref, Tags: []string{tag}, BlobKey: "clips/k/" + ref + ".pcm"})
		require.NoError(t, err)
		clk.Advance(time.Second) // distinct created_at so ordering is stable
	}
	mk("first", "alpha")
	mk("second", "beta")
	mk("third", "alpha")

	all, err := repo.List(ctx, corpus.ListFilter{})
	require.NoError(t, err)
	require.Len(t, all, 3)
	require.Equal(t, "third", all[0].ReferenceText, "newest first (created_at DESC)")

	alpha, err := repo.List(ctx, corpus.ListFilter{TagContains: "alpha"})
	require.NoError(t, err)
	require.Len(t, alpha, 2, "tag filter narrows to matching clips")
}

func TestRepository_CountCreatedInWindowUsesUTCAndHalfOpenBounds(t *testing.T) {
	ctx := context.Background()
	db := newSchemaDB(t)
	repo := corpus.NewSQLiteRepository(db, scheduletest.New(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)))
	for i, created := range []time.Time{
		time.Date(2026, 6, 29, 9, 59, 59, 0, time.UTC),
		time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 29, 11, 59, 59, 0, time.UTC),
		time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC),
	} {
		_, err := repo.Create(ctx, corpus.Clip{ID: string(rune('a' + i)), ReferenceText: "x", BlobKey: "clips/k/" + string(rune('a'+i)), CreatedAt: created})
		require.NoError(t, err)
	}
	// These local-time bounds are the same UTC instants as 10:00 and 12:00.
	count, err := repo.CountCreatedInWindow(ctx,
		time.Date(2026, 6, 29, 6, 0, 0, 0, time.FixedZone("EDT", -4*60*60)),
		time.Date(2026, 6, 29, 8, 0, 0, 0, time.FixedZone("EDT", -4*60*60)))
	require.NoError(t, err)
	require.EqualValues(t, 2, count, "start is inclusive and end is exclusive")
	now := time.Now()
	_, err = repo.CountCreatedInWindow(ctx, now, now)
	require.ErrorContains(t, err, "start must be before end")
}

func TestRepository_DeleteAndNotFound(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)

	_, err := repo.Get(ctx, "missing")
	require.ErrorAs(t, err, &corpus.ErrClipNotFound{})

	saved, err := repo.Create(ctx, corpus.Clip{ReferenceText: "x", BlobKey: "clips/k/x.pcm"})
	require.NoError(t, err)
	require.NoError(t, repo.Delete(ctx, saved.ID))
	require.ErrorAs(t, repo.Delete(ctx, saved.ID), &corpus.ErrClipNotFound{}, "second delete reports not-found")
}

func TestRepository_CreateRequiresBlobKey(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.Create(context.Background(), corpus.Clip{ReferenceText: "no blob"})
	require.Error(t, err, "a clip without a blob key is rejected")
}
