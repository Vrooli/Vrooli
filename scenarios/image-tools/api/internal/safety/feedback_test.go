package safety

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func TestBuildFeedbackCaseMapsAPerVariationFinding(t *testing.T) {
	verdict := MinorVerdict{
		Action: ActionRefuse, ReasonCode: ReasonSexualMinorFace, Minor: true, Sexual: true,
		Images: []MinorImageFinding{
			{
				NSFWScore: 0.97, Sexual: true, Action: ActionRefuse, ReasonCode: ReasonSexualMinorFace, FacesFound: 1, AssessedFaces: 1, MaxPUnder20: 0.92, MinorFace: true, Refuse: true,
				Faces: []MinorFaceFinding{{SizePx: 120, PUnder20: 0.92}},
			},
			{NSFWScore: 0.81, Sexual: true, Action: ActionFlag, ReasonCode: ReasonSexualAdult, FacesFound: 1, AssessedFaces: 1, MaxPUnder20: 0.01},
		},
		Thresholds: map[string]float64{"falconsai-nsfw": 0.3, "age": 0.2},
		Models:     []string{"falconsai-nsfw", "adamcodd-vit-nsfw", "yunet", "fairface-vit-age"},
		Strict:     true, ProfileName: "local-balanced", ReviewerVersion: "rev-1",
	}
	evidence, err := json.Marshal(verdict)
	if err != nil {
		t.Fatal(err)
	}
	observation := FeedbackObservation{
		JobID: "job-1", OutputRef: "out/b.png", Operation: "text_to_image", ModelID: "flux",
		Action: "refuse", ReasonCode: "sexual_minor_face", SafetyReview: "partial_refused", Score: 0.97,
		Evidence:     string(evidence),
		ReviewedRefs: []string{"out/a.png", "out/b.png"},
		RefusedRefs:  []string{"out/a.png"},
	}
	item, err := BuildFeedbackCase(observation, FeedbackFalsePositive, "  the second face is an adult  ", "operator")
	if err != nil {
		t.Fatal(err)
	}
	if item.Action != string(ActionFlag) || item.ReasonCode != string(ReasonSexualAdult) {
		t.Fatalf("variation action/reason = %q/%q", item.Action, item.ReasonCode)
	}
	if item.Score != 0.81 {
		t.Fatalf("score = %v, want the variation's 0.81", item.Score)
	}
	if item.ProfileName != "local-balanced" || item.ReviewerVersion != "rev-1" {
		t.Fatalf("profile/version = %q/%q", item.ProfileName, item.ReviewerVersion)
	}
	if !strings.Contains(item.Finding, `"p_under_20":0.01`) && !strings.Contains(item.Finding, `"nsfw_score":0.81`) {
		t.Fatalf("finding did not snapshot the variation: %s", item.Finding)
	}
	if item.Note != "the second face is an adult" || item.Actor != "operator" {
		t.Fatalf("note/actor = %q/%q", item.Note, item.Actor)
	}
}

