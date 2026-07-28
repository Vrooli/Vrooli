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

func TestReviewRunPersistsEveryModeAndBlocksOnFailure(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(posttypes.Schema), database.SchemaProviderFunc(review.Schema)))
	_, err := d.ExecContext(context.Background(), `INSERT INTO drafts (id, campaign_id, post_type_id, body, status, created_at, updated_at) VALUES ('draft-1', 'campaign-1', 'dev-log', '', 'reviewed', '2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = d.ExecContext(context.Background(), `INSERT INTO post_types (id, status) VALUES ('dev-log', 'active')`)
	require.NoError(t, err)
	_, err = d.ExecContext(context.Background(), `INSERT INTO post_type_failure_modes (post_type_id, failure_mode) VALUES ('dev-log', 'disclosure'), ('dev-log', 'fabricated_testimonial')`)
	require.NoError(t, err)
	s := review.NewService(d)
	run, err := s.Record(context.Background(), "draft-1", []review.Verdict{{Mode: "disclosure", Passed: true}, {Mode: "fabricated_testimonial", Passed: false, Finding: "unsupported quote"}})
	require.NoError(t, err)
	require.Equal(t, review.OutcomeBlocked, run.Outcome)
	runs, err := s.List(context.Background())
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Len(t, runs[0].Verdicts, 2)
	require.Equal(t, "fabricated_testimonial", runs[0].Verdicts[0].Mode)
}

func TestReviewRunRefusesIncompleteOrUndeclaredModes(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(posttypes.Schema), database.SchemaProviderFunc(review.Schema)))
	_, err := d.ExecContext(context.Background(), `INSERT INTO drafts (id, campaign_id, post_type_id, body, status, created_at, updated_at) VALUES ('draft-2', 'campaign-1', 'dev-log', '', 'reviewed', '2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = d.ExecContext(context.Background(), `INSERT INTO post_types (id, status) VALUES ('dev-log', 'active')`)
	require.NoError(t, err)
	_, err = d.ExecContext(context.Background(), `INSERT INTO post_type_failure_modes (post_type_id, failure_mode) VALUES ('dev-log', 'disclosure'), ('dev-log', 'credential_claim')`)
	require.NoError(t, err)
	s := review.NewService(d)
	_, err = s.Record(context.Background(), "draft-2", []review.Verdict{{Mode: "disclosure", Passed: true}})
	require.ErrorContains(t, err, "missing declared failure mode")
	_, err = s.Record(context.Background(), "draft-2", []review.Verdict{{Mode: "disclosure", Passed: true}, {Mode: "unregistered", Passed: true}})
	require.ErrorContains(t, err, "not declared")
}
