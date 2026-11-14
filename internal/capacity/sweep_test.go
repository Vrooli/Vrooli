package capacity

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/testenv"
)

// residentClaim is a resident (resource) VRAM claim for the named owner.
func residentClaim(owner string) CapacityClaim {
	return CapacityClaim{
		OwnerKind:      OwnerKindResource,
		OwnerID:        owner,
		ResourceKind:   ResourceKindVRAM,
		GPUIndex:       gpu(0),
		AmountBytes:    3 << 30,
		PreferredBytes: 3 << 30,
		FloorBytes:     1 << 30,
		Priority:       PriorityService,
	}
}

// A resident claim whose owner is still observed on the GPU is heartbeated, not
// expired — even after its deadline would otherwise have lapsed.
func TestSweepRefreshesObservedResidentClaim(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})

	created, err := store.CreateClaim(ctx, residentClaim("whisper"), DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}

	// Time advances past the original deadline; without a sweep the claim would
	// be expired. whisper is still observed holding VRAM.
	clk.Advance(2 * DefaultHeartbeatTTL)
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "whisper", UsedBytes: 3 * uint64(gib)})
	attr := fakeAttributor{1000: {ContainerName: "/vrooli-whisper-1", OwnerID: "whisper"}}

	result, err := Sweep(ctx, store, snap, attr, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("Sweep() error = %v", err)
	}
	if len(result.Refreshed) != 1 || result.Refreshed[0].ClaimID != created.ClaimID {
		t.Fatalf("refreshed = %+v, want [%s]", result.Refreshed, created.ClaimID)
	}
	if len(result.Expired) != 0 {
		t.Fatalf("expired = %+v, want none (observed owner rescued)", result.Expired)
	}

	got, err := store.GetClaim(ctx, created.ClaimID)
	if err != nil {
		t.Fatalf("GetClaim() error = %v", err)
	}
	if got.Status != StatusGranted {
		t.Errorf("status = %q, want granted (refreshed)", got.Status)
	}
	wantDeadline := clk.Now().Add(DefaultHeartbeatTTL)
	if got.HeartbeatDeadlineAt == nil || !got.HeartbeatDeadlineAt.Equal(wantDeadline) {
		t.Errorf("deadline = %v, want %v", got.HeartbeatDeadlineAt, wantDeadline)
	}
	// Heartbeat must NOT bump the generation (a re-read signal only).
	if got.Generation != created.Generation {
		t.Errorf("generation = %d, want %d (heartbeat must not bump)", got.Generation, created.Generation)
	}
}

// A resident claim whose owner is no longer observed and whose deadline lapsed
// is swept to expired.
func TestSweepExpiresUnobservedResidentClaim(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})

	created, err := store.CreateClaim(ctx, residentClaim("kyutai-stt"), DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	clk.Advance(2 * DefaultHeartbeatTTL)

	// No GPU process attributes to kyutai-stt (container gone).
	snap := snapshotWithProcs()
	snap.FieldProvenance = map[string]hostinventory.Provenance{"gpu_processes": {ObservedAt: clk.Now()}}
	result, err := Sweep(ctx, store, snap, fakeAttributor{}, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("Sweep() error = %v", err)
	}
	if len(result.Refreshed) != 0 {
		t.Fatalf("refreshed = %+v, want none", result.Refreshed)
	}
	if len(result.Expired) != 1 || result.Expired[0].ClaimID != created.ClaimID {
		t.Fatalf("expired = %+v, want [%s]", result.Expired, created.ClaimID)
	}
	got, err := store.GetClaim(ctx, created.ClaimID)
	if err != nil {
		t.Fatalf("GetClaim() error = %v", err)
	}
	if got.Status != StatusExpired {
		t.Errorf("status = %q, want expired", got.Status)
	}
}

// Op-scoped claims own their own claim->run->release lifecycle and are NEVER
// presence-refreshed, even when their owner process is observed.
func TestSweepIgnoresOpScopedClaims(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})

	opClaim := residentClaim("image-tools:job-1")
	opClaim.OwnerKind = OwnerKindOp
	created, err := store.CreateClaim(ctx, opClaim, DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	clk.Advance(2 * DefaultHeartbeatTTL)

	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 2000, ProcessName: "sd", UsedBytes: 3 * uint64(gib)})
	attr := fakeAttributor{2000: {ContainerName: "/vrooli-image-tools-1", OwnerID: "image-tools"}}

	result, err := Sweep(ctx, store, snap, attr, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("Sweep() error = %v", err)
	}
	if len(result.Refreshed) != 0 {
		t.Fatalf("refreshed = %+v, want none (op claims not presence-refreshed)", result.Refreshed)
	}
	// The op claim's deadline lapsed and it was not refreshed, so it expires.
	if len(result.Expired) != 1 || result.Expired[0].ClaimID != created.ClaimID {
		t.Fatalf("expired = %+v, want [%s]", result.Expired, created.ClaimID)
	}
}

