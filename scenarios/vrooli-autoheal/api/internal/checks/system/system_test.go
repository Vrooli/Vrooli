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
