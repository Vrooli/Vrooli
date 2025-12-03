package migrations

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/testutil"
)

// mockEnvRenderer implements EnvRenderer for testing.
type mockEnvRenderer struct{}

func (m *mockEnvRenderer) RenderValue(input string) string {
	return input
}

func (m *mockEnvRenderer) RenderArgs(args []string) []string {
	return args
}

// mockLogProvider implements LogProvider for testing.
type mockLogProvider struct {
	file infra.File
	err  error
}

func (m *mockLogProvider) LogWriter(svc manifest.Service) (infra.File, string, error) {
	return m.file, "/tmp/test.log", m.err
}

// mockFile implements infra.File for testing.
type mockFile struct{}

func (f *mockFile) Write(p []byte) (n int, err error) { return len(p), nil }
func (f *mockFile) Close() error                      { return nil }
func (f *mockFile) Sync() error                       { return nil }

// mockTelemetry implements telemetry.Recorder for testing.
type mockTelemetry struct {
	events []string
}

func (m *mockTelemetry) Record(event string, details map[string]interface{}) error {
	m.events = append(m.events, event)
	return nil
}

func TestTrackerLoad(t *testing.T) {
	tmp := t.TempDir()

	t.Run("returns empty state for missing file", func(t *testing.T) {
		tracker := NewTracker(filepath.Join(tmp, "nonexistent", "migrations.json"), infra.RealFileSystem{})
		state, err := tracker.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if state.Applied == nil {
			t.Error("Load() returned nil Applied map")
		}
		if len(state.Applied) != 0 {
			t.Errorf("Load() Applied = %v, want empty", state.Applied)
		}
	})

	t.Run("loads existing state", func(t *testing.T) {
		migPath := filepath.Join(tmp, "migrations.json")
		state := State{
			AppVersion: "1.0.0",
			Applied: map[string][]string{
				"api": {"v1", "v2"},
			},
		}
		data, _ := json.Marshal(state)
		if err := os.WriteFile(migPath, data, 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		tracker := NewTracker(migPath, infra.RealFileSystem{})
		got, err := tracker.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got.AppVersion != "1.0.0" {
			t.Errorf("AppVersion = %q, want %q", got.AppVersion, "1.0.0")
		}
		if len(got.Applied["api"]) != 2 {
			t.Errorf("Applied[api] = %v, want 2 items", got.Applied["api"])
		}
	})
}

func TestTrackerPersist(t *testing.T) {
	tmp := t.TempDir()
	migPath := filepath.Join(tmp, "subdir", "migrations.json")

	tracker := NewTracker(migPath, infra.RealFileSystem{})
	state := State{
		AppVersion: "2.0.0",
		Applied: map[string][]string{
			"api":    {"v1", "v2", "v3"},
			"worker": {"v1"},
		},
	}

	if err := tracker.Persist(state); err != nil {
		t.Fatalf("Persist() error = %v", err)
	}

	// Verify file was created with correct content.
	data, err := os.ReadFile(migPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got State
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.AppVersion != "2.0.0" {
		t.Errorf("persisted AppVersion = %q, want %q", got.AppVersion, "2.0.0")
	}
	if len(got.Applied["api"]) != 3 {
		t.Errorf("persisted Applied[api] = %v, want 3 items", got.Applied["api"])
	}
}

func TestPhase(t *testing.T) {
	tests := []struct {
		name           string
		savedVersion   string
		currentVersion string
		want           string
	}{
		{"first install", "", "1.0.0", "first_install"},
		{"upgrade", "1.0.0", "2.0.0", "upgrade"},
		{"current version", "1.0.0", "1.0.0", "current"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := State{AppVersion: tt.savedVersion}
			got := Phase(state, tt.currentVersion)
			if got != tt.want {
				t.Errorf("Phase() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShouldRun(t *testing.T) {
	tests := []struct {
		name   string
		runOn  string
		phase  string
		expect bool
	}{
		{"always on first_install", "always", "first_install", true},
		{"always on upgrade", "always", "upgrade", true},
		{"always on current", "always", "current", true},
		{"first_install on first_install", "first_install", "first_install", true},
		{"first_install on upgrade", "first_install", "upgrade", false},
		{"first_install on current", "first_install", "current", false},
		{"upgrade on first_install", "upgrade", "first_install", false},
		{"upgrade on upgrade", "upgrade", "upgrade", true},
		{"upgrade on current", "upgrade", "current", false},
		{"empty defaults to always", "", "upgrade", true},
		{"unknown run_on", "invalid", "upgrade", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := manifest.Migration{RunOn: tt.runOn}
			got := ShouldRun(m, tt.phase)
			if got != tt.expect {
				t.Errorf("ShouldRun() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestBuildAppliedSet(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
		check    string
		exists   bool
	}{
		{"empty list", []string{}, "v1", false},
		{"version exists", []string{"v1", "v2", "v3"}, "v2", true},
		{"version missing", []string{"v1", "v2", "v3"}, "v4", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := BuildAppliedSet(tt.versions)
			if set[tt.check] != tt.exists {
				t.Errorf("BuildAppliedSet()[%q] = %v, want %v", tt.check, set[tt.check], tt.exists)
			}
		})
	}
}

func TestExecutorEnsureAppVersionRecorded(t *testing.T) {
	tmp := t.TempDir()
	migPath := filepath.Join(tmp, "migrations.json")

	mockFS := testutil.NewMockFileSystem()
	tracker := NewTracker(migPath, mockFS)
	telem := &mockTelemetry{}

	exec := NewExecutor(
		ExecutorConfig{
			BundlePath: tmp,
			AppVersion: "3.0.0",
			Tracker:    tracker,
			Telemetry:  telem,
		},
		&mockEnvRenderer{},
		&mockLogProvider{},
	)
	exec.SetState(NewState())

	// Run with no migrations should still record version.
	svc := manifest.Service{ID: "test"}
	bin := manifest.Binary{}
	if err := exec.Run(context.Background(), svc, bin, nil); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	state := exec.State()
	if state.AppVersion != "3.0.0" {
		t.Errorf("State().AppVersion = %q, want %q", state.AppVersion, "3.0.0")
	}
}

func TestExecutorRunMigrations(t *testing.T) {
	tmp := t.TempDir()
	migPath := filepath.Join(tmp, "migrations.json")

	mockFS := testutil.NewMockFileSystem()
	mockProc := testutil.NewMockProcess(12345)
	mockProcRunner := testutil.NewMockProcessRunner()
	mockProcRunner.SetProcesses([]*testutil.MockProcess{mockProc})

	tracker := NewTracker(migPath, mockFS)
	telem := &mockTelemetry{}

	exec := NewExecutor(
		ExecutorConfig{
			BundlePath: tmp,
			AppVersion: "1.0.0",
			Tracker:    tracker,
			ProcRunner: mockProcRunner,
			Telemetry:  telem,
		},
		&mockEnvRenderer{},
		&mockLogProvider{file: &mockFile{}},
	)
	exec.SetState(NewState())

	// Create a service with a migration.
	svc := manifest.Service{
		ID: "test",
		Migrations: []manifest.Migration{
			{Version: "v1", Command: []string{"echo", "hello"}, RunOn: "always"},
		},
	}
	bin := manifest.Binary{}

	// Exit the mock process successfully.
	go func() {
		mockProc.Exit(nil)
	}()

	if err := exec.Run(context.Background(), svc, bin, nil); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	state := exec.State()
	if len(state.Applied["test"]) != 1 {
		t.Errorf("Applied[test] = %v, want 1 migration", state.Applied["test"])
	}
	if state.Applied["test"][0] != "v1" {
		t.Errorf("Applied[test][0] = %q, want %q", state.Applied["test"][0], "v1")
	}
}

func TestExecutorSkipsAppliedMigrations(t *testing.T) {
	tmp := t.TempDir()
	migPath := filepath.Join(tmp, "migrations.json")

	mockFS := testutil.NewMockFileSystem()
	tracker := NewTracker(migPath, mockFS)
	telem := &mockTelemetry{}

	exec := NewExecutor(
		ExecutorConfig{
			BundlePath: tmp,
			AppVersion: "1.0.0",
			Tracker:    tracker,
			Telemetry:  telem,
		},
		&mockEnvRenderer{},
		&mockLogProvider{},
	)

	// Set up state with already applied migration.
	state := NewState()
	state.Applied["test"] = []string{"v1"}
	exec.SetState(state)

	// Service has the same migration.
	svc := manifest.Service{
		ID: "test",
		Migrations: []manifest.Migration{
			{Version: "v1", Command: []string{"echo", "hello"}, RunOn: "always"},
		},
	}
	bin := manifest.Binary{}

	// This should not try to run any process since v1 is already applied.
	if err := exec.Run(context.Background(), svc, bin, nil); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestExecutorMigrationMissingVersion(t *testing.T) {
	tmp := t.TempDir()
	migPath := filepath.Join(tmp, "migrations.json")

	mockFS := testutil.NewMockFileSystem()
	tracker := NewTracker(migPath, mockFS)
	telem := &mockTelemetry{}

	exec := NewExecutor(
		ExecutorConfig{
			BundlePath: tmp,
			AppVersion: "1.0.0",
			Tracker:    tracker,
			Telemetry:  telem,
		},
		&mockEnvRenderer{},
		&mockLogProvider{},
	)
	exec.SetState(NewState())

	svc := manifest.Service{
		ID: "test",
		Migrations: []manifest.Migration{
			{Version: "", Command: []string{"echo", "hello"}}, // Missing version
		},
	}
	bin := manifest.Binary{}

	err := exec.Run(context.Background(), svc, bin, nil)
	if err == nil {
		t.Fatal("Run() expected error for missing version")
	}
}

func TestExecutorMigrationNoCommand(t *testing.T) {
	tmp := t.TempDir()
	migPath := filepath.Join(tmp, "migrations.json")

	mockFS := testutil.NewMockFileSystem()
	tracker := NewTracker(migPath, mockFS)
	telem := &mockTelemetry{}

	exec := NewExecutor(
		ExecutorConfig{
			BundlePath: tmp,
			AppVersion: "1.0.0",
			Tracker:    tracker,
			Telemetry:  telem,
		},
		&mockEnvRenderer{},
		&mockLogProvider{},
	)
	exec.SetState(NewState())

	svc := manifest.Service{
		ID: "test",
		Migrations: []manifest.Migration{
			{Version: "v1", Command: []string{}}, // No command
		},
	}
	bin := manifest.Binary{}

	err := exec.Run(context.Background(), svc, bin, nil)
	if err == nil {
		t.Fatal("Run() expected error for empty command")
	}
}

func TestExecutorMigrationFailure(t *testing.T) {
	tmp := t.TempDir()
	migPath := filepath.Join(tmp, "migrations.json")

	mockFS := testutil.NewMockFileSystem()
	mockProc := testutil.NewMockProcess(12345)
	mockProcRunner := testutil.NewMockProcessRunner()
	mockProcRunner.SetProcesses([]*testutil.MockProcess{mockProc})

	tracker := NewTracker(migPath, mockFS)
	telem := &mockTelemetry{}

	exec := NewExecutor(
		ExecutorConfig{
			BundlePath: tmp,
			AppVersion: "1.0.0",
			Tracker:    tracker,
			ProcRunner: mockProcRunner,
			Telemetry:  telem,
		},
		&mockEnvRenderer{},
		&mockLogProvider{file: &mockFile{}},
	)
	exec.SetState(NewState())

	svc := manifest.Service{
		ID: "test",
		Migrations: []manifest.Migration{
			{Version: "v1", Command: []string{"false"}, RunOn: "always"},
		},
	}
	bin := manifest.Binary{}

	// Exit the mock process with an error.
	go func() {
		mockProc.Exit(errors.New("migration failed"))
	}()

	err := exec.Run(context.Background(), svc, bin, nil)
	if err == nil {
		t.Fatal("Run() expected error for failed migration")
	}

	// Check telemetry recorded failure.
	hasFailure := false
	for _, e := range telem.events {
		if e == "migration_failed" {
			hasFailure = true
			break
		}
	}
	if !hasFailure {
		t.Error("expected migration_failed telemetry event")
	}
}
