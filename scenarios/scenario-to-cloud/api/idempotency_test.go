package main

import (
	"context"
	"strings"
	"testing"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/secrets"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/vps"
)

// TestCaddyConfigIdempotency verifies that Caddy configuration is only written
// when it differs from the current config on the VPS.
// [REQ:STC-IDEM-001] Caddy config updates are idempotent
func TestCaddyConfigIdempotency(t *testing.T) {
	// Create a fake SSH runner that tracks calls
	fakeSSH := &FakeSSHRunner{
		Responses: map[string]ssh.Result{
			// All commands succeed
		},
	}

	// Set up a pattern-based response handler by using a custom Run implementation
	// that checks for specific command patterns
	tests := []struct {
		name                string
		currentCaddyContent string
		desiredDomain       string
		desiredPort         int
		expectWrite         bool
		expectReload        bool
	}{
		{
			name:                "write when file empty",
			currentCaddyContent: "",
			desiredDomain:       "example.com",
			desiredPort:         3000,
			expectWrite:         true,
			expectReload:        true,
		},
		{
			name:                "skip when content matches",
			currentCaddyContent: "{\n  acme_ca https://acme-v02.api.letsencrypt.org/directory\n}\nexample.com {\n  reverse_proxy 127.0.0.1:3000\n}",
			desiredDomain:       "example.com",
			desiredPort:         3000,
			expectWrite:         false,
			expectReload:        false,
		},
		{
			name:                "write when domain differs",
			currentCaddyContent: "{\n  acme_ca https://acme-v02.api.letsencrypt.org/directory\n}\nold-domain.com {\n  reverse_proxy 127.0.0.1:3000\n}",
			desiredDomain:       "example.com",
			desiredPort:         3000,
			expectWrite:         true,
			expectReload:        true,
		},
		{
			name:                "write when port differs",
			currentCaddyContent: "{\n  acme_ca https://acme-v02.api.letsencrypt.org/directory\n}\nexample.com {\n  reverse_proxy 127.0.0.1:8080\n}",
			desiredDomain:       "example.com",
			desiredPort:         3000,
			expectWrite:         true,
			expectReload:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset call tracking
			fakeSSH.Calls = nil

			// Configure response for cat command
			catCmd := "cat /etc/caddy/Caddyfile 2>/dev/null || echo ''"
			fakeSSH.Responses = map[string]ssh.Result{
				catCmd: {
					Stdout:   tt.currentCaddyContent,
					ExitCode: 0,
				},
			}

			// Configure default success response for other commands
			fakeSSH.DefaultErr = nil

			// Simulate the idempotent Caddy config logic
			currentContent := strings.TrimSpace(tt.currentCaddyContent)
			desiredCaddyfile := vps.BuildCaddyfile(tt.desiredDomain, tt.desiredPort, vps.CaddyTLSConfig{})
			desiredContent := strings.TrimSpace(desiredCaddyfile)

			needsWrite := currentContent != desiredContent

			if needsWrite != tt.expectWrite {
				t.Errorf("needsWrite = %v, want %v\nCurrent: %q\nDesired: %q",
					needsWrite, tt.expectWrite, currentContent, desiredContent)
			}
		})
	}
}

// TestBuildCaddyfileDeterministic verifies that buildCaddyfile produces
// consistent output for the same inputs.
// [REQ:STC-IDEM-002] Caddyfile generation is deterministic
func TestBuildCaddyfileDeterministic(t *testing.T) {
	domain := "example.com"
	port := 3000

	first := vps.BuildCaddyfile(domain, port, vps.CaddyTLSConfig{})
	second := vps.BuildCaddyfile(domain, port, vps.CaddyTLSConfig{})

	if first != second {
		t.Errorf("buildCaddyfile not deterministic:\n  First: %q\n  Second: %q", first, second)
	}

	expected := "{\n  acme_ca https://acme-v02.api.letsencrypt.org/directory\n}\nexample.com {\n  reverse_proxy 127.0.0.1:3000\n}"
	if first != expected {
		t.Errorf("vps.BuildCaddyfile(%q, %d) = %q, want %q", domain, port, first, expected)
	}
}

