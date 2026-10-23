package surfaces

import (
	"net"
	"testing"
)

func TestPortalDiscoveryUsesCurrentGatewayPortAndTLS(t *testing.T) {
	for _, port := range []string{"21345", "25678"} {
		c, err := PortalDiscovery(port, "portal.home.example", true, net.ParseIP("192.168.1.12"))
		if err != nil || c.URL != "https://portal.home.example:"+port || c.Service != "_vrooli-portal._tcp.local" || c.TXT["protocol"] != "portal-catalog-v1" {
			t.Fatalf("bad discovery: %+v %v", c, err)
		}
	}
	c, err := PortalDiscovery("23456", "", false, net.ParseIP("192.168.1.12"))
	if err != nil || c.URL != "http://192.168.1.12:23456" {
		t.Fatalf("plaintext service must be labeled truthfully: %+v %v", c, err)
	}
	other, err := PortalDiscovery("23456", "", false, net.ParseIP("192.168.1.13"))
	if err != nil || other.Instance == c.Instance || other.Host == c.Host {
		t.Fatal("Different home servers must not collide in DNS-SD")
	}
}

func TestPortalDiscoveryRejectsInvalidGateway(t *testing.T) {
	for _, c := range []struct{ port, host, ip string }{
		{"0", "", "192.168.1.12"}, {"65536", "", "192.168.1.12"}, {"80", "", "127.0.0.1"}, {"80", "", "8.8.8.8"}, {"80", "user@host", "192.168.1.12"}, {"80", "host/path", "192.168.1.12"},
	} {
		if _, err := PortalDiscovery(c.port, c.host, true, net.ParseIP(c.ip)); err == nil {
			t.Fatalf("accepted invalid discovery %+v", c)
		}
	}
}
