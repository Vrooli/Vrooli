package review_test

import (
	"context"
	"testing"

	"content-desk/internal/artifacts"
	localdb "content-desk/internal/database"
	"content-desk/internal/posttypes"
	"content-desk/internal/review"
	"content-desk/internal/testutil/db"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
)

// [REQ:CONTENTD-P0-009]
func TestReviewRunPersistsEveryModeAndBlocksOnFailure(t *testing.T) {
	t.Run("[CONTENTD-P0-009] review records every policy and craft mode", func(t *testing.T) {
		d := db.NewSQLite(t)
		require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(posttypes.Schema), database.SchemaProviderFunc(review.Schema)))
		_, err := d.ExecContext(context.Background(), `INSERT INTO drafts (id, campaign_id, post_type_id, body, status, created_at, updated_at) VALUES ('draft-1', 'campaign-1', 'dev-log', '', 'reviewed', '2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z')`)
		require.NoError(t, err)
		_, err = d.ExecContext(context.Background(), `UPDATE post_types SET status = 'active' WHERE id = 'dev-log'`)
		require.NoError(t, err)
		_, err = d.ExecContext(context.Background(), `DELETE FROM post_type_failure_modes WHERE post_type_id = 'dev-log'`)
		require.NoError(t, err)
		_, err = d.ExecContext(context.Background(), `INSERT INTO post_type_failure_modes (post_type_id, failure_mode) VALUES ('dev-log', 'disclosure'), ('dev-log', 'fabricated_testimonial')`)
		require.NoError(t, err)
		s := review.NewService(d)
		run, err := s.Record(context.Background(), "draft-1", []review.Verdict{
			{Mode: "disclosure", Passed: true},
			{Mode: "fabricated_testimonial", Passed: false, Finding: "unsupported quote"},
			{Mode: "credential_claim_by_persona", Passed: true},
			{Mode: "real_person_impersonation", Passed: true},
			{Mode: "fabricated_customer_testimonial", Passed: true},
			{Mode: "missing_platform_disclosure", Passed: true},
		})
		require.NoError(t, err)
		require.Equal(t, review.OutcomeBlocked, run.Outcome)
		runs, err := s.List(context.Background())
		require.NoError(t, err)
		require.Len(t, runs, 1)
		require.Len(t, runs[0].Verdicts, 6)
		require.Equal(t, "fabricated_testimonial", runs[0].Verdicts[0].Mode)
	})
}

func TestReviewRunRefusesIncompleteOrUndeclaredModes(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(posttypes.Schema), database.SchemaProviderFunc(review.Schema)))
	_, err := d.ExecContext(context.Background(), `INSERT INTO drafts (id, campaign_id, post_type_id, body, status, created_at, updated_at) VALUES ('draft-2', 'campaign-1', 'dev-log', '', 'reviewed', '2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = d.ExecContext(context.Background(), `UPDATE post_types SET status = 'active' WHERE id = 'dev-log'`)
	require.NoError(t, err)
	_, err = d.ExecContext(context.Background(), `DELETE FROM post_type_failure_modes WHERE post_type_id = 'dev-log'`)
	require.NoError(t, err)
	_, err = d.ExecContext(context.Background(), `INSERT INTO post_type_failure_modes (post_type_id, failure_mode) VALUES ('dev-log', 'disclosure'), ('dev-log', 'credential_claim')`)
	require.NoError(t, err)
	s := review.NewService(d)
	_, err = s.Record(context.Background(), "draft-2", []review.Verdict{{Mode: "disclosure", Passed: true}})
	require.ErrorContains(t, err, "missing declared failure mode")
	_, err = s.Record(context.Background(), "draft-2", []review.Verdict{{Mode: "disclosure", Passed: true}, {Mode: "unregistered", Passed: true}})
	require.ErrorContains(t, err, "not declared")
}

func TestReviewRerunSupersedesWithoutDeletingPriorRun(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(posttypes.Schema), database.SchemaProviderFunc(review.Schema)))
	_, err := d.ExecContext(context.Background(), `INSERT INTO drafts (id, campaign_id, post_type_id, body, status, created_at, updated_at) VALUES ('draft-3', 'campaign-1', 'dev-log', '', 'reviewed', '2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z')`)
	require.NoError(t, err)
	s := review.NewService(d)
	first, err := s.Record(context.Background(), "draft-3", devLogVerdicts())
	require.NoError(t, err)
	second, err := s.Record(context.Background(), "draft-3", devLogVerdicts())
	require.NoError(t, err)
	runs, err := s.List(context.Background())
	require.NoError(t, err)
	require.Len(t, runs, 2)
	var superseding string
	require.NoError(t, d.QueryRowContext(context.Background(), `SELECT superseding_run_id FROM review_supersessions WHERE superseded_run_id = ?`, first.ID).Scan(&superseding))
	require.Equal(t, second.ID, superseding)
}

func devLogVerdicts() []review.Verdict {
	return []review.Verdict{
		{Mode: "what_without_why", Passed: true},
		{Mode: "internal_vocabulary_leakage", Passed: true},
		{Mode: "credential_claim_by_persona", Passed: true},
		{Mode: "real_person_impersonation", Passed: true},
		{Mode: "fabricated_customer_testimonial", Passed: true},
		{Mode: "missing_platform_disclosure", Passed: true},
	}
}
