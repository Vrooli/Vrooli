package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
)

// autoAcquireEngine is a text_to_image engine whose model starts uninstalled
// and becomes installed only through AcquireModel.
type autoAcquireEngine struct {
	*Engine
	mu        sync.Mutex
	installed bool
	notReady  string // lifecycle reason while installed but not runtime-ready
	acquired  []string
	refuse    error
	events    []string
}

func newAutoAcquireEngine(t *testing.T) (*autoAcquireEngine, string) {
	t.Helper()
	eng, _, modelID := newTestEngine(t, "text_to_image", &fakeProvider{})
	a := &autoAcquireEngine{Engine: eng}
	eng.deps.ModelInstalled = func(string) bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.installed
	}
	eng.deps.AutoAcquirable = func(_ context.Context, m models.Model) error { return a.refuse }
	eng.deps.AcquireModel = func(ctx context.Context, id, jobID string, emit func(int, string)) error {
		emit(10, "acquiring weights for "+id)
		a.mu.Lock()
		defer a.mu.Unlock()
		a.acquired = append(a.acquired, id+"@"+jobID)
		a.events = append(a.events, "acquire")
		a.installed, a.notReady = true, ""
		return nil
	}
	eng.deps.ModelRuntimeReadiness = func(context.Context, string) (bool, string) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.notReady != "" {
			return false, a.notReady
		}
		return true, "ready"
	}
	eng.deps.Revalidatable = func(_ context.Context, m models.Model, reason string) error {
		if reason != "smoke_pending" {
			return errors.New("cannot revalidate " + reason)
		}
		return a.refuse
	}
	eng.deps.AcquireModelUse = func(string) (func(), error) {
		a.mu.Lock()
		a.events = append(a.events, "use")
		a.mu.Unlock()
		return func() {}, nil
	}
	eng.deps.InstallEstimateSeconds = func(sizeMB int) int { return 1000 }
	return a, modelID
}

func TestPlanSchedulesAutoAcquireForEligibleMissingModel(t *testing.T) {
	a, modelID := newAutoAcquireEngine(t)
	installed, _, _ := newTestEngine(t, "text_to_image", &fakeProvider{})
	baseline, err := installed.Plan(context.Background(), PlanRequest{Operation: "text_to_image"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := a.Plan(context.Background(), PlanRequest{Operation: "text_to_image"})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if !plan.AutoAcquire || plan.ModelID != modelID {
		t.Fatalf("plan=%+v, want auto-acquire of %s", plan, modelID)
	}
	if plan.EstimatedSeconds != baseline.EstimatedSeconds+1000 {
		t.Fatalf("ETA=%d, want baseline %d plus the acquisition estimate", plan.EstimatedSeconds, baseline.EstimatedSeconds)
	}
	if !strings.Contains(strings.Join(plan.Warnings, "\n"), "will acquire its weights") {
		t.Fatalf("warnings %v do not announce the acquisition", plan.Warnings)
	}
	if len(a.acquired) != 0 {
		t.Fatal("planning must not download weights")
	}
}

func TestPlanRefusesMissingModelWhenAutoAcquireIsNotPossible(t *testing.T) {
	declined := false
	cases := []struct {
		name   string
		req    PlanRequest
		refuse error
		want   string
	}{
		{"caller opt-out", PlanRequest{Operation: "text_to_image", AutoAcquire: &declined}, nil, "declined by the caller"},
		{"ineligible model", PlanRequest{Operation: "text_to_image"}, errors.New("has a manual source"), "has a manual source"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, modelID := newAutoAcquireEngine(t)
			a.refuse = tc.refuse
			_, err := a.Plan(context.Background(), tc.req)
			if !errors.Is(err, ErrModelNotInstalled) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want ErrModelNotInstalled mentioning %q", err, tc.want)
			}
			if tc.refuse == nil && !strings.Contains(err.Error(), "models install "+modelID) {
				t.Fatalf("err=%v lacks the explicit install command", err)
			}
		})
	}
	a, _ := newAutoAcquireEngine(t)
	a.deps.AutoAcquirable, a.deps.AcquireModel = nil, nil
	if _, err := a.Plan(context.Background(), PlanRequest{Operation: "text_to_image"}); !errors.Is(err, ErrModelNotInstalled) {
		t.Fatalf("unwired auto-acquire: err=%v, want ErrModelNotInstalled", err)
	}
}

