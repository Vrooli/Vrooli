package envx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOSReader_ReadsLiveProcessEnv guards the production seam: OS{}
// must read os.Getenv. The unit test uses t.Setenv (test-local) to
// prove the wire works without trapping any in-package fakes; domain
// tests should use testutil/mocks.FakeEnv instead of mutating process
// env from their own scope.
func TestOSReader_ReadsLiveProcessEnv(t *testing.T) {
	const key = "AUDIO_TOOLS_ENVX_TEST_PROBE"
	t.Setenv(key, "hello")
	var r Reader = OS{}
	require.Equal(t, "hello", r.Get(key))
}
