package runtime_test

import (
	"errors"
	"testing"

	"workspace-sandbox/internal/config"
	driverexec "workspace-sandbox/internal/driver/exec"
	"workspace-sandbox/internal/runtime"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/types"
)

func TestProfileResolver_FallsBackToDefault(t *testing.T) {
	r := &runtime.ProfileResolver{
		Profiles:  map[string]config.IsolationProfile{"full": {ID: "full"}},
		DefaultID: "full",
	}
	got, err := r.Resolve("")
	if err != nil {
		t.Fatalf("Resolve(): %v", err)
	}
	if got.ID != "full" {
		t.Errorf("got %q, want full", got.ID)
	}
}

func TestProfileResolver_FallsBackToBuiltinFull(t *testing.T) {
	r := &runtime.ProfileResolver{
		Profiles: map[string]config.IsolationProfile{"full": {ID: "full"}},
	}
	got, err := r.Resolve("")
	if err != nil {
		t.Fatalf("Resolve(): %v", err)
	}
	if got.ID != "full" {
		t.Errorf("got %q, want full", got.ID)
	}
}

func TestProfileResolver_UnknownReturnsTypedError(t *testing.T) {
	r := &runtime.ProfileResolver{
		Profiles: map[string]config.IsolationProfile{"full": {ID: "full"}},
	}
	_, err := r.Resolve("nope")
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
	var notFound *types.IsolationProfileNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected *IsolationProfileNotFoundError, got %T", err)
	}
}

func TestProfileResolver_EmptySnapshotReturnsTypedError(t *testing.T) {
	r := &runtime.ProfileResolver{}
	_, err := r.Resolve("full")
	if err == nil {
		t.Fatal("expected error when snapshot is empty")
	}
	var notFound *types.IsolationProfileNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected *IsolationProfileNotFoundError, got %T", err)
	}
}

func TestLoadProfiles_BuildsMapFromStore(t *testing.T) {
	store := mocks.NewFakeProfileStore(
		config.IsolationProfile{ID: "A"},
		config.IsolationProfile{ID: "B"},
	)
	snapshot, err := runtime.LoadProfiles(store)
	if err != nil {
		t.Fatalf("LoadProfiles: %v", err)
	}
	if len(snapshot) != 2 {
		t.Errorf("snapshot size = %d, want 2", len(snapshot))
	}
	if _, ok := snapshot["A"]; !ok {
		t.Error("missing A from snapshot")
	}
	if _, ok := snapshot["B"]; !ok {
		t.Error("missing B from snapshot")
	}
}

func TestLoadProfiles_PropagatesListError(t *testing.T) {
	store := mocks.NewFakeProfileStore()
	store.ListErr = errors.New("disk read failed")
	_, err := runtime.LoadProfiles(store)
	if err == nil {
		t.Fatal("expected error from List()")
	}
}

func TestLoadProfiles_RejectsNilStore(t *testing.T) {
	_, err := runtime.LoadProfiles(nil)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

// TestResolveProfile_StartupCacheRejectsUnknown is the contract test
// from the Round 4 plan: a snapshot-built resolver must not see
// post-construction Store mutations. Register A and B; verify C
// raises the typed not-found; then delete A from the store and
// confirm Resolve("A") still returns A from the snapshot.
func TestResolveProfile_StartupCacheRejectsUnknown(t *testing.T) {
	store := mocks.NewFakeProfileStore(
		config.IsolationProfile{ID: "A"},
		config.IsolationProfile{ID: "B"},
	)
	snapshot, err := runtime.LoadProfiles(store)
	if err != nil {
		t.Fatalf("LoadProfiles: %v", err)
	}
	r := &runtime.ProfileResolver{Profiles: snapshot}

	if _, err := r.Resolve("C"); err == nil {
		t.Fatal("expected typed error for unknown profile C")
	} else {
		var notFound *types.IsolationProfileNotFoundError
		if !errors.As(err, &notFound) {
			t.Fatalf("expected *IsolationProfileNotFoundError, got %T", err)
		}
	}

	// Deletion of A from the underlying store must not affect the
	// resolver's snapshot — that is the whole point of the new design.
	if err := store.Delete("A"); err != nil {
		t.Fatalf("store.Delete(A): %v", err)
	}
	got, err := r.Resolve("A")
	if err != nil {
		t.Fatalf("Resolve(A) after delete: %v", err)
	}
	if got.ID != "A" {
		t.Errorf("Resolve(A).ID = %q, want A — snapshot leaked Store mutation", got.ID)
	}
}

// TestProfileResolver_ResolveReturnsCopy confirms callers cannot
// mutate the snapshot through the returned pointer.
func TestProfileResolver_ResolveReturnsCopy(t *testing.T) {
	r := &runtime.ProfileResolver{
		Profiles: map[string]config.IsolationProfile{
			"full": {ID: "full", Name: "original"},
		},
	}
	got, err := r.Resolve("full")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	got.Name = "mutated"

	again, err := r.Resolve("full")
	if err != nil {
		t.Fatalf("Resolve again: %v", err)
	}
	if again.Name != "original" {
		t.Errorf("snapshot leaked mutation: Name = %q, want original", again.Name)
	}
}

func TestApplyResourceLimitDefaults_FillsZeros(t *testing.T) {
	cfg := config.ExecutionConfig{
		DefaultResourceLimits: config.ResourceLimitsConfig{
			MemoryLimitMB: 256,
			CPUTimeSec:    60,
			TimeoutSec:    30,
		},
	}
	got := runtime.ApplyResourceLimitDefaults(driverexec.ResourceLimits{}, cfg)
	if got.MemoryLimitMB != 256 {
		t.Errorf("MemoryLimitMB = %d, want 256", got.MemoryLimitMB)
	}
	if got.CPUTimeSec != 60 {
		t.Errorf("CPUTimeSec = %d, want 60", got.CPUTimeSec)
	}
	if got.TimeoutSec != 30 {
		t.Errorf("TimeoutSec = %d, want 30", got.TimeoutSec)
	}
}

func TestApplyResourceLimitDefaults_ClampsAtMaxes(t *testing.T) {
	cfg := config.ExecutionConfig{
		MaxResourceLimits: config.ResourceLimitsConfig{MemoryLimitMB: 100},
	}
	got := runtime.ApplyResourceLimitDefaults(driverexec.ResourceLimits{MemoryLimitMB: 9999}, cfg)
	if got.MemoryLimitMB != 100 {
		t.Errorf("MemoryLimitMB = %d, want 100 (clamped)", got.MemoryLimitMB)
	}
}
