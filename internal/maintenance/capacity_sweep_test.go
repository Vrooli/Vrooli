package maintenance

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/testenv"
)

// seedCapacityClaim opens a temp ledger, records one resident VRAM claim with a
// tiny TTL (so it is immediately past its heartbeat deadline), and returns the
// db path + the claim id.
func seedCapacityClaim(t *testing.T, owner string) (string, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	store, err := capacity.NewSQLiteStore(context.Background(), capacity.Config{DBPath: dbPath})
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()
	gpu := 0
	created, err := store.CreateClaim(context.Background(), capacity.CapacityClaim{
		OwnerKind:    capacity.OwnerKindResource,
		OwnerID:      owner,
		ResourceKind: capacity.ResourceKindVRAM,
		GPUIndex:     &gpu,
		AmountBytes:  3 << 30,
		Priority:     capacity.PriorityService,
	}, time.Nanosecond)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	return dbPath, created.ClaimID
}

// withCapacitySweepSeams swaps the injectable seams for the duration of a test
// and restores them after.
func withCapacitySweepSeams(t *testing.T, dbPath string, snap hostinventory.Snapshot, snapErr error, attr capacity.Attributor, now time.Time) {
	t.Helper()
	origOpen, origSnap, origAttr, origNow := openCapacityStoreFn, capacitySnapshotFn, capacityAttributorFn, capacityNowFn
	openCapacityStoreFn = func(ctx context.Context, _ string) (capacitySweepStore, error) {
		return capacity.NewSQLiteStore(ctx, capacity.Config{DBPath: dbPath})
	}
	capacitySnapshotFn = func(context.Context) (hostinventory.Snapshot, error) { return snap, snapErr }
	capacityAttributorFn = func() capacity.Attributor { return attr }
	capacityNowFn = func() time.Time { return now }
	t.Cleanup(func() {
		openCapacityStoreFn, capacitySnapshotFn, capacityAttributorFn, capacityNowFn = origOpen, origSnap, origAttr, origNow
	})
}

// The maintenance pass expires a dead resident's claim and surfaces it as a
// StopReport item.
func TestSweepCapacityClaimsExpiresDeadResident(t *testing.T) {
	ctx := context.Background()
	dbPath, claimID := seedCapacityClaim(t, "kyutai-stt")
	// A complete, successful empty process probe proves absence.
	now := time.Now() // seed TTL is one nanosecond; store and probe use the real clock.
	snapshot := hostinventory.Snapshot{ProbeStatuses: map[string]string{"nvidia_gpu": "ok", "nvidia_gpu_processes": "ok"}, FieldProvenance: map[string]hostinventory.Provenance{"gpu_processes": {ObservedAt: now}}}
	withCapacitySweepSeams(t, dbPath, snapshot, nil, fakeMaintAttributor{}, now)

	c := NewController(t.TempDir(), t.TempDir())
	items, err := c.sweepCapacityClaims(ctx)
	if err != nil {
		t.Fatalf("sweepCapacityClaims() error = %v", err)
	}
	if len(items.Observed) != 1 || items.Observed[0].Name != "kyutai-stt" || len(items.Stopped) != 0 {
		t.Fatalf("items = %+v, want one expired item for kyutai-stt (claim %s)", items, claimID)
	}
}

// A sensing failure is a clean no-op: the claim is NOT expired even though its
// deadline lapsed (never expire what we cannot verify on the GPU).
func TestSweepCapacityClaimsSensingDownNoExpiry(t *testing.T) {
	ctx := context.Background()
	dbPath, claimID := seedCapacityClaim(t, "whisper")
	withCapacitySweepSeams(t, dbPath, hostinventory.Snapshot{}, errMaintSensing, fakeMaintAttributor{}, time.Now().Add(time.Hour))

	c := NewController(t.TempDir(), t.TempDir())
	items, err := c.sweepCapacityClaims(ctx)
	if err != nil {
		t.Fatalf("sweepCapacityClaims() error = %v", err)
	}
	if len(items.Stopped) != 0 {
		t.Fatalf("items = %+v, want none (sensing down)", items)
	}
	// Confirm the claim survived.
	store, err := capacity.NewSQLiteStore(ctx, capacity.Config{DBPath: dbPath})
	if err != nil {
		t.Fatalf("reopen store error = %v", err)
	}
	defer store.Close()
	got, err := store.GetClaim(ctx, claimID)
	if err != nil {
		t.Fatalf("GetClaim() error = %v", err)
	}
	if got.Status != capacity.StatusGranted {
		t.Errorf("status = %q, want granted (sensing-down must not expire)", got.Status)
	}
}

