package analysis

import (
	"context"
	"errors"
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
			if request.RAMBytes != 512<<20 || request.CPUMilli != 1000 || request.Priority != "interactive" || !request.NoReclaim {
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
	if readers.Load() != 1 {
		t.Fatal("test did not hold artifact reader")
	}
	if err := admission.Release(t.Context()); err != nil {
		t.Fatal(err)
	}
	if readers.Load() != 0 {
		t.Fatal("failed admission leaked artifact reader")
	}
}
