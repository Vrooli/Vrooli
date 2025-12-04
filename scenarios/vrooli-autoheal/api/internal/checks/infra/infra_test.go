// Package infra tests for infrastructure health checks
// [REQ:INFRA-NET-001] [REQ:INFRA-DNS-001] [REQ:INFRA-DOCKER-001]
package infra

import (
	"context"
	"testing"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// testTarget and testDomain are explicit values for testing.
// Production defaults are defined in the bootstrap package.
const (
	testTarget = "8.8.8.8:53"
	testDomain = "google.com"
)

// testCaps returns platform capabilities for testing
func testCaps() *platform.Capabilities {
	return &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}
}

// TestNetworkCheckInterface verifies NetworkCheck implements Check
// [REQ:INFRA-NET-001]
func TestNetworkCheckInterface(t *testing.T) {
	var _ checks.Check = (*NetworkCheck)(nil)

	check := NewNetworkCheck(testTarget)
	if check.ID() != "infra-network" {
		t.Errorf("ID() = %q, want %q", check.ID(), "infra-network")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}
	// Should run on all platforms
	if check.Platforms() != nil {
		t.Error("NetworkCheck should run on all platforms")
	}
}

// TestNetworkCheckTarget verifies target is used exactly as provided
func TestNetworkCheckTarget(t *testing.T) {
	customTarget := "1.1.1.1:53"
	check := NewNetworkCheck(customTarget)
	if check.target != customTarget {
		t.Errorf("target = %q, want %q", check.target, customTarget)
	}
}

// TestNetworkCheckRun verifies network check execution
// [REQ:INFRA-NET-001]
func TestNetworkCheckRun(t *testing.T) {
	check := NewNetworkCheck(testTarget)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := check.Run(ctx)

	if result.CheckID != check.ID() {
		t.Errorf("result.CheckID = %q, want %q", result.CheckID, check.ID())
	}

	// We expect either OK (network available) or Critical (network unavailable)
	validStatuses := map[checks.Status]bool{checks.StatusOK: true, checks.StatusCritical: true}
	if !validStatuses[result.Status] {
		t.Errorf("result.Status = %v, want OK or Critical", result.Status)
	}

	if result.Message == "" {
		t.Error("result.Message is empty")
	}

	t.Logf("Network check result: %s - %s", result.Status, result.Message)
}

// TestDNSCheckInterface verifies DNSCheck implements Check
// [REQ:INFRA-DNS-001]
func TestDNSCheckInterface(t *testing.T) {
	var _ checks.Check = (*DNSCheck)(nil)

	check := NewDNSCheck(testDomain)
	if check.ID() != "infra-dns" {
		t.Errorf("ID() = %q, want %q", check.ID(), "infra-dns")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}
}

// TestDNSCheckDomain verifies domain is used exactly as provided
func TestDNSCheckDomain(t *testing.T) {
	customDomain := "cloudflare.com"
	check := NewDNSCheck(customDomain)
	if check.domain != customDomain {
		t.Errorf("domain = %q, want %q", check.domain, customDomain)
	}
}

// TestDNSCheckRun verifies DNS check execution
// [REQ:INFRA-DNS-001]
func TestDNSCheckRun(t *testing.T) {
	check := NewDNSCheck(testDomain)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := check.Run(ctx)

	if result.CheckID != check.ID() {
		t.Errorf("result.CheckID = %q, want %q", result.CheckID, check.ID())
	}

	validStatuses := map[checks.Status]bool{checks.StatusOK: true, checks.StatusCritical: true}
	if !validStatuses[result.Status] {
		t.Errorf("result.Status = %v, want OK or Critical", result.Status)
	}

	if result.Message == "" {
		t.Error("result.Message is empty")
	}

	// Check that domain is in details
	if result.Details != nil {
		if _, ok := result.Details["domain"]; !ok {
			t.Error("details should contain domain")
		}
	}

	t.Logf("DNS check result: %s - %s", result.Status, result.Message)
}

