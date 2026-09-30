package system

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

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

func TestSwapCheck_PagingRateMatrix(t *testing.T) {
	cases := []struct {
		name string
		used int64
		rate float64
		err  error
		want checks.Status
	}{
		{"low usage and quiet", 20, 0, nil, checks.StatusOK},
		{"low usage and active", 20, 2, nil, checks.StatusWarning},
		{"low usage and high", 20, 128, nil, checks.StatusCritical},
		{"warning usage and quiet", 60, 0, nil, checks.StatusWarning},
		{"critical usage and quiet", 90, 0, nil, checks.StatusCritical},
		{"rate unavailable", 20, 0, context.DeadlineExceeded, checks.StatusWarning},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &mockProcReader{memInfo: &checks.MemInfo{SwapTotal: 100, SwapFree: uint64(100 - tc.used)}}
			c := NewSwapCheck(WithSwapProcReader(reader), WithSwapRateReader(func() (float64, error) { return tc.rate, tc.err }))
			if got := c.Run(context.Background()).Status; got != tc.want {
				t.Fatalf("status = %s, want %s", got, tc.want)
			}
		})
	}
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