func TestBuildFeedbackCaseJobLevelSnapshotsTheRefusedFinding(t *testing.T) {
	verdict := MinorVerdict{
		Action: ActionRefuse, ReasonCode: ReasonSexualMinorFace, Minor: true, Sexual: true,
		Images: []MinorImageFinding{
			{NSFWScore: 0.12, Sexual: false, Action: ActionAllow, ReasonCode: ReasonClean},
			{NSFWScore: 0.97, Sexual: true, Action: ActionRefuse, ReasonCode: ReasonSexualMinorFace, FacesFound: 1, AssessedFaces: 1, MaxPUnder20: 0.9, MinorFace: true, Refuse: true},
		},
	}
	evidence, _ := json.Marshal(verdict)
	item, err := BuildFeedbackCase(FeedbackObservation{
		JobID: "job", Action: "refuse", ReasonCode: "sexual_minor_face", SafetyReview: "refused",
		Evidence: string(evidence), ReviewedRefs: []string{"out/a.png", "out/b.png"}, RefusedRefs: []string{"out/b.png"},
	}, FeedbackConfirmed, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if item.OutputRef != "" {
		t.Fatalf("job-level case named a ref: %q", item.OutputRef)
	}
	if item.Finding == "" || !strings.Contains(item.Finding, `"nsfw_score":0.97`) {
		t.Fatalf("job-level finding did not snapshot the refused image: %s", item.Finding)
	}
	if item.Score != 0.97 {
		t.Fatalf("score = %v, want the refused image's 0.97", item.Score)
	}
}

func TestBuildFeedbackCaseRejectsUnsafeInputs(t *testing.T) {
	evidence, _ := json.Marshal(MinorVerdict{Action: ActionAllow, ReasonCode: ReasonClean})
	base := FeedbackObservation{
		JobID: "job", SafetyReview: "passed", Action: "allow", Evidence: string(evidence),
		ReviewedRefs: []string{"out/a.png"},
	}
	if _, err := BuildFeedbackCase(base, FeedbackFalsePositive, "", ""); !errors.Is(err, ErrNoSafetyVerdict) {
		t.Fatalf("clean job: %v", err)
	}
	refused := base
	refused.SafetyReview = "refused"
	refused.Action = "refuse"
	if _, err := BuildFeedbackCase(refused, FeedbackFalsePositive, "", ""); err != nil {
		t.Fatalf("aggregate-only refusal should be labelable: %v", err)
	}
	withRef := refused
	withRef.OutputRef = "out/other.png"
	if _, err := BuildFeedbackCase(withRef, FeedbackFalsePositive, "", ""); !errors.Is(err, ErrFeedbackRefAbsent) {
		t.Fatalf("unknown ref: %v", err)
	}
	if _, err := BuildFeedbackCase(refused, "legitimate", "", ""); !errors.Is(err, ErrFeedbackLabel) {
		t.Fatalf("bad label: %v", err)
	}
	if _, err := BuildFeedbackCase(refused, FeedbackConfirmed, strings.Repeat("x", FeedbackNoteLimit+1), ""); !errors.Is(err, ErrFeedbackNote) {
		t.Fatalf("long note: %v", err)
	}
	noEvidence := refused
	noEvidence.Evidence = ""
	if _, err := BuildFeedbackCase(noEvidence, FeedbackConfirmed, "", ""); !errors.Is(err, ErrFeedbackNoEvidence) {
		t.Fatalf("no evidence: %v", err)
	}
}

func TestFeedbackStoreUpsertsStatsAndExports(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	store := NewFeedbackStore(db)

	refused := MinorVerdict{
		Action: ActionRefuse, ReasonCode: ReasonSexualMinorFace, Minor: true, Sexual: true,
		Images:      []MinorImageFinding{{NSFWScore: 0.9, Sexual: true, Action: ActionRefuse, ReasonCode: ReasonSexualMinorFace, MinorFace: true, Refuse: true}},
		ProfileName: "local-balanced", ReviewerVersion: "rev-1",
	}
	refusedJSON, _ := json.Marshal(refused)
	base := FeedbackObservation{
		JobID: "job-1", Operation: "text_to_image", ModelID: "flux", Action: "refuse",
		ReasonCode: "sexual_minor_face", SafetyReview: "refused", Score: 0.9,
		Evidence: string(refusedJSON), ReviewedRefs: []string{"out/a.png"}, RefusedRefs: []string{"out/a.png"},
	}
	first, err := BuildFeedbackCase(base, FeedbackFalsePositive, "adult subject", "operator")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Record(ctx, first); err != nil {
		t.Fatal(err)
	}

	unassessable := base
	unassessable.JobID = "job-2"
	unassessable.ReasonCode = "sexual_unassessable"
	second, _ := BuildFeedbackCase(unassessable, FeedbackFalsePositive, "", "operator")
	if _, err := store.Record(ctx, second); err != nil {
		t.Fatal(err)
	}

	confirmed := base
	confirmed.JobID = "job-3"
	third, _ := BuildFeedbackCase(confirmed, FeedbackConfirmed, "", "api")
	if _, err := store.Record(ctx, third); err != nil {
		t.Fatal(err)
	}

	stats, err := store.Stats(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 3 || stats.FalsePositives != 2 || stats.Confirmed != 1 || stats.Uncertain != 0 {
		t.Fatalf("stats = %+v", stats)
	}
	if stats.FalsePositiveRate < 0.66 || stats.FalsePositiveRate > 0.67 {
		t.Fatalf("false-positive rate = %v", stats.FalsePositiveRate)
	}
	if len(stats.ByReason) == 0 || stats.ByReason[0].Key != "sexual_minor_face" || stats.ByReason[0].Cases != 2 {
		t.Fatalf("by reason = %+v", stats.ByReason)
	}
	if len(stats.ByProfile) != 1 || stats.ByProfile[0].Key != "local-balanced" {
		t.Fatalf("by profile = %+v", stats.ByProfile)
	}

	// Re-labelling the same job/output replaces it, never duplicates it.
	first.Label = FeedbackConfirmed
	if _, err := store.Record(ctx, first); err != nil {
		t.Fatal(err)
	}
	list, total, err := store.List(ctx, FeedbackFilters{}, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(list) != 3 {
		t.Fatalf("list total/len = %d/%d", total, len(list))
	}
	stats, _ = store.Stats(ctx, time.Time{})
	if stats.FalsePositives != 1 || stats.Confirmed != 2 {
		t.Fatalf("after relabel: %+v", stats)
	}

	jsonl, count, err := store.Export(ctx, FeedbackFilters{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 || strings.Count(jsonl, "\n") != 3 {
		t.Fatalf("export count/lines = %d/%d", count, strings.Count(jsonl, "\n"))
	}
	if !strings.Contains(jsonl, `"schema_version":1`) || !strings.Contains(jsonl, `"profile_name":"local-balanced"`) {
		t.Fatalf("export missing fields: %s", jsonl)
	}
	if !strings.Contains(jsonl, `"finding":{`) {
		t.Fatalf("export did not snapshot a job-level finding: %s", jsonl)
	}

	cleared, err := store.Clear(ctx, "job-1", "")
	if err != nil || !cleared {
		t.Fatalf("clear = %v/%v", cleared, err)
	}
	if again, _ := store.Clear(ctx, "job-1", ""); again {
		t.Fatal("clearing twice reported a row")
	}
}
