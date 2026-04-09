package pipeline

import (
	"context"
	"sync"
	"time"
)

// mockTimeProvider provides a fixed time for deterministic testing.
type mockTimeProvider struct {
	now int64
}

func (m *mockTimeProvider) Now() int64 {
	return m.now
}

// mockStage is a test stage that can be configured to succeed or fail.
type mockStage struct {
	name        string
	shouldFail  bool
	shouldSkip  bool
	executeCh   chan struct{}
	executeTime time.Duration
}

func (s *mockStage) Name() string {
	return s.name
}

func (s *mockStage) Dependencies() []string {
	return nil
}

func (s *mockStage) CanSkip(input *StageInput) bool {
	return s.shouldSkip
}

func (s *mockStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	if s.executeCh != nil {
		close(s.executeCh)
	}

	if s.executeTime > 0 {
		select {
		case <-ctx.Done():
			return &StageResult{
				Stage:       s.name,
				Status:      StatusCancelled,
				CompletedAt: time.Now().Unix(),
			}
		case <-time.After(s.executeTime):
		}
	}

	if s.shouldFail {
		return &StageResult{
			Stage:       s.name,
			Status:      StatusFailed,
			Error:       "mock failure",
			CompletedAt: time.Now().Unix(),
		}
	}

	return &StageResult{
		Stage:       s.name,
		Status:      StatusCompleted,
		CompletedAt: time.Now().Unix(),
	}
}

// mockLogger is a no-op logger for testing.
type mockLogger struct{}

func (m *mockLogger) Info(msg string, args ...interface{})  {}
func (m *mockLogger) Warn(msg string, args ...interface{})  {}
func (m *mockLogger) Error(msg string, args ...interface{}) {}
func (m *mockLogger) Debug(msg string, args ...interface{}) {}

// trackingStage records when it was executed for test verification.
type trackingStage struct {
	name     string
	executed *[]string
	mu       *sync.Mutex
}

func (s *trackingStage) Name() string {
	return s.name
}

func (s *trackingStage) Dependencies() []string {
	return nil
}

func (s *trackingStage) CanSkip(input *StageInput) bool {
	return false
}

func (s *trackingStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	s.mu.Lock()
	*s.executed = append(*s.executed, s.name)
	s.mu.Unlock()

	return &StageResult{
		Stage:       s.name,
		Status:      StatusCompleted,
		CompletedAt: time.Now().Unix(),
	}
}
