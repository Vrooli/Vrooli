package safety

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"connectrpc.com/connect"
	internaljobs "image-tools/internal/jobs"
	internalsafety "image-tools/internal/safety"

	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"google.golang.org/protobuf/types/known/timestamppb"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety"
)

type fakeJobs map[string]internaljobs.Job

func (f fakeJobs) Get(_ context.Context, id string) (internaljobs.Job, error) {
	job, ok := f[id]
	if !ok {
		return internaljobs.Job{}, internaljobs.ErrNotFound
	}
	return job, nil
}

func newFeedbackHandler(t *testing.T, jobs JobSource) *connectHandler {
	t.Helper()
	db := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(internalsafety.Schema)); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return NewConnectHandlerWithDeps(Deps{
		Policy:   internalsafety.PolicyFor(internalsafety.TierLocal),
		Feedback: internalsafety.NewFeedbackStore(db),
		Jobs:     jobs,
	})
}

func refusedJob(id string) internaljobs.Job {
	verdict := internalsafety.MinorVerdict{
		Action: internalsafety.ActionRefuse, ReasonCode: internalsafety.ReasonSexualMinorFace,
		Minor: true, Sexual: true,
		Images: []internalsafety.MinorImageFinding{{
			NSFWScore: 0.96, Sexual: true, Action: internalsafety.ActionRefuse,
			ReasonCode: internalsafety.ReasonSexualMinorFace, MinorFace: true, Refuse: true,
			FacesFound: 1, AssessedFaces: 1, MaxPUnder20: 0.91,
		}},
		ProfileName: "local-balanced", ReviewerVersion: "2026-11-23",
	}
	evidence, _ := json.Marshal(verdict)
	return internaljobs.Job{
		ID: id, Operation: "text_to_image",
		Payload: []byte(`{"model_id":"flux-2-klein-4b"}`),
		Meta: map[string]string{
			internaljobs.MetaSafetyReview:       "refused",
			internaljobs.MetaSafetyAction:       "refuse",
			internaljobs.MetaSafetyReason:       "sexual_minor_face",
			internaljobs.MetaSafetyEvidence:     string(evidence),
			internaljobs.MetaSafetyReviewedRefs: "out/a.png",
			internaljobs.MetaSafetyRefusedRefs:  "out/a.png",
			internaljobs.MetaNSFWScore:          "0.9600",
		},
	}
}

