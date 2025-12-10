package containment

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// --- Mock implementations for seam testing ---

// MockCommandLookup simulates command path lookup.
type MockCommandLookup struct {
	available map[string]bool
	lookups   []string // Records which commands were looked up
}

func NewMockCommandLookup() *MockCommandLookup {
	return &MockCommandLookup{
		available: make(map[string]bool),
	}
}

func (m *MockCommandLookup) SetAvailable(cmd string, available bool) {
	m.available[cmd] = available
}

func (m *MockCommandLookup) LookPath(file string) (string, error) {
	m.lookups = append(m.lookups, file)
	if m.available[file] {
		return "/usr/bin/" + file, nil
	}
	return "", exec.ErrNotFound
}

func (m *MockCommandLookup) GetLookups() []string {
	return m.lookups
}

// MockCommandRunner simulates command execution.
type MockCommandRunner struct {
	results   map[string]error // command name -> result
	runLog    []string         // Records which commands were run
	shouldErr bool             // Global error flag for simplicity
}

func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		results: make(map[string]error),
	}
}

func (m *MockCommandRunner) SetResult(cmd string, err error) {
	m.results[cmd] = err
}

func (m *MockCommandRunner) SetShouldError(shouldErr bool) {
	m.shouldErr = shouldErr
}

func (m *MockCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	fullCmd := name + " " + strings.Join(args, " ")
	m.runLog = append(m.runLog, fullCmd)

	if m.shouldErr {
		return errors.New("mock command error")
	}

	if err, ok := m.results[name]; ok {
		return err
	}
	return nil
}

func (m *MockCommandRunner) GetRunLog() []string {
	return m.runLog
}

// --- Tests ---

func TestDockerProviderSeams(t *testing.T) {
	t.Run("IsAvailable uses CommandLookup seam", func(t *testing.T) {
		lookup := NewMockCommandLookup()
		runner := NewMockCommandRunner()

		// Docker not installed
		lookup.SetAvailable("docker", false)

		provider := NewDockerProvider(
			WithCommandLookup(lookup),
			WithCommandRunner(runner),
		)

		ctx := context.Background()
		available := provider.IsAvailable(ctx)

		if available {
			t.Error("Expected Docker to be unavailable when LookPath fails")
		}

		lookups := lookup.GetLookups()
		if len(lookups) == 0 || lookups[0] != "docker" {
			t.Errorf("Expected LookPath to be called for 'docker', got: %v", lookups)
		}
	})

	t.Run("IsAvailable uses CommandRunner seam when binary exists", func(t *testing.T) {
		lookup := NewMockCommandLookup()
		runner := NewMockCommandRunner()

		// Docker installed but daemon not running
		lookup.SetAvailable("docker", true)
		runner.SetShouldError(true)

		provider := NewDockerProvider(
			WithCommandLookup(lookup),
			WithCommandRunner(runner),
		)

		ctx := context.Background()
		available := provider.IsAvailable(ctx)

		if available {
			t.Error("Expected Docker to be unavailable when daemon is not running")
		}

		runLog := runner.GetRunLog()
		if len(runLog) == 0 {
			t.Error("Expected CommandRunner.Run to be called for 'docker info'")
		}
		if !strings.Contains(runLog[0], "docker") {
			t.Errorf("Expected 'docker info' command, got: %s", runLog[0])
		}
	})

	t.Run("IsAvailable returns true when Docker is fully available", func(t *testing.T) {
		lookup := NewMockCommandLookup()
		runner := NewMockCommandRunner()

		// Docker installed and daemon running
		lookup.SetAvailable("docker", true)
		runner.SetShouldError(false)

		provider := NewDockerProvider(
			WithCommandLookup(lookup),
			WithCommandRunner(runner),
		)

		ctx := context.Background()
		available := provider.IsAvailable(ctx)

		if !available {
			t.Error("Expected Docker to be available when binary exists and daemon runs")
		}
	})
}

