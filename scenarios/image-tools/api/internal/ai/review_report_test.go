package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
)

// batchReview adapts a one-image fake to the batch reviewer seam. It keeps the
// existing tests focused on one verdict while exercising the production shape.
func batchReview(one func(context.Context, []byte) (ScanVerdict, error)) OutputScanner {
	return func(ctx context.Context, images [][]byte) (ReviewBatch, error) {
		verdicts := make([]ScanVerdict, 0, len(images))
		for _, image := range images {
			verdict, err := one(ctx, image)
			if err != nil {
				return ReviewBatch{}, err
			}
			verdicts = append(verdicts, verdict)
		}
		return ReviewBatch{Verdicts: verdicts}, nil
	}
}

// runJobResult is runJob that keeps the whole result (the review records on its Meta).
func runJobResult(t *testing.T, eng *Engine, op string, pl Payload) (internaljobs.Result, error) {
	t.Helper()
	raw, err := marshalPinnedPayload(t, eng, op, pl)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	runner := eng.BuildRunners()[op]
	if runner == nil {
		t.Fatalf("no runner for %q", op)
	}
	return runner(context.Background(), internaljobs.Job{ID: "job-under-test", Operation: op, Payload: raw}, func(int, string) {})
}

func t2iPayload(modelID string) Payload {
	return Payload{Operation: "text_to_image", ModelID: modelID, Params: map[string]string{"prompt": "x"}}
}

func TestReviewRecordsTheSensitiveFlagAndScoreOnTheJob(t *testing.T) {
	eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	eng.deps.ReviewOutput = batchReview(func(context.Context, []byte) (ScanVerdict, error) {
		return ScanVerdict{NSFW: true, Score: 0.93, Detail: "passed: 1 image(s), max nsfw 0.93"}, nil
	})
	result, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID))
	if err != nil {
		t.Fatal(err)
	}
	if result.Meta[MetaSensitive] != "true" {
		t.Errorf("sensitive = %q, want true; meta=%v", result.Meta[MetaSensitive], result.Meta)
	}
	if result.Meta[MetaSensitiveRefs] != result.Ref {
		t.Errorf("sensitive_refs = %q, want the output ref %q", result.Meta[MetaSensitiveRefs], result.Ref)
	}
	if result.Meta[MetaNSFWScore] != "0.9300" || result.Meta[MetaSafetyReview] != "flagged" {
		t.Errorf("score/review meta = %q/%q", result.Meta[MetaNSFWScore], result.Meta[MetaSafetyReview])
	}
	if result.Meta[MetaSafetyAction] != "flag" || result.Meta[MetaSafetyReason] != "sexual_adult" {
		t.Errorf("action/reason = %q/%q", result.Meta[MetaSafetyAction], result.Meta[MetaSafetyReason])
	}
	if !strings.Contains(result.Meta[MetaSafetyReviewNotes], "max nsfw 0.93") {
		t.Errorf("review detail not recorded: %q", result.Meta[MetaSafetyReviewNotes])
	}
}

// forceNSFWCapable swaps the engine's registry for one where the model carries the
// given nsfw_capable label, so a test asserts the rule rather than the seed's data.
func forceNSFWCapable(t *testing.T, eng *Engine, modelID string, capable bool) {
	t.Helper()
	base := eng.deps.Registry
	all := base.Models()
	for i := range all {
		if all[i].ID == modelID {
			all[i].CapabilityLabels.NSFWCapable = capable
		}
	}
	encoded, err := json.Marshal(struct {
		SchemaVersion string                  `json:"schema_version"`
		Models        []models.Model          `json:"models"`
		Blocklist     []models.BlocklistEntry `json:"blocklist"`
	}{base.SchemaVersion(), all, base.Blocklist()})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := models.Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	eng.deps.Registry = registry
}

func TestReviewFlagsEveryOutputOfAnNSFWCapableModelEvenWhenTheClassifierIsClear(t *testing.T) {
	eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	forceNSFWCapable(t, eng, modelID, true)
	eng.deps.ReviewOutput = batchReview(func(context.Context, []byte) (ScanVerdict, error) { return ScanVerdict{Score: 0.01}, nil })
	result, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID))
	if err != nil {
		t.Fatal(err)
	}
	if result.Meta[MetaSensitive] != "true" || result.Meta[MetaSensitiveReason] != "model" || result.Meta[MetaSensitiveRefs] != result.Ref {
		t.Errorf("nsfw-capable model output not flagged for its model: %v", result.Meta)
	}
}

