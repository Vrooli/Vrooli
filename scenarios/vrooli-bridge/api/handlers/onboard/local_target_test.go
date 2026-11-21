package onboard

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRejectSelfTargetRecognizesControlPlaneHostname(t *testing.T) {
	host, err := os.Hostname()
	require.NoError(t, err)
	require.Error(t, rejectSelfTarget(host))
}

func TestRejectSelfTargetRecognizesLoopback(t *testing.T) {
	require.Error(t, rejectSelfTarget("127.0.0.1"))
	require.Error(t, rejectSelfTarget("localhost"))
}

func TestRejectSelfTargetAllowsUnrelatedRemoteAddress(t *testing.T) {
	require.NoError(t, rejectSelfTarget("198.51.100.23"))
}
