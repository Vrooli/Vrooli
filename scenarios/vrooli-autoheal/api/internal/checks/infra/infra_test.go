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