func TestDockerProviderDefaults(t *testing.T) {
	t.Run("NewDockerProvider uses secure defaults", func(t *testing.T) {
		provider := NewDockerProvider()
		cfg := provider.GetConfig()
		expectedImage := "ubuntu:22.04" // Default from Config

		if cfg.DockerImage != expectedImage {
			t.Errorf("Expected image %s, got %s", expectedImage, cfg.DockerImage)
		}

		if !cfg.NoNewPrivileges {
			t.Error("Expected NoNewPrivileges to be true by default")
		}

		if !cfg.DropAllCapabilities {
			t.Error("Expected DropAllCapabilities to be true by default")
		}

		if cfg.ReadOnlyRootFS {
			t.Error("Expected ReadOnlyRootFS to be false by default (for writing test files)")
		}
	})

	t.Run("Type returns ContainmentTypeDocker", func(t *testing.T) {
		provider := NewDockerProvider()
		if provider.Type() != ContainmentTypeDocker {
			t.Errorf("Expected type %s, got %s", ContainmentTypeDocker, provider.Type())
		}
	})

	t.Run("Info returns correct security level", func(t *testing.T) {
		provider := NewDockerProvider()
		info := provider.Info()

		if info.Type != ContainmentTypeDocker {
			t.Errorf("Expected type %s, got %s", ContainmentTypeDocker, info.Type)
		}

		if info.SecurityLevel != 7 {
			t.Errorf("Expected security level 7, got %d", info.SecurityLevel)
		}
	})
}

func TestDockerProviderPrepareCommand(t *testing.T) {
	t.Run("PrepareCommand returns error for empty command", func(t *testing.T) {
		provider := NewDockerProvider()
		ctx := context.Background()

		config := ExecutionConfig{
			Command: []string{},
		}

		_, err := provider.PrepareCommand(ctx, config)
		if !errors.Is(err, ErrNoCommand) {
			t.Errorf("Expected ErrNoCommand, got %v", err)
		}
	})

	t.Run("PrepareCommand includes security options", func(t *testing.T) {
		provider := NewDockerProvider()
		ctx := context.Background()

		config := ExecutionConfig{
			WorkingDir: "/tmp/test",
			Command:    []string{"echo", "hello"},
		}

		cmd, err := provider.PrepareCommand(ctx, config)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		args := strings.Join(cmd.Args, " ")

		// Check security options
		if !strings.Contains(args, "--security-opt=no-new-privileges:true") {
			t.Error("Expected --security-opt=no-new-privileges:true in args")
		}
		if !strings.Contains(args, "--cap-drop=ALL") {
			t.Error("Expected --cap-drop=ALL in args")
		}
	})

	t.Run("PrepareCommand mounts working directory", func(t *testing.T) {
		provider := NewDockerProvider()
		ctx := context.Background()

		config := ExecutionConfig{
			WorkingDir: "/home/user/project",
			Command:    []string{"make", "test"},
		}

		cmd, err := provider.PrepareCommand(ctx, config)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		args := strings.Join(cmd.Args, " ")

		if !strings.Contains(args, "-v /home/user/project:/home/user/project") {
			t.Error("Expected working directory to be mounted")
		}
		if !strings.Contains(args, "-w /home/user/project") {
			t.Error("Expected working directory to be set")
		}
	})

	t.Run("PrepareCommand disables network by default", func(t *testing.T) {
		provider := NewDockerProvider()
		ctx := context.Background()

		config := ExecutionConfig{
			Command:       []string{"echo", "hello"},
			NetworkAccess: false,
		}

		cmd, err := provider.PrepareCommand(ctx, config)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		args := strings.Join(cmd.Args, " ")

		if !strings.Contains(args, "--network=none") {
			t.Error("Expected --network=none when NetworkAccess is false")
		}
	})

	t.Run("PrepareCommand enables network when requested", func(t *testing.T) {
		provider := NewDockerProvider()
		ctx := context.Background()

		config := ExecutionConfig{
			Command:       []string{"echo", "hello"},
			NetworkAccess: true,
		}

		cmd, err := provider.PrepareCommand(ctx, config)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		args := strings.Join(cmd.Args, " ")

		if strings.Contains(args, "--network=none") {
			t.Error("Did not expect --network=none when NetworkAccess is true")
		}
	})

	t.Run("PrepareCommand applies resource limits", func(t *testing.T) {
		provider := NewDockerProvider()
		ctx := context.Background()

		config := ExecutionConfig{
			Command:       []string{"echo", "hello"},
			MaxMemoryMB:   512,
			MaxCPUPercent: 100,
		}

		cmd, err := provider.PrepareCommand(ctx, config)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		args := strings.Join(cmd.Args, " ")

		if !strings.Contains(args, "--memory=512m") {
			t.Error("Expected --memory=512m in args")
		}
		if !strings.Contains(args, "--memory-swap=512m") {
			t.Error("Expected --memory-swap=512m in args")
		}
		if !strings.Contains(args, "--cpu-quota=100000") {
			t.Error("Expected --cpu-quota=100000 in args")
		}
	})
}

