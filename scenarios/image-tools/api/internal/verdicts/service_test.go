package verdicts

import (
	"context"
	"errors"
	"fmt"
	"testing"

	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"

	"image-tools/internal/measures"
)

func newService(t *testing.T) (*Service, SQLExecutor) {
	t.Helper()
	d := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(measures.Schema), apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	svc, err := NewService(d)
	if err != nil {
		t.Fatal(err)
	}
	return svc, d
}

func trace(t *testing.T, d SQLExecutor, job, op, model, state string, at int) {
	t.Helper()
	if _, err := d.ExecContext(context.Background(),
		`INSERT INTO job_trace (job_id, operation, model_id, state, recorded_at) VALUES (?,?,?,?,?)`,
		job, op, model, state, "2026-11-21T00:00:"+twoDigits(at)+"Z"); err != nil {
		t.Fatal(err)
	}
}

func twoDigits(n int) string {
	return string(rune('0'+n/10%10)) + string(rune('0'+n%10))
}

func TestRateRequiresASucceededTracedJob(t *testing.T) {
	ctx := context.Background()
	svc, d := newService(t)
	trace(t, d, "ok", "edit_instruct", "flux-2-klein-4b", "succeeded", 1)
	trace(t, d, "bad", "edit_instruct", "flux-2-klein-4b", "failed", 2)

	if _, err := svc.Ratings.Rate(ctx, "missing", 4, "", "t"); !errors.Is(err, ErrJobUnknown) {
		t.Fatalf("unknown job: %v", err)
	}
	if _, err := svc.Ratings.Rate(ctx, "bad", 4, "", "t"); !errors.Is(err, ErrJobNotRateable) {
		t.Fatalf("failed job: %v", err)
	}
	for _, score := range []int{0, 6, -1} {
		if _, err := svc.Ratings.Rate(ctx, "ok", score, "", "t"); !errors.Is(err, ErrBadScore) {
			t.Fatalf("score %d: %v", score, err)
		}
	}
	long := make([]rune, MaxNoteLength+1)
	for i := range long {
		long[i] = 'x'
	}
	if _, err := svc.Ratings.Rate(ctx, "ok", 3, string(long), "t"); !errors.Is(err, ErrNoteTooLong) {
		t.Fatalf("long note: %v", err)
	}

	r, err := svc.Ratings.Rate(ctx, "ok", 5, "  kept the face  ", "operator")
	if err != nil {
		t.Fatal(err)
	}
	if r.ModelID != "flux-2-klein-4b" || r.Operation != "edit_instruct" || r.Note != "kept the face" {
		t.Fatalf("rating = %+v", r)
	}
	// Re-rating replaces.
	if _, err := svc.Ratings.Rate(ctx, "ok", 2, "", "operator"); err != nil {
		t.Fatal(err)
	}
	got, ok, err := svc.Ratings.Get(ctx, "ok")
	if err != nil || !ok || got.Score != 2 || got.Note != "" {
		t.Fatalf("after re-rate: %+v ok=%t err=%v", got, ok, err)
	}
	if err := svc.Ratings.Clear(ctx, "ok"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Ratings.Clear(ctx, "ok"); !errors.Is(err, ErrNotRated) {
		t.Fatalf("second clear: %v", err)
	}
}

// TestRatingsFeedVerdicts proves the loop end to end: ratings written through
// the store change the derived verdict of the model they were given to.
func TestRatingsFeedVerdicts(t *testing.T) {
	ctx := context.Background()
	svc, d := newService(t)
	// A model with no curated claim and no job history other than three rated runs.
	for i := 1; i <= 3; i++ {
		id := "j" + twoDigits(i)
		trace(t, d, id, "upscale", "some-new-model", "succeeded", i)
	}
	board, err := svc.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v := board.For("some-new-model", "upscale"); v.Grade != Works {
		t.Fatalf("three successful jobs: %s (%s)", v.Grade, v.Reason)
	}
	for i := 1; i <= 3; i++ {
		if _, err := svc.Ratings.Rate(ctx, "j"+twoDigits(i), 1, "soft and smeared", "operator"); err != nil {
			t.Fatal(err)
		}
	}
	board, err = svc.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v := board.For("some-new-model", "upscale")
	if v.Grade != NotRecommended {
		t.Fatalf("three 1-star ratings: %s (%s)", v.Grade, v.Reason)
	}
	if per, roll := board.ForModel("some-new-model", []string{"upscale", "denoise"}); len(per) != 2 || roll.Grade != NotRecommended {
		t.Fatalf("per-op %v roll-up %v", per, roll)
	}
	// Another model's ratings and an op the model does not serve do not leak.
	if v := board.For("other-model", "upscale"); v.Grade != Untested {
		t.Fatalf("other model: %s", v.Grade)
	}
}

func TestJobHistoryIsRecentAndIgnoresCancels(t *testing.T) {
	ctx := context.Background()
	svc, d := newService(t)
	// 30 old failures then 20 recent successes: the old failures fall out of the window.
	for i := 0; i < 30; i++ {
		trace(t, d, "old"+twoDigits(i), "text_to_image", "m", "failed", 0)
	}
	for i := 0; i < RecentJobs; i++ {
		trace(t, d, "new"+twoDigits(i), "text_to_image", "m", "succeeded", 30+i%30)
	}
	trace(t, d, "cancelled", "text_to_image", "m", "canceled", 59)
	if _, err := d.ExecContext(ctx, `UPDATE job_trace SET recorded_at='2026-11-20T00:00:00Z' WHERE job_id LIKE 'old%'`); err != nil {
		t.Fatal(err)
	}
	board, err := svc.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v := board.For("m", "text_to_image")
	if v.Grade != Works {
		t.Fatalf("grade = %s (%s)", v.Grade, v.Reason)
	}
}

// A refusal, a request the model cannot serve as given and a restart say nothing
// about whether the model works; a fault does, and a record from before the
// kind existed still counts as a fault.
func TestOnlyFaultsLowerTheJobHistoryOfAModel(t *testing.T) {
	ctx := context.Background()
	svc, d := newService(t)
	add := func(job, state, kind string, at int) {
		t.Helper()
		if _, err := d.ExecContext(ctx,
			`INSERT INTO job_trace (job_id, operation, model_id, state, failure_kind, recorded_at) VALUES (?,?,?,?,?,?)`,
			job, "segment", "segformer-b3-fashion", state, kind, "2026-11-21T00:00:"+twoDigits(at)+"Z"); err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= 3; i++ {
		add(fmt.Sprintf("ok%d", i), "succeeded", "", i)
	}
	add("no-label-1", "failed", "invalid_input", 4)
	add("no-label-2", "failed", "invalid_input", 5)
	add("consent", "failed", "refused", 6)
	add("restart", "failed", "interrupted", 7)
	board, err := svc.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := board.signals[key{"segformer-b3-fashion", "segment"}].Jobs; got.Succeeded != 3 || got.Failed != 0 {
		t.Fatalf("history = %+v, want 3 succeeded and no failures counted", got)
	}
	add("crash", "failed", "fault", 8)
	add("legacy", "failed", "", 9)
	board, err = svc.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := board.signals[key{"segformer-b3-fashion", "segment"}].Jobs; got.Succeeded != 3 || got.Failed != 2 {
		t.Fatalf("history = %+v, want the fault and the legacy record counted", got)
	}
}
