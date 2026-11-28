package discovery

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	mdns "github.com/vrooli/mdns-go"
)

// MDNSBrowser adapts the shared service-agnostic DNS-SD browser to the agent's
// control-plane discovery seam. The agent owns only the service vocabulary and
// URL projection; packet encoding and record resolution live in mdns-go.
type MDNSBrowser struct {
	Browser *mdns.Browser
}

var _ Browser = (*MDNSBrowser)(nil)

func (b *MDNSBrowser) Browse(ctx context.Context, service string, timeout time.Duration) ([]ServiceInstance, error) {
	browser := b.Browser
	if browser == nil {
		browser = mdns.NewBrowser()
	}
	if timeout > 0 {
		browser.Window = timeout
	}
	instances, err := browser.Browse(ctx, service)
	if err != nil {
		return nil, err
	}
	result := make([]ServiceInstance, 0, len(instances))
	for _, instance := range instances {
		host := strings.TrimSpace(instance.Host)
		if host == "" && len(instance.Addrs) > 0 {
			host = instance.Addrs[0].String()
		}
		if host == "" || instance.Port <= 0 {
			continue
		}
		result = append(result, ServiceInstance{
			Host: host, Port: instance.Port,
			URL: serviceInstanceURL(instance, host),
		})
	}
	return result, nil
}

// serviceInstanceURL prefers the endpoint explicitly advertised by the
// control plane. Bridge commonly serves plain HTTP on a trusted LAN, so
// synthesizing https from SRV alone would turn a successful discovery into a
// TLS connection failure. Generic DNS-SD records without a URL TXT value use
// HTTP as the conservative local-network fallback.
func serviceInstanceURL(instance mdns.ServiceInstance, host string) string {
	if advertised := strings.TrimSpace(instance.TXT["url"]); advertised != "" {
		return advertised
	}
	return fmt.Sprintf("http://%s", net.JoinHostPort(host, fmt.Sprint(instance.Port)))
}