func TestRecordFeedbackSnapshotsTheReviewerEvidence(t *testing.T) {
	handler := newFeedbackHandler(t, fakeJobs{"job-1": refusedJob("job-1")})
	resp, err := handler.RecordSafetyFeedback(context.Background(), connect.NewRequest(&safetyv1.RecordSafetyFeedbackRequest{
		JobId: "job-1", Label: safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_FALSE_POSITIVE,
		Note: "the subject is an adult", Actor: "operator",
	}))
	if err != nil {
		t.Fatal(err)
	}
	item := resp.Msg
	if item.GetLabel() != safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_FALSE_POSITIVE {
		t.Fatalf("label = %v", item.GetLabel())
	}
	if item.GetReasonCode() != "sexual_minor_face" || item.GetModelId() != "flux-2-klein-4b" {
		t.Fatalf("reason/model = %q/%q", item.GetReasonCode(), item.GetModelId())
	}
	if item.GetProfileName() != "local-balanced" || item.GetReviewerVersion() != "2026-11-23" {
		t.Fatalf("profile/version = %q/%q", item.GetProfileName(), item.GetReviewerVersion())
	}
	if item.GetEvidenceJson() == "" {
		t.Fatal("evidence was not snapshotted")
	}
	if item.GetRecordedAt() == nil {
		t.Fatal("recorded_at is unset")
	}

	perRef, err := handler.RecordSafetyFeedback(context.Background(), connect.NewRequest(&safetyv1.RecordSafetyFeedbackRequest{
		JobId: "job-1", OutputRef: "out/a.png",
		Label: safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_CONFIRMED,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if perRef.Msg.GetOutputRef() != "out/a.png" || perRef.Msg.GetFindingJson() == "" {
		t.Fatalf("per-ref case = %+v", perRef.Msg)
	}

	_, err = handler.RecordSafetyFeedback(context.Background(), connect.NewRequest(&safetyv1.RecordSafetyFeedbackRequest{
		JobId: "job-1", OutputRef: "out/other.png",
		Label: safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_CONFIRMED,
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unknown ref code = %v, want failed_precondition", connect.CodeOf(err))
	}
}

func TestRecordFeedbackRejectsAJobWithNoVerdictAndABadLabel(t *testing.T) {
	clean := internaljobs.Job{ID: "job-2", Operation: "text_to_image", Meta: map[string]string{
		internaljobs.MetaSafetyReview: "passed", internaljobs.MetaSafetyAction: "allow",
		internaljobs.MetaSafetyEvidence: `{"action":"allow","reason_code":"clean"}`,
	}}
	handler := newFeedbackHandler(t, fakeJobs{"job-2": clean})
	_, err := handler.RecordSafetyFeedback(context.Background(), connect.NewRequest(&safetyv1.RecordSafetyFeedbackRequest{
		JobId: "job-2", Label: safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_FALSE_POSITIVE,
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("clean job code = %v, want failed_precondition", connect.CodeOf(err))
	}
	_, err = handler.RecordSafetyFeedback(context.Background(), connect.NewRequest(&safetyv1.RecordSafetyFeedbackRequest{
		JobId: "job-2", Label: safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_UNSPECIFIED,
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unspecified label code = %v, want invalid_argument", connect.CodeOf(err))
	}
	_, err = handler.RecordSafetyFeedback(context.Background(), connect.NewRequest(&safetyv1.RecordSafetyFeedbackRequest{
		JobId: "missing", Label: safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_CONFIRMED,
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing job code = %v, want not_found", connect.CodeOf(err))
	}
}

func TestFeedbackListStatsExportAndClear(t *testing.T) {
	jobs := fakeJobs{
		"job-1": refusedJob("job-1"),
		"job-2": refusedJob("job-2"),
	}
	handler := newFeedbackHandler(t, jobs)
	ctx := context.Background()
	for jobID, label := range map[string]safetyv1.SafetyFeedbackLabel{
		"job-1": safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_FALSE_POSITIVE,
		"job-2": safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_CONFIRMED,
	} {
		if _, err := handler.RecordSafetyFeedback(ctx, connect.NewRequest(&safetyv1.RecordSafetyFeedbackRequest{
			JobId: jobID, Label: label,
		})); err != nil {
			t.Fatal(err)
		}
	}

	list, err := handler.ListSafetyFeedback(ctx, connect.NewRequest(&safetyv1.ListSafetyFeedbackRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if list.Msg.GetTotal() != 2 || len(list.Msg.GetCases()) != 2 {
		t.Fatalf("list total/cases = %d/%d", list.Msg.GetTotal(), len(list.Msg.GetCases()))
	}
	filtered, err := handler.ListSafetyFeedback(ctx, connect.NewRequest(&safetyv1.ListSafetyFeedbackRequest{
		Label: safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_CONFIRMED,
	}))
	if err != nil || filtered.Msg.GetTotal() != 1 {
		t.Fatalf("filtered = %v/%v", filtered, err)
	}

	stats, err := handler.GetSafetyFeedbackStats(ctx, connect.NewRequest(&safetyv1.GetSafetyFeedbackStatsRequest{
		Since: timestamppb.New(time.Now().Add(time.Hour)),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if stats.Msg.GetTotal() != 0 {
		t.Fatalf("future-since total = %d, want 0", stats.Msg.GetTotal())
	}
	stats, err = handler.GetSafetyFeedbackStats(ctx, connect.NewRequest(&safetyv1.GetSafetyFeedbackStatsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if stats.Msg.GetFalsePositives() != 1 || stats.Msg.GetConfirmed() != 1 {
		t.Fatalf("stats = %+v", stats.Msg)
	}

	corpus, err := handler.ExportSafetyCorpus(ctx, connect.NewRequest(&safetyv1.ExportSafetyCorpusRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if corpus.Msg.GetCases() != 2 || corpus.Msg.GetJsonl() == "" {
		t.Fatalf("corpus cases/jsonl = %d/%q", corpus.Msg.GetCases(), corpus.Msg.GetJsonl())
	}

	cleared, err := handler.ClearSafetyFeedback(ctx, connect.NewRequest(&safetyv1.ClearSafetyFeedbackRequest{JobId: "job-1"}))
	if err != nil || !cleared.Msg.GetCleared() {
		t.Fatalf("clear = %v/%v", cleared, err)
	}
}