func admitPayload(t *testing.T, a *autoAcquireEngine, modelID string, autoAcquire bool) internaljobs.Job {
	t.Helper()
	a.installed = true // pin the provider through a normal plan
	raw, err := marshalPinnedPayload(t, a.Engine, "text_to_image", Payload{ModelID: modelID})
	a.installed = false
	if err != nil {
		t.Fatal(err)
	}
	var pl Payload
	if err := json.Unmarshal(raw, &pl); err != nil {
		t.Fatal(err)
	}
	pl.AutoAcquire = autoAcquire
	raw, _ = json.Marshal(pl)
	return internaljobs.Job{ID: "job-auto", Operation: "text_to_image", Payload: raw}
}

func TestAdmitAcquiresMissingWeightsBeforeAnyLease(t *testing.T) {
	a, modelID := newAutoAcquireEngine(t)
	job := admitPayload(t, a, modelID, true)
	var progress []string
	ctx := internaljobs.WithAdmissionProgress(context.Background(), func(_ int, m string) { progress = append(progress, m) })
	admission, err := a.Admit(ctx, job)
	if admission.Release != nil {
		defer func() { _ = admission.Release(context.Background()) }()
	}
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	if len(a.acquired) != 1 || a.acquired[0] != modelID+"@job-auto" {
		t.Fatalf("acquired=%v", a.acquired)
	}
	if len(a.events) < 2 || a.events[0] != "acquire" || a.events[1] != "use" {
		t.Fatalf("events=%v, want acquisition before the model use lease", a.events)
	}
	if len(progress) == 0 || !strings.Contains(progress[0], "acquiring weights") {
		t.Fatalf("admission progress=%v", progress)
	}
}

func TestAdmitRefusesMissingWeightsWithoutAutoAcquire(t *testing.T) {
	a, modelID := newAutoAcquireEngine(t)
	job := admitPayload(t, a, modelID, false)
	admission, err := a.Admit(context.Background(), job)
	if admission.Release != nil {
		_ = admission.Release(context.Background())
	}
	if !errors.Is(err, ErrModelNotInstalled) || len(a.acquired) != 0 {
		t.Fatalf("err=%v acquired=%v, want refusal without download", err, a.acquired)
	}
}

func TestAdmitRetriesWhenSharedDiskIsBusy(t *testing.T) {
	a, modelID := newAutoAcquireEngine(t)
	a.deps.AcquireModel = func(context.Context, string, string, func(int, string)) error {
		return fmt.Errorf("modelstorage: %w: waiting", internaljobs.ErrCapacityBusy)
	}
	job := admitPayload(t, a, modelID, true)
	admission, err := a.Admit(context.Background(), job)
	if admission.Release != nil {
		_ = admission.Release(context.Background())
	}
	if !errors.Is(err, internaljobs.ErrCapacityBusy) || errors.Is(err, ErrModelNotInstalled) {
		t.Fatalf("err=%v, want a retryable capacity refusal, not a missing-model failure", err)
	}
}

func TestAdmitRetriesWhenAnotherWriterHoldsTheModel(t *testing.T) {
	a, modelID := newAutoAcquireEngine(t)
	a.deps.AcquireModel = func(context.Context, string, string, func(int, string)) error {
		return models.ErrModelInUse
	}
	job := admitPayload(t, a, modelID, true)
	admission, err := a.Admit(context.Background(), job)
	if admission.Release != nil {
		_ = admission.Release(context.Background())
	}
	if !errors.Is(err, internaljobs.ErrCapacityBusy) {
		t.Fatalf("err=%v, want a retryable capacity refusal", err)
	}
}