func TestDockerProviderWithOptions(t *testing.T) {
	t.Run("WithContainmentConfig sets custom config", func(t *testing.T) {
		customCfg := DefaultConfig()
		customCfg.DockerImage = "custom:image"
		customCfg.MaxMemoryMB = 4096
		customCfg.MaxCPUPercent = 400

		provider := NewDockerProvider(WithContainmentConfig(customCfg))
		cfg := provider.GetConfig()

		if cfg.DockerImage != "custom:image" {
			t.Errorf("Expected image 'custom:image', got %s", cfg.DockerImage)
		}
		if cfg.MaxMemoryMB != 4096 {
			t.Errorf("Expected MaxMemoryMB 4096, got %d", cfg.MaxMemoryMB)
		}
		if cfg.MaxCPUPercent != 400 {
			t.Errorf("Expected MaxCPUPercent 400, got %d", cfg.MaxCPUPercent)
		}
	})

	t.Run("WithExtraArgs adds arguments", func(t *testing.T) {
		provider := NewDockerProvider().WithExtraArgs("--privileged", "--user=1000")

		if len(provider.ExtraDockerArgs) != 2 {
			t.Errorf("Expected 2 extra args, got %d", len(provider.ExtraDockerArgs))
		}
	})
}

func TestFallbackProvider(t *testing.T) {
	t.Run("FallbackProvider is always available", func(t *testing.T) {
		provider := NewFallbackProvider()
		ctx := context.Background()

		if !provider.IsAvailable(ctx) {
			t.Error("FallbackProvider should always be available")
		}
	})

	t.Run("FallbackProvider type is None", func(t *testing.T) {
		provider := NewFallbackProvider()

		if provider.Type() != ContainmentTypeNone {
			t.Errorf("Expected type %s, got %s", ContainmentTypeNone, provider.Type())
		}
	})

	t.Run("FallbackProvider has security level 0", func(t *testing.T) {
		provider := NewFallbackProvider()
		info := provider.Info()

		if info.SecurityLevel != 0 {
			t.Errorf("Expected security level 0, got %d", info.SecurityLevel)
		}
	})

	t.Run("FallbackProvider PrepareCommand returns error for empty command", func(t *testing.T) {
		provider := NewFallbackProvider()
		ctx := context.Background()

		config := ExecutionConfig{
			Command: []string{},
		}

		_, err := provider.PrepareCommand(ctx, config)
		if !errors.Is(err, ErrNoCommand) {
			t.Errorf("Expected ErrNoCommand, got %v", err)
		}
	})

	t.Run("FallbackProvider PrepareCommand creates basic command", func(t *testing.T) {
		provider := NewFallbackProvider()
		ctx := context.Background()

		config := ExecutionConfig{
			WorkingDir: "/tmp/test",
			Command:    []string{"echo", "hello"},
		}

		cmd, err := provider.PrepareCommand(ctx, config)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if cmd.Dir != "/tmp/test" {
			t.Errorf("Expected Dir to be '/tmp/test', got %s", cmd.Dir)
		}

		if cmd.Path == "" {
			t.Error("Expected cmd.Path to be set")
		}
	})
}

