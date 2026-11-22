package intents_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"content-desk/internal/artifacts"
	localdb "content-desk/internal/database"
	"content-desk/internal/intents"
	"content-desk/internal/review"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

type source struct{ repo artifacts.Repository }

func (s source) CurrentRevision(ctx context.Context, id string) (intents.CurrentRevision, error) {
	r, e := s.repo.GetCurrentRevision(ctx, id)
	if e != nil {
		return intents.CurrentRevision{}, e
	}
	return intents.CurrentRevision{DraftID: r.DraftID, Body: r.Body, RevisionID: r.RevisionID}, nil
}
func setup(t *testing.T) (context.Context, *sql.DB, artifacts.Repository, intents.Service) {
	t.Helper()
	ctx := context.Background()
	d := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(intents.Schema), database.SchemaProviderFunc(review.Schema)))
	_, err := d.ExecContext(ctx, `INSERT INTO drafts (id,campaign_id,post_type_id,body,status,created_at,updated_at) VALUES ('d1','','dev-log','exact words','draft','2026-07-01T00:00:00Z','2026-07-01T00:00:00Z')`)
	require.NoError(t, err)
	repo := artifacts.NewSQLiteRepository(d)
	return ctx, d, repo, intents.NewService(d, source{repo})
}
func input(rev string) intents.SetInput {
	return intents.SetInput{DraftID: "d1", Destination: "linkedin", DesiredLocalDate: "2026-12-01", TimeZone: "America/New_York", Daypart: "morning", ExpectedRevisionID: rev, ExpectedBody: "exact words", ExpectedIntentVersion: 0, IdempotencyKey: "intent-1"}
}
func TestSetIntentBindsDateAndDisplayedRevisionAndReplaysImmutableOperation(t *testing.T) {
	ctx, d, repo, s := setup(t)
	current, err := repo.GetCurrentRevision(ctx, "d1")
	require.NoError(t, err)
	stale := input(current.RevisionID)
	stale.ExpectedBody = "changed on screen"
	_, err = s.Set(ctx, stale)
	require.ErrorContains(t, err, "draft changed")
	first, err := s.Set(ctx, input(current.RevisionID))
	require.NoError(t, err)
	require.Equal(t, "2026-12-01", first.DesiredLocalDate)
	require.Equal(t, "morning", first.Daypart)
	require.Equal(t, "desired", first.Status)
	require.Equal(t, int64(1), first.Version)
	replay, err := s.Set(ctx, input(current.RevisionID))
	require.NoError(t, err)
	require.Equal(t, first.ID, replay.ID)
	update := input(current.RevisionID)
	update.ExpectedIntentVersion = 1
	update.IdempotencyKey = "intent-2"
	update.DesiredLocalDate = "2026-12-03"
	update.Daypart = "evening"
	second, err := s.Set(ctx, update)
	require.NoError(t, err)
	require.Equal(t, int64(2), second.Version)
	replay, err = s.Set(ctx, input(current.RevisionID))
	require.NoError(t, err)
	require.Equal(t, first.DesiredLocalDate, replay.DesiredLocalDate, "replay returns original immutable revision")
	conflict := update
	conflict.DesiredLocalDate = "2026-12-04"
	_, err = s.Set(ctx, conflict)
	require.ErrorContains(t, err, "reused with a different")
	staleVersion := update
	staleVersion.IdempotencyKey = "intent-3"
	_, err = s.Set(ctx, staleVersion)
	require.ErrorContains(t, err, "changed")
	history, err := s.List(ctx, "d1", "linkedin")
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, "2026-12-03", history[0].DesiredLocalDate)
	_, err = d.ExecContext(ctx, `UPDATE drafts SET body='next words' WHERE id='d1'`)
	require.NoError(t, err)
	latest, err := repo.GetCurrentRevision(ctx, "d1")
	require.NoError(t, err)
	changed := input(latest.RevisionID)
	changed.ExpectedBody = "exact words"
	changed.IdempotencyKey = "intent-4"
	changed.ExpectedIntentVersion = 2
	_, err = s.Set(ctx, changed)
	require.ErrorContains(t, err, "draft changed")
}
func TestSetIntentRejectsInvalidDateZoneAndDaypart(t *testing.T) {
	ctx, _, repo, s := setup(t)
	current, err := repo.GetCurrentRevision(ctx, "d1")
	require.NoError(t, err)
	for _, mutate := range []func(*intents.SetInput){func(x *intents.SetInput) { x.DesiredLocalDate = "2026-2-1" }, func(x *intents.SetInput) { x.TimeZone = "Mars/Olympus" }, func(x *intents.SetInput) { x.Daypart = "night" }} {
		x := input(current.RevisionID)
		mutate(&x)
		_, err = s.Set(ctx, x)
		require.Error(t, err)
	}
}

func TestConcurrentSameKeyConvergesOnOneIntentRevision(t *testing.T) {
	ctx, d, repo, svc := setup(t)
	current, err := repo.GetCurrentRevision(ctx, "d1")
	require.NoError(t, err)
	const n = 12
	start := make(chan struct{})
	got := make(chan intents.PublicationIntent, n)
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			item, err := svc.Set(ctx, input(current.RevisionID))
			if err != nil {
				errs <- err
				return
			}
			got <- item
		}()
	}
	close(start)
	wg.Wait()
	close(got)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var first string
	for item := range got {
		if first == "" {
			first = item.ID
		}
		require.Equal(t, first, item.ID)
	}
	var count int
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM publication_intent_revisions`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM publication_intent_operations`).Scan(&count))
	require.Equal(t, 1, count)
}