func TestPlanRevalidatesInstalledModelThatIsNotRuntimeReady(t *testing.T) {
	a, modelID := newAutoAcquireEngine(t)
	a.installed, a.notReady = true, "smoke_pending"
	plan, err := a.Plan(context.Background(), PlanRequest{Operation: "text_to_image"})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if !plan.AutoAcquire || !strings.Contains(strings.Join(plan.Warnings, "\n"), "will revalidate it") {
		t.Fatalf("plan=%+v, want a scheduled revalidation of %s", plan, modelID)
	}
	declined := false
	if _, err := a.Plan(context.Background(), PlanRequest{Operation: "text_to_image", AutoAcquire: &declined}); !errors.Is(err, ErrModelNotRuntimeReady) || !strings.Contains(err.Error(), "declined") {
		t.Fatalf("opt-out: err=%v, want the not-runtime-ready refusal", err)
	}
	a.notReady = "smoke_failed"
	if _, err := a.Plan(context.Background(), PlanRequest{Operation: "text_to_image"}); !errors.Is(err, ErrModelNotRuntimeReady) {
		t.Fatalf("unresolvable reason: err=%v, want ErrModelNotRuntimeReady", err)
	}
}

func TestAdmitRevalidatesBeforeAnyLease(t *testing.T) {
	a, modelID := newAutoAcquireEngine(t)
	job := admitPayload(t, a, modelID, true)
	a.installed, a.notReady = true, "smoke_pending"
	admission, err := a.Admit(context.Background(), job)
	if admission.Release != nil {
		defer func() { _ = admission.Release(context.Background()) }()
	}
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	if len(a.events) < 2 || a.events[0] != "acquire" || a.events[1] != "use" {
		t.Fatalf("events=%v, want revalidation before the model use lease", a.events)
	}
}

// The safety review's weights are acquired while the job is still admitting,
// before any lease and before any output exists, so a wait for them reads as
// admitting and can be returned to the queue for a restart.
func TestAdmitAcquiresTheSafetyReviewWeightsAndReportsTheWaitAsAdmission(t *testing.T) {
	a, modelID := newAutoAcquireEngine(t)
	a.installed = true
	reviewInstalled := false
	var order []string
	a.deps.ModelInstalled = func(id string) bool {
		if id == modelID {
			return true
		}
		return reviewInstalled
	}
	a.deps.ReviewOutput = func(context.Context, []byte) (ScanVerdict, error) { return ScanVerdict{}, nil }
	a.deps.ReviewModels = []string{"review-a"}
	a.deps.AcquireModel = func(_ context.Context, id, jobID string, emit func(int, string)) error {
		order = append(order, "acquire:"+id+"@"+jobID)
		emit(95, "waiting for model smoke capacity")
		reviewInstalled = true
		return nil
	}
	job := admitPayload(t, a, modelID, true)
	a.installed = true
	var progress []string
	ctx := internaljobs.WithAdmissionProgress(context.Background(), func(_ int, m string) { progress = append(progress, m) })
	admission, err := a.Admit(ctx, job)
	if admission.Release != nil {
		defer func() { _ = admission.Release(context.Background()) }()
	}
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	if len(order) != 1 || order[0] != "acquire:review-a@job-auto" {
		t.Fatalf("review weights must be acquired once during admission: %v", order)
	}
	if got := strings.Join(progress, "|"); !strings.Contains(got, "safety review weights: waiting for model smoke capacity") {
		t.Fatalf("the wait must be reported as admission progress, got %q", got)
	}
}

func TestAdmitWaitsInsteadOfFailingWhenTheReviewWeightsHitBusyCapacity(t *testing.T) {
	a, modelID := newAutoAcquireEngine(t)
	a.installed = true
	a.deps.ModelInstalled = func(id string) bool { return id == modelID }
	a.deps.ReviewOutput = func(context.Context, []byte) (ScanVerdict, error) { return ScanVerdict{}, nil }
	a.deps.ReviewModels = []string{"review-a"}
	a.deps.AcquireModel = func(context.Context, string, string, func(int, string)) error {
		return fmt.Errorf("%w: model smoke capacity", internaljobs.ErrCapacityBusy)
	}
	job := admitPayload(t, a, modelID, true)
	a.installed = true
	admission, err := a.Admit(context.Background(), job)
	if admission.Release != nil {
		defer func() { _ = admission.Release(context.Background()) }()
	}
	if !errors.Is(err, internaljobs.ErrCapacityBusy) {
		t.Fatalf("err = %v, want the job requeued as capacity-busy", err)
	}
}
