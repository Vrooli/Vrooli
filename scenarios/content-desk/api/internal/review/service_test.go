package review_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"content-desk/internal/artifacts"
	localdb "content-desk/internal/database"
	"content-desk/internal/posttypes"
	"content-desk/internal/review"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
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
		s := review.NewService(d, testRevisionSource{repo: artifacts.NewSQLiteRepository(d)})
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
	s := review.NewService(d, testRevisionSource{repo: artifacts.NewSQLiteRepository(d)})
	_, err = s.Record(context.Background(), "draft-2", []review.Verdict{{Mode: "disclosure", Passed: true}})
	require.ErrorContains(t, err, "missing declared failure mode")
	_, err = s.Record(context.Background(), "draft-2", []review.Verdict{{Mode: "disclosure", Passed: true}, {Mode: "unregistered", Passed: true}})
	require.ErrorContains(t, err, "not declared")
}

func TestRecordExactBindsReviewToCurrentRevisionAndReportsRequiredModes(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(posttypes.Schema), database.SchemaProviderFunc(review.Schema)))
	_, err := d.ExecContext(ctx, `INSERT INTO drafts (id,campaign_id,post_type_id,body,status,created_at,updated_at) VALUES ('exact-review','','dev-log','synthetic evidence-backed note','checking','2026-07-01T00:00:00Z','2026-07-01T00:00:00Z')`)
	require.NoError(t, err)
	repo := artifacts.NewSQLiteRepository(d)
	current, err := repo.GetCurrentRevision(ctx, "exact-review")
	require.NoError(t, err)
	svc := review.NewService(d, testRevisionSource{repo: repo})
	modes, err := svc.ListRequiredModes(ctx, "dev-log")
	require.NoError(t, err)
	require.Contains(t, modes, "credential_claim_by_persona")
	require.Contains(t, modes, "missing_platform_disclosure")
	verdicts := devLogVerdicts()
	for i := range verdicts {
		verdicts[i].Evidence = "Synthetic local evidence review."
	}
	run, err := svc.RecordExact(ctx, "exact-review", current.RevisionID, current.Body, verdicts)
	require.NoError(t, err)
	require.Equal(t, current.RevisionID, run.SourceRevisionID)

	_, err = repo.UpdateBody(ctx, "exact-review", "new synthetic wording")
	require.NoError(t, err)
	_, err = svc.RecordExact(ctx, "exact-review", current.RevisionID, current.Body, verdicts)
	require.ErrorContains(t, err, "changed")
	runs, err := svc.List(ctx)
	require.NoError(t, err)
	require.Len(t, runs, 1, "a stale retry must not create or supersede a review run")
	var bases int
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM review_run_basis`).Scan(&bases))
	require.Equal(t, 1, bases)
}

func TestRecordExactRequiresEvidenceAndFailureFinding(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(posttypes.Schema), database.SchemaProviderFunc(review.Schema)))
	_, err := d.ExecContext(ctx, `INSERT INTO drafts (id,campaign_id,post_type_id,body,status,created_at,updated_at) VALUES ('evidence-review','','dev-log','synthetic text','checking','2026-07-01T00:00:00Z','2026-07-01T00:00:00Z')`)
	require.NoError(t, err)
	repo := artifacts.NewSQLiteRepository(d)
	current, err := repo.GetCurrentRevision(ctx, "evidence-review")
	require.NoError(t, err)
	svc := review.NewService(d, testRevisionSource{repo: repo})
	verdicts := devLogVerdicts()
	for i := range verdicts {
		verdicts[i].Evidence = "Synthetic evidence."
	}
	verdicts[0].Evidence = " "
	_, err = svc.RecordExact(ctx, "evidence-review", current.RevisionID, current.Body, verdicts)
	require.ErrorContains(t, err, "evidence is required")
	verdicts[0] = review.Verdict{Mode: "what_without_why", Passed: false, Evidence: "A synthetic checklist found a gap."}
	_, err = svc.RecordExact(ctx, "evidence-review", current.RevisionID, current.Body, verdicts)
	require.ErrorContains(t, err, "finding is required")
	var runs int
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM review_runs`).Scan(&runs))
	require.Zero(t, runs)
}

