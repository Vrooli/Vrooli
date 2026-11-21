// Package bootstrap tests for CheckFactory
// [REQ:TEST-SEAM-001] Verify factory implementations work correctly
package bootstrap

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/userconfig"
)

// =============================================================================
// DefaultCheckFactory Tests
// =============================================================================

// TestNewDefaultCheckFactory verifies factory initialization
func TestNewDefaultCheckFactory(t *testing.T) {
	t.Setenv(NetworkTargetEnv, "resolver.test:53")
	t.Setenv(DNSDomainEnv, "resolver.test")
	factory := NewDefaultCheckFactory()

	if factory == nil {
		t.Fatal("NewDefaultCheckFactory() returned nil")
	}
	if factory.networkTarget != "resolver.test:53" {
		t.Errorf("networkTarget = %q, want resolver.test:53", factory.networkTarget)
	}
	if factory.dnsDomain != "resolver.test" {
		t.Errorf("dnsDomain = %q, want resolver.test", factory.dnsDomain)
	}
	if len(factory.criticalScenarios) != 0 {
		t.Error("criticalScenarios must come from the canonical supervision set")
	}
	if len(factory.nonCriticalScenarios) != 0 {
		t.Error("nonCriticalScenarios must come from the canonical supervision set")
	}
	if len(factory.resources) != 0 {
		t.Error("resources must come from the canonical supervision set")
	}
}

// TestNewDefaultCheckFactoryWithOptions verifies option configuration
func TestNewDefaultCheckFactoryWithOptions(t *testing.T) {
	factory := NewDefaultCheckFactoryWithOptions(
		WithNetworkTarget("1.2.3.4:53"),
		WithDNSDomain("custom.com"),
		WithCriticalScenarios([]string{"scenario-a"}),
		WithNonCriticalScenarios([]string{"scenario-b", "scenario-c"}),
		WithResources([]string{"postgres", "redis"}),
	)

	if factory.networkTarget != "1.2.3.4:53" {
		t.Errorf("networkTarget = %q, want %q", factory.networkTarget, "1.2.3.4:53")
	}
	if factory.dnsDomain != "custom.com" {
		t.Errorf("dnsDomain = %q, want %q", factory.dnsDomain, "custom.com")
	}
	if len(factory.criticalScenarios) != 1 || factory.criticalScenarios[0] != "scenario-a" {
		t.Errorf("criticalScenarios = %v, want [scenario-a]", factory.criticalScenarios)
	}
	if len(factory.nonCriticalScenarios) != 2 {
		t.Errorf("nonCriticalScenarios count = %d, want 2", len(factory.nonCriticalScenarios))
	}
	if len(factory.resources) != 2 {
		t.Errorf("resources count = %d, want 2", len(factory.resources))
	}
}

// TestDefaultCheckFactory_CreateInfrastructureChecks verifies infrastructure check creation
func TestDefaultCheckFactory_CreateInfrastructureChecks(t *testing.T) {
	factory := NewDefaultCheckFactory()
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		HasDocker:       true,
		SupportsSystemd: true,
	}

	infraChecks := factory.CreateInfrastructureChecks(caps)

	if len(infraChecks) == 0 {
		t.Fatal("expected infrastructure checks to be created")
	}

	// Verify expected checks are present
	checkIDs := make(map[string]bool)
	for _, check := range infraChecks {
		checkIDs[check.ID()] = true
	}

	expectedIDs := []string{
		"infra-network",
		"infra-dns",
		"infra-docker",
		"infra-cloudflared",
		"infra-rdp",
		"infra-ntp",
		"infra-resolved",
		"infra-certificate",
		"infra-display",
	}

	for _, id := range expectedIDs {
		if !checkIDs[id] {
			t.Errorf("expected check %q to be created", id)
		}
	}
}

func TestNewCheckFactoryFromConfigManager_CloudflaredSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	schemaPath := filepath.Join(tmpDir, "config.schema.json")

	mgr := userconfig.NewManager(configPath, schemaPath)

	cfg := mgr.Get()
	if cfg.Checks == nil {
		cfg.Checks = make(map[string]userconfig.Check)
	}
	cfg.Checks["infra-cloudflared"] = userconfig.Check{
		Settings: &userconfig.CheckSettings{
			TunnelTestURL: "https://app-monitor.example.com/",
		},
	}
	if err := mgr.Update(cfg); err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	factory := NewCheckFactoryFromConfigManager(mgr)
	if factory.cloudflaredExternalURL != "https://app-monitor.example.com/" {
		t.Fatalf("cloudflaredExternalURL = %q, want %q", factory.cloudflaredExternalURL, "https://app-monitor.example.com/")
	}
}

// TestDefaultCheckFactory_CreateSystemChecks verifies system check creation
func TestDefaultCheckFactory_CreateSystemChecks(t *testing.T) {
	factory := NewDefaultCheckFactory()

	systemChecks := factory.CreateSystemChecks()

	if len(systemChecks) == 0 {
		t.Fatal("expected system checks to be created")
	}

	// Verify expected checks are present
	checkIDs := make(map[string]bool)
	for _, check := range systemChecks {
		checkIDs[check.ID()] = true
	}

	expectedIDs := []string{
		"system-disk",
		"system-inode",
		"system-swap",
		"system-zombies",
		"system-ports",
		"system-claude-cache",
	}

	for _, id := range expectedIDs {
		if !checkIDs[id] {
			t.Errorf("expected check %q to be created", id)
		}
	}
}