// TestDockerCheckInterface verifies DockerCheck implements Check
// [REQ:INFRA-DOCKER-001]
func TestDockerCheckInterface(t *testing.T) {
	var _ checks.Check = (*DockerCheck)(nil)

	check := NewDockerCheck()
	if check.ID() != "infra-docker" {
		t.Errorf("ID() = %q, want %q", check.ID(), "infra-docker")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}

	// Docker check should be platform-specific
	platforms := check.Platforms()
	if len(platforms) == 0 {
		t.Error("DockerCheck should specify platforms")
	}

	// Should include Linux and macOS
	hasLinux := false
	hasMacOS := false
	for _, p := range platforms {
		if p == platform.Linux {
			hasLinux = true
		}
		if p == platform.MacOS {
			hasMacOS = true
		}
	}
	if !hasLinux || !hasMacOS {
		t.Errorf("DockerCheck platforms = %v, should include Linux and macOS", platforms)
	}
}

// TestDockerCheckRun verifies Docker check execution
// [REQ:INFRA-DOCKER-001]
func TestDockerCheckRun(t *testing.T) {
	check := NewDockerCheck()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := check.Run(ctx)

	if result.CheckID != check.ID() {
		t.Errorf("result.CheckID = %q, want %q", result.CheckID, check.ID())
	}

	// Valid status depends on whether Docker is installed
	validStatuses := map[checks.Status]bool{checks.StatusOK: true, checks.StatusCritical: true}
	if !validStatuses[result.Status] {
		t.Errorf("result.Status = %v, want OK or Critical", result.Status)
	}

	if result.Message == "" {
		t.Error("result.Message is empty")
	}

	t.Logf("Docker check result: %s - %s", result.Status, result.Message)
	if result.Status == checks.StatusOK && result.Details != nil {
		t.Logf("Docker details: %v", result.Details)
	}
}

// TestCloudflaredCheckInterface verifies CloudflaredCheck implements Check
func TestCloudflaredCheckInterface(t *testing.T) {
	var _ checks.Check = (*CloudflaredCheck)(nil)

	check := NewCloudflaredCheck(testCaps())
	if check.ID() != "infra-cloudflared" {
		t.Errorf("ID() = %q, want %q", check.ID(), "infra-cloudflared")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
}

// TestCloudflaredCheckRun verifies cloudflared check execution
func TestCloudflaredCheckRun(t *testing.T) {
	check := NewCloudflaredCheck(testCaps())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := check.Run(ctx)

	if result.CheckID != check.ID() {
		t.Errorf("result.CheckID = %q, want %q", result.CheckID, check.ID())
	}

	// Valid statuses: OK (healthy), Warning (not installed), Critical (not active)
	validStatuses := map[checks.Status]bool{checks.StatusOK: true, checks.StatusWarning: true, checks.StatusCritical: true}
	if !validStatuses[result.Status] {
		t.Errorf("result.Status = %v", result.Status)
	}

	t.Logf("Cloudflared check result: %s - %s", result.Status, result.Message)
}

// TestCloudflaredCheckUsesInjectedCaps verifies platform caps are used
func TestCloudflaredCheckUsesInjectedCaps(t *testing.T) {
	// Test with systemd disabled - should not try to check systemd service
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: false,
	}
	check := NewCloudflaredCheck(caps)
	if check.caps != caps {
		t.Error("CloudflaredCheck should store injected capabilities")
	}
}

// TestRDPCheckInterface verifies RDPCheck implements Check
func TestRDPCheckInterface(t *testing.T) {
	var _ checks.Check = (*RDPCheck)(nil)

	check := NewRDPCheck(testCaps())
	if check.ID() != "infra-rdp" {
		t.Errorf("ID() = %q, want %q", check.ID(), "infra-rdp")
	}

	// RDP check should be platform-specific (Linux and Windows)
	platforms := check.Platforms()
	if len(platforms) == 0 {
		t.Error("RDPCheck should specify platforms")
	}
}

// TestRDPCheckRun verifies RDP check execution
func TestRDPCheckRun(t *testing.T) {
	check := NewRDPCheck(testCaps())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := check.Run(ctx)

	if result.CheckID != check.ID() {
		t.Errorf("result.CheckID = %q, want %q", result.CheckID, check.ID())
	}

	t.Logf("RDP check result: %s - %s", result.Status, result.Message)
}

