package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	internaljobs "image-tools/internal/jobs"
)

// reviewKeys splits the meta refs written by the batch review.
func splitRefs(meta map[string]string, key string) []string {
	raw := strings.TrimSpace(meta[key])
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

func TestReviewBatchesEveryVariationInOneCall(t *testing.T) {
	eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	forceNSFWCapable(t, eng, modelID, false)
	calls, seen := 0, 0
	eng.deps.ReviewOutput = func(_ context.Context, images [][]byte) (ReviewBatch, error) {
		calls++
		seen = len(images)
		verdicts := make([]ScanVerdict, len(images))
		for i := range images {
			verdicts[i] = ScanVerdict{Action: "allow", ReasonCode: "clean", Score: 0.01}
		}
		return ReviewBatch{Verdicts: verdicts, Evidence: `{"reviewer_version":"test"}`}, nil
	}
	payload := t2iPayload(modelID)
	payload.Variations = 3
	result, err := runJobResult(t, eng, "text_to_image", payload)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || seen != 3 {
		t.Fatalf("review calls=%d images=%d, want one call with all 3 variations", calls, seen)
	}
	if result.Meta[MetaSafetyReview] != "passed" || result.Meta[MetaSafetyEvidence] == "" {
		t.Fatalf("batch evidence not recorded: %v", result.Meta)
	}
}

func TestReviewShadowRecordsARefusalWithoutDeletingOrFailing(t *testing.T) {
	eng, store, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	rec := &recordingStore{Store: store}
	eng.deps.Store = rec
	eng.deps.ReviewOutput = batchReview(func(context.Context, []byte) (ScanVerdict, error) {
		return ScanVerdict{
			Action: "refuse", ReasonCode: "sexual_unassessable", Detail: "age could not be established",
			Score: 0.95, Shadow: true, HardRefusal: "age could not be established",
		}, nil
	})
	result, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID))
	if err != nil {
		t.Fatalf("shadow mode must not fail the job: %v", err)
	}
	if len(rec.deleted) != 0 {
		t.Fatalf("shadow mode deleted outputs: %v", rec.deleted)
	}
	if result.Meta[MetaSafetyReview] != "shadow_refused" || result.Meta[MetaSensitive] != "true" {
		t.Fatalf("shadow evidence = %v", result.Meta)
	}
	if len(splitRefs(result.Meta, MetaSafetyShadowRefs)) != 1 {
		t.Fatalf("shadow refs = %q", result.Meta[MetaSafetyShadowRefs])
	}
}

func TestReviewDropVariationKeepsThePassingImages(t *testing.T) {
	eng, store, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	rec := &recordingStore{Store: store}
	eng.deps.Store = rec
	eng.deps.ReviewOutput = func(_ context.Context, images [][]byte) (ReviewBatch, error) {
		if len(images) != 2 {
			t.Fatalf("images = %d, want 2", len(images))
		}
		return ReviewBatch{Verdicts: []ScanVerdict{
			{Action: "refuse", ReasonCode: "sexual_unassessable", Detail: "no readable face", HardRefusal: "no readable face", DropVariation: true},
			{Action: "allow", ReasonCode: "clean", Score: 0.01},
		}}, nil
	}
	payload := t2iPayload(modelID)
	payload.Variations = 2
	result, err := runJobResult(t, eng, "text_to_image", payload)
	if err != nil {
		t.Fatalf("dropping one variation must not fail the job: %v", err)
	}
	allowed := splitRefs(result.Meta, MetaSafetyAllowedRefs)
	refused := splitRefs(result.Meta, MetaSafetyRefusedRefs)
	if len(allowed) != 1 || len(refused) != 1 {
		t.Fatalf("allowed=%v refused=%v", allowed, refused)
	}
	if result.Ref != allowed[0] {
		t.Fatalf("result ref = %q, want the allowed variation %q", result.Ref, allowed[0])
	}
	if len(rec.deleted) != 1 || rec.deleted[0] != refused[0] {
		t.Fatalf("deleted = %v, want only %q", rec.deleted, refused[0])
	}
	if result.Meta[MetaSafetyReview] != "partial_refused" || result.Meta[MetaSafetyReason] != "sexual_unassessable" {
		t.Fatalf("partial refusal evidence = %v", result.Meta)
	}
}

func TestReviewFailJobModeDeletesEveryVariationAndKeepsEvidence(t *testing.T) {
	eng, store, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	rec := &recordingStore{Store: store}
	eng.deps.Store = rec
	eng.deps.ReviewOutput = func(_ context.Context, images [][]byte) (ReviewBatch, error) {
		return ReviewBatch{Verdicts: []ScanVerdict{
			{Action: "allow", ReasonCode: "clean", Score: 0.01},
			{Action: "refuse", ReasonCode: "sexual_minor_face", HardRefusal: "minor-appearing face", DropVariation: false},
		}}, nil
	}
	payload := t2iPayload(modelID)
	payload.Variations = 2
	result, err := runJobResult(t, eng, "text_to_image", payload)
	if err == nil {
		t.Fatal("fail_job mode allowed a refused variation")
	}
	if len(rec.deleted) != 2 {
		t.Fatalf("deleted = %v, want every variation", rec.deleted)
	}
	if result.Meta[MetaSafetyReview] != "refused" || result.Meta[MetaSafetyReason] != "sexual_minor_face" {
		t.Fatalf("refusal evidence = %v", result.Meta)
	}
	if len(splitRefs(result.Meta, MetaSafetyRefusedRefs)) != 1 {
		t.Fatalf("refused refs = %q", result.Meta[MetaSafetyRefusedRefs])
	}
}