// TestDefaultCheckFactory_CreateVrooliChecks verifies Vrooli check creation
func TestDefaultCheckFactory_CreateVrooliChecks(t *testing.T) {
	factory := NewDefaultCheckFactory()
	caps := &platform.Capabilities{Platform: platform.Linux}

	vrooliChecks := factory.CreateVrooliChecks(caps)

	if len(vrooliChecks) == 0 {
		t.Fatal("expected Vrooli checks to be created")
	}

	// Count by type. A monitored resource produces one base check, plus one
	// accelerator-placement check when it declares a non-CPU backend, so the
	// assertion is coverage — every monitored resource is represented, and
	// every extra check is a mode-drift check for one of them — rather than a
	// pinned total that breaks whenever a resource declares an accelerator.
	baseChecks := map[string]bool{}
	driftChecks := map[string]bool{}
	scenarioCount := 0

	for _, check := range vrooliChecks {
		id := check.ID()
		switch {
		case strings.HasSuffix(id, "-mode-drift"):
			driftChecks[strings.TrimSuffix(strings.TrimPrefix(id, "resource-"), "-mode-drift")] = true
		case strings.HasPrefix(id, "resource-"):
			baseChecks[strings.TrimPrefix(id, "resource-")] = true
		case strings.HasPrefix(id, "scenario-"):
			scenarioCount++
		}
	}

	monitored, _ := PruneMissingResources(factory.resources)
	for _, name := range monitored {
		if !baseChecks[name] {
			t.Errorf("monitored resource %q has no resource check", name)
		}
	}
	if len(baseChecks) != len(monitored) {
		t.Errorf("resource check count = %d, want %d (monitored: %v)", len(baseChecks), len(monitored), monitored)
	}
	for name := range driftChecks {
		if !baseChecks[name] {
			t.Errorf("mode-drift check for %q has no matching resource check", name)
		}
	}
	// And every accelerator-declaring monitored resource has one.
	for _, name := range acceleratedResources(monitored) {
		if !driftChecks[name] {
			t.Errorf("accelerated resource %q has no mode-drift check", name)
		}
	}

	expectedScenarios := len(factory.criticalScenarios) + len(factory.nonCriticalScenarios)
	if scenarioCount != expectedScenarios {
		t.Errorf("scenario check count = %d, want %d", scenarioCount, expectedScenarios)
	}
}

// TestDefaultCheckFactory_ChecksImplementInterface verifies all checks implement Check interface
func TestDefaultCheckFactory_ChecksImplementInterface(t *testing.T) {
	factory := NewDefaultCheckFactory()
	caps := &platform.Capabilities{Platform: platform.Linux}

	allChecks := []checks.Check{}
	allChecks = append(allChecks, factory.CreateInfrastructureChecks(caps)...)
	allChecks = append(allChecks, factory.CreateSystemChecks()...)
	allChecks = append(allChecks, factory.CreateVrooliChecks(caps)...)

	for _, check := range allChecks {
		// Verify interface methods don't panic
		id := check.ID()
		if id == "" {
			t.Error("check ID should not be empty")
		}
		title := check.Title()
		if title == "" {
			t.Errorf("check %q Title should not be empty", id)
		}
		desc := check.Description()
		if desc == "" {
			t.Errorf("check %q Description should not be empty", id)
		}
		importance := check.Importance()
		if importance == "" {
			t.Errorf("check %q Importance should not be empty", id)
		}
		_ = check.Category()
		interval := check.IntervalSeconds()
		if interval <= 0 {
			t.Errorf("check %q IntervalSeconds = %d, should be > 0", id, interval)
		}
		_ = check.Platforms()
	}

	t.Logf("Verified %d checks implement Check interface correctly", len(allChecks))
}

// =============================================================================
// RegisterChecksWithFactory Tests
// =============================================================================

