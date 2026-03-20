package livedesktop

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindAvailablePort_FindsPort(t *testing.T) {
	// Use a high port range that is likely free
	port, err := findAvailablePort(19500, 19600)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, port, 19500)
	assert.LessOrEqual(t, port, 19600)
}

func TestCheckVNCDependencies(t *testing.T) {
	// This test just verifies the function returns a helpful message
	// when tools are missing (which they might be in CI)
	err := checkVNCDependencies()
	if err != nil {
		assert.Contains(t, err.Error(), "required tools not installed")
		assert.Contains(t, err.Error(), "apt-get install")
		assert.Contains(t, err.Error(), "manage.sh setup")
	}
	// If no error, the tools are installed — also fine
}

func TestFindAvailablePort_AllUsed(t *testing.T) {
	// Bind a small range of ports
	start := 19700
	end := 19702
	var listeners []net.Listener
	for p := start; p <= end; p++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
		if err != nil {
			t.Skipf("could not bind port %d for test setup: %v", p, err)
		}
		listeners = append(listeners, ln)
	}
	defer func() {
		for _, ln := range listeners {
			_ = ln.Close()
		}
	}()

	_, err := findAvailablePort(start, end)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no available port")
}
