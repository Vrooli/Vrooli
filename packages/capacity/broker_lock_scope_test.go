package capacity

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

type slowHostSource struct {
	delay time.Duration
	gpus  []hostinventory.GPU
}

func (s slowHostSource) Snapshot(ctx context.Context) (hostinventory.Snapshot, error) {
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return hostinventory.Snapshot{}, ctx.Err()
	}
	snap := hostinventory.Snapshot{GPUs: s.gpus}
	snap.Memory.TotalBytes, snap.Memory.AvailableBytes = 64<<30, 48<<30
	return snap, nil
}

// admissionLockHold polls the shared admission lock and returns the longest
// stretch during which it could not be taken while run executed.
func admissionLockHold(t *testing.T, store *internalcapacity.SQLiteStore, run func()) time.Duration {
	t.Helper()
	var mu sync.Mutex
	var longest time.Duration
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		var since time.Time
		for {
			select {
			case <-stop:
				return
			default:
			}
			probe, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
			release, err := store.LockAdmission(probe)
			cancel()
			if err == nil {
				release()
				since = time.Time{}
			} else {
				if since.IsZero() {
					since = time.Now()
				}
				mu.Lock()
				if d := time.Since(since); d > longest {
					longest = d
				}
				mu.Unlock()
			}
			time.Sleep(time.Millisecond)
		}
	}()
	run()
	close(stop)
	<-done
	mu.Lock()
	defer mu.Unlock()
	return longest
}

// Acquire used to sense the host (nvidia-smi, docker, systemd) while holding the
// fleet-wide admission lock, so one slow probe stalled every release and
// admission on the host. The lock now covers only the ledger read and write.
func TestBrokerAcquireDoesNotHoldAdmissionLockWhileSensingHost(t *testing.T) {
	ctx := context.Background()
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: filepath.Join(t.TempDir(), "capacity.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	const sensing = 600 * time.Millisecond
	gpu := hostinventory.GPU{Index: 0, UUID: "GPU-x", Name: "test", VRAMBytes: 24 << 30}
	broker := NewBrokerWithSource(store, slowHostSource{delay: sensing, gpus: []hostinventory.GPU{gpu}})

	var hold time.Duration
	started := time.Now()
	hold = admissionLockHold(t, store, func() {
		lease, verdict, err := broker.Acquire(ctx, "test:owner:ram", 1<<30, 0)
		if err != nil || lease == nil || verdict.Kind != "grant" {
			t.Errorf("acquire = %v %+v %v", lease != nil, verdict, err)
			return
		}
		_ = lease.Release(ctx)
	})
	if elapsed := time.Since(started); elapsed < sensing {
		t.Fatalf("acquire returned in %v, before sensing finished", elapsed)
	}
	if hold > sensing/3 {
		t.Fatalf("admission lock unavailable for %v during a %v sensing pass", hold, sensing)
	}
	t.Logf("Acquire: admission lock longest hold %v while sensing took %v", hold, sensing)

	broker2 := NewBrokerWithSource(store, slowHostSource{delay: sensing, gpus: []hostinventory.GPU{gpu}})
	hold = admissionLockHold(t, store, func() {
		lease, verdict, err := broker2.AcquireGPU(ctx, "test:owner:gpu", "GPU-x", 1<<30)
		if err != nil || lease == nil || verdict.Kind != "grant" {
			t.Errorf("acquire GPU = %v %+v %v", lease != nil, verdict, err)
			return
		}
		_ = lease.Release(ctx)
	})
	if hold > sensing/3 {
		t.Fatalf("GPU admission held the lock for %v during a %v sensing pass", hold, sensing)
	}
	t.Logf("AcquireGPU: admission lock longest hold %v while sensing took %v", hold, sensing)
}