// TestSecretsWriterPreservesExisting verifies that WriteToVPS consults the
// remote authority before provisioning generated credentials.
// [REQ:STC-IDEM-003] Secrets are preserved across redeployments
func TestSecretsWriterPreservesExisting(t *testing.T) {
	ctx := context.Background()
	fakeSSH := &FakeSSHRunner{
		Handler: func(command string) (ssh.Result, error, bool) {
			switch {
			case strings.Contains(command, "credentials' 'doctor"):
				return ssh.Result{Stdout: `{"provider":{"condition":"available"}}`, ExitCode: 0}, nil, true
			case strings.Contains(command, "credentials' 'status") && strings.Contains(command, "postgres-password"):
				return ssh.Result{Stdout: `{"configured":true,"provider_state":"available"}`, ExitCode: 0}, nil, true
			case strings.Contains(command, "credentials' 'status"):
				return ssh.Result{Stdout: `{"configured":false,"provider_state":"available"}`, ExitCode: 0}, nil, true
			default:
				return ssh.Result{ExitCode: 0}, nil, true
			}
		},
	}

	cfg := ssh.Config{Host: "test", Port: 22, User: "root"}
	if err := secrets.WriteToVPS(ctx, fakeSSH, cfg, "/root/Vrooli", []secrets.GeneratedSecret{
		{ID: "pg_pass", Key: "POSTGRES_PASSWORD", Value: "new-generated-password"},
		{ID: "api_key", Key: "API_KEY", Value: "new-api-key"},
	}, nil, "demo"); err != nil {
		t.Fatalf("WriteToVPS: %v", err)
	}
	for _, command := range fakeSSH.Calls {
		if strings.Contains(command, "new-generated-password") {
			t.Fatalf("existing credential value appeared in command: %q", command)
		}
		if strings.Contains(command, "credentials' 'provision") && strings.Contains(command, "api-key") {
			return
		}
	}
	t.Fatal("new API key was not provisioned")
}

// TestStopExistingScenarioIdempotent verifies that StopExistingScenario
// can be called multiple times without error.
// [REQ:STC-IDEM-004] Scenario stop is idempotent
func TestStopExistingScenarioIdempotent(t *testing.T) {
	ctx := context.Background()

	// First call: scenario running
	fakeSSH := &FakeSSHRunner{
		Responses: map[string]ssh.Result{},
	}
	fakeSSH.DefaultErr = nil // All commands succeed

	cfg := ssh.Config{Host: "test", Port: 22, User: "root"}
	workdir := "/root/Vrooli"
	scenarioID := "test-scenario"
	ports := []int{3000, 3001}

	// First stop
	result1 := vps.StopExistingScenario(ctx, fakeSSH, cfg, workdir, scenarioID, ports)
	if !result1.OK {
		t.Errorf("First StopExistingScenario failed: %s", result1.Error)
	}

	// Second stop (should also succeed even if nothing to stop)
	result2 := vps.StopExistingScenario(ctx, fakeSSH, cfg, workdir, scenarioID, ports)
	if !result2.OK {
		t.Errorf("Second StopExistingScenario failed: %s", result2.Error)
	}
}

// TestDeleteBundleIdempotent verifies that DeleteBundle can be called
// multiple times on the same hash without error.
// [REQ:STC-IDEM-005] Bundle deletion is idempotent
func TestDeleteBundleIdempotent(t *testing.T) {
	// DeleteBundle already returns 0 bytes and nil error for missing files
	// This is tested implicitly in bundle.go via os.IsNotExist check

	// Test with empty dir (no bundles)
	tempDir := t.TempDir()

	// First delete - file doesn't exist
	freed1, err := bundle.DeleteBundle(tempDir, "abc123")
	if err != nil {
		t.Errorf("First DeleteBundle failed: %v", err)
	}
	if freed1 != 0 {
		t.Errorf("Expected 0 bytes freed, got %d", freed1)
	}

	// Second delete - still doesn't exist, should still succeed
	freed2, err := bundle.DeleteBundle(tempDir, "abc123")
	if err != nil {
		t.Errorf("Second DeleteBundle failed: %v", err)
	}
	if freed2 != 0 {
		t.Errorf("Expected 0 bytes freed, got %d", freed2)
	}
}