func TestReviewNamesBothReasonsWhenModelAndClassifierAgree(t *testing.T) {
	eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	forceNSFWCapable(t, eng, modelID, true)
	eng.deps.ReviewOutput = batchReview(func(context.Context, []byte) (ScanVerdict, error) { return ScanVerdict{NSFW: true, Score: 0.9}, nil })
	result, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID))
	if err != nil {
		t.Fatal(err)
	}
	if result.Meta[MetaSensitiveReason] != "classifier,model" || result.Meta[MetaSensitiveRefs] != result.Ref {
		t.Errorf("reasons/refs = %q/%q (an output must be listed once)", result.Meta[MetaSensitiveReason], result.Meta[MetaSensitiveRefs])
	}
}

func TestReviewLeavesAClearOutputOfAnOrdinaryModelUnflagged(t *testing.T) {
	eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	forceNSFWCapable(t, eng, modelID, false)
	eng.deps.ReviewOutput = batchReview(func(context.Context, []byte) (ScanVerdict, error) { return ScanVerdict{Score: 0.01}, nil })
	result, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID))
	if err != nil {
		t.Fatal(err)
	}
	if _, flagged := result.Meta[MetaSensitive]; flagged {
		t.Errorf("a clear output was flagged sensitive: %v", result.Meta)
	}
	if result.Meta[MetaSafetyReview] != "passed" {
		t.Errorf("a passed review must still be recorded: %v", result.Meta)
	}
}

func TestReviewKnowsWhetherTheProducingModelCanMakeExplicitImages(t *testing.T) {
	eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	model, ok := eng.ModelByID(modelID)
	if !ok {
		t.Fatalf("model %q missing", modelID)
	}
	var seen []bool
	eng.deps.ReviewOutput = batchReview(func(ctx context.Context, _ []byte) (ScanVerdict, error) {
		seen = append(seen, ReviewSourceFromContext(ctx).NSFWCapable)
		return ScanVerdict{}, nil
	})
	if _, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID)); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0] != model.CapabilityLabels.NSFWCapable {
		t.Fatalf("reviewer saw nsfw_capable=%v, model label is %v", seen, model.CapabilityLabels.NSFWCapable)
	}
}

// useLedger records the order of acquisitions and releases around the review.
type useLedger struct {
	mu     sync.Mutex
	events []string
}

func (l *useLedger) add(e string) { l.mu.Lock(); l.events = append(l.events, e); l.mu.Unlock() }

func TestReviewAcquiresMissingWeightsAndHoldsEveryUseLeaseAroundTheReview(t *testing.T) {
	eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	ledger := &useLedger{}
	installed := map[string]bool{"review-a": true, "review-b": false}
	eng.deps.ModelInstalled = func(id string) bool { return id == modelID || installed[id] }
	eng.deps.ReviewModels = []string{"review-a", "review-b"}
	eng.deps.AcquireModel = func(_ context.Context, id, jobID string, _ func(int, string)) error {
		ledger.add("acquire:" + id + "@" + jobID)
		installed[id] = true
		return nil
	}
	eng.deps.AcquireModelUse = func(id string) (func(), error) {
		if id == modelID {
			return func() {}, nil
		}
		ledger.add("hold:" + id)
		return func() { ledger.add("release:" + id) }, nil
	}
	eng.deps.ReviewOutput = batchReview(func(context.Context, []byte) (ScanVerdict, error) {
		ledger.add("review")
		return ScanVerdict{}, nil
	})
	if _, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID)); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(ledger.events, ",")
	// Admission acquires the missing weights first (the job reads as admitting
	// while it waits); the review then only holds the use leases.
	want := "acquire:review-b@job-under-test,hold:review-a,hold:review-b,review,release:review-b,release:review-a"
	if got != want {
		t.Fatalf("events\n got %s\nwant %s", got, want)
	}
}