func TestManager(t *testing.T) {
	t.Run("SelectProvider returns first available provider", func(t *testing.T) {
		lookup := NewMockCommandLookup()
		runner := NewMockCommandRunner()

		// Docker is available
		lookup.SetAvailable("docker", true)
		runner.SetShouldError(false)

		dockerProvider := NewDockerProvider(
			WithCommandLookup(lookup),
			WithCommandRunner(runner),
		)
		fallbackProvider := NewFallbackProvider()

		manager := NewManager([]Provider{dockerProvider}, fallbackProvider)
		ctx := context.Background()

		selected := manager.SelectProvider(ctx)

		if selected.Type() != ContainmentTypeDocker {
			t.Errorf("Expected Docker provider when available, got %s", selected.Type())
		}
	})

	t.Run("SelectProvider returns fallback when no providers available", func(t *testing.T) {
		lookup := NewMockCommandLookup()

		// Docker is not available
		lookup.SetAvailable("docker", false)

		dockerProvider := NewDockerProvider(
			WithCommandLookup(lookup),
		)
		fallbackProvider := NewFallbackProvider()

		manager := NewManager([]Provider{dockerProvider}, fallbackProvider)
		ctx := context.Background()

		selected := manager.SelectProvider(ctx)

		if selected.Type() != ContainmentTypeNone {
			t.Errorf("Expected Fallback provider when Docker unavailable, got %s", selected.Type())
		}
	})

	t.Run("GetStatus includes warnings when no containment available", func(t *testing.T) {
		lookup := NewMockCommandLookup()
		lookup.SetAvailable("docker", false)

		dockerProvider := NewDockerProvider(
			WithCommandLookup(lookup),
		)
		fallbackProvider := NewFallbackProvider()

		manager := NewManager([]Provider{dockerProvider}, fallbackProvider)
		ctx := context.Background()

		status := manager.GetStatus(ctx)

		if status.ActiveProvider != ContainmentTypeNone {
			t.Errorf("Expected ActiveProvider to be None, got %s", status.ActiveProvider)
		}

		if len(status.Warnings) == 0 {
			t.Error("Expected warnings when no containment available")
		}

		hasNoContainmentWarning := false
		for _, w := range status.Warnings {
			if strings.Contains(w, "No containment available") {
				hasNoContainmentWarning = true
				break
			}
		}
		if !hasNoContainmentWarning {
			t.Error("Expected 'No containment available' warning")
		}
	})

	t.Run("ListProviders returns all registered providers", func(t *testing.T) {
		lookup := NewMockCommandLookup()
		lookup.SetAvailable("docker", true)

		dockerProvider := NewDockerProvider(
			WithCommandLookup(lookup),
			WithCommandRunner(NewMockCommandRunner()),
		)
		fallbackProvider := NewFallbackProvider()

		manager := NewManager([]Provider{dockerProvider}, fallbackProvider)
		ctx := context.Background()

		providers := manager.ListProviders(ctx)

		if len(providers) != 2 {
			t.Errorf("Expected 2 providers, got %d", len(providers))
		}
	})
}

func TestManagerInterfaceCompliance(t *testing.T) {
	t.Run("Manager implements ProviderSelector", func(t *testing.T) {
		var _ ProviderSelector = (*Manager)(nil)
	})

	t.Run("DockerProvider implements Provider", func(t *testing.T) {
		var _ Provider = (*DockerProvider)(nil)
	})

	t.Run("FallbackProvider implements Provider", func(t *testing.T) {
		var _ Provider = (*FallbackProvider)(nil)
	})
}