// A refused job on the real submit path carries route evidence. The failure
// narrowing in run() must not drop the reviewer's safety_* evidence, or the
// retained quarantine entry cannot be labelled: safety feedback reads the job's
// evidence to snapshot the verdict.
func TestRefusedJobKeepsSafetyEvidenceAlongsideRouteEvidence(t *testing.T) {
	eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	eng.deps.ReviewOutput = func(context.Context, [][]byte) (ReviewBatch, error) {
		return ReviewBatch{
			Verdicts: []ScanVerdict{{
				Action: "refuse", ReasonCode: "sexual_unassessable", HardRefusal: "no readable face",
			}},
			Evidence: `{"reviewer_version":"test"}`,
		}, nil
	}
	payload := t2iPayload(modelID)
	payload.Route = &RouteState{
		Choice: routeChoiceLocal, LocalModel: modelID, MaxCostUSD: 0.25, WaitBudgetSeconds: 600,
	}
	result, err := runJobResult(t, eng, "text_to_image", payload)
	if err == nil {
		t.Fatal("a refusal must fail the job")
	}
	for _, key := range []string{
		MetaSafetyReview, MetaSafetyAction, MetaSafetyReason, MetaSafetyEvidence,
		MetaSafetyRefusedRefs, MetaNSFWScore,
	} {
		if result.Meta[key] == "" {
			t.Errorf("refused job dropped %q: %v", key, result.Meta)
		}
	}
	if result.Meta["route_choice"] != routeChoiceLocal {
		t.Errorf("route evidence also dropped: %v", result.Meta)
	}
}

func TestReviewTechnicalFailureIsClassifiedAsSafetyUnavailable(t *testing.T) {
	eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	eng.deps.ReviewOutput = func(context.Context, [][]byte) (ReviewBatch, error) {
		return ReviewBatch{}, context.DeadlineExceeded
	}
	_, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID))
	if err == nil {
		t.Fatal("a reviewer failure must fail closed")
	}
	if got := internaljobs.KindOf(err); got != internaljobs.FailureSafetyUnavailable {
		t.Fatalf("failure kind = %q, want safety_unavailable", got)
	}
}

func TestReviewQuarantinesARefusedVariationBeforeDeletingIt(t *testing.T) {
	eng, store, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	rec := &recordingStore{Store: store}
	eng.deps.Store = rec
	var quarantined QuarantineItem
	var quarantinedBytes int
	eng.deps.Quarantine = func(_ context.Context, item QuarantineItem, data []byte) (string, error) {
		quarantined = item
		quarantinedBytes = len(data)
		return "quarantine/job/refused.png", nil
	}
	eng.deps.ReviewOutput = func(_ context.Context, images [][]byte) (ReviewBatch, error) {
		return ReviewBatch{Verdicts: []ScanVerdict{
			{Action: "refuse", ReasonCode: "sexual_unassessable", Detail: "no readable face", HardRefusal: "no readable face", DropVariation: true, Finding: `{"refuse":true}`},
			{Action: "allow", ReasonCode: "clean", Score: 0.01},
		}}, nil
	}
	payload := t2iPayload(modelID)
	payload.Variations = 2
	result, err := runJobResult(t, eng, "text_to_image", payload)
	if err != nil {
		t.Fatal(err)
	}
	if quarantined.OriginalRef == "" || quarantined.JobID != "job-under-test" || quarantined.ReasonCode != "sexual_unassessable" {
		t.Fatalf("quarantine item = %+v", quarantined)
	}
	if quarantined.Finding == "" || quarantinedBytes == 0 {
		t.Fatalf("quarantine lost the finding or bytes: finding=%q bytes=%d", quarantined.Finding, quarantinedBytes)
	}
	if result.Meta[MetaSafetyQuarantineRefs] != "quarantine/job/refused.png" {
		t.Fatalf("quarantine meta = %q", result.Meta[MetaSafetyQuarantineRefs])
	}
	if len(rec.deleted) != 1 || rec.deleted[0] != quarantined.OriginalRef {
		t.Fatalf("deleted = %v, want only the original refused ref %q", rec.deleted, quarantined.OriginalRef)
	}
}

func TestQuarantineFailureDeletesTheRefusalAndRecordsWhy(t *testing.T) {
	eng, store, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	rec := &recordingStore{Store: store}
	eng.deps.Store = rec
	eng.deps.Quarantine = func(context.Context, QuarantineItem, []byte) (string, error) {
		return "", errors.New("quarantine disk full")
	}
	eng.deps.ReviewOutput = func(context.Context, [][]byte) (ReviewBatch, error) {
		return ReviewBatch{Verdicts: []ScanVerdict{{
			Action: "refuse", ReasonCode: "sexual_unassessable", HardRefusal: "no readable face",
		}}}, nil
	}
	result, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID))
	if err == nil {
		t.Fatal("a refusal with a failed quarantine must still fail")
	}
	if len(rec.deleted) != 1 {
		t.Fatalf("deleted = %v, want the refused bytes deleted as the fail-closed fallback", rec.deleted)
	}
	if !strings.Contains(result.Meta[MetaSafetyQuarantineError], "quarantine disk full") {
		t.Fatalf("quarantine error = %q", result.Meta[MetaSafetyQuarantineError])
	}
	if result.Meta[MetaSafetyQuarantineRefs] != "" {
		t.Fatalf("a failed quarantine recorded a ref: %q", result.Meta[MetaSafetyQuarantineRefs])
	}
}
