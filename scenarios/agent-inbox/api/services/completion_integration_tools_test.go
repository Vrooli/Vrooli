package services

// =============================================================================
// Integration Tests for CompletionService
// =============================================================================
//
// These tests verify completion-service orchestration helpers used by async
// guidance coverage. Provider tool injection was removed in favor of
// search-hub command context.
//
// Unlike unit tests which test individual functions in isolation, these tests
// verify that the pieces work correctly together.

// createTestCompletionService creates a CompletionService with injected mocks
// for integration testing.
func createTestCompletionService(tracker *AsyncTrackerService) *CompletionService {
	return &CompletionService{
		asyncTracker: tracker,
		// Other dependencies will be nil - tests should only exercise paths
		// that don't need them, or should provide mocks
	}
}
