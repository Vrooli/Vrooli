package system

import (
	"context"
	"testing"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

func TestDiskCheck_Interface(t *testing.T) {
	c := NewDiskCheck()

	if c.ID() != "system-disk" {
		t.Errorf("ID() = %q, want %q", c.ID(), "system-disk")
	}
	if c.Title() == "" {
		t.Error("Title() should not be empty")
	}
	if c.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if c.Importance() == "" {
		t.Error("Importance() should not be empty")
	}
	if c.Category() != checks.CategorySystem {
		t.Errorf("Category() = %v, want %v", c.Category(), checks.CategorySystem)
	}
	if c.IntervalSeconds() <= 0 {
		t.Errorf("IntervalSeconds() = %d, want positive", c.IntervalSeconds())
	}
}

func TestDiskCheck_WithOptions(t *testing.T) {
	c := NewDiskCheck(
		WithPartitions([]string{"/", "/home", "/tmp"}),
		WithDiskThresholds(75, 95),
	)

	if len(c.partitions) != 3 {
		t.Errorf("partitions = %v, want 3 elements", c.partitions)
	}
	if c.warningThreshold != 75 {
		t.Errorf("warningThreshold = %d, want 75", c.warningThreshold)
	}
	if c.criticalThreshold != 95 {
		t.Errorf("criticalThreshold = %d, want 95", c.criticalThreshold)
	}
}

func TestDiskCheck_Run(t *testing.T) {
	c := NewDiskCheck()
	result := c.Run(context.Background())

	if result.CheckID != "system-disk" {
		t.Errorf("CheckID = %q, want %q", result.CheckID, "system-disk")
	}
	if result.Status == "" {
		t.Error("Status should not be empty")
	}
	if result.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestInodeCheck_Interface(t *testing.T) {
	c := NewInodeCheck()

	if c.ID() != "system-inode" {
		t.Errorf("ID() = %q, want %q", c.ID(), "system-inode")
	}
	if c.Category() != checks.CategorySystem {
		t.Errorf("Category() = %v, want %v", c.Category(), checks.CategorySystem)
	}
	// Inode check is Linux-only
	platforms := c.Platforms()
	if len(platforms) != 1 || platforms[0] != platform.Linux {
		t.Errorf("Platforms() = %v, want [Linux]", platforms)
	}
}

func TestInodeCheck_WithOptions(t *testing.T) {
	c := NewInodeCheck(
		WithInodePartitions([]string{"/var"}),
		WithInodeThresholds(70, 85),
	)

	if len(c.partitions) != 1 || c.partitions[0] != "/var" {
		t.Errorf("partitions = %v, want [/var]", c.partitions)
	}
	if c.warningThreshold != 70 {
		t.Errorf("warningThreshold = %d, want 70", c.warningThreshold)
	}
}

func TestSwapCheck_Interface(t *testing.T) {
	c := NewSwapCheck()

	if c.ID() != "system-swap" {
		t.Errorf("ID() = %q, want %q", c.ID(), "system-swap")
	}
	if c.Category() != checks.CategorySystem {
		t.Errorf("Category() = %v, want %v", c.Category(), checks.CategorySystem)
	}
}

func TestSwapCheck_WithOptions(t *testing.T) {
	c := NewSwapCheck(WithSwapThresholds(40, 70))

	if c.warningThreshold != 40 {
		t.Errorf("warningThreshold = %d, want 40", c.warningThreshold)
	}
	if c.criticalThreshold != 70 {
		t.Errorf("criticalThreshold = %d, want 70", c.criticalThreshold)
	}
}

func TestZombieCheck_Interface(t *testing.T) {
	c := NewZombieCheck()

	if c.ID() != "system-zombies" {
		t.Errorf("ID() = %q, want %q", c.ID(), "system-zombies")
	}
	if c.Category() != checks.CategorySystem {
		t.Errorf("Category() = %v, want %v", c.Category(), checks.CategorySystem)
	}
}

func TestZombieCheck_WithOptions(t *testing.T) {
	c := NewZombieCheck(WithZombieThresholds(10, 50))

	if c.warningThreshold != 10 {
		t.Errorf("warningThreshold = %d, want 10", c.warningThreshold)
	}
	if c.criticalThreshold != 50 {
		t.Errorf("criticalThreshold = %d, want 50", c.criticalThreshold)
	}
}

func TestZombieCheck_Run(t *testing.T) {
	c := NewZombieCheck()
	result := c.Run(context.Background())

	if result.CheckID != "system-zombies" {
		t.Errorf("CheckID = %q, want %q", result.CheckID, "system-zombies")
	}
	// On a healthy system, we expect OK or at least a valid status
	if result.Status == "" {
		t.Error("Status should not be empty")
	}
}

func TestPortCheck_Interface(t *testing.T) {
	c := NewPortCheck()

	if c.ID() != "system-ports" {
		t.Errorf("ID() = %q, want %q", c.ID(), "system-ports")
	}
	if c.Category() != checks.CategorySystem {
		t.Errorf("Category() = %v, want %v", c.Category(), checks.CategorySystem)
	}
}

func TestPortCheck_WithOptions(t *testing.T) {
	c := NewPortCheck(WithPortThresholds(60, 80))

	if c.warningThreshold != 60 {
		t.Errorf("warningThreshold = %d, want 60", c.warningThreshold)
	}
	if c.criticalThreshold != 80 {
		t.Errorf("criticalThreshold = %d, want 80", c.criticalThreshold)
	}
}

func TestPortCheck_Run(t *testing.T) {
	c := NewPortCheck()
	result := c.Run(context.Background())

	if result.CheckID != "system-ports" {
		t.Errorf("CheckID = %q, want %q", result.CheckID, "system-ports")
	}
	if result.Status == "" {
		t.Error("Status should not be empty")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := formatBytes(tt.input)
			if got != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestAllChecksImplementInterface ensures all system checks implement the Check interface
func TestAllChecksImplementInterface(t *testing.T) {
	allChecks := []checks.Check{
		NewDiskCheck(),
		NewInodeCheck(),
		NewSwapCheck(),
		NewZombieCheck(),
		NewPortCheck(),
	}

	for _, c := range allChecks {
		t.Run(c.ID(), func(t *testing.T) {
			if c.ID() == "" {
				t.Error("ID() should not be empty")
			}
			if c.Title() == "" {
				t.Error("Title() should not be empty")
			}
			if c.Description() == "" {
				t.Error("Description() should not be empty")
			}
			if c.Importance() == "" {
				t.Error("Importance() should not be empty")
			}
			if c.IntervalSeconds() <= 0 {
				t.Error("IntervalSeconds() should be positive")
			}
		})
	}
}

// TestZombieCheck_Healable verifies ZombieCheck implements HealableCheck
// [REQ:HEAL-ACTION-001]
func TestZombieCheck_Healable(t *testing.T) {
	var _ checks.HealableCheck = (*ZombieCheck)(nil)

	c := NewZombieCheck()
	actions := c.RecoveryActions(nil)

	if len(actions) == 0 {
		t.Error("ZombieCheck should have recovery actions")
	}

	// Should have reap and list actions
	actionIDs := make(map[string]bool)
	for _, a := range actions {
		actionIDs[a.ID] = true
	}
	expectedActions := []string{"reap", "list"}
	for _, expected := range expectedActions {
		if !actionIDs[expected] {
			t.Errorf("ZombieCheck should have %s action", expected)
		}
	}
}

// TestZombieCheck_ReapActionAvailability verifies reap action is only available when zombies exist
func TestZombieCheck_ReapActionAvailability(t *testing.T) {
	c := NewZombieCheck()

	// With no zombies
	resultNoZombies := &checks.Result{
		Details: map[string]interface{}{"zombieCount": 0},
	}
	actions := c.RecoveryActions(resultNoZombies)
	for _, a := range actions {
		if a.ID == "reap" && a.Available {
			t.Error("reap action should not be available when no zombies")
		}
	}

	// With zombies
	resultWithZombies := &checks.Result{
		Details: map[string]interface{}{"zombieCount": 5},
	}
	actions = c.RecoveryActions(resultWithZombies)
	for _, a := range actions {
		if a.ID == "reap" && !a.Available {
			t.Error("reap action should be available when zombies exist")
		}
	}
}

// TestPortCheck_Healable verifies PortCheck implements HealableCheck
// [REQ:HEAL-ACTION-001]
func TestPortCheck_Healable(t *testing.T) {
	var _ checks.HealableCheck = (*PortCheck)(nil)

	c := NewPortCheck()
	actions := c.RecoveryActions(nil)

	if len(actions) == 0 {
		t.Error("PortCheck should have recovery actions")
	}

	// Should have analyze, time-wait, and kill-port actions
	actionIDs := make(map[string]bool)
	for _, a := range actions {
		actionIDs[a.ID] = true
	}
	expectedActions := []string{"analyze", "time-wait", "kill-port"}
	for _, expected := range expectedActions {
		if !actionIDs[expected] {
			t.Errorf("PortCheck should have %s action", expected)
		}
	}
}

// TestPortCheck_TimeWaitActionAvailability verifies time-wait action availability
func TestPortCheck_TimeWaitActionAvailability(t *testing.T) {
	c := NewPortCheck()

	// With low port usage
	resultLow := &checks.Result{
		Details: map[string]interface{}{"usedPercent": 30},
	}
	actions := c.RecoveryActions(resultLow)
	for _, a := range actions {
		if a.ID == "time-wait" && a.Available {
			t.Error("time-wait action should not be available with low port usage")
		}
	}

	// With high port usage
	resultHigh := &checks.Result{
		Details: map[string]interface{}{"usedPercent": 75},
	}
	actions = c.RecoveryActions(resultHigh)
	for _, a := range actions {
		if a.ID == "time-wait" && !a.Available {
			t.Error("time-wait action should be available with high port usage")
		}
	}
}

// TestPortCheck_ExecuteAction verifies action execution
func TestPortCheck_ExecuteAction(t *testing.T) {
	c := NewPortCheck()
	ctx := context.Background()

	// Test unknown action
	result := c.ExecuteAction(ctx, "unknown")
	if result.Success {
		t.Error("unknown action should fail")
	}
	if result.Error == "" {
		t.Error("unknown action should have error message")
	}

	// Test kill-port returns helpful info (doesn't actually kill)
	result = c.ExecuteAction(ctx, "kill-port")
	if !result.Success {
		t.Error("kill-port should return successfully with instructions")
	}
	if result.Output == "" {
		t.Error("kill-port should provide instructions")
	}
}

// --- Mock-based Tests for Better Coverage ---

// mockFSReader implements checks.FileSystemReader for testing.
type mockFSReader struct {
	statfsResult *checks.StatfsResult
	statfsErr    error
}

func (m *mockFSReader) Statfs(path string) (*checks.StatfsResult, error) {
	return m.statfsResult, m.statfsErr
}

func TestDiskCheck_WithMockReader(t *testing.T) {
	t.Run("healthy disk", func(t *testing.T) {
		reader := &mockFSReader{
			statfsResult: &checks.StatfsResult{
				Blocks: 1000000,
				Bfree:  500000,
				Bsize:  4096,
			},
		}
		c := NewDiskCheck(
			WithPartitions([]string{"/"}),
			WithFileSystemReader(reader),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusOK {
			t.Errorf("expected OK status for 50%% usage, got %s", result.Status)
		}
	})

	t.Run("warning threshold", func(t *testing.T) {
		reader := &mockFSReader{
			statfsResult: &checks.StatfsResult{
				Blocks: 1000000,
				Bfree:  150000, // 85% used
				Bsize:  4096,
			},
		}
		c := NewDiskCheck(
			WithPartitions([]string{"/"}),
			WithDiskThresholds(80, 90),
			WithFileSystemReader(reader),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusWarning {
			t.Errorf("expected Warning status for 85%% usage, got %s", result.Status)
		}
	})

	t.Run("critical threshold", func(t *testing.T) {
		reader := &mockFSReader{
			statfsResult: &checks.StatfsResult{
				Blocks: 1000000,
				Bfree:  50000, // 95% used
				Bsize:  4096,
			},
		}
		c := NewDiskCheck(
			WithPartitions([]string{"/"}),
			WithDiskThresholds(80, 90),
			WithFileSystemReader(reader),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusCritical {
			t.Errorf("expected Critical status for 95%% usage, got %s", result.Status)
		}
	})

	t.Run("statfs error", func(t *testing.T) {
		reader := &mockFSReader{
			statfsErr: context.DeadlineExceeded,
		}
		c := NewDiskCheck(
			WithPartitions([]string{"/nonexistent"}),
			WithFileSystemReader(reader),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusCritical {
			t.Errorf("expected Critical status on error, got %s", result.Status)
		}
	})

	t.Run("multiple partitions", func(t *testing.T) {
		callCount := 0
		reader := &mockFSReader{
			statfsResult: &checks.StatfsResult{
				Blocks: 1000000,
				Bfree:  500000,
				Bsize:  4096,
			},
		}
		// Override to track calls
		customReader := &multiCallFSReader{
			results: map[string]*checks.StatfsResult{
				"/":     {Blocks: 1000000, Bfree: 500000, Bsize: 4096},
				"/home": {Blocks: 1000000, Bfree: 100000, Bsize: 4096}, // 90% used
			},
		}
		_ = reader
		_ = callCount

		c := NewDiskCheck(
			WithPartitions([]string{"/", "/home"}),
			WithDiskThresholds(80, 95),
			WithFileSystemReader(customReader),
		)

		result := c.Run(context.Background())
		// Should be warning because /home is at 90%
		if result.Status != checks.StatusWarning {
			t.Errorf("expected Warning status, got %s", result.Status)
		}

		partitions, ok := result.Details["partitions"].([]map[string]interface{})
		if !ok {
			t.Fatal("partitions field missing")
		}
		if len(partitions) != 2 {
			t.Errorf("expected 2 partitions, got %d", len(partitions))
		}
	})
}

// multiCallFSReader returns different results per path
type multiCallFSReader struct {
	results map[string]*checks.StatfsResult
}

func (m *multiCallFSReader) Statfs(path string) (*checks.StatfsResult, error) {
	if result, ok := m.results[path]; ok {
		return result, nil
	}
	return nil, context.DeadlineExceeded
}

// mockPortReader implements checks.PortReader for testing.
type mockPortReader struct {
	portInfo *checks.PortInfo
	err      error
}

func (m *mockPortReader) ReadPortStats() (*checks.PortInfo, error) {
	return m.portInfo, m.err
}

func TestPortCheck_WithMockReader(t *testing.T) {
	t.Run("healthy port usage", func(t *testing.T) {
		reader := &mockPortReader{
			portInfo: &checks.PortInfo{
				TotalPorts:  28232,
				UsedPorts:   1000,
				UsedPercent: 3,
				TimeWait:    50,
			},
		}
		c := NewPortCheck(WithPortReader(reader))

		result := c.Run(context.Background())
		if result.Status != checks.StatusOK {
			t.Errorf("expected OK status, got %s", result.Status)
		}
	})

	t.Run("warning threshold", func(t *testing.T) {
		reader := &mockPortReader{
			portInfo: &checks.PortInfo{
				TotalPorts:  28232,
				UsedPorts:   20000,
				UsedPercent: 75,
				TimeWait:    5000,
			},
		}
		c := NewPortCheck(
			WithPortReader(reader),
			WithPortThresholds(70, 85),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusWarning {
			t.Errorf("expected Warning status for 75%% usage, got %s", result.Status)
		}
	})

	t.Run("critical threshold", func(t *testing.T) {
		reader := &mockPortReader{
			portInfo: &checks.PortInfo{
				TotalPorts:  28232,
				UsedPorts:   25000,
				UsedPercent: 90,
				TimeWait:    10000,
			},
		}
		c := NewPortCheck(
			WithPortReader(reader),
			WithPortThresholds(70, 85),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusCritical {
			t.Errorf("expected Critical status for 90%% usage, got %s", result.Status)
		}
	})

	t.Run("read error", func(t *testing.T) {
		reader := &mockPortReader{
			err: context.DeadlineExceeded,
		}
		c := NewPortCheck(WithPortReader(reader))

		result := c.Run(context.Background())
		if result.Status != checks.StatusCritical {
			t.Errorf("expected Critical status on error, got %s", result.Status)
		}
	})
}

// mockExecutor implements checks.CommandExecutor for testing.
type mockExecutor struct {
	combinedOutput []byte
	combinedErr    error
	output         []byte
	outputErr      error
	runErr         error
}

func (m *mockExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return m.combinedOutput, m.combinedErr
}

func (m *mockExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return m.output, m.outputErr
}

func (m *mockExecutor) Run(ctx context.Context, name string, args ...string) error {
	return m.runErr
}

func TestPortCheck_ExecuteAction_WithMock(t *testing.T) {
	t.Run("analyze success", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutput: []byte("LISTEN 8080 pid=1234"),
		}
		c := NewPortCheck(WithPortExecutor(exec))

		result := c.ExecuteAction(context.Background(), "analyze")
		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if result.Output != "LISTEN 8080 pid=1234" {
			t.Errorf("unexpected output: %s", result.Output)
		}
	})

	t.Run("time-wait success", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutput: []byte("State      Recv-Q Send-Q\nTIME-WAIT  0      0\nTIME-WAIT  0      0"),
		}
		c := NewPortCheck(WithPortExecutor(exec))

		result := c.ExecuteAction(context.Background(), "time-wait")
		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
	})
}

// TestZombieCheck_WithMock tests zombie check with mock executor
func TestZombieCheck_WithMock(t *testing.T) {
	t.Run("no zombies", func(t *testing.T) {
		exec := &mockExecutor{
			combinedOutput: []byte(""),
		}
		c := NewZombieCheck(WithZombieExecutor(exec))

		result := c.Run(context.Background())
		if result.Status != checks.StatusOK {
			t.Errorf("expected OK status, got %s", result.Status)
		}
	})
}

// TestZombieCheck_ExecuteAction tests zombie action execution
func TestZombieCheck_ExecuteAction(t *testing.T) {
	c := NewZombieCheck()

	t.Run("list action", func(t *testing.T) {
		result := c.ExecuteAction(context.Background(), "list")
		if !result.Success {
			t.Errorf("list action should succeed, got error: %s", result.Error)
		}
	})

	t.Run("unknown action", func(t *testing.T) {
		result := c.ExecuteAction(context.Background(), "unknown")
		if result.Success {
			t.Error("unknown action should fail")
		}
	})
}

// =============================================================================
// Inode Check Mock Tests
// =============================================================================

func TestInodeCheck_WithMockReader(t *testing.T) {
	t.Run("healthy inode usage", func(t *testing.T) {
		reader := &mockFSReader{
			statfsResult: &checks.StatfsResult{
				Files: 1000000, // Total inodes
				Ffree: 700000,  // Free inodes (30% used)
			},
		}
		c := NewInodeCheck(
			WithInodePartitions([]string{"/"}),
			WithInodeFileSystemReader(reader),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusOK {
			t.Errorf("expected OK status for 30%% inode usage, got %s", result.Status)
		}
		if result.Metrics == nil {
			t.Fatal("expected metrics to be set")
		}
		if result.Metrics.Score == nil {
			t.Fatal("expected score to be set")
		}
		// Score should be ~70 (100 - 30%)
		if *result.Metrics.Score < 65 || *result.Metrics.Score > 75 {
			t.Errorf("expected score around 70, got %d", *result.Metrics.Score)
		}
	})

	t.Run("warning threshold", func(t *testing.T) {
		reader := &mockFSReader{
			statfsResult: &checks.StatfsResult{
				Files: 1000000,
				Ffree: 150000, // 85% used
			},
		}
		c := NewInodeCheck(
			WithInodePartitions([]string{"/"}),
			WithInodeThresholds(80, 90),
			WithInodeFileSystemReader(reader),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusWarning {
			t.Errorf("expected Warning status for 85%% inode usage, got %s", result.Status)
		}
	})

	t.Run("critical threshold", func(t *testing.T) {
		reader := &mockFSReader{
			statfsResult: &checks.StatfsResult{
				Files: 1000000,
				Ffree: 50000, // 95% used
			},
		}
		c := NewInodeCheck(
			WithInodePartitions([]string{"/"}),
			WithInodeThresholds(80, 90),
			WithInodeFileSystemReader(reader),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusCritical {
			t.Errorf("expected Critical status for 95%% inode usage, got %s", result.Status)
		}
	})

	t.Run("statfs error", func(t *testing.T) {
		reader := &mockFSReader{
			statfsErr: context.DeadlineExceeded,
		}
		c := NewInodeCheck(
			WithInodePartitions([]string{"/nonexistent"}),
			WithInodeFileSystemReader(reader),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusCritical {
			t.Errorf("expected Critical status on error, got %s", result.Status)
		}
	})

	t.Run("multiple partitions worst status", func(t *testing.T) {
		reader := &multiCallFSReader{
			results: map[string]*checks.StatfsResult{
				"/":     {Files: 1000000, Ffree: 700000}, // 30% used (OK)
				"/var":  {Files: 1000000, Ffree: 100000}, // 90% used (Critical)
				"/home": {Files: 1000000, Ffree: 200000}, // 80% used (Warning)
			},
		}
		c := NewInodeCheck(
			WithInodePartitions([]string{"/", "/var", "/home"}),
			WithInodeThresholds(75, 85),
			WithInodeFileSystemReader(reader),
		)

		result := c.Run(context.Background())
		// Should be critical because /var is at 90%
		if result.Status != checks.StatusCritical {
			t.Errorf("expected Critical status (worst case), got %s", result.Status)
		}

		// Check subchecks
		if result.Metrics == nil || len(result.Metrics.SubChecks) != 3 {
			t.Fatalf("expected 3 subchecks, got %v", result.Metrics)
		}
	})

	t.Run("zero inodes filesystem", func(t *testing.T) {
		reader := &mockFSReader{
			statfsResult: &checks.StatfsResult{
				Files: 0, // No inode limit (some filesystems)
				Ffree: 0,
			},
		}
		c := NewInodeCheck(
			WithInodePartitions([]string{"/"}),
			WithInodeFileSystemReader(reader),
		)

		result := c.Run(context.Background())
		// 0 inodes should result in 0% usage
		if result.Status != checks.StatusOK {
			t.Errorf("expected OK status for 0 inodes, got %s", result.Status)
		}
	})
}

// =============================================================================
// Swap Check Mock Tests
// =============================================================================

// mockProcReader implements checks.ProcReader for testing.
type mockProcReader struct {
	memInfo        *checks.MemInfo
	memInfoErr     error
	processes      []checks.ProcessInfo
	processesErr   error
}

func (m *mockProcReader) ReadMeminfo() (*checks.MemInfo, error) {
	return m.memInfo, m.memInfoErr
}

func (m *mockProcReader) ListProcesses() ([]checks.ProcessInfo, error) {
	return m.processes, m.processesErr
}

func TestSwapCheck_WithMockReader(t *testing.T) {
	t.Run("healthy swap usage", func(t *testing.T) {
		reader := &mockProcReader{
			memInfo: &checks.MemInfo{
				SwapTotal: 8388608, // 8GB in KB
				SwapFree:  6291456, // 6GB free (25% used)
			},
		}
		c := NewSwapCheck(WithSwapProcReader(reader))

		result := c.Run(context.Background())
		if result.Status != checks.StatusOK {
			t.Errorf("expected OK status for 25%% swap usage, got %s", result.Status)
		}
		if result.Metrics == nil {
			t.Fatal("expected metrics to be set")
		}
		if result.Metrics.Score == nil {
			t.Fatal("expected score to be set")
		}
		// Score should be ~75 (100 - 25%)
		if *result.Metrics.Score < 70 || *result.Metrics.Score > 80 {
			t.Errorf("expected score around 75, got %d", *result.Metrics.Score)
		}
	})

	t.Run("warning threshold", func(t *testing.T) {
		reader := &mockProcReader{
			memInfo: &checks.MemInfo{
				SwapTotal: 8388608, // 8GB
				SwapFree:  3355443, // 40% free (60% used)
			},
		}
		c := NewSwapCheck(
			WithSwapProcReader(reader),
			WithSwapThresholds(50, 80),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusWarning {
			t.Errorf("expected Warning status for 60%% swap usage, got %s", result.Status)
		}
	})

	t.Run("critical threshold", func(t *testing.T) {
		reader := &mockProcReader{
			memInfo: &checks.MemInfo{
				SwapTotal: 8388608, // 8GB
				SwapFree:  838861,  // 10% free (90% used)
			},
		}
		c := NewSwapCheck(
			WithSwapProcReader(reader),
			WithSwapThresholds(50, 80),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusCritical {
			t.Errorf("expected Critical status for 90%% swap usage, got %s", result.Status)
		}
	})

	t.Run("no swap configured", func(t *testing.T) {
		reader := &mockProcReader{
			memInfo: &checks.MemInfo{
				SwapTotal: 0,
				SwapFree:  0,
			},
		}
		c := NewSwapCheck(WithSwapProcReader(reader))

		result := c.Run(context.Background())
		if result.Status != checks.StatusWarning {
			t.Errorf("expected Warning status when no swap configured, got %s", result.Status)
		}
		if swapConfigured, ok := result.Details["swapConfigured"].(bool); !ok || swapConfigured {
			t.Error("expected swapConfigured=false")
		}
	})

	t.Run("read error", func(t *testing.T) {
		reader := &mockProcReader{
			memInfoErr: context.DeadlineExceeded,
		}
		c := NewSwapCheck(WithSwapProcReader(reader))

		result := c.Run(context.Background())
		if result.Status != checks.StatusCritical {
			t.Errorf("expected Critical status on error, got %s", result.Status)
		}
		if _, ok := result.Details["error"]; !ok {
			t.Error("expected error details")
		}
	})

	t.Run("full swap usage", func(t *testing.T) {
		reader := &mockProcReader{
			memInfo: &checks.MemInfo{
				SwapTotal: 8388608, // 8GB
				SwapFree:  0,       // 0% free (100% used)
			},
		}
		c := NewSwapCheck(WithSwapProcReader(reader))

		result := c.Run(context.Background())
		if result.Status != checks.StatusCritical {
			t.Errorf("expected Critical status for 100%% swap usage, got %s", result.Status)
		}
		// Score should be 0
		if result.Metrics != nil && result.Metrics.Score != nil && *result.Metrics.Score != 0 {
			t.Errorf("expected score 0 for full swap, got %d", *result.Metrics.Score)
		}
	})

	t.Run("details contain all fields", func(t *testing.T) {
		reader := &mockProcReader{
			memInfo: &checks.MemInfo{
				SwapTotal: 8388608,
				SwapFree:  4194304, // 50% used
			},
		}
		c := NewSwapCheck(WithSwapProcReader(reader))

		result := c.Run(context.Background())

		// Verify all expected fields are present
		expectedFields := []string{
			"swapTotalKB", "swapFreeKB", "swapTotalBytes", "swapFreeBytes",
			"swapUsedKB", "swapUsedBytes", "usedPercent", "swapConfigured",
			"warningThreshold", "criticalThreshold",
		}
		for _, field := range expectedFields {
			if _, ok := result.Details[field]; !ok {
				t.Errorf("expected field %s in details", field)
			}
		}
	})
}

// =============================================================================
// Zombie Check Mock Tests (additional scenarios)
// =============================================================================

func TestZombieCheck_WithMock_MultipleZombies(t *testing.T) {
	t.Run("many zombies - warning", func(t *testing.T) {
		// Simulate 15 zombie processes using ProcReader
		processes := make([]checks.ProcessInfo, 15)
		for i := 0; i < 15; i++ {
			processes[i] = checks.ProcessInfo{
				PID:   1000 + i,
				PPid:  1,
				Comm:  "defunct",
				State: "Z",
			}
		}
		reader := &mockProcReader{
			processes: processes,
		}
		c := NewZombieCheck(
			WithZombieProcReader(reader),
			WithZombieThresholds(10, 50),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusWarning {
			t.Errorf("expected Warning status for 15 zombies, got %s", result.Status)
		}
	})

	t.Run("critical zombies", func(t *testing.T) {
		// Simulate 60 zombie processes using ProcReader
		processes := make([]checks.ProcessInfo, 60)
		for i := 0; i < 60; i++ {
			processes[i] = checks.ProcessInfo{
				PID:   1000 + i,
				PPid:  1,
				Comm:  "defunct",
				State: "Z",
			}
		}
		reader := &mockProcReader{
			processes: processes,
		}
		c := NewZombieCheck(
			WithZombieProcReader(reader),
			WithZombieThresholds(10, 50),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusCritical {
			t.Errorf("expected Critical status for 60 zombies, got %s", result.Status)
		}
	})

	t.Run("no zombies", func(t *testing.T) {
		// No zombie processes
		processes := []checks.ProcessInfo{
			{PID: 1, PPid: 0, Comm: "init", State: "S"},
			{PID: 100, PPid: 1, Comm: "bash", State: "S"},
		}
		reader := &mockProcReader{
			processes: processes,
		}
		c := NewZombieCheck(WithZombieProcReader(reader))

		result := c.Run(context.Background())
		if result.Status != checks.StatusOK {
			t.Errorf("expected OK status for no zombies, got %s", result.Status)
		}
	})

	t.Run("proc read error", func(t *testing.T) {
		reader := &mockProcReader{
			processesErr: context.DeadlineExceeded,
		}
		c := NewZombieCheck(WithZombieProcReader(reader))

		result := c.Run(context.Background())
		if result.Status != checks.StatusCritical {
			t.Errorf("expected Critical status on error, got %s", result.Status)
		}
	})

	t.Run("some zombies below threshold", func(t *testing.T) {
		// 3 zombies but threshold is 5
		processes := []checks.ProcessInfo{
			{PID: 1000, PPid: 1, Comm: "defunct1", State: "Z"},
			{PID: 1001, PPid: 1, Comm: "defunct2", State: "Z"},
			{PID: 1002, PPid: 1, Comm: "defunct3", State: "Z"},
			{PID: 100, PPid: 1, Comm: "bash", State: "S"},
		}
		reader := &mockProcReader{
			processes: processes,
		}
		c := NewZombieCheck(
			WithZombieProcReader(reader),
			WithZombieThresholds(5, 20),
		)

		result := c.Run(context.Background())
		if result.Status != checks.StatusOK {
			t.Errorf("expected OK status for zombies below threshold, got %s", result.Status)
		}
		if count, ok := result.Details["zombieCount"].(int); !ok || count != 3 {
			t.Errorf("expected zombieCount=3, got %v", result.Details["zombieCount"])
		}
	})
}
