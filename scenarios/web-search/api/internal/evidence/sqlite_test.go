package evidence_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"

	"web-search/internal/evidence"
)

func newRepo(t *testing.T) (evidence.Repository, *sql.DB) {
	t.Helper()
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(evidence.Schema)))
	return evidence.NewSQLiteRepository(db, nil), db
}

func TestReceiptAndPassageSurviveReadback(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	receipt, err := repo.CreateReceipt(ctx, evidence.NewObservation{URL: "https://example.test/page", Content: []byte("alpha 🙂 omega"), ExtractionRevision: "extract-v1"})
	require.NoError(t, err)
	require.NotEmpty(t, receipt.ReceiptID)
	require.NotEmpty(t, receipt.ObservationID)

	passage, err := repo.CreatePassage(ctx, receipt.ReceiptID, 0, len([]byte("alpha 🙂")))
	require.NoError(t, err)
	got, err := repo.GetPassage(ctx, passage.PassageID)
	require.NoError(t, err)
	require.Equal(t, passage.Content, got.Content)
	require.Equal(t, passage.Hash, got.Hash)
}

func TestReceiptPreservesProducerExecutionIdentity(t *testing.T) {
	repo, _ := newRepo(t)
	receipt, err := repo.CreateReceipt(context.Background(), evidence.NewObservation{
		URL:                 "https://example.test/spa",
		ProducerExecutionID: "bas-execution-1",
		Content:             []byte("rendered"),
		ExtractionRevision:  "bas-readable-text-v1",
	})
	require.NoError(t, err)
	require.Equal(t, "bas-execution-1", receipt.ProducerExecutionID)
	got, err := repo.GetReceipt(context.Background(), receipt.ReceiptID)
	require.NoError(t, err)
	require.Equal(t, "bas-execution-1", got.ProducerExecutionID)
}

func TestReceiptPreservesRedirectProvenance(t *testing.T) {
	repo, _ := newRepo(t)
	input := evidence.NewObservation{
		URL: "https://example.test/start", FinalURL: "https://example.test/final",
		RedirectURLs: []string{"https://example.test/middle", "https://example.test/final"},
		Content:      []byte("redirected"), ExtractionRevision: "extract-v1",
	}
	receipt, err := repo.CreateReceipt(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, input.FinalURL, receipt.FinalURL)
	require.Equal(t, input.RedirectURLs, receipt.RedirectURLs)
	got, err := repo.GetReceipt(context.Background(), receipt.ReceiptID)
	require.NoError(t, err)
	require.Equal(t, receipt, got)
}

func TestValidationObservationGetsDistinctReceiptAndRetainsValidators(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	first, err := repo.CreateReceipt(ctx, evidence.NewObservation{URL: "https://example.test/page", RetrievedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Content: []byte("same"), ExtractionRevision: "extract-v1", ETag: `"v1"`, LastModified: "Tue, 01 Sep 2026 00:00:00 GMT"})
	require.NoError(t, err)
	second, err := repo.CreateReceipt(ctx, evidence.NewObservation{URL: "https://example.test/page", RetrievedAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), Content: []byte{}, ExtractionRevision: "extract-v1", Retention: "metadata-only", FailureCode: "not_modified", ETag: `"v1"`, LastModified: "Tue, 01 Sep 2026 00:00:00 GMT"})
	require.NoError(t, err)
	require.NotEqual(t, first.ReceiptID, second.ReceiptID)
	require.NotEqual(t, first.ObservationID, second.ObservationID)
	require.Equal(t, "not_modified", second.FailureCode)
	require.Equal(t, `"v1"`, second.ETag)
	got, err := repo.GetReceipt(ctx, second.ReceiptID)
	require.NoError(t, err)
	require.Equal(t, second, got)
}

