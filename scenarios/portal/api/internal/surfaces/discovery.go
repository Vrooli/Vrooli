package surfaces

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	mdns "github.com/vrooli/mdns-go"
)

// PortalDiscovery advertises the lifecycle-assigned UI gateway, never the API
// or sibling application ports. Advertisements are hints, not authentication.
func PortalDiscovery(port, host string, secure bool, ip net.IP) (mdns.ResponderConfig, error) {
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 || ip == nil || !ip.IsPrivate() || ip.IsLoopback() {
		return mdns.ResponderConfig{}, fmt.Errorf("local Portal discovery requires a UI port and private interface")
	}
	scheme := "http"
	if secure {
		scheme = "https"
	}
	if host == "" {
		host = ip.String()
	}
	u := &url.URL{Scheme: scheme, Host: net.JoinHostPort(host, port)}
	if parsed, err := url.Parse(u.String()); err != nil || parsed.Hostname() != host || strings.ContainsAny(host, "/?#@\\ \t\r\n") {
		return mdns.ResponderConfig{}, fmt.Errorf("invalid Portal discovery hostname")
	}
	identity := "portal-" + strings.ReplaceAll(ip.String(), ".", "-")
	return mdns.ResponderConfig{Service: "_vrooli-portal._tcp.local", Instance: identity, Host: "vrooli-" + identity + ".local", Port: p, Address: ip, URL: u.String(), TXT: map[string]string{"protocol": "portal-catalog-v1"}}, nil
}

// StartPortalDiscovery uses the existing DNS-SD owner package. It is optional:
// missing multicast must not prevent normal Portal startup or manual access.
func StartPortalDiscovery(ctx context.Context, port, host string, secure bool) (func(), error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return func() {}, err
	}
	var responders []*mdns.Responder
	var lastError error
	for i := range interfaces {
		iface := &interfaces[i]
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err != nil || ip.To4() == nil || !ip.IsPrivate() {
				continue
			}
			config, err := PortalDiscovery(port, host, secure, ip)
			if err != nil {
				lastError = err
				continue
			}
			config.Interface = iface
			responder := mdns.NewResponder(config)
			if err := responder.Start(ctx); err != nil {
				lastError = err
				continue
			}
			responders = append(responders, responder)
			break
		}
	}
	stop := func() {
		for _, responder := range responders {
			_ = responder.Close()
		}
	}
	if len(responders) == 0 {
		if lastError == nil {
			lastError = fmt.Errorf("no private multicast interface")
		}
		return stop, lastError
	}
	return stop, nil
}