// TestRDPCheckUsesInjectedCaps verifies platform caps are used
func TestRDPCheckUsesInjectedCaps(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Windows,
		SupportsSystemd: false,
	}
	check := NewRDPCheck(caps)
	if check.caps != caps {
		t.Error("RDPCheck should store injected capabilities")
	}
}

// TestNTPCheckInterface verifies NTPCheck implements Check
// [REQ:INFRA-NTP-001]
func TestNTPCheckInterface(t *testing.T) {
	var _ checks.Check = (*NTPCheck)(nil)

	check := NewNTPCheck(testCaps())
	if check.ID() != "infra-ntp" {
		t.Errorf("ID() = %q, want %q", check.ID(), "infra-ntp")
	}
	if check.Title() == "" {
		t.Error("Title() is empty")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.Importance() == "" {
		t.Error("Importance() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}
	if check.Category() != checks.CategoryInfrastructure {
		t.Error("Category should be infrastructure")
	}

	// Should be Linux-only
	platforms := check.Platforms()
	if len(platforms) == 0 {
		t.Error("NTPCheck should specify platforms")
	}
	hasLinux := false
	for _, p := range platforms {
		if p == platform.Linux {
			hasLinux = true
		}
	}
	if !hasLinux {
		t.Error("NTPCheck should include Linux platform")
	}
}

// TestNTPCheckRun verifies NTP check execution
// [REQ:INFRA-NTP-001]
func TestNTPCheckRun(t *testing.T) {
	check := NewNTPCheck(testCaps())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := check.Run(ctx)

	if result.CheckID != check.ID() {
		t.Errorf("result.CheckID = %q, want %q", result.CheckID, check.ID())
	}

	// Valid statuses depend on platform and timedatectl availability
	validStatuses := map[checks.Status]bool{checks.StatusOK: true, checks.StatusWarning: true}
	if !validStatuses[result.Status] {
		t.Errorf("result.Status = %v, want OK or Warning", result.Status)
	}

	if result.Message == "" {
		t.Error("result.Message is empty")
	}

	t.Logf("NTP check result: %s - %s", result.Status, result.Message)
}

// TestNTPCheckHealable verifies NTPCheck implements HealableCheck
func TestNTPCheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*NTPCheck)(nil)

	check := NewNTPCheck(testCaps())
	actions := check.RecoveryActions(nil)

	if len(actions) == 0 {
		t.Error("NTPCheck should have recovery actions")
	}

	// Should have enable-ntp and force-sync actions
	actionIDs := make(map[string]bool)
	for _, a := range actions {
		actionIDs[a.ID] = true
	}
	if !actionIDs["enable-ntp"] {
		t.Error("NTPCheck should have enable-ntp action")
	}
	if !actionIDs["force-sync"] {
		t.Error("NTPCheck should have force-sync action")
	}
}

// TestResolvedCheckInterface verifies ResolvedCheck implements Check
// [REQ:INFRA-RESOLVED-001]
func TestResolvedCheckInterface(t *testing.T) {
	var _ checks.Check = (*ResolvedCheck)(nil)

	check := NewResolvedCheck(testCaps())
	if check.ID() != "infra-resolved" {
		t.Errorf("ID() = %q, want %q", check.ID(), "infra-resolved")
	}
	if check.Title() == "" {
		t.Error("Title() is empty")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.Importance() == "" {
		t.Error("Importance() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}

	// Should be Linux-only
	platforms := check.Platforms()
	if len(platforms) == 0 {
		t.Error("ResolvedCheck should specify platforms")
	}
}

// TestResolvedCheckRun verifies systemd-resolved check execution
// [REQ:INFRA-RESOLVED-001]
func TestResolvedCheckRun(t *testing.T) {
	check := NewResolvedCheck(testCaps())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := check.Run(ctx)

	if result.CheckID != check.ID() {
		t.Errorf("result.CheckID = %q, want %q", result.CheckID, check.ID())
	}

	// Valid statuses depend on whether systemd-resolved is installed
	validStatuses := map[checks.Status]bool{
		checks.StatusOK:       true,
		checks.StatusWarning:  true,
		checks.StatusCritical: true,
	}
	if !validStatuses[result.Status] {
		t.Errorf("result.Status = %v", result.Status)
	}

	if result.Message == "" {
		t.Error("result.Message is empty")
	}

	t.Logf("Resolved check result: %s - %s", result.Status, result.Message)
}

// TestResolvedCheckHealable verifies ResolvedCheck implements HealableCheck
func TestResolvedCheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*ResolvedCheck)(nil)

	check := NewResolvedCheck(testCaps())
	actions := check.RecoveryActions(nil)

	if len(actions) == 0 {
		t.Error("ResolvedCheck should have recovery actions")
	}

	// Should have start, restart, flush-cache, logs actions
	actionIDs := make(map[string]bool)
	for _, a := range actions {
		actionIDs[a.ID] = true
	}
	expectedActions := []string{"start", "restart", "flush-cache", "logs"}
	for _, expected := range expectedActions {
		if !actionIDs[expected] {
			t.Errorf("ResolvedCheck should have %s action", expected)
		}
	}
}

// TestCloudflaredCheckHealable verifies CloudflaredCheck implements HealableCheck
func TestCloudflaredCheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*CloudflaredCheck)(nil)

	check := NewCloudflaredCheck(testCaps())
	actions := check.RecoveryActions(nil)

	if len(actions) == 0 {
		t.Error("CloudflaredCheck should have recovery actions")
	}

	// Should have start, restart, test-tunnel, logs, diagnose actions
	actionIDs := make(map[string]bool)
	for _, a := range actions {
		actionIDs[a.ID] = true
	}
	expectedActions := []string{"start", "restart", "test-tunnel", "logs", "diagnose"}
	for _, expected := range expectedActions {
		if !actionIDs[expected] {
			t.Errorf("CloudflaredCheck should have %s action", expected)
		}
	}
}