func TestReviewRerunSupersedesWithoutDeletingPriorRun(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(posttypes.Schema), database.SchemaProviderFunc(review.Schema)))
	_, err := d.ExecContext(context.Background(), `INSERT INTO drafts (id, campaign_id, post_type_id, body, status, created_at, updated_at) VALUES ('draft-3', 'campaign-1', 'dev-log', '', 'reviewed', '2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z')`)
	require.NoError(t, err)
	s := review.NewService(d, testRevisionSource{repo: artifacts.NewSQLiteRepository(d)})
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

type testRevisionSource struct{ repo artifacts.Repository }

func (s testRevisionSource) CurrentRevision(ctx context.Context, draftID string) (review.RevisionSnapshot, error) {
	current, err := s.repo.GetCurrentRevision(ctx, draftID)
	if err != nil {
		return review.RevisionSnapshot{}, err
	}
	return review.RevisionSnapshot{DraftID: current.DraftID, Body: current.Body, RevisionID: current.RevisionID}, nil
}

func (s testRevisionSource) CurrentEditorialSnapshotTx(ctx context.Context, tx *sql.Tx, draftID string) (review.EditorialSnapshot, error) {
	var snap review.EditorialSnapshot
	var latestBody string
	var actorKind, capacity, revisionCreatedAt string
	err := tx.QueryRowContext(ctx, `SELECT d.id,d.body,COALESCE(r.id,''),COALESCE(r.body,''),COALESCE(sl.channel,''),COALESCE(sl.format,''),d.campaign_id,COALESCE(pr.project_id,''),d.post_type_id,d.lane,d.sku,d.scenario_name,COALESCE(r.actor_kind,''),COALESCE(r.capacity,''),COALESCE(r.created_at,'') FROM drafts d LEFT JOIN draft_slots sl ON sl.draft_id=d.id LEFT JOIN draft_project_refs pr ON pr.draft_id=d.id LEFT JOIN draft_revisions r ON r.id=(SELECT id FROM draft_revisions WHERE draft_id=d.id ORDER BY created_at DESC,id DESC LIMIT 1) WHERE d.id=?`, draftID).Scan(&snap.DraftID, &snap.Body, &snap.RevisionID, &latestBody, &snap.Context.Channel, &snap.Context.Format, &snap.Context.CampaignID, &snap.Context.ProjectID, &snap.Context.PostTypeID, &snap.Context.Lane, &snap.Context.SKU, &snap.Context.ScenarioName, &actorKind, &capacity, &revisionCreatedAt)
	if err != nil {
		return review.EditorialSnapshot{}, err
	}
	if snap.RevisionID != "" && latestBody == snap.Body {
		snap.RevisionProvenance = &review.RevisionProvenance{ActorKind: actorKind, Capacity: capacity, CreatedAt: revisionCreatedAt}
	}
	if latestBody != snap.Body {
		snap.RevisionID = ""
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,asset_id,role,aspect_ratio,alt_text,position FROM draft_attachments WHERE draft_id=? ORDER BY position,id`, draftID)
	if err != nil {
		return review.EditorialSnapshot{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var a review.EditorialAttachment
		if err := rows.Scan(&a.ID, &a.AssetID, &a.Role, &a.AspectRatio, &a.AltText, &a.Position); err != nil {
			return review.EditorialSnapshot{}, err
		}
		snap.Context.Attachments = append(snap.Context.Attachments, a)
	}
	return snap, rows.Err()
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

// [REQ:CONTENTD-P0-022] Wording packages are immutable and bound to the exact current body.
func TestEditorialPackageCapturesExactWordingAndRejectsStaleRevision(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(posttypes.Schema), database.SchemaProviderFunc(review.Schema)))
	_, err := d.ExecContext(ctx, `INSERT INTO drafts (id,campaign_id,post_type_id,body,status,created_at,updated_at) VALUES ('package-draft','','dev-log','original wording','draft','2026-07-01T00:00:00Z','2026-07-01T00:00:00Z')`)
	require.NoError(t, err)
	repo := artifacts.NewSQLiteRepository(d)
	_, err = d.ExecContext(ctx, `INSERT INTO draft_slots (draft_id,campaign_id,channel,format) VALUES ('package-draft','','linkedin','article')`)
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, `INSERT INTO draft_project_refs (draft_id,project_id) VALUES ('package-draft','project-1')`)
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, `INSERT INTO draft_attachments (id,draft_id,asset_id,role,aspect_ratio,alt_text,position,created_at) VALUES ('attachment-1','package-draft','asset-rev-1','hero','16:9','diagram caption',0,'2026-07-01T00:00:00Z')`)
	require.NoError(t, err)
	s := review.NewService(d, testRevisionSource{repo: repo})
	_, err = s.CreateEditorialPackage(ctx, "package-draft", "", "stale visible wording", "stale-visible-op")
	require.ErrorContains(t, err, "wording changed")

	first, err := s.CreateEditorialPackage(ctx, "package-draft", "", "original wording", "package-op-1")
	require.NoError(t, err)
	require.Equal(t, "original wording", first.Body)
	require.Equal(t, review.EditorialPackageScope, first.Scope)
	require.NotEmpty(t, first.BodySHA256)
	require.NotEmpty(t, first.PackageSHA256)
	require.Contains(t, first.ManifestJSON, `"channel":"linkedin"`)
	require.Contains(t, first.ManifestJSON, `"project_id":"project-1"`)
	require.Contains(t, first.ManifestJSON, `"asset_id":"asset-rev-1"`)
	require.Contains(t, first.ManifestJSON, `"alt_text":"diagram caption"`)
	require.Empty(t, first.SourceRevisionID, "a legacy draft has no stored revision id, so its body digest still binds the snapshot")
	replay, err := s.CreateEditorialPackage(ctx, "package-draft", "", "original wording", "package-op-1")
	require.NoError(t, err)
	require.Equal(t, first.ID, replay.ID, "same-key exact request returns the original package")

	_, err = repo.UpdateBody(ctx, "package-draft", "later wording")
	require.NoError(t, err)
	current, err := repo.GetCurrentRevision(ctx, "package-draft")
	require.NoError(t, err)
	require.NotEmpty(t, current.RevisionID)
	_, err = s.CreateEditorialPackage(ctx, "package-draft", current.RevisionID, "later wording", "package-op-1")
	require.ErrorContains(t, err, "reused with a different")
	replayAfterEdit, err := s.CreateEditorialPackage(ctx, "package-draft", "", "original wording", "package-op-1")
	require.NoError(t, err)
	require.Equal(t, first.ID, replayAfterEdit.ID)
	_, err = s.CreateEditorialPackage(ctx, "package-draft", "", "original wording", "package-op-2")
	require.ErrorContains(t, err, "revision changed")

	second, err := s.CreateEditorialPackage(ctx, "package-draft", current.RevisionID, "later wording", "package-op-3")
	require.NoError(t, err)
	require.Equal(t, "later wording", second.Body)
	require.Equal(t, current.RevisionID, second.SourceRevisionID)
	_, err = d.ExecContext(ctx, `UPDATE draft_attachments SET alt_text='changed metadata' WHERE id='attachment-1'`)
	require.NoError(t, err)
	third, err := s.CreateEditorialPackage(ctx, "package-draft", current.RevisionID, "later wording", "package-op-4")
	require.NoError(t, err)
	require.NotEqual(t, second.PackageSHA256, third.PackageSHA256, "the same body/revision with changed attachment context has a distinct package digest")
	oldManifest, err := s.GetEditorialPackage(ctx, first.ID)
	require.NoError(t, err)
	require.Contains(t, oldManifest.ManifestJSON, `"alt_text":"diagram caption"`, "immutable manifests retain the old attachment context")
	readBack, err := s.GetEditorialPackage(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, first.PackageSHA256, readBack.PackageSHA256)
	require.Equal(t, "original wording", readBack.Body, "later edits must not mutate an earlier package")

	_, err = d.ExecContext(ctx, `UPDATE editorial_review_package_manifests SET manifest_json='{}' WHERE package_id = ?`, first.ID)
	require.NoError(t, err)
	_, err = s.GetEditorialPackage(ctx, first.ID)
	require.ErrorContains(t, err, "digest check")
}

// [REQ:CONTENTD-P0-022] A package fingerprints the revision's persisted authoring provenance.
func TestEditorialPackageIncludesRevisionAuthoringProvenance(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(posttypes.Schema), database.SchemaProviderFunc(review.Schema)))
	_, err := d.ExecContext(ctx, `INSERT INTO drafts (id,campaign_id,post_type_id,body,status,created_at,updated_at) VALUES ('provenance-draft','','dev-log','exact body','draft','2026-07-01T00:00:00Z','2026-07-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, `INSERT INTO draft_revisions (id,draft_id,body,actor_kind,capacity,created_at) VALUES ('rev-provenance','provenance-draft','exact body','agent','author','2026-07-01T01:00:00Z')`)
	require.NoError(t, err)
	s := review.NewService(d, testRevisionSource{repo: artifacts.NewSQLiteRepository(d)})
	first, err := s.CreateEditorialPackage(ctx, "provenance-draft", "rev-provenance", "exact body", "provenance-op-1")
	require.NoError(t, err)
	require.Contains(t, first.ManifestJSON, `"revision_authoring":{"actor_kind":"agent","capacity":"author","created_at":"2026-07-01T01:00:00Z"}`)
	_, err = d.ExecContext(ctx, `UPDATE draft_revisions SET capacity='reviewer' WHERE id='rev-provenance'`)
	require.NoError(t, err)
	second, err := s.CreateEditorialPackage(ctx, "provenance-draft", "rev-provenance", "exact body", "provenance-op-2")
	require.NoError(t, err)
	require.NotEqual(t, first.PackageSHA256, second.PackageSHA256, "changed persisted revision provenance must change the package digest")
	readBack, err := s.GetEditorialPackage(ctx, first.ID)
	require.NoError(t, err)
	require.Contains(t, readBack.ManifestJSON, `"capacity":"author"`, "later metadata changes cannot rewrite prior package context")
}

func TestEditorialPackageConcurrentSameKeyReturnsOneReceipt(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(posttypes.Schema), database.SchemaProviderFunc(review.Schema)))
	_, err := d.ExecContext(ctx, `INSERT INTO drafts (id,campaign_id,post_type_id,body,status,created_at,updated_at) VALUES ('concurrent-package','','dev-log','same body','draft','2026-07-01T00:00:00Z','2026-07-01T00:00:00Z')`)
	require.NoError(t, err)
	repo := artifacts.NewSQLiteRepository(d)
	s := review.NewService(d, testRevisionSource{repo: repo})
	const callers = 12
	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make([]review.EditorialPackage, callers)
	errors := make([]error, callers)
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			results[index], errors[index] = s.CreateEditorialPackage(ctx, "concurrent-package", "", "same body", "same-concurrent-key")
		}(i)
	}
	close(start)
	wg.Wait()
	for _, err := range errors {
		require.NoError(t, err)
	}
	for _, result := range results {
		require.Equal(t, results[0].ID, result.ID)
	}
	var count int
	require.NoError(t, d.QueryRowContext(ctx, `SELECT count(*) FROM editorial_review_packages WHERE draft_id = 'concurrent-package'`).Scan(&count))
	require.Equal(t, 1, count)
}

