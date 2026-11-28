package jobs

import (
	"context"
	"log"
	"testing"
	"time"

	"connectrpc.com/connect"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"

	internaljobs "image-tools/internal/jobs"
	internalmeasures "image-tools/internal/measures"
	"image-tools/internal/verdicts"

	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
)

func newRatingHandler(t *testing.T, jobs ...internaljobs.Job) *connectHandler {
	t.Helper()
	d := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(internalmeasures.Schema), apidb.SchemaProviderFunc(verdicts.Schema)); err != nil {
		t.Fatal(err)
	}
	for _, j := range jobs {
		state := "succeeded"
		if j.State == internaljobs.StateFailed {
			state = "failed"
		}
		if _, err := d.ExecContext(context.Background(), `INSERT INTO job_trace (job_id, operation, model_id, state, recorded_at) VALUES (?,?,?,?,?)`,
			j.ID, j.Operation, j.Meta["model"], state, time.Now().UTC().Format(time.RFC3339)); err != nil {
			t.Fatal(err)
		}
	}
	m := &fakeManager{list: jobs}
	if len(jobs) > 0 {
		m.job = jobs[0]
	}
	return NewConnectHandler(Deps{Manager: m, Ratings: verdicts.NewRatings(d), Logger: log.New(testWriter{}, "", 0)})
}

func succeededJob(id, model string) internaljobs.Job {
	return internaljobs.Job{ID: id, Operation: "edit_instruct", State: internaljobs.StateSucceeded, Meta: map[string]string{"model": model}}
}

// TestRateJobThroughTheHandler covers the wire contract: rate, read it back on
// the job and in the list, replace it, filter the ratings, clear it, and get an
// actionable status for every refusal.
func TestRateJobThroughTheHandler(t *testing.T) {
	ctx := context.Background()
	failed := succeededJob("bad", "flux-2-klein-4b")
	failed.State = internaljobs.StateFailed
	h := newRatingHandler(t, succeededJob("ok", "flux-2-klein-4b"), succeededJob("other", "sd-1.5"), failed)

	rated, err := h.RateJob(ctx, connect.NewRequest(&jobsv1.RateJobRequest{JobId: "ok", Score: 4, Note: "kept the face", Actor: "operator"}))
	if err != nil {
		t.Fatal(err)
	}
	if r := rated.Msg.Rating; r.Score != 4 || r.ModelId != "flux-2-klein-4b" || r.Operation != "edit_instruct" || r.RatedBy != "operator" || r.RatedAt == nil {
		t.Fatalf("rating = %+v", r)
	}
	got, err := h.GetJob(ctx, connect.NewRequest(&jobsv1.GetJobRequest{Id: "ok"}))
	if err != nil || got.Msg.Job.Rating.GetScore() != 4 || got.Msg.Job.Rating.GetNote() != "kept the face" {
		t.Fatalf("GetJob rating = %+v (%v)", got.Msg.Job.GetRating(), err)
	}
	list, err := h.ListJobs(ctx, connect.NewRequest(&jobsv1.ListJobsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range list.Msg.Jobs {
		if (j.Id == "ok") != (j.Rating != nil) {
			t.Fatalf("job %s rating = %+v", j.Id, j.Rating)
		}
	}

	if _, err := h.RateJob(ctx, connect.NewRequest(&jobsv1.RateJobRequest{JobId: "ok", Score: 2})); err != nil {
		t.Fatal(err)
	}
	byModel, err := h.ListJobRatings(ctx, connect.NewRequest(&jobsv1.ListJobRatingsRequest{ModelId: "flux-2-klein-4b"}))
	if err != nil || len(byModel.Msg.Ratings) != 1 || byModel.Msg.Ratings[0].Score != 2 || byModel.Msg.Ratings[0].RatedBy != "api" {
		t.Fatalf("ratings = %+v (%v)", byModel.Msg.GetRatings(), err)
	}
	if none, _ := h.ListJobRatings(ctx, connect.NewRequest(&jobsv1.ListJobRatingsRequest{ModelId: "sd-1.5"})); len(none.Msg.Ratings) != 0 {
		t.Fatalf("sd-1.5 has no rating: %+v", none.Msg.Ratings)
	}

	for name, tc := range map[string]struct {
		req  *jobsv1.RateJobRequest
		code connect.Code
	}{
		"unknown job":   {&jobsv1.RateJobRequest{JobId: "nope", Score: 3}, connect.CodeNotFound},
		"failed job":    {&jobsv1.RateJobRequest{JobId: "bad", Score: 3}, connect.CodeFailedPrecondition},
		"score too low": {&jobsv1.RateJobRequest{JobId: "ok", Score: 0}, connect.CodeInvalidArgument},
		"score too big": {&jobsv1.RateJobRequest{JobId: "ok", Score: 6}, connect.CodeInvalidArgument},
	} {
		if _, err := h.RateJob(ctx, connect.NewRequest(tc.req)); connect.CodeOf(err) != tc.code {
			t.Errorf("%s: code = %v (%v), want %v", name, connect.CodeOf(err), err, tc.code)
		}
	}

	if _, err := h.ClearJobRating(ctx, connect.NewRequest(&jobsv1.ClearJobRatingRequest{JobId: "ok"})); err != nil {
		t.Fatal(err)
	}
	if _, err := h.ClearJobRating(ctx, connect.NewRequest(&jobsv1.ClearJobRatingRequest{JobId: "ok"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("second clear: %v", err)
	}
	if got, _ := h.GetJob(ctx, connect.NewRequest(&jobsv1.GetJobRequest{Id: "ok"})); got.Msg.Job.Rating != nil {
		t.Fatalf("cleared rating still on the job: %+v", got.Msg.Job.Rating)
	}
}

func TestRatingVerbsAreUnimplementedWithoutAStore(t *testing.T) {
	h := newHandler(&fakeManager{})
	if _, err := h.RateJob(context.Background(), connect.NewRequest(&jobsv1.RateJobRequest{JobId: "j", Score: 3})); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("RateJob: %v", err)
	}
	if _, err := h.ListJobRatings(context.Background(), connect.NewRequest(&jobsv1.ListJobRatingsRequest{})); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("ListJobRatings: %v", err)
	}
}
