package discovery

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	mdns "github.com/vrooli/mdns-go"
)

type fakeBrowser struct {
	instances []ServiceInstance
	called    bool
}

func (f *fakeBrowser) Browse(_ context.Context, _ string, _ time.Duration) ([]ServiceInstance, error) {
	f.called = true
	return f.instances, nil
}

func TestResolve_DiscoversAdvertisedControlPlane(t *testing.T) {
	browser := &fakeBrowser{instances: []ServiceInstance{{Host: "10.0.0.9", Port: 8443, URL: "https://10.0.0.9:8443"}}}
	result, err := Resolve(context.Background(), "", browser, time.Second)
	require.NoError(t, err)
	require.True(t, browser.called)
	require.Equal(t, SourceDiscovered, result.Source)
	require.Equal(t, "https://10.0.0.9:8443", result.URL)
}

func TestResolve_ManualURLSkipsBrowser(t *testing.T) {
	browser := &fakeBrowser{}
	result, err := Resolve(context.Background(), "https://control-plane.example", browser, time.Second)
	require.NoError(t, err)
	require.Equal(t, SourceManual, result.Source)
	require.False(t, browser.called)
}

func TestServiceInstanceURL_UsesAdvertisedScheme(t *testing.T) {
	got := serviceInstanceURL(mdns.ServiceInstance{TXT: map[string]string{"url": "http://192.168.1.173:18767"}}, "192.168.1.173")
	require.Equal(t, "http://192.168.1.173:18767", got)
}

func TestServiceInstanceURL_FallsBackToHTTP(t *testing.T) {
	got := serviceInstanceURL(mdns.ServiceInstance{Port: 8443}, "192.168.1.173")
	require.Equal(t, "http://192.168.1.173:8443", got)
}

func TestLiveBrowseBridgeAdvertisementWhenRequested(t *testing.T) {
	if os.Getenv("MDNS_BRIDGE_LIVE_CHECK") != "1" {
		t.Skip("set MDNS_BRIDGE_LIVE_CHECK=1 for an operator-network witness")
	}
	interfaces, err := net.Interfaces()
	require.NoError(t, err)
	var lanInterface *net.Interface
	for i := range interfaces {
		candidate := &interfaces[i]
		if candidate.Flags&net.FlagUp != 0 && candidate.Flags&net.FlagMulticast != 0 && candidate.Flags&net.FlagLoopback == 0 {
			lanInterface = candidate
			break
		}
	}
	if lanInterface == nil {
		t.Skip("no non-loopback multicast interface")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	instances, err := (&MDNSBrowser{Browser: &mdns.Browser{Interfaces: []*net.Interface{lanInterface}}}).Browse(ctx, defaultServiceType, 0)
	require.NoError(t, err)
	require.NotEmpty(t, instances)
	require.Contains(t, instances[0].URL, "://")
	require.Equal(t, 18767, instances[0].Port)
}
