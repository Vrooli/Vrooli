package main

import (
	"errors"
	"net"
	"reflect"
	"testing"
)

func TestBootstrapScriptCandidatesPreferDeclaredRoots(t *testing.T) {
	got := bootstrapScriptCandidates(
		"/tmp/api",
		"/srv/vrooli-bridge",
		"/srv/legacy-scenario",
		"/srv/vrooli",
		"/tmp/bin/vrooli-bridge-api",
	)
	if len(got) == 0 || got[0] != "/srv/vrooli-bridge/bootstrap/bootstrap.sh" {
		t.Fatalf("first bootstrap candidate = %v, want scenario root", got)
	}
	if got[0] == "/tmp/api/bootstrap/bootstrap.sh" {
		t.Fatal("working-directory candidate must not precede declared roots")
	}
}

func TestCanonicalControlPlaneEndpointPrecedence(t *testing.T) {
	t.Setenv("BRIDGE_CONTROL_PLANE_URL", "https://configured.example.test")
	t.Setenv("BRIDGE_TUNNEL_URL", "https://tunnel.example.test")
	if got, source := canonicalControlPlaneEndpoint(); got != "https://configured.example.test" || source != "configured" {
		t.Fatalf("configured endpoint = %q (%s)", got, source)
	}
	t.Setenv("BRIDGE_CONTROL_PLANE_URL", "")
	if got, source := canonicalControlPlaneEndpoint(); got != "https://tunnel.example.test" || source != "tunnel" {
		t.Fatalf("tunnel endpoint = %q (%s)", got, source)
	}
}

func TestResolveOutboundIPReportsMissingRoute(t *testing.T) {
	ip, err := resolveOutboundIP(func(string, string) (net.Conn, error) {
		return nil, errors.New("network is offline")
	})
	if ip != "127.0.0.1" || err == nil {
		t.Fatalf("resolveOutboundIP() = (%q, %v), want loopback plus diagnostic", ip, err)
	}
}

func TestResolveOutboundIPUsesSelectedInterface(t *testing.T) {
	want := net.ParseIP("192.0.2.44")
	ip, err := resolveOutboundIP(func(string, string) (net.Conn, error) {
		return probeConn{addr: &net.UDPAddr{IP: want}}, nil
	})
	if err != nil || ip != want.String() {
		t.Fatalf("resolveOutboundIP() = (%q, %v), want %q", ip, err, want)
	}
}

func TestCollectInterfaceIPsReturnsAllNonLoopbackCandidates(t *testing.T) {
	interfaces := []net.Interface{
		{Name: "lan0", Flags: net.FlagUp},
		{Name: "wifi0", Flags: net.FlagUp},
		{Name: "lo", Flags: net.FlagUp | net.FlagLoopback},
	}
	addresses := map[string][]net.Addr{
		"lan0":  {&net.IPNet{IP: net.ParseIP("192.0.2.10")}, &net.IPNet{IP: net.ParseIP("192.0.2.10")}},
		"wifi0": {&net.IPAddr{IP: net.ParseIP("2001:db8::10")}},
		"lo":    {&net.IPAddr{IP: net.ParseIP("127.0.0.1")}},
	}
	candidates, err := collectInterfaceIPs(interfaces, func(iface net.Interface) ([]net.Addr, error) {
		return addresses[iface.Name], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"192.0.2.10", "2001:db8::10"}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("collectInterfaceIPs() = %v, want %v", candidates, want)
	}
}

type probeConn struct {
	net.Conn
	addr net.Addr
}

func (c probeConn) LocalAddr() net.Addr { return c.addr }
func (c probeConn) Close() error        { return nil }