// Sweep with a nil attributor never panics and refreshes nothing.
func TestSweepNilAttributorSafe(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	if _, err := store.CreateClaim(ctx, residentClaim("whisper"), DefaultHeartbeatTTL); err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	snap := snapshotWithProcs(hostinventory.GPUProcess{GPUIndex: 0, PID: 1000, ProcessName: "whisper", UsedBytes: 3 * uint64(gib)})
	result, err := Sweep(ctx, store, snap, nil, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("Sweep() error = %v", err)
	}
	if len(result.Refreshed) != 0 {
		t.Errorf("refreshed = %+v, want none (nil attributor => unknown owners)", result.Refreshed)
	}
}

func TestSweepUnknownProcessProbeDoesNotExpireResidentOrEraseUsage(t *testing.T) {
	for _, status := range []string{"failed", "incomplete", ""} {
		t.Run(status, func(t *testing.T) {
			ctx := context.Background()
			clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
			store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk}) })
			c, err := store.CreateClaim(ctx, residentClaim("whisper"), time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			c, err = store.RecordObserved(ctx, c.ClaimID, 2*gib, 3*gib, clk.Now(), nil)
			if err != nil {
				t.Fatal(err)
			}
			clk.Advance(2 * time.Minute)
			snap := snapshotWithProcs()
			snap.ProbeStatuses = map[string]string{"nvidia_gpu": "ok", "nvidia_gpu_processes": status}
			result, err := Sweep(ctx, store, snap, fakeAttributor{}, DefaultPolicy(), clk.Now())
			if err != nil {
				t.Fatal(err)
			}
			got, err := store.GetClaim(ctx, c.ClaimID)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Expired) != 0 || got.Status != StatusGranted || got.ObservedBytes != 2*gib || !got.ObservedAt.Equal(*c.ObservedAt) {
				t.Fatalf("unknown probe expired/zero-sampled resident: result=%+v claim=%+v", result, got)
			}
		})
	}
}

func TestSweepAmbiguousAndUnknownOwnerIsNotAbsence(t *testing.T) {
	for _, mode := range []string{"duplicate", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
			store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk}) })
			c, err := store.CreateClaim(ctx, residentClaim("whisper"), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "duplicate" {
				if _, err := store.CreateClaim(ctx, residentClaim("whisper"), time.Second); err != nil {
					t.Fatal(err)
				}
			}
			clk.Advance(2 * time.Second)
			snap := snapshotWithProcs(hostinventory.GPUProcess{PID: 1, GPUIndex: 0, UsedBytes: uint64(gib), ProcessName: "python"})
			attr := fakeAttributor{}
			if mode == "duplicate" {
				attr[1] = Attribution{OwnerID: "whisper"}
			}
			result, err := Sweep(ctx, store, snap, attr, DefaultPolicy(), clk.Now())
			if err != nil {
				t.Fatal(err)
			}
			got, err := store.GetClaim(ctx, c.ClaimID)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Expired) != 0 || len(result.Refreshed) != 0 || got.Status != StatusGranted || got.ObservedAt != nil || !result.Deferred.Contains(c.ClaimID) {
				t.Fatalf("ambiguous/unknown presence was invented or erased: %+v / %+v", result, got)
			}
		})
	}
}

func TestSweepAbsenceRequiresFreshClaimRelevantProvenance(t *testing.T) {
	for _, mode := range []string{"missing", "future", "stale", "before-created", "before-heartbeat", "before-activity", "fresh"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
			store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk}) })
			c, err := store.CreateClaim(ctx, residentClaim("whisper"), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			c, err = store.RecordObserved(ctx, c.ClaimID, 2*gib, 3*gib, clk.Now(), nil)
			if err != nil {
				t.Fatal(err)
			}
			created := clk.Now()
			clk.Advance(2 * time.Minute)
			observed := clk.Now()
			switch mode {
			case "missing":
				observed = time.Time{}
			case "future":
				observed = clk.Now().Add(time.Minute)
			case "stale":
				observed = clk.Now().Add(-31 * time.Second)
			case "before-created":
				observed = created.Add(-time.Nanosecond)
			case "before-heartbeat":
				c, err = store.HeartbeatClaim(ctx, c.ClaimID, c.Generation, time.Second)
				if err != nil {
					t.Fatal(err)
				}
				observed = clk.Now().Add(-time.Nanosecond)
			case "before-activity":
				c, err = store.ReportActivity(ctx, c.ClaimID, c.Generation, ActivityActive)
				if err != nil {
					t.Fatal(err)
				}
				observed = clk.Now().Add(-time.Nanosecond)
			}
			clk.Advance(2 * time.Second)
			snap := snapshotWithProcs()
			snap.FieldProvenance = map[string]hostinventory.Provenance{"gpu_processes": {ObservedAt: observed}}
			sampled := SampleObservedUsage(ctx, store, []CapacityClaim{c}, snap, fakeAttributor{}, DefaultPolicy(), clk.Now())
			got, err := store.GetClaim(ctx, c.ClaimID)
			if err != nil {
				t.Fatal(err)
			}
			if mode != "fresh" && (sampled != 0 || got.ObservedBytes != 2*gib || !got.ObservedAt.Equal(*c.ObservedAt)) {
				t.Fatalf("untrusted absence erased observed usage: sampled=%d claim=%+v", sampled, got)
			}
			result, err := Sweep(ctx, store, snap, fakeAttributor{}, DefaultPolicy(), clk.Now())
			if err != nil {
				t.Fatal(err)
			}
			got, err = store.GetClaim(ctx, c.ClaimID)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "fresh" {
				if len(result.Expired) != 1 || got.Status != StatusExpired {
					t.Fatalf("fresh verified absence not expired: %+v %+v", result, got)
				}
			} else if len(result.Expired) != 0 || !result.Deferred.Contains(c.ClaimID) || got.Status != StatusGranted {
				t.Fatalf("untrusted absence expired resident: %+v %+v", result, got)
			}
		})
	}
}