// A resident claim that already has sampled GPU usage must be rescued when its
// deadline lapses. This covers native model-server processes whose attribution
// is process-name based rather than container based (the reranker is one such
// resource); observed resident bytes are evidence to refresh, not a reason to
// expire the claim.
func TestSweepCapacityClaimsRefreshesObservedResident(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	store, err := capacity.NewSQLiteStore(ctx, capacity.Config{DBPath: dbPath})
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	observedAt := time.Now().UTC().Add(-time.Hour)
	gpu := 0
	created, err := store.CreateClaim(ctx, capacity.CapacityClaim{
		OwnerKind:     capacity.OwnerKindResource,
		OwnerID:       "reranker",
		ResourceKind:  capacity.ResourceKindVRAM,
		GPUIndex:      &gpu,
		AmountBytes:   3 << 30,
		Priority:      capacity.PriorityService,
		ObservedBytes: 3 << 30,
		ObservedAt:    &observedAt,
	}, time.Nanosecond)
	store.Close()
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	sweepAt := time.Now().UTC()

	withCapacitySweepSeams(t, dbPath, hostinventory.Snapshot{GPUProcesses: []hostinventory.GPUProcess{{
		GPUIndex: 0, PID: 1000, ProcessName: "reranker_linux_amd64", UsedBytes: 3 << 30,
	}}}, nil, fakeMaintAttributor{}, sweepAt)

	c := NewController(t.TempDir(), t.TempDir())
	items, err := c.sweepCapacityClaims(ctx)
	if err != nil {
		t.Fatalf("sweepCapacityClaims() error = %v", err)
	}
	if len(items.Stopped) != 0 {
		t.Fatalf("items = %+v, want no expiry for observed reranker claim %s", items, created.ClaimID)
	}

	store, err = capacity.NewSQLiteStore(ctx, capacity.Config{DBPath: dbPath})
	if err != nil {
		t.Fatalf("reopen store error = %v", err)
	}
	defer store.Close()
	got, err := store.GetClaim(ctx, created.ClaimID)
	if err != nil {
		t.Fatalf("GetClaim() error = %v", err)
	}
	if got.Status != capacity.StatusGranted {
		t.Fatalf("status = %q, want granted", got.Status)
	}
	if got.HeartbeatDeadlineAt == nil || !got.HeartbeatDeadlineAt.After(sweepAt) {
		t.Fatalf("heartbeat deadline = %v, want refreshed beyond sweep time", got.HeartbeatDeadlineAt)
	}
}

// No ledger yet on the host → nothing to sweep, no error.
func TestSweepCapacityClaimsNoLedgerIsNoop(t *testing.T) {
	ctx := context.Background()
	c := NewController(t.TempDir(), t.TempDir())
	items, err := c.sweepCapacityClaims(ctx)
	if err != nil {
		t.Fatalf("sweepCapacityClaims() error = %v", err)
	}
	if len(items.Stopped) != 0 {
		t.Fatalf("items = %+v, want none (no ledger)", items)
	}
}

type fakeMaintAttributor struct{}

func (fakeMaintAttributor) Attribute(context.Context, int) capacity.Attribution {
	return capacity.Attribution{OwnerID: capacity.OwnerUnknown}
}

var errMaintSensing = maintSensingError("nvidia-smi unavailable")

type maintSensingError string

func (e maintSensingError) Error() string { return string(e) }

type maintenanceApplyFunc func(context.Context, string, string, []string) error

func (f maintenanceApplyFunc) Apply(ctx context.Context, owner, verb string, args []string) error {
	return f(ctx, owner, verb, args)
}

type activateOnReadStore struct {
	*capacity.SQLiteStore
	activateID string
	once       bool
}

func (s *activateOnReadStore) GetClaim(ctx context.Context, id string) (capacity.CapacityClaim, error) {
	if !s.once {
		s.once = true
		c, err := s.SQLiteStore.GetClaim(ctx, s.activateID)
		if err != nil {
			return c, err
		}
		if _, err := s.SQLiteStore.ReportActivity(ctx, c.ClaimID, c.Generation, capacity.ActivityActive); err != nil {
			return c, err
		}
	}
	return s.SQLiteStore.GetClaim(ctx, id)
}