// TestCloudflaredCheckOptions verifies CloudflaredCheck configuration options
// [REQ:INFRA-CLOUDFLARED-001]
func TestCloudflaredCheckOptions(t *testing.T) {
	check := NewCloudflaredCheck(
		testCaps(),
		WithLocalTestPort(8080),
		WithExternalURL("https://example.com"),
		WithConnectTimeout(10*time.Second),
	)

	if check.localTestPort != 8080 {
		t.Errorf("localTestPort = %d, want %d", check.localTestPort, 8080)
	}
	if check.externalURL != "https://example.com" {
		t.Errorf("externalURL = %q, want %q", check.externalURL, "https://example.com")
	}
	if check.connectTimeout != 10*time.Second {
		t.Errorf("connectTimeout = %v, want %v", check.connectTimeout, 10*time.Second)
	}
}

// TestCloudflaredCheckDefaultOptions verifies CloudflaredCheck default options
func TestCloudflaredCheckDefaultOptions(t *testing.T) {
	check := NewCloudflaredCheck(testCaps())

	if check.localTestPort != 21774 {
		t.Errorf("default localTestPort = %d, want %d", check.localTestPort, 21774)
	}
	if check.externalURL != "" {
		t.Errorf("default externalURL = %q, want empty", check.externalURL)
	}
	if check.connectTimeout != 5*time.Second {
		t.Errorf("default connectTimeout = %v, want %v", check.connectTimeout, 5*time.Second)
	}
}

// TestDockerCheckHealable verifies DockerCheck implements HealableCheck
// [REQ:HEAL-ACTION-001]
func TestDockerCheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*DockerCheck)(nil)

	check := NewDockerCheck(testCaps())
	actions := check.RecoveryActions(nil)

	if len(actions) == 0 {
		t.Error("DockerCheck should have recovery actions")
	}

	// Should have restart, start, prune, logs, info actions
	actionIDs := make(map[string]bool)
	for _, a := range actions {
		actionIDs[a.ID] = true
	}
	expectedActions := []string{"restart", "start", "prune", "logs", "info"}
	for _, expected := range expectedActions {
		if !actionIDs[expected] {
			t.Errorf("DockerCheck should have %s action", expected)
		}
	}
}