type absenceBarrierStore struct {
	ClaimRepository
	beforeExpiry func()
	beforeSample func()
}

func (s *absenceBarrierStore) ExpireStaleClaims(ctx context.Context, at time.Time, e AbsenceProofs) ([]CapacityClaim, error) {
	if s.beforeExpiry != nil {
		s.beforeExpiry()
	}
	return s.ClaimRepository.ExpireStaleClaims(ctx, at, e)
}
func (s *absenceBarrierStore) RecordObserved(ctx context.Context, id string, observed, peak int64, at time.Time, proof *AbsenceProof) (CapacityClaim, error) {
	if s.beforeSample != nil {
		s.beforeSample()
	}
	return s.ClaimRepository.RecordObserved(ctx, id, observed, peak, at, proof)
}

func TestAbsenceProofCannotCrossConcurrentLivenessOrNewClaim(t *testing.T) {
	for _, mode := range []string{"expire-activity", "expire-heartbeat", "expire-new", "sample-activity", "sample-heartbeat"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
			var dbPath string
			store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
				dbPath = path
				return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk})
			})
			c, err := store.CreateClaim(ctx, residentClaim("whisper"), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			c, err = store.RecordObserved(ctx, c.ClaimID, 2*gib, 3*gib, clk.Now(), nil)
			if err != nil {
				t.Fatal(err)
			}
			clk.Advance(2 * time.Minute)
			probeAt := clk.Now()
			snap := snapshotWithProcs()
			snap.FieldProvenance = map[string]hostinventory.Provenance{"gpu_processes": {ObservedAt: probeAt}}
			target := c
			hit := false
			barrier := func() {
				hit = true
				clk.Advance(time.Second)
				var err error
				switch mode {
				case "expire-activity", "sample-activity":
					target, err = store.ReportActivity(ctx, c.ClaimID, c.Generation, ActivityActive)
				case "expire-heartbeat", "sample-heartbeat":
					target, err = store.HeartbeatClaim(ctx, c.ClaimID, c.Generation, time.Nanosecond)
				case "expire-new":
					// A create begun before sensing commits after the captured cohort.
					delayed, openErr := NewSQLiteStore(ctx, Config{DBPath: dbPath, Clock: testenv.NewClock(probeAt.Add(-time.Minute))})
					if openErr != nil {
						t.Fatal(openErr)
					}
					target, err = delayed.CreateClaim(ctx, residentClaim("new-resident"), time.Nanosecond)
					delayed.Close()
				}
				if err != nil {
					t.Fatal(err)
				}
				clk.Advance(time.Second)
			}
			wrapped := &absenceBarrierStore{ClaimRepository: store}
			if strings.HasPrefix(mode, "sample") {
				wrapped.beforeSample = barrier
				SampleObservedUsage(ctx, wrapped, []CapacityClaim{c}, snap, fakeAttributor{}, DefaultPolicy(), probeAt)
			} else {
				wrapped.beforeExpiry = barrier
				if _, err := Sweep(ctx, wrapped, snap, fakeAttributor{}, DefaultPolicy(), probeAt); err != nil {
					t.Fatal(err)
				}
			}
			if !hit {
				t.Fatal("desired write barrier not reached")
			}
			got, err := store.GetClaim(ctx, target.ClaimID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != StatusGranted {
				t.Fatalf("old absence expired newer liveness/unlisted resident: %+v", got)
			}
			if !strings.HasSuffix(mode, "new") && got.ObservedBytes != 2*gib {
				t.Fatalf("old absence erased newer liveness sample: %+v", got)
			}
		})
	}
}
