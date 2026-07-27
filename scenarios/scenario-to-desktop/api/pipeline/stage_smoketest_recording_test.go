package pipeline

import (
	"context"
	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/smoketest/mocks"
	"testing"
	"time"
)

// mockSmokeTestService implements smoketest.Service for testing.
type mockSmokeTestService struct {
	performCalled bool
}

func (m *mockSmokeTestService) PerformSmokeTest(_ context.Context, _, _, _, _ string) {
	m.performCalled = true
}

func (m *mockSmokeTestService) CurrentPlatform() string {
	return "linux"
}

func TestSmokeTestStage_DefaultRecordingConfig(t *testing.T) {
	store := mocks.NewMockStore()
	svc := &mockSmokeTestService{}

	stage := NewSmokeTestStage(
		WithSmokeTestService(svc),
		WithSmokeTestStore(store),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName: "test-app",
		},
		BuildResult: &build.Status{
			PlatformResults: map[string]*build.PlatformResult{
				"linux": {Status: "ready", Artifact: "/tmp/test.AppImage"},
			},
		},
	}

	// Use a short-lived context so the poll loop exits quickly after the store save
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = stage.Execute(ctx, input)

	// Verify the store received a status with RecordingConfig set
	if len(store.Statuses) == 0 {
		t.Fatal("expected store to have a saved status")
	}

	for _, status := range store.Statuses {
		if status.RecordingConfig == nil {
			t.Fatal("expected RecordingConfig to be set")
		}
		if !status.RecordingConfig.Enabled {
			t.Error("expected RecordingConfig.Enabled to be true")
		}
		if status.RecordingConfig.DisplayWidth != 1920 {
			t.Errorf("expected DisplayWidth=1920, got %d", status.RecordingConfig.DisplayWidth)
		}
		if status.RecordingConfig.DisplayHeight != 1080 {
			t.Errorf("expected DisplayHeight=1080, got %d", status.RecordingConfig.DisplayHeight)
		}
		if status.RecordingConfig.FPS != 15 {
			t.Errorf("expected FPS=15, got %d", status.RecordingConfig.FPS)
		}
	}
}

func TestSmokeTestStage_RecordingConfigNotSetWhenSkipped(t *testing.T) {
	store := mocks.NewMockStore()
	svc := &mockSmokeTestService{}

	stage := NewSmokeTestStage(
		WithSmokeTestService(svc),
		WithSmokeTestStore(store),
	)

	input := &StageInput{
		Config: &Config{
			ScenarioName:  "test-app",
			SkipSmokeTest: true,
		},
	}

	result := stage.Execute(context.Background(), input)
	if result.Status != "skipped" {
		t.Errorf("expected skipped, got %s", result.Status)
	}

	// No status should be saved when skipped
	if len(store.Statuses) != 0 {
		t.Error("expected no statuses saved when smoke test is skipped")
	}
}
