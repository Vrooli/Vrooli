package pipeline

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"scenario-to-desktop-api/generation"
)

// trackingBuildStore tracks Get/Update calls for test verification.
type trackingBuildStore struct {
	mu          sync.RWMutex
	statuses    map[string]*generation.BuildStatus
	getCalls    int
	updateCalls int
}

func newTrackingBuildStore() *trackingBuildStore {
	return &trackingBuildStore{
		statuses: make(map[string]*generation.BuildStatus),
	}
}

func (s *trackingBuildStore) Create(buildID string) *generation.BuildStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := &generation.BuildStatus{
		BuildID: buildID,
		Status:  "building",
	}
	s.statuses[buildID] = status
	return status
}

func (s *trackingBuildStore) Get(buildID string) (*generation.BuildStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getCalls++
	status, ok := s.statuses[buildID]
	if !ok {
		return nil, false
	}
	// Return a copy to prevent external modification
	copy := *status
	return &copy, true
}

func (s *trackingBuildStore) Update(buildID string, fn func(*generation.BuildStatus)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateCalls++
	if status, ok := s.statuses[buildID]; ok {
		fn(status)
	}
}

func (s *trackingBuildStore) GetCalls() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getCalls
}

func TestWaitForGeneration_PollsStoreNotPointer(t *testing.T) {
	// Setup: mock store that updates status asynchronously
	store := newTrackingBuildStore()

	// Create initial "building" status
	buildID := "test-build-123"
	store.Create(buildID)

	stage := NewGenerateStage(
		WithGenerateBuildStore(store),
		WithGenerateTimeProvider(NewRealTimeProvider()),
	)

	// Simulate async completion after 600ms (after first poll at 500ms)
	go func() {
		time.Sleep(600 * time.Millisecond)
		store.Update(buildID, func(status *generation.BuildStatus) {
			status.Status = "ready"
			status.OutputPath = "/output/path"
		})
	}()

	// Create stale pointer (simulates what QueueBuild returns)
	stalePointer := &generation.BuildStatus{
		BuildID: buildID,
		Status:  "building", // Will never change
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This should succeed by polling the store, not the stale pointer
	path, err := stage.waitForGeneration(ctx, buildID, stalePointer)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if path != "/output/path" {
		t.Errorf("expected /output/path, got %s", path)
	}
	// Should have at least polled once before completion (the store's Get is called each poll)
	if store.GetCalls() < 1 {
		t.Errorf("expected at least 1 Get() call (polling), got %d", store.GetCalls())
	}
}

func TestWaitForGeneration_HandlesFailure(t *testing.T) {
	store := newTrackingBuildStore()
	buildID := "build-fail"
	store.Create(buildID)

	stage := NewGenerateStage(WithGenerateBuildStore(store))

	go func() {
		time.Sleep(50 * time.Millisecond)
		store.Update(buildID, func(status *generation.BuildStatus) {
			status.Status = "failed"
			status.ErrorLog = []string{"template error"}
		})
	}()

	ctx := context.Background()
	_, err := stage.waitForGeneration(ctx, buildID, &generation.BuildStatus{Status: "building"})

	if err == nil || !strings.Contains(err.Error(), "template error") {
		t.Errorf("expected failure with error message, got: %v", err)
	}
}

func TestWaitForGeneration_RespectsContext(t *testing.T) {
	store := newTrackingBuildStore()
	buildID := "build-ctx"
	store.Create(buildID)

	stage := NewGenerateStage(WithGenerateBuildStore(store))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel() // Cancel before completion
	}()

	_, err := stage.waitForGeneration(ctx, buildID, &generation.BuildStatus{Status: "building"})

	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("expected cancellation error, got: %v", err)
	}
}

func TestWaitForGeneration_ImmediateSuccess(t *testing.T) {
	store := newTrackingBuildStore()
	stage := NewGenerateStage(WithGenerateBuildStore(store))

	// If the initial status is already "ready", should return immediately
	initialStatus := &generation.BuildStatus{
		BuildID:    "immediate-123",
		Status:     "ready",
		OutputPath: "/immediate/path",
	}

	ctx := context.Background()
	path, err := stage.waitForGeneration(ctx, "immediate-123", initialStatus)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if path != "/immediate/path" {
		t.Errorf("expected /immediate/path, got %s", path)
	}
	// Should not have polled the store since initial status was ready
	if store.GetCalls() != 0 {
		t.Errorf("expected 0 Get() calls for immediate success, got %d", store.GetCalls())
	}
}

func TestWaitForGeneration_BuildNotFound(t *testing.T) {
	store := newTrackingBuildStore()
	// Don't create any status - simulate missing build

	stage := NewGenerateStage(WithGenerateBuildStore(store))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := stage.waitForGeneration(ctx, "nonexistent-build", &generation.BuildStatus{Status: "building"})

	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestWaitForGeneration_NilStore(t *testing.T) {
	// Without a buildStore, it should fall back to checking the initialStatus pointer
	// This won't work for async updates, but maintains backward compatibility
	stage := NewGenerateStage() // No buildStore

	initialStatus := &generation.BuildStatus{
		BuildID:    "no-store-build",
		Status:     "ready",
		OutputPath: "/no-store/path",
	}

	ctx := context.Background()
	path, err := stage.waitForGeneration(ctx, "no-store-build", initialStatus)
	if err != nil {
		t.Fatalf("expected success with nil store and ready status, got error: %v", err)
	}
	if path != "/no-store/path" {
		t.Errorf("expected /no-store/path, got %s", path)
	}
}

func TestGenerateStage_WithBuildStore(t *testing.T) {
	store := newTrackingBuildStore()
	stage := NewGenerateStage(
		WithGenerateBuildStore(store),
	)

	if stage.buildStore == nil {
		t.Error("expected buildStore to be set")
	}
}

// mockAnalyzer, capturingService, and bundled mode tests are in stage_generate_bundle_test.go
