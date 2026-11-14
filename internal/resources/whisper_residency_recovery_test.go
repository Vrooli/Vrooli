package resources

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	capacityengine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

func TestWhisperOwnerRecoveryReleasesExactClaimAfterVerifiedUnload(t *testing.T) {
	ctx := context.Background()
	service, store, claim, started, _ := newWhisperRecoveryFixture(t, 128)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	setWhisperRecoveryInventory(&service, nil)
	supervisor := newManagedServiceSupervisor(filepath.Join(t.TempDir(), "missing.json"), filepath.Join(t.TempDir(), "service.log"))
	manifest := ResourceManifest{Name: "whisper"}
	if err := recoverWhisperOpenResidencyWithService(ctx, &Controller{Root: t.TempDir()}, manifest, supervisor, service); err != nil {
		t.Fatalf("recover exact stopped Whisper unload: %v", err)
	}
	reopened, err := service.OpenStore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	after, err := reopened.GetClaim(ctx, claim.ClaimID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != capacityengine.StatusReleased || after.Generation != claim.Generation+1 {
		t.Fatalf("claim after verified owner recovery = status %s generation %d; want released/%d", after.Status, after.Generation, claim.Generation+1)
	}
	history, err := service.ResidencyHistory(ctx, started.Transition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 3 || history[0].EventKind != "started" || history[1].EventKind != "recovered" || history[2].EventKind != "claim_released" {
		t.Fatalf("owner recovery audit events = %+v", history)
	}
}

func TestWhisperOwnerRecoveryAuditsUnknownAndKeepsClaimLocked(t *testing.T) {
	ctx := context.Background()
	service, store, claim, started, _ := newWhisperRecoveryFixture(t, 17)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	setWhisperRecoveryInventory(&service, []hostinventory.GPUProcess{{GPUIndex: 0, PID: 77, ProcessName: "whisper", UsedBytes: 17}})
	supervisor := newManagedServiceSupervisor(filepath.Join(t.TempDir(), "missing.json"), filepath.Join(t.TempDir(), "service.log"))
	err := recoverWhisperOpenResidencyWithService(ctx, &Controller{Root: t.TempDir()}, ResourceManifest{Name: "whisper"}, supervisor, service)
	if err == nil || !strings.Contains(err.Error(), "remains unresolved") {
		t.Fatalf("unknown owner recovery error = %v", err)
	}
	reopened, openErr := service.OpenStore(ctx)
	if openErr != nil {
		t.Fatal(openErr)
	}
	defer reopened.Close()
	after, getErr := reopened.GetClaim(ctx, claim.ClaimID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if after.Status != capacityengine.StatusGranted || after.Generation != claim.Generation {
		t.Fatalf("unknown recovery changed claim = %+v", after)
	}
	history, historyErr := service.ResidencyHistory(ctx, started.Transition.ID)
	if historyErr != nil {
		t.Fatal(historyErr)
	}
	if len(history) != 2 || history[1].EventKind != "recovery_unknown" || history[1].Record.Transition.Observed.Known {
		t.Fatalf("unknown recovery audit events = %+v", history)
	}
}

func TestWhisperOwnerRecoverySensingFailureLeavesStartedTransitionAndClaimLocked(t *testing.T) {
	ctx := context.Background()
	service, store, claim, started, _ := newWhisperRecoveryFixture(t, 17)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	service.Source = capacityengine.StaticSource{Err: errors.New("GPU inventory unavailable")}
	supervisor := newManagedServiceSupervisor(filepath.Join(t.TempDir(), "missing.json"), filepath.Join(t.TempDir(), "service.log"))
	err := recoverWhisperOpenResidencyWithService(ctx, &Controller{Root: t.TempDir()}, ResourceManifest{Name: "whisper"}, supervisor, service)
	if err == nil || !strings.Contains(err.Error(), "GPU inventory unavailable") {
		t.Fatalf("sensing failure recovery error=%v", err)
	}
	reopened, err := service.OpenStore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	after, err := reopened.GetClaim(ctx, claim.ClaimID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != capacityengine.StatusGranted || after.Generation != claim.Generation {
		t.Fatalf("sensing failure changed claim=%+v", after)
	}
	history, err := service.ResidencyHistory(ctx, started.Transition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].EventKind != "started" {
		t.Fatalf("sensing failure fabricated recovery evidence: %+v", history)
	}
}

func TestWhisperOwnerRecoveryHandlesExactTerminalGenerationWithoutSecondRelease(t *testing.T) {
	ctx := context.Background()
	service, store, claim, started, dbPath := newWhisperRecoveryFixture(t, 128)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	terminalAt := time.Now().UTC()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `UPDATE capacity_claims SET status=?, generation=generation+1, observed_bytes=0, observed_peak_bytes=0, observed_at=?, updated_at=? WHERE claim_id=?`, capacityengine.StatusReleased, terminalAt.Format(time.RFC3339Nano), terminalAt.Format(time.RFC3339Nano), claim.ClaimID)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	setWhisperRecoveryInventory(&service, nil)
	supervisor := newManagedServiceSupervisor(filepath.Join(t.TempDir(), "missing.json"), filepath.Join(t.TempDir(), "service.log"))
	if err := recoverWhisperOpenResidencyWithService(ctx, &Controller{Root: t.TempDir()}, ResourceManifest{Name: "whisper"}, supervisor, service); err != nil {
		t.Fatalf("recover exact terminal Whisper claim: %v", err)
	}
	reopened, err := service.OpenStore(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	after, err := reopened.GetClaim(ctx, claim.ClaimID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != capacityengine.StatusReleased || after.Generation != claim.Generation+1 {
		t.Fatalf("terminal claim changed during recovery = status %s generation %d", after.Status, after.Generation)
	}
	history, err := service.ResidencyHistory(ctx, started.Transition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[1].EventKind != "recovered" || history[1].Record.Transition.Observed.Backend != capacityengine.ResidencyBackendUnloaded {
		t.Fatalf("terminal recovery audit events = %+v", history)
	}
}

func newWhisperRecoveryFixture(t *testing.T, initialBytes int64) (capacityapp.Service, *capacityengine.SQLiteStore, capacityengine.CapacityClaim, capacityengine.ResidencyTransitionRecord, string) {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	zeroFloor := int64(0)
	open := func(ctx context.Context) (capacityapp.Store, error) {
		return capacityengine.NewSQLiteStore(ctx, capacityengine.Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
	}
	service := capacityapp.Service{OpenStore: open, Source: capacityengine.StaticSource{}, Attributor: whisperRecoveryAttributor{}}
	store, err := capacityengine.NewSQLiteStore(ctx, capacityengine.Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	model := capacityengine.ModelIdentity{Name: "ggml-medium.bin", Digest: "sha256:medium"}
	startedAt := time.Now().UTC().Add(-time.Second)
	claim, err := store.CreateClaim(ctx, capacityengine.CapacityClaim{
		OwnerKind: capacityengine.OwnerKindResource, OwnerID: "whisper", ResourceKind: capacityengine.ResourceKindVRAM,
		AmountBytes: 1 << 30, PreferredBytes: 1 << 30, FloorBytes: 0, Status: capacityengine.StatusGranted, ActivityState: capacityengine.ActivityIdle,
		ObservedBytes: initialBytes, ObservedAt: &startedAt,
		DegradeProfile: &capacityengine.DegradeProfile{Model: model},
	}, time.Hour)
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	transition := capacityengine.ResidencyTransition{
		ID: "whisper-owner-recovery", Model: model,
		Desired: capacityengine.ResidencyPlacement{Backend: capacityengine.ResidencyBackendUnloaded, Known: true},
		Reason:  "idle_unload", StartedAt: startedAt,
		PreFootprint: capacityengine.ResidencyFootprint{Bytes: initialBytes, Source: capacityengine.FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true},
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	started, err := service.ResidencyBegin(ctx, capacityapp.ResidencyRequest{OwnerID: "whisper", ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation, Transition: transition})
	if err != nil {
		t.Fatal(err)
	}
	storeRaw, err := capacityengine.NewSQLiteStore(ctx, capacityengine.Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	return service, storeRaw, claim, started, dbPath
}

type whisperRecoveryAttributor struct{}

func (whisperRecoveryAttributor) Attribute(_ context.Context, pid int) capacityengine.Attribution {
	return capacityengine.Attribution{PID: pid, OwnerID: "whisper"}
}

type whisperRecoverySource struct {
	processes []hostinventory.GPUProcess
}

func (s whisperRecoverySource) Snapshot(context.Context) (hostinventory.Snapshot, error) {
	return hostinventory.Snapshot{
		GPUs:          []hostinventory.GPU{{Index: 0, Name: "test gpu", Source: "nvidia-smi", VRAMBytes: 1024}},
		GPUProcesses:  s.processes,
		ProbeStatuses: map[string]string{"nvidia_gpu": "ok", "nvidia_gpu_processes": "ok"},
		FieldProvenance: map[string]hostinventory.Provenance{
			"gpu_processes": {ObservedAt: time.Now().UTC()},
		},
	}, nil
}

func setWhisperRecoveryInventory(service *capacityapp.Service, processes []hostinventory.GPUProcess) {
	service.Source = whisperRecoverySource{processes: processes}
}

func TestWhisperRecoveryClaimAcceptsExactTerminalGeneration(t *testing.T) {
	claim := capacityapp.ClaimView{ClaimID: "claim-1", OwnerKind: capacityengine.OwnerKindResource, OwnerID: "whisper", ResourceKind: capacityengine.ResourceKindVRAM, Status: capacityengine.StatusReleased, Generation: 8}
	got, terminal, err := whisperRecoveryClaim(capacityapp.ListOutput{Claims: []capacityapp.ClaimView{claim}}, "whisper", "claim-1", 7)
	if err != nil || got == nil || got.ClaimID != claim.ClaimID || !terminal {
		t.Fatalf("terminal claim = %+v, terminal=%v, err=%v", got, terminal, err)
	}
}

func TestWhisperRecoveryClaimRejectsUnknownTerminalGeneration(t *testing.T) {
	claim := capacityapp.ClaimView{ClaimID: "claim-1", OwnerKind: capacityengine.OwnerKindResource, OwnerID: "whisper", ResourceKind: capacityengine.ResourceKindVRAM, Status: capacityengine.StatusReleased, Generation: 9}
	if _, _, err := whisperRecoveryClaim(capacityapp.ListOutput{Claims: []capacityapp.ClaimView{claim}}, "whisper", "claim-1", 7); err == nil || !strings.Contains(err.Error(), "generation changed unexpectedly") {
		t.Fatalf("unexpected terminal generation error = %v", err)
	}
}

func TestWhisperRecoveryClaimRejectsActiveReplacement(t *testing.T) {
	claims := []capacityapp.ClaimView{
		{ClaimID: "claim-1", OwnerKind: capacityengine.OwnerKindResource, OwnerID: "whisper", ResourceKind: capacityengine.ResourceKindVRAM, Status: capacityengine.StatusReleased, Generation: 8},
		{ClaimID: "claim-2", OwnerKind: capacityengine.OwnerKindResource, OwnerID: "whisper", ResourceKind: capacityengine.ResourceKindVRAM, Status: capacityengine.StatusGranted, Generation: 1},
	}
	if _, _, err := whisperRecoveryClaim(capacityapp.ListOutput{Claims: claims}, "whisper", "claim-1", 7); err == nil || !strings.Contains(err.Error(), "another resource claim is active") {
		t.Fatalf("active replacement error = %v", err)
	}
}