// A weight that was present at admission and offloaded before the review is
// re-acquired by the review itself; if that fails the review fails closed.
func TestReviewFailsClosedAndDeletesOutputsWhenItsWeightsCannotBeAcquired(t *testing.T) {
	eng, store, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	rec := &recordingStore{Store: store}
	eng.deps.Store = rec
	reviewChecks := 0
	eng.deps.ModelInstalled = func(id string) bool {
		if id == "review-a" {
			reviewChecks++
			return reviewChecks == 1 // present at admission, offloaded afterwards
		}
		return id == modelID
	}
	eng.deps.ReviewModels = []string{"review-a"}
	eng.deps.AcquireModel = func(context.Context, string, string, func(int, string)) error { return errors.New("network down") }
	reviewed := false
	eng.deps.ReviewOutput = batchReview(func(context.Context, []byte) (ScanVerdict, error) { reviewed = true; return ScanVerdict{}, nil })

	_, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID))
	if !errors.Is(err, ErrModelNotInstalled) || !strings.Contains(err.Error(), "safety review") {
		t.Fatalf("err = %v, want ErrModelNotInstalled naming the safety review", err)
	}
	if reviewed {
		t.Error("the reviewer ran without its weights")
	}
	if len(rec.deleted) != 1 {
		t.Fatalf("deleted %v, want the unreviewed output removed", rec.deleted)
	}
}

func TestReviewWithoutAnyAcquirerRefusesMissingWeightsInsteadOfSkippingTheReview(t *testing.T) {
	eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	eng.deps.ModelInstalled = func(id string) bool { return id == modelID }
	eng.deps.ReviewModels = []string{"review-a"}
	eng.deps.ReviewOutput = batchReview(func(context.Context, []byte) (ScanVerdict, error) { return ScanVerdict{}, nil })
	if _, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID)); !errors.Is(err, ErrModelNotInstalled) {
		t.Fatalf("err = %v, want ErrModelNotInstalled", err)
	}
}

// The same missing weights are refused in admission, before the provider has
// produced anything, so no output exists to delete and nothing was billed.
func TestMissingReviewWeightsFailAdmissionBeforeAnyOutputExists(t *testing.T) {
	provider := &fakeProvider{}
	eng, store, modelID := newTestEngine(t, "text_to_image", provider)
	rec := &recordingStore{Store: store}
	eng.deps.Store = rec
	eng.deps.ModelInstalled = func(id string) bool { return id == modelID }
	eng.deps.ReviewModels = []string{"review-a"}
	eng.deps.AcquireModel = func(context.Context, string, string, func(int, string)) error { return errors.New("network down") }
	eng.deps.ReviewOutput = batchReview(func(context.Context, []byte) (ScanVerdict, error) { return ScanVerdict{}, nil })
	_, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID))
	if !errors.Is(err, ErrModelNotInstalled) || !strings.Contains(err.Error(), "safety review") {
		t.Fatalf("err = %v, want ErrModelNotInstalled naming the safety review", err)
	}
	if provider.execN != 0 || len(rec.deleted) != 0 {
		t.Fatalf("admission must fail before the provider runs: executions=%d deleted=%v", provider.execN, rec.deleted)
	}
}

func TestReviewAcquiresTheAgeWeightsUpFrontOnlyForAnNSFWCapableProducer(t *testing.T) {
	for _, capable := range []bool{false, true} {
		eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
		forceNSFWCapable(t, eng, modelID, capable)
		var acquired []string
		installed := map[string]bool{}
		eng.deps.ModelInstalled = func(id string) bool { return id == modelID || installed[id] }
		eng.deps.ReviewModels = []string{"classifier"}
		eng.deps.ReviewEscalationModels = []string{"faces", "age"}
		eng.deps.AcquireModel = func(_ context.Context, id, _ string, _ func(int, string)) error {
			acquired = append(acquired, id)
			installed[id] = true
			return nil
		}
		eng.deps.ReviewOutput = batchReview(func(context.Context, []byte) (ScanVerdict, error) { return ScanVerdict{}, nil })
		if _, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID)); err != nil {
			t.Fatal(err)
		}
		want := "classifier"
		if capable {
			want = "classifier,faces,age"
		}
		if got := strings.Join(acquired, ","); got != want {
			t.Errorf("nsfw_capable=%v acquired %q, want %q: ordinary work must not depend on the age model", capable, got, want)
		}
	}
}

