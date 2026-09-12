package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// failingBrowser fails the test if its Browse method is ever invoked. It proves
// the manual-URL path is independent of mDNS: when a manual URL is supplied,
// Resolve must short-circuit before touching the Browser.
//
// [REQ:BRG-P1-006]
type failingBrowser struct{ t *testing.T }

func (b failingBrowser) Browse(context.Context, string, time.Duration) ([]ServiceInstance, error) {
	b.t.Fatal("manual URL must win without ever invoking mDNS discovery")
	return nil, nil
}

// [REQ:BRG-P1-006] Off-LAN bootstrap uses the manual URL+code path WITHOUT
// depending on mDNS: when a manual control-plane URL is supplied it is returned
// immediately and the Browser is never invoked. The failingBrowser fails the
// test if Resolve consults it, so the independence is enforced, not merely
// observed.
func TestResolve_ManualURLWinsWithoutInvokingMDNS(t *testing.T) {
	got, err := Resolve(context.Background(), "  https://cp.example:8443  ", failingBrowser{t}, time.Second)
	require.NoError(t, err)
	require.True(t, got.Found())
	require.Equal(t, SourceManual, got.Source)
	require.Equal(t, "https://cp.example:8443", got.URL, "the manual URL is trimmed and returned verbatim")
	require.Equal(t, ServiceInstance{}, got.Instance, "no discovered instance accompanies a manual resolution")
}

// [REQ:BRG-P1-006] The manual path is the cross-network default even with no
// browser wired at all: a manual URL resolves cleanly when discovery is
// disabled.
func TestResolve_ManualURLResolvesWithNoBrowser(t *testing.T) {
	got, err := Resolve(context.Background(), "https://cp.example:8443", nil, time.Second)
	require.NoError(t, err)
	require.Equal(t, SourceManual, got.Source)
	require.Equal(t, "https://cp.example:8443", got.URL)
}
