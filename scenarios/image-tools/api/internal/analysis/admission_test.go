package analysis

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	"image-tools/internal/jobs"
)

type analysisLease struct{ releases atomic.Int32 }

func (*analysisLease) Heartbeat(context.Context) error { return nil }
func (l *analysisLease) Release(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	l.releases.Add(1)
	return nil
}

func TestPreparedAnalysisUsesDeclaredBudgetAndRetainsClaimsUntilCleanup(t *testing.T) {
	lease := &analysisLease{}
	var acquisitions atomic.Int32
	service, err := NewService(Config{
		ModelInstalled: func(string) bool { return true },
		LookPath:       func(string) (string, error) { return "tesseract", nil },
		Run:            func(context.Context, string, []string) ([]byte, error) { return []byte("synthetic fixture"), nil },
		AcquireCapacity: func(ctx context.Context, request sharedcapacity.Request) (*sharedcapacity.Scope, error) {
			acquisitions.Add(1)
			if request.RAMBytes != 512<<20 || request.CPUMilli != int64(cpuThreadsFor(OpOCR))*1000 || request.Priority != "interactive" || !request.NoReclaim {
				t.Fatalf("unbounded or wrong admission: %+v", request)
			}
			return sharedcapacity.Maintain(ctx, lease), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	admission, err := service.Admit(ctx, jobs.Job{ID: "ocr", Operation: OpOCR, Priority: jobs.PriorityInteractive})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OCR(admission.Context, []byte("synthetic")); err != nil {
		t.Fatal(err)
	}
	if acquisitions.Load() != 1 || lease.releases.Load() != 0 {
		t.Fatal("prepared analysis reacquired or prematurely released capacity")
	}
	cancel()
	if lease.releases.Load() != 0 {
		t.Fatal("cancellation released before worker ownership ended")
	}
	if err := admission.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if lease.releases.Load() != 1 {
		t.Fatal("cleanup did not release exact claim")
	}
}

func TestAnalysisAdmissionDenialRollsBackModelReader(t *testing.T) {
	var readers atomic.Int32
	service, err := NewService(Config{
		ModelInstalled:  func(string) bool { return true },
		ModelRAMBytes:   func(string) (int64, error) { return 4 << 30, nil },
		AcquireModelUse: func(string) (func(), error) { readers.Add(1); return func() { readers.Add(-1) }, nil },
		AcquireCapacity: func(context.Context, sharedcapacity.Request) (*sharedcapacity.Scope, error) {
			return nil, &sharedcapacity.UnavailableError{Verdict: sharedcapacity.Verdict{Kind: "queue", Reason: "occupied"}}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	admission, err := service.Admit(t.Context(), jobs.Job{ID: "classification", Operation: OpNSFW})
	if !errors.Is(err, jobs.ErrCapacityBusy) {
		t.Fatalf("denial lost retryability: %v", err)
	}
	if readers.Load() != 2 {
		t.Fatal("test did not hold both classifier readers")
	}
	if err := admission.Release(t.Context()); err != nil {
		t.Fatal(err)
	}
	if readers.Load() != 0 {
		t.Fatal("failed admission leaked artifact readers")
	}
}

func TestAnalysisAdmissionAcquiresMissingModelBeforeReaderLease(t *testing.T) {
	var events []string
	installed := false
	service, err := NewService(Config{
		ModelInstalled: func(string) bool { return installed },
		ModelRAMBytes:  func(string) (int64, error) { return 1 << 30, nil },
		AcquireModel: func(ctx context.Context, id, jobID string, emit func(int, string)) error {
			emit(5, "acquiring weights for "+id)
			events = append(events, "acquire:"+id+"@"+jobID)
			installed = true
			return nil
		},
		AcquireModelUse: func(id string) (func(), error) { events = append(events, "use:"+id); return func() {}, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	var progress []string
	ctx := jobs.WithAdmissionProgress(t.Context(), func(_ int, m string) { progress = append(progress, m) })
	admission, err := service.Admit(ctx, jobs.Job{ID: "scan-1", Operation: OpNSFW})
	if admission.Release != nil {
		defer func() { _ = admission.Release(context.Background()) }()
	}
	if err != nil {
		t.Fatal(err)
	}
	wantEvents := []string{"acquire:falconsai-nsfw@scan-1", "use:falconsai-nsfw", "use:adamcodd-vit-nsfw"}
	if len(events) != len(wantEvents) {
		t.Fatalf("events=%v, want %v", events, wantEvents)
	}
	for i := range wantEvents {
		if events[i] != wantEvents[i] {
			t.Fatalf("events=%v, want %v", events, wantEvents)
		}
	}
	if len(progress) != 1 {
		t.Fatalf("progress=%v", progress)
	}

	installed = false
	service.acquireModel = func(context.Context, string, string, func(int, string)) error { return errors.New("manual source") }
	if _, err := service.Admit(t.Context(), jobs.Job{ID: "scan-2", Operation: OpNSFW}); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("err=%v, want ErrBackendUnavailable", err)
	}
}

func TestAnalysisAdmissionRevalidatesInstalledModelBeforeReaderLease(t *testing.T) {
	var events []string
	state := "smoke_pending"
	service, err := NewService(Config{
		ModelInstalled: func(string) bool { return true },
		ModelRAMBytes:  func(string) (int64, error) { return 1 << 30, nil },
		ModelReady: func(context.Context, string) (bool, string) {
			events = append(events, "ready?"+state)
			return state == "ready", state
		},
		AcquireModel: func(_ context.Context, id, jobID string, emit func(int, string)) error {
			emit(5, "revalidating "+id)
			events = append(events, "revalidate:"+id+"@"+jobID)
			state = "ready"
			return nil
		},
		AcquireModelUse: func(id string) (func(), error) { events = append(events, "use:"+id); return func() {}, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	admission, err := service.Admit(t.Context(), jobs.Job{ID: "scan-1", Operation: OpNSFW})
	if admission.Release != nil {
		defer func() { _ = admission.Release(context.Background()) }()
	}
	if err != nil {
		t.Fatalf("a smoke_pending model must be revalidated and admitted: %v", err)
	}
	want := []string{
		"ready?smoke_pending", "revalidate:falconsai-nsfw@scan-1", "use:falconsai-nsfw", "ready?ready",
		"ready?ready", "use:adamcodd-vit-nsfw", "ready?ready",
	}
	if len(events) != len(want) {
		t.Fatalf("events=%v, want %v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("events=%v, want %v", events, want)
		}
	}
}

func TestAnalysisAdmissionRefusesWhatRevalidationCannotResolve(t *testing.T) {
	cases := []struct {
		name      string
		acquire   func(context.Context, string, string, func(int, string)) error
		wantBusy  bool
		wantInErr string
	}{
		{name: "installer leaves a failed smoke untouched", acquire: func(context.Context, string, string, func(int, string)) error { return nil }, wantInErr: "smoke_failed"},
		{name: "revalidation itself fails", acquire: func(context.Context, string, string, func(int, string)) error { return errors.New("smoke crashed") }, wantInErr: "could not be revalidated"},
		{name: "exclusive model lease is busy", acquire: func(context.Context, string, string, func(int, string)) error {
			return sharedcapacity.ErrAssetInUse
		}, wantBusy: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var held atomic.Int32
			service, err := NewService(Config{
				ModelInstalled:  func(string) bool { return true },
				ModelRAMBytes:   func(string) (int64, error) { return 1 << 30, nil },
				ModelReady:      func(context.Context, string) (bool, string) { return false, "smoke_failed" },
				AcquireModel:    tc.acquire,
				AcquireModelUse: func(string) (func(), error) { held.Add(1); return func() { held.Add(-1) }, nil },
			})
			if err != nil {
				t.Fatal(err)
			}
			admission, err := service.Admit(t.Context(), jobs.Job{ID: "scan", Operation: OpOCR})
			if admission.Release != nil {
				_ = admission.Release(context.Background())
			}
			if held.Load() != 0 {
				t.Fatal("a refused model must not keep its reader lease")
			}
			if tc.wantBusy {
				if !errors.Is(err, jobs.ErrCapacityBusy) {
					t.Fatalf("err=%v, want capacity busy so the scheduler retries", err)
				}
				return
			}
			if !errors.Is(err, ErrBackendUnavailable) || !strings.Contains(err.Error(), tc.wantInErr) {
				t.Fatalf("err=%v, want ErrBackendUnavailable naming %q", err, tc.wantInErr)
			}
		})
	}
}
