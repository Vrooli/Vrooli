package bundleruntime

import (
	"context"
	"errors"
	"sync"
	"testing"

	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/migrations"
)

// =============================================================================
// Mock Migration Runner for testing runMigrations
// =============================================================================

// mockMigrationRunner implements migrations.Runner for testing.
type mockMigrationRunner struct {
	mu      sync.Mutex
	state   migrations.State
	runErr  error
	runArgs []mockMigrationRunArgs
}

type mockMigrationRunArgs struct {
	serviceID  string
	binaryPath string
	envCount   int
}

func newMockMigrationRunner() *mockMigrationRunner {
	return &mockMigrationRunner{
		state: migrations.NewState(),
	}
}

func (m *mockMigrationRunner) setRunErr(err error) {
	m.mu.Lock()
	m.runErr = err
	m.mu.Unlock()
}

func (m *mockMigrationRunner) SetState(state migrations.State) {
	m.mu.Lock()
	m.state = state
	m.mu.Unlock()
}

func (m *mockMigrationRunner) Run(ctx context.Context, svc manifest.Service, bin manifest.Binary, baseEnv map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runArgs = append(m.runArgs, mockMigrationRunArgs{
		serviceID:  svc.ID,
		binaryPath: bin.Path,
		envCount:   len(baseEnv),
	})
	return m.runErr
}

func (m *mockMigrationRunner) State() migrations.State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *mockMigrationRunner) runCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.runArgs)
}

func (m *mockMigrationRunner) runArgsAt(n int) mockMigrationRunArgs {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n < len(m.runArgs) {
		return m.runArgs[n]
	}
	return mockMigrationRunArgs{}
}

// Ensure mockMigrationRunner implements migrations.Runner.
var _ migrations.Runner = (*mockMigrationRunner)(nil)

func TestRunMigrations_NilExecutor(t *testing.T) {
	// Create a minimal supervisor without migration executor.
	s := &Supervisor{
		migrationExecutor: nil,
	}

	svc := manifest.Service{ID: "test-service"}
	bin := manifest.Binary{Path: "/bin/test"}
	env := map[string]string{"KEY": "VALUE"}

	err := s.runMigrations(context.Background(), svc, bin, env)
	if err != nil {
		t.Errorf("runMigrations() with nil executor expected nil error, got %v", err)
	}
}

func TestRunMigrations_DelegatesToExecutor(t *testing.T) {
	mockRunner := newMockMigrationRunner()
	s := &Supervisor{
		migrationExecutor: mockRunner,
	}

	svc := manifest.Service{ID: "api"}
	bin := manifest.Binary{Path: "/app/api-binary"}
	env := map[string]string{"DB_URL": "postgres://localhost", "PORT": "8080"}

	err := s.runMigrations(context.Background(), svc, bin, env)
	if err != nil {
		t.Fatalf("runMigrations() unexpected error: %v", err)
	}

	// Verify executor was called with correct arguments.
	if mockRunner.runCallCount() != 1 {
		t.Errorf("expected 1 Run call, got %d", mockRunner.runCallCount())
	}

	args := mockRunner.runArgsAt(0)
	if args.serviceID != "api" {
		t.Errorf("Run() serviceID = %q, want %q", args.serviceID, "api")
	}
	if args.binaryPath != "/app/api-binary" {
		t.Errorf("Run() binaryPath = %q, want %q", args.binaryPath, "/app/api-binary")
	}
	if args.envCount != 2 {
		t.Errorf("Run() envCount = %d, want %d", args.envCount, 2)
	}
}

func TestRunMigrations_SyncsStateAfterRun(t *testing.T) {
	mockRunner := newMockMigrationRunner()

	// Configure executor to return a specific state.
	expectedState := migrations.State{
		AppVersion: "2.0.0",
		Applied: map[string][]string{
			"api":    {"v1", "v2"},
			"worker": {"v1"},
		},
	}
	mockRunner.SetState(expectedState)

	s := &Supervisor{
		migrationExecutor: mockRunner,
		migrations:        migrations.NewState(), // Start with empty state
	}

	svc := manifest.Service{ID: "api"}
	bin := manifest.Binary{Path: "/bin/api"}

	err := s.runMigrations(context.Background(), svc, bin, nil)
	if err != nil {
		t.Fatalf("runMigrations() unexpected error: %v", err)
	}

	// Verify state was synced back to supervisor.
	if s.migrations.AppVersion != "2.0.0" {
		t.Errorf("migrations.AppVersion = %q, want %q", s.migrations.AppVersion, "2.0.0")
	}
	if len(s.migrations.Applied["api"]) != 2 {
		t.Errorf("migrations.Applied[api] = %v, want 2 items", s.migrations.Applied["api"])
	}
	if len(s.migrations.Applied["worker"]) != 1 {
		t.Errorf("migrations.Applied[worker] = %v, want 1 item", s.migrations.Applied["worker"])
	}
}

func TestRunMigrations_PropagatesError(t *testing.T) {
	mockRunner := newMockMigrationRunner()
	mockRunner.setRunErr(errors.New("migration failed: database error"))

	s := &Supervisor{
		migrationExecutor: mockRunner,
	}

	svc := manifest.Service{ID: "api"}
	bin := manifest.Binary{Path: "/bin/api"}

	err := s.runMigrations(context.Background(), svc, bin, nil)
	if err == nil {
		t.Fatal("runMigrations() expected error, got nil")
	}
	if err.Error() != "migration failed: database error" {
		t.Errorf("runMigrations() error = %q, want %q", err.Error(), "migration failed: database error")
	}
}

func TestRunMigrations_DoesNotSyncStateOnError(t *testing.T) {
	mockRunner := newMockMigrationRunner()

	// Configure executor with state that should NOT be synced due to error.
	mockRunner.SetState(migrations.State{
		AppVersion: "2.0.0",
		Applied:    map[string][]string{"api": {"v1"}},
	})
	mockRunner.setRunErr(errors.New("migration failed"))

	initialState := migrations.NewState()
	initialState.AppVersion = "1.0.0"
	s := &Supervisor{
		migrationExecutor: mockRunner,
		migrations:        initialState,
	}

	svc := manifest.Service{ID: "api"}
	bin := manifest.Binary{Path: "/bin/api"}

	_ = s.runMigrations(context.Background(), svc, bin, nil)

	// Verify state was NOT updated due to error.
	if s.migrations.AppVersion != "1.0.0" {
		t.Errorf("migrations.AppVersion = %q, want %q (should remain unchanged)", s.migrations.AppVersion, "1.0.0")
	}
}

func TestRunMigrations_WithEmptyEnv(t *testing.T) {
	mockRunner := newMockMigrationRunner()
	s := &Supervisor{
		migrationExecutor: mockRunner,
	}

	svc := manifest.Service{ID: "worker"}
	bin := manifest.Binary{Path: "/bin/worker"}

	err := s.runMigrations(context.Background(), svc, bin, nil)
	if err != nil {
		t.Fatalf("runMigrations() unexpected error: %v", err)
	}

	args := mockRunner.runArgsAt(0)
	if args.envCount != 0 {
		t.Errorf("Run() envCount = %d, want 0", args.envCount)
	}
}

func TestRunMigrations_WithCancelledContext(t *testing.T) {
	mockRunner := newMockMigrationRunner()
	mockRunner.setRunErr(context.Canceled)

	s := &Supervisor{
		migrationExecutor: mockRunner,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	svc := manifest.Service{ID: "api"}
	bin := manifest.Binary{Path: "/bin/api"}

	err := s.runMigrations(ctx, svc, bin, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("runMigrations() error = %v, want context.Canceled", err)
	}
}