type revisionMutatingSource struct{ repo artifacts.Repository }

func (s revisionMutatingSource) CurrentRevision(ctx context.Context, draftID string) (review.RevisionSnapshot, error) {
	current, err := s.repo.GetCurrentRevision(ctx, draftID)
	if err != nil {
		return review.RevisionSnapshot{}, err
	}
	if _, err := s.repo.UpdateBody(ctx, draftID, "racing edit after snapshot read"); err != nil {
		return review.RevisionSnapshot{}, err
	}
	return review.RevisionSnapshot{DraftID: current.DraftID, Body: current.Body, RevisionID: current.RevisionID}, nil
}
func (s revisionMutatingSource) CurrentEditorialSnapshotTx(ctx context.Context, tx *sql.Tx, draftID string) (review.EditorialSnapshot, error) {
	return testRevisionSource{repo: s.repo}.CurrentEditorialSnapshotTx(ctx, tx, draftID)
}

func TestEditorialPackageRechecksBodyAndRevisionInsideAdmissionTransaction(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(posttypes.Schema), database.SchemaProviderFunc(review.Schema)))
	_, err := d.ExecContext(ctx, `INSERT INTO drafts (id,campaign_id,post_type_id,body,status,created_at,updated_at) VALUES ('racing-draft','','dev-log','initial exact body','draft','2026-07-01T00:00:00Z','2026-07-01T00:00:00Z')`)
	require.NoError(t, err)
	repo := artifacts.NewSQLiteRepository(d)
	svc := review.NewService(d, revisionMutatingSource{repo})
	_, err = svc.CreateEditorialPackage(ctx, "racing-draft", "", "initial exact body", "racing-package-key")
	require.ErrorContains(t, err, "changed before package admission")
	var packages, ops int
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM editorial_review_packages`).Scan(&packages))
	require.Zero(t, packages)
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM editorial_review_package_operations`).Scan(&ops))
	require.Zero(t, ops)
}