// TestRegisterChecksWithFactory_UsesFactory verifies factory is called correctly
func TestRegisterChecksWithFactory_UsesFactory(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)

	// Create mock checks
	infraCheck := &simpleCheck{id: "mock-infra"}
	systemCheck := &simpleCheck{id: "mock-system"}
	vrooliCheck := &simpleCheck{id: "mock-vrooli"}

	factory := newMockCheckFactory()
	factory.infraChecks = []checks.Check{infraCheck}
	factory.systemChecks = []checks.Check{systemCheck}
	factory.vrooliChecks = []checks.Check{vrooliCheck}

	// Register with factory
	RegisterChecksWithFactory(registry, caps, factory)

	// Verify factory was called
	if factory.callCounts["infrastructure"] != 1 {
		t.Errorf("infrastructure called %d times, want 1", factory.callCounts["infrastructure"])
	}
	if factory.callCounts["system"] != 1 {
		t.Errorf("system called %d times, want 1", factory.callCounts["system"])
	}
	if factory.callCounts["vrooli"] != 1 {
		t.Errorf("vrooli called %d times, want 1", factory.callCounts["vrooli"])
	}

	// Verify checks were registered
	registeredChecks := registry.ListChecks()
	if len(registeredChecks) != 5 {
		t.Errorf("registered checks = %d, want 5", len(registeredChecks))
	}

	// Verify check IDs
	foundIDs := make(map[string]bool)
	for _, info := range registeredChecks {
		foundIDs[info.ID] = true
	}

	if !foundIDs["mock-infra"] {
		t.Error("expected mock-infra to be registered")
	}
	if !foundIDs["mock-system"] {
		t.Error("expected mock-system to be registered")
	}
	if !foundIDs["mock-vrooli"] {
		t.Error("expected mock-vrooli to be registered")
	}
	if !foundIDs["coverage-remediation-reach"] || !foundIDs["coverage-delivery-reach"] {
		t.Error("expected coverage reach checks to be registered")
	}
}

// TestRegisterChecksWithFactory_EmptyFactory verifies empty factory works
func TestRegisterChecksWithFactory_EmptyFactory(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)

	factory := newMockCheckFactory()
	// Factory returns empty slices

	// Should not panic
	RegisterChecksWithFactory(registry, caps, factory)

	// Coverage checks remain registered so an empty source is reported as
	// unreadable instead of silently disappearing from the instrument.
	registeredChecks := registry.ListChecks()
	if len(registeredChecks) != 2 {
		t.Errorf("registered checks = %d, want 2", len(registeredChecks))
	}
}

// TestRegisterDefaultChecks_UsesDefaultFactory verifies default function works
func TestRegisterDefaultChecks_UsesDefaultFactory(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)

	RegisterDefaultChecks(registry, caps)

	// Should have registered all default checks
	registeredChecks := registry.ListChecks()
	if len(registeredChecks) == 0 {
		t.Error("expected checks to be registered")
	}

	// Count by category
	categories := make(map[checks.Category]int)
	for _, info := range registeredChecks {
		categories[info.Category]++
	}

	t.Logf("Registered checks by category: %v", categories)
}

// simpleCheck is a minimal Check implementation for testing
type simpleCheck struct {
	id string
}

func (c *simpleCheck) ID() string                 { return c.id }
func (c *simpleCheck) Title() string              { return "Simple Check " + c.id }
func (c *simpleCheck) Description() string        { return "A simple test check" }
func (c *simpleCheck) Importance() string         { return "For testing" }
func (c *simpleCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *simpleCheck) IntervalSeconds() int       { return 60 }
func (c *simpleCheck) Platforms() []platform.Type { return nil }
func (c *simpleCheck) Run(ctx context.Context) checks.Result {
	return checks.Result{
		CheckID: c.id,
		Status:  checks.StatusOK,
		Message: "Simple check OK",
	}
}

// =============================================================================
// Factory Options Tests
// =============================================================================

// TestWithNetworkTarget verifies option sets correctly
func TestWithNetworkTarget(t *testing.T) {
	factory := &DefaultCheckFactory{}
	WithNetworkTarget("custom:53")(factory)

	if factory.networkTarget != "custom:53" {
		t.Errorf("networkTarget = %q, want %q", factory.networkTarget, "custom:53")
	}
}

// TestWithDNSDomain verifies option sets correctly
func TestWithDNSDomain(t *testing.T) {
	factory := &DefaultCheckFactory{}
	WithDNSDomain("test.com")(factory)

	if factory.dnsDomain != "test.com" {
		t.Errorf("dnsDomain = %q, want %q", factory.dnsDomain, "test.com")
	}
}

// TestWithCriticalScenarios verifies option sets correctly
func TestWithCriticalScenarios(t *testing.T) {
	factory := &DefaultCheckFactory{}
	WithCriticalScenarios([]string{"a", "b"})(factory)

	if len(factory.criticalScenarios) != 2 {
		t.Errorf("criticalScenarios count = %d, want 2", len(factory.criticalScenarios))
	}
}

// TestWithNonCriticalScenarios verifies option sets correctly
func TestWithNonCriticalScenarios(t *testing.T) {
	factory := &DefaultCheckFactory{}
	WithNonCriticalScenarios([]string{"x", "y", "z"})(factory)

	if len(factory.nonCriticalScenarios) != 3 {
		t.Errorf("nonCriticalScenarios count = %d, want 3", len(factory.nonCriticalScenarios))
	}
}

// TestWithResources verifies option sets correctly
func TestWithResources(t *testing.T) {
	factory := &DefaultCheckFactory{}
	WithResources([]string{"postgres"})(factory)

	if len(factory.resources) != 1 {
		t.Errorf("resources count = %d, want 1", len(factory.resources))
	}
}

// =============================================================================
// GetDefaultFactory/SetDefaultFactory Tests
// =============================================================================

// TestGetDefaultFactory verifies default factory exists
func TestGetDefaultFactory(t *testing.T) {
	factory := GetDefaultFactory()

	if factory == nil {
		t.Error("GetDefaultFactory() should not return nil")
	}
}