func TestAssessmentPersistsOwnerIdentity(t *testing.T) {
	repo, db := newRepo(t)
	writer, ok := repo.(evidence.AssessmentWriter)
	require.True(t, ok)
	stored, err := writer.CreateAssessment(context.Background(), evidence.Assessment{
		ClaimID: "claim-1", Disposition: "supported", PolicyRevision: "policy-v1",
		EvidenceJSON: `[{"passage_id":"passage-1"}]`,
	})
	require.NoError(t, err)
	require.NotEmpty(t, stored.AssessmentID)
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM evidence_assessments WHERE assessment_id = ? AND claim_id = ?`, stored.AssessmentID, "claim-1").Scan(&count))
	require.Equal(t, 1, count)
	reader, ok := repo.(evidence.AssessmentReader)
	require.True(t, ok)
	readback, err := reader.GetAssessment(context.Background(), stored.AssessmentID)
	require.NoError(t, err)
	require.Equal(t, stored, readback)
}

func TestPassageInsideMultibyteRuneRejected(t *testing.T) {
	repo, _ := newRepo(t)
	receipt, err := repo.CreateReceipt(context.Background(), evidence.NewObservation{URL: "https://example.test/page", Content: []byte("a🙂b"), ExtractionRevision: "extract-v1"})
	require.NoError(t, err)
	_, err = repo.CreatePassage(context.Background(), receipt.ReceiptID, 2, 5)
	require.Error(t, err)
}

func TestReceiptUsesDistinctObservationIdentityForSameContent(t *testing.T) {
	repo, _ := newRepo(t)
	in := evidence.NewObservation{URL: "https://example.test/page", Content: []byte("same"), ExtractionRevision: "extract-v1"}
	a, err := repo.CreateReceipt(context.Background(), in)
	require.NoError(t, err)
	b, err := repo.CreateReceipt(context.Background(), in)
	require.NoError(t, err)
	require.NotEqual(t, a.ReceiptID, b.ReceiptID)
	require.NotEqual(t, a.ObservationID, b.ObservationID)
	require.Equal(t, a.ContentHash, b.ContentHash)
}

func TestInterruptedReceiptPersistenceRollsBackArtifactAndReceipt(t *testing.T) {
	repo, db := newRepo(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `CREATE TRIGGER fail_receipt_insert BEFORE INSERT ON evidence_receipts BEGIN SELECT RAISE(ABORT, 'injected receipt interruption'); END`)
	require.NoError(t, err)

	input := evidence.NewObservation{URL: "https://example.test/interrupted", Content: []byte("transactional evidence"), ExtractionRevision: "extract-v1"}
	_, err = repo.CreateReceipt(ctx, input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insert evidence receipt")

	hash := evidence.ContentHash(input.Content)
	var artifacts, receipts int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM evidence_artifacts WHERE content_hash = ?`, hash).Scan(&artifacts))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM evidence_receipts WHERE url = ?`, input.URL).Scan(&receipts))
	require.Equal(t, 0, artifacts, "an interrupted receipt must not orphan its content artifact")
	require.Equal(t, 0, receipts, "an interrupted receipt must not expose a success-shaped row")
}

func TestReceiptCannotResolveAnotherInstanceArtifact(t *testing.T) {
	primaryRepo, _ := newRepo(t)
	shadowRepo, _ := newRepo(t)
	ctx := context.Background()

	primaryReceipt, err := primaryRepo.CreateReceipt(ctx, evidence.NewObservation{
		URL:                "https://primary.example/page",
		Content:            []byte("primary instance content"),
		ExtractionRevision: "extract-v1",
	})
	require.NoError(t, err)
	primaryPassage, err := primaryRepo.CreatePassage(ctx, primaryReceipt.ReceiptID, 0, len("primary instance"))
	require.NoError(t, err)

	_, err = shadowRepo.GetReceipt(ctx, primaryReceipt.ReceiptID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
	_, err = shadowRepo.GetPassage(ctx, primaryPassage.PassageID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestCorruptedArtifactFailsIntegrityCheck(t *testing.T) {
	repo, db := newRepo(t)
	ctx := context.Background()
	receipt, err := repo.CreateReceipt(ctx, evidence.NewObservation{URL: "https://example.test/page", Content: []byte("trusted content"), ExtractionRevision: "extract-v1"})
	require.NoError(t, err)
	passage, err := repo.CreatePassage(ctx, receipt.ReceiptID, 0, len("trusted content"))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE evidence_artifacts SET content = ? WHERE artifact_id = ?`, []byte("tampered content"), receipt.ArtifactID)
	require.NoError(t, err)
	_, err = repo.GetPassage(ctx, passage.PassageID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "integrity")
}

func TestFailureReceiptRetainsMetadataWithoutContent(t *testing.T) {
	repo, _ := newRepo(t)
	receipt, err := repo.CreateReceipt(context.Background(), evidence.NewObservation{URL: "https://example.test/unavailable", ExtractionRevision: "fetch-failure-v1", Retention: "metadata-only", FailureCode: "timeout"})
	require.NoError(t, err)
	require.Equal(t, "timeout", receipt.FailureCode)
	require.Equal(t, "metadata-only", receipt.Retention)
	require.NotEmpty(t, receipt.ContentHash)
}

func TestContentExpiryPreservesReceiptMetadata(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	receipt, err := repo.CreateReceipt(ctx, evidence.NewObservation{URL: "https://example.test/old", Content: []byte("old content"), ExtractionRevision: "extract-v1"})
	require.NoError(t, err)
	_, err = repo.ExpireContent(ctx, time.Now().Add(time.Second))
	require.NoError(t, err)
	got, err := repo.GetReceipt(ctx, receipt.ReceiptID)
	require.NoError(t, err)
	require.Equal(t, receipt.ContentHash, got.ContentHash)
	_, err = repo.CreatePassage(ctx, receipt.ReceiptID, 0, 3)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unavailable")
}

func TestRetentionPreviewAndExpiryProtectReferencedArtifact(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	protected, err := repo.CreateReceipt(ctx, evidence.NewObservation{URL: "https://example.test/protected", Content: []byte("keep me"), ExtractionRevision: "extract-v1"})
	require.NoError(t, err)
	removable, err := repo.CreateReceipt(ctx, evidence.NewObservation{URL: "https://example.test/removable", Content: []byte("remove me"), ExtractionRevision: "extract-v1"})
	require.NoError(t, err)
	op, ok := repo.(evidence.RetentionOperator)
	require.True(t, ok)
	preview, err := op.PreviewContentExpiry(ctx, time.Now().Add(time.Second), []string{protected.ArtifactID})
	require.NoError(t, err)
	require.Equal(t, 1, preview.ArtifactCount)
	require.Equal(t, int64(len("remove me")), preview.ContentBytes)
	count, err := op.ExpireContentExcept(ctx, time.Now().Add(time.Second), []string{protected.ArtifactID})
	require.NoError(t, err)
	require.Equal(t, 1, count)
	_, err = repo.CreatePassage(ctx, protected.ReceiptID, 0, len("keep me"))
	require.NoError(t, err)
	_, err = repo.CreatePassage(ctx, removable.ReceiptID, 0, len("remove me"))
	require.Error(t, err)
}