func TestCapacityMaintenanceReportsActualOutcomes(t *testing.T) {
	for _, kind := range []string{"unload", "upshift"} {
		for _, mode := range []string{"advisory", "applied", "failed", "debounced", "stale"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				ctx := context.Background()
				clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
				db := filepath.Join(t.TempDir(), "capacity.db")
				store, err := capacity.NewSQLiteStore(ctx, capacity.Config{DBPath: db, Clock: clk})
				if err != nil {
					t.Fatal(err)
				}
				gpu := 0
				amount := int64(8 << 30)
				status := capacity.StatusGranted
				ttl := time.Second
				if kind == "upshift" {
					amount = 4 << 30
					status = capacity.StatusDegraded
					ttl = 0
				}
				if mode == "debounced" {
					status = capacity.StatusDegraded
				}
				c, err := store.CreateClaim(ctx, capacity.CapacityClaim{OwnerKind: "resource", OwnerID: "fixture", ResourceKind: "vram", GPUIndex: &gpu, AmountBytes: amount, PreferredBytes: 8 << 30, FloorBytes: 2 << 30, Priority: capacity.PriorityService, Status: status, IdleUnloadTTL: ttl, DegradeProfile: &capacity.DegradeProfile{Steps: []capacity.DegradeStep{{Label: "large", AmountBytes: 8 << 30}, {Label: "medium", AmountBytes: 4 << 30}, {Label: "small", AmountBytes: 2 << 30}}, Upshift: kind == "upshift", Apply: capacity.DegradeApply{Verb: "fixture", Argv: []string{"{label}"}}}}, time.Hour)
				if err != nil {
					t.Fatal(err)
				}
				store.Close()
				clk.Advance(time.Minute)
				withCapacitySweepSeams(t, db, hostinventory.Snapshot{GPUs: []hostinventory.GPU{{Index: 0, VRAMBytes: 16 << 30, VRAMUsedBytes: uint64(amount)}}}, nil, fakeMaintAttributor{}, clk.Now())
				oldSweep, oldExec, oldEnforce, oldOpen := capacitySweepFn, capacityExecFn, capacityEnforceFn, openCapacityStoreFn
				t.Cleanup(func() {
					capacitySweepFn, capacityExecFn, capacityEnforceFn, openCapacityStoreFn = oldSweep, oldExec, oldEnforce, oldOpen
				})
				capacitySweepFn = func(context.Context, capacity.ClaimRepository, hostinventory.Snapshot, capacity.Attributor, capacity.Policy, time.Time) (capacity.SweepResult, error) {
					return capacity.SweepResult{}, nil
				}
				openCapacityStoreFn = func(ctx context.Context, _ string) (capacitySweepStore, error) {
					s, err := capacity.NewSQLiteStore(ctx, capacity.Config{DBPath: db, Clock: clk})
					if err != nil {
						return nil, err
					}
					if mode == "debounced" {
						if _, err := s.SetPolicyKey(ctx, "degrade_debounce", "2m"); err != nil {
							t.Fatal(err)
						}
					}
					if mode == "stale" {
						return &activateOnReadStore{SQLiteStore: s, activateID: c.ClaimID}, nil
					}
					return s, nil
				}
				capacityEnforceFn = func(capacity.Policy) string {
					if mode == "advisory" {
						return capacity.EnforceAdvisory
					}
					return capacity.EnforceOn
				}
				calls := 0
				capacityExecFn = func() capacity.ApplyExecutor {
					return maintenanceApplyFunc(func(context.Context, string, string, []string) error {
						calls++
						if mode == "failed" {
							return errors.New("desired fake resize failure")
						}
						return nil
					})
				}
				report, err := NewController(t.TempDir(), t.TempDir()).sweepCapacityClaims(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if mode == "failed" {
					if len(report.Failed) != 1 || !strings.Contains(report.Failed[0].Error, "desired fake resize failure") {
						t.Fatalf("failed callback presented as success: %+v", report)
					}
				} else if mode == "applied" && kind == "unload" {
					if len(report.Stopped) != 1 || !strings.Contains(report.Stopped[0].Message, "idle-unloaded") {
						t.Fatalf("applied unload missing: %+v", report)
					}
				} else {
					if len(report.Observed) != 1 {
						t.Fatalf("non-stop outcome not observed: %+v", report)
					}
				}
				wantCalls := 0
				if mode == "failed" || mode == "applied" {
					wantCalls = 1
				}
				if calls != wantCalls {
					t.Fatalf("callbacks=%d want=%d", calls, wantCalls)
				}
				if !(mode == "applied" && kind == "unload") && len(report.Stopped) != 0 {
					t.Fatalf("non-stop outcome counted stopped: %+v", report)
				}
			})
		}
	}
}

func TestCleanStaleLocksDoesNotCountLedgerObservationAsStop(t *testing.T) {
	db, _ := seedCapacityClaim(t, "gone")
	now := time.Now()
	snap := hostinventory.Snapshot{ProbeStatuses: map[string]string{"nvidia_gpu": "ok", "nvidia_gpu_processes": "ok"}, FieldProvenance: map[string]hostinventory.Provenance{"gpu_processes": {ObservedAt: now}}}
	withCapacitySweepSeams(t, db, snap, nil, fakeMaintAttributor{}, now)
	stubListenerSnapshot(t, true, nil)
	report, err := NewController(t.TempDir(), t.TempDir()).CleanStaleLocks()
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Stopped) != 0 || len(report.Failed) != 0 || len(report.Observed) != 1 || !strings.Contains(report.Message, "Stopped 0 targets, 0 failed") {
		t.Fatalf("ledger observation became physical stop: %+v", report)
	}
}