func TestReviewAcquiresMissingEscalationWeightsOnDemandAndRetriesOnce(t *testing.T) {
	eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	forceNSFWCapable(t, eng, modelID, false)
	installed := map[string]bool{"classifier": true}
	var acquired []string
	eng.deps.ModelInstalled = func(id string) bool { return id == modelID || installed[id] }
	eng.deps.ReviewModels = []string{"classifier"}
	eng.deps.ReviewEscalationModels = []string{"faces", "age"}
	eng.deps.AcquireModel = func(_ context.Context, id, _ string, _ func(int, string)) error {
		acquired = append(acquired, id)
		installed[id] = true
		return nil
	}
	calls := 0
	eng.deps.ReviewOutput = batchReview(func(context.Context, []byte) (ScanVerdict, error) {
		calls++
		if !installed["faces"] || !installed["age"] {
			return ScanVerdict{}, &ReviewWeightsMissing{IDs: []string{"faces", "age"}}
		}
		return ScanVerdict{NSFW: true, Score: 0.8}, nil
	})
	result, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID))
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || strings.Join(acquired, ",") != "faces,age" {
		t.Fatalf("review calls = %d, acquired = %v; want one retry after acquiring faces and age", calls, acquired)
	}
	if result.Meta[MetaSensitive] != "true" || result.Meta[MetaSensitiveReason] != "classifier" {
		t.Errorf("retried verdict not recorded: %v", result.Meta)
	}
}

func TestReviewFailsClosedWhenTheOnDemandWeightsCannotBeAcquired(t *testing.T) {
	eng, store, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	forceNSFWCapable(t, eng, modelID, false)
	rec := &recordingStore{Store: store}
	eng.deps.Store = rec
	eng.deps.ModelInstalled = func(id string) bool { return id == modelID || id == "classifier" }
	eng.deps.ReviewModels = []string{"classifier"}
	eng.deps.AcquireModel = func(context.Context, string, string, func(int, string)) error { return errors.New("offline") }
	eng.deps.ReviewOutput = batchReview(func(context.Context, []byte) (ScanVerdict, error) {
		return ScanVerdict{}, &ReviewWeightsMissing{IDs: []string{"age"}}
	})
	_, err := runJobResult(t, eng, "text_to_image", t2iPayload(modelID))
	if !errors.Is(err, ErrModelNotInstalled) {
		t.Fatalf("err = %v, want ErrModelNotInstalled", err)
	}
	if len(rec.deleted) != 1 {
		t.Fatalf("deleted %v, want the unreviewed output removed", rec.deleted)
	}
}

// A warm worker's reservation cancels the admitted context when the engine
// releases the execution claims. The review runs after that release and must
// not inherit the cancellation, or every warm-worker job would fail its review.
func TestReviewRunsOnTheJobContextAfterTheAdmittedContextIsReleased(t *testing.T) {
	eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	var reviewCtxErr error
	reviewed := 0
	eng.deps.ReviewOutput = batchReview(func(ctx context.Context, _ []byte) (ScanVerdict, error) {
		reviewed++
		reviewCtxErr = ctx.Err()
		return ScanVerdict{}, nil
	})
	job := internaljobs.Job{ID: "job-under-test", Operation: "text_to_image"}
	raw, err := marshalPinnedPayload(t, eng, "text_to_image", t2iPayload(modelID))
	if err != nil {
		t.Fatal(err)
	}
	job.Payload = raw
	admission, err := eng.Admit(context.Background(), job)
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	admitted, release := context.WithCancel(admission.Context)
	defer release()
	prepared, ok := admitted.Value(executionAdmissionKey{}).(*preparedExecution)
	if !ok {
		t.Fatal("admission carries no prepared execution")
	}
	prepared.finishInference = func(context.Context) error { release(); return nil }

	if _, err := eng.BuildRunners()["text_to_image"](admitted, job, func(int, string) {}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if reviewed != 1 {
		t.Fatalf("reviewed %d outputs, want 1", reviewed)
	}
	if reviewCtxErr != nil {
		t.Fatalf("review ran on a canceled context: %v", reviewCtxErr)
	}
}
