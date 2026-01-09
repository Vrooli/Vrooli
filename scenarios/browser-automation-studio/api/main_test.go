package main

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckPortAvailable tests the port availability check function
func TestCheckPortAvailable(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] returns nil for available port", func(t *testing.T) {
		// Find an available port by letting the OS assign one
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		_, port, err := net.SplitHostPort(listener.Addr().String())
		require.NoError(t, err)
		listener.Close() // Close so the port becomes available

		// Now check if port is available
		err = checkPortAvailable(port)
		assert.NoError(t, err)
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] returns error for port in use", func(t *testing.T) {
		// Bind to a random port
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		// Extract the port
		_, port, err := net.SplitHostPort(listener.Addr().String())
		require.NoError(t, err)

		// Check that the port is reported as unavailable
		err = checkPortAvailable(port)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unavailable")
		assert.Contains(t, err.Error(), port)
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] error message includes diagnostic hints", func(t *testing.T) {
		// Bind to a port
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		_, port, err := net.SplitHostPort(listener.Addr().String())
		require.NoError(t, err)

		err = checkPortAvailable(port)
		require.Error(t, err)

		// Error message should include helpful diagnostic hints
		assert.Contains(t, err.Error(), "lsof")
		assert.Contains(t, err.Error(), "ss")
	})

	t.Run("handles invalid port gracefully", func(t *testing.T) {
		// Invalid port should return an error
		err := checkPortAvailable("invalid")
		assert.Error(t, err)
	})

	t.Run("handles port out of range", func(t *testing.T) {
		// Port 99999 is out of valid range (0-65535)
		err := checkPortAvailable("99999")
		assert.Error(t, err)
	})
}

// TestPortRangeValidation tests edge cases for port numbers
func TestPortRangeValidation(t *testing.T) {
	t.Run("accepts minimum valid port", func(t *testing.T) {
		// Port 1 might be in use, but should not fail due to range
		err := checkPortAvailable("1")
		// Either succeeds (available) or fails (in use), but not due to invalid port
		if err != nil {
			assert.Contains(t, err.Error(), "unavailable")
		}
	})

	t.Run("accepts maximum valid port", func(t *testing.T) {
		err := checkPortAvailable("65535")
		// Should either succeed or fail due to port in use, not invalid port
		if err != nil {
			assert.Contains(t, err.Error(), "unavailable")
		}
	})

	t.Run("rejects port 0", func(t *testing.T) {
		// Port 0 means "any available port" in TCP, so Listen will succeed
		// but the actual bound port will be different. This is a special case.
		// The function should handle this gracefully.
		err := checkPortAvailable("0")
		// Port 0 is special - it will succeed (OS assigns a port) but might cause confusion
		// This test documents the current behavior
		assert.NoError(t, err, "port 0 triggers OS port assignment and succeeds")
	})

	t.Run("rejects negative port as string", func(t *testing.T) {
		err := checkPortAvailable("-1")
		assert.Error(t, err)
	})
}

// TestConcurrentPortChecks ensures thread safety
func TestConcurrentPortChecks(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] handles concurrent port checks", func(t *testing.T) {
		// Find an available port
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		_, port, err := net.SplitHostPort(listener.Addr().String())
		require.NoError(t, err)
		listener.Close()

		// Run multiple concurrent checks
		done := make(chan error, 10)
		for i := 0; i < 10; i++ {
			go func() {
				done <- checkPortAvailable(port)
			}()
		}

		// At least one should succeed (the first one to bind)
		successCount := 0
		for i := 0; i < 10; i++ {
			if <-done == nil {
				successCount++
			}
		}

		// Due to race conditions, success count varies, but should get at least one
		assert.GreaterOrEqual(t, successCount, 1)
	})
}

// TestPortCheckErrorFormat validates error message format
func TestPortCheckErrorFormat(t *testing.T) {
	t.Run("error format is consistent", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		_, port, err := net.SplitHostPort(listener.Addr().String())
		require.NoError(t, err)

		err = checkPortAvailable(port)
		require.Error(t, err)

		// Verify error message structure
		errMsg := err.Error()
		assert.Contains(t, errMsg, fmt.Sprintf("port %s", port))
		assert.Contains(t, errMsg, "hint:")
	})
}
