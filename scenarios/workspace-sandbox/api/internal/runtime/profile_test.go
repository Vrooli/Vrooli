package runtime

import (
	"errors"
	"testing"

	"workspace-sandbox/internal/config"
	driverexec "workspace-sandbox/internal/driver/exec"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/types"
)

func TestProfileResolver_FallsBackToDefault(t *testing.T) {
	store := mocks.NewFakeProfileStore(config.IsolationProfile{ID: "full"})
	r := &ProfileResolver{Store: store, DefaultID: "full"}
	got, err := r.Resolve("")
	if err != nil {
		t.Fatalf("Resolve(): %v", err)
	}
	if got.ID != "full" {
		t.Errorf("got %q, want full", got.ID)
	}
}

func TestProfileResolver_FallsBackToBuiltinFull(t *testing.T) {
	store := mocks.NewFakeProfileStore(config.IsolationProfile{ID: "full"})
	r := &ProfileResolver{Store: store}
	got, err := r.Resolve("")
	if err != nil {
		t.Fatalf("Resolve(): %v", err)
	}
	if got.ID != "full" {
		t.Errorf("got %q, want full", got.ID)
	}
}

func TestProfileResolver_UnknownReturnsTypedError(t *testing.T) {
	r := &ProfileResolver{Store: mocks.NewFakeProfileStore()}
	_, err := r.Resolve("nope")
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
	var notFound *types.IsolationProfileNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected *IsolationProfileNotFoundError, got %T", err)
	}
}

func TestProfileResolver_NilStoreReturnsTypedError(t *testing.T) {
	r := &ProfileResolver{}
	_, err := r.Resolve("full")
	if err == nil {
		t.Fatal("expected error when store is nil")
	}
	var notFound *types.IsolationProfileNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected *IsolationProfileNotFoundError, got %T", err)
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
	got := ApplyResourceLimitDefaults(driverexec.ResourceLimits{}, cfg)
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
	got := ApplyResourceLimitDefaults(driverexec.ResourceLimits{MemoryLimitMB: 9999}, cfg)
	if got.MemoryLimitMB != 100 {
		t.Errorf("MemoryLimitMB = %d, want 100 (clamped)", got.MemoryLimitMB)
	}
}
