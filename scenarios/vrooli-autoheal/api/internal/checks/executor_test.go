// Package checks tests for real executor implementations
// [REQ:TEST-SEAM-001] Verify real executor works correctly
package checks

import (
	"context"
	"testing"
	"time"
)

// =============================================================================
// RealExecutor Tests
// =============================================================================

// TestRealExecutorOutput verifies Output runs a command
func TestRealExecutorOutput(t *testing.T) {
	exec := &RealExecutor{}
	ctx := context.Background()

	output, err := exec.Output(ctx, "echo", "hello")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(output) != "hello\n" {
		t.Errorf("Output = %q, want %q", output, "hello\n")
	}
}

// TestRealExecutorOutputError verifies error handling
func TestRealExecutorOutputError(t *testing.T) {
	exec := &RealExecutor{}
	ctx := context.Background()

	_, err := exec.Output(ctx, "nonexistent-command-12345")

	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}

// TestRealExecutorCombinedOutput verifies CombinedOutput runs a command
func TestRealExecutorCombinedOutput(t *testing.T) {
	exec := &RealExecutor{}
	ctx := context.Background()

	output, err := exec.CombinedOutput(ctx, "echo", "test")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(output) != "test\n" {
		t.Errorf("CombinedOutput = %q, want %q", output, "test\n")
	}
}

// TestRealExecutorRun verifies Run executes a command
func TestRealExecutorRun(t *testing.T) {
	exec := &RealExecutor{}
	ctx := context.Background()

	err := exec.Run(ctx, "true")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRealExecutorRunError verifies Run error handling
func TestRealExecutorRunError(t *testing.T) {
	exec := &RealExecutor{}
	ctx := context.Background()

	err := exec.Run(ctx, "false")

	if err == nil {
		t.Error("expected error for 'false' command")
	}
}

// TestRealExecutorContextTimeout verifies context cancellation
func TestRealExecutorContextTimeout(t *testing.T) {
	exec := &RealExecutor{}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Sleep to ensure context times out
	time.Sleep(1 * time.Millisecond)

	_, err := exec.Output(ctx, "sleep", "10")

	if err == nil {
		t.Error("expected context timeout error")
	}
}

// TestDefaultExecutorExists verifies DefaultExecutor is set
func TestDefaultExecutorExists(t *testing.T) {
	if DefaultExecutor == nil {
		t.Error("DefaultExecutor should not be nil")
	}

	// Verify it's a RealExecutor
	_, ok := DefaultExecutor.(*RealExecutor)
	if !ok {
		t.Error("DefaultExecutor should be *RealExecutor")
	}
}

// =============================================================================
// RealDialer Tests
// =============================================================================

// TestRealDialerDialTimeout verifies real dialer works
func TestRealDialerDialTimeout(t *testing.T) {
	dialer := &RealDialer{}

	// Connect to localhost on a port that should exist (SSH or some common service)
	// If no service is running, this will fail with connection refused, which is fine
	// We're just testing the interface works
	_, err := dialer.DialTimeout("tcp", "127.0.0.1:12345", 100*time.Millisecond)

	// We expect either a successful connection or a connection refused error
	// Not testing the actual connection, just that the method works
	if err == nil {
		t.Log("Unexpectedly found a service on port 12345")
	} else {
		t.Logf("Expected error (no service on port 12345): %v", err)
	}
}

// TestRealDialerDialTimeoutExpired verifies timeout behavior
func TestRealDialerDialTimeoutExpired(t *testing.T) {
	dialer := &RealDialer{}

	// Try to connect to a non-routable IP with a very short timeout
	start := time.Now()
	_, err := dialer.DialTimeout("tcp", "10.255.255.1:80", 50*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Skip("Unexpectedly connected to 10.255.255.1")
	}

	// Should timeout within a reasonable window (give some slack for test environments)
	if elapsed > 1*time.Second {
		t.Errorf("DialTimeout took %v, expected faster timeout", elapsed)
	}
}

// TestDefaultDialerExists verifies DefaultDialer is set
func TestDefaultDialerExists(t *testing.T) {
	if DefaultDialer == nil {
		t.Error("DefaultDialer should not be nil")
	}

	// Verify it's a RealDialer
	_, ok := DefaultDialer.(*RealDialer)
	if !ok {
		t.Error("DefaultDialer should be *RealDialer")
	}
}

// =============================================================================
// DefaultHTTPClient Tests
// =============================================================================

// TestDefaultHTTPClientExists verifies DefaultHTTPClient is set
func TestDefaultHTTPClientExists(t *testing.T) {
	if DefaultHTTPClient == nil {
		t.Error("DefaultHTTPClient should not be nil")
	}
}
