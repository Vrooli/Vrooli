package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

const MaxRemoteImageBytes int64 = 64 << 20

var nonPublicPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
}

var (
	ErrUnsafeRemoteURL     = errors.New("fetch: unsafe remote image URL")
	ErrRemoteImageTooLarge = errors.New("fetch: remote image exceeds byte limit")
)

// RemoteImage is an untrusted URL fetcher with an explicit public-network
// boundary. It deliberately does not follow redirects: callers must submit the
// final public image URL so redirects cannot escape the validated destination.
type RemoteImage struct {
	Resolver *net.Resolver
	MaxBytes int64
	Timeout  time.Duration
	lookupIP func(context.Context, string) ([]netip.Addr, error)
	dial     func(context.Context, string, string) (net.Conn, error)
}

func (f RemoteImage) Fetch(ctx context.Context, raw string) ([]byte, string, error) {
	timeout := f.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !u.IsAbs() || u.User != nil || u.Opaque != "" {
		return nil, "", ErrUnsafeRemoteURL
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, "", ErrUnsafeRemoteURL
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return nil, "", ErrUnsafeRemoteURL
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	if (u.Scheme == "https" && port != "443") || (u.Scheme == "http" && port != "80") {
		return nil, "", ErrUnsafeRemoteURL
	}
	ips, err := f.publicIPs(ctx, host)
	if err != nil {
		return nil, "", fmt.Errorf("fetch: resolve remote image host: %w", err)
	}
	maxBytes := f.MaxBytes
	if maxBytes <= 0 || maxBytes > MaxRemoteImageBytes {
		maxBytes = MaxRemoteImageBytes
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: nil, // never let ambient proxy configuration route around the IP policy
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			_, requestedPort, splitErr := net.SplitHostPort(address)
			if splitErr != nil || requestedPort != port {
				return nil, ErrUnsafeRemoteURL
			}
			var lastErr error
			for _, ip := range ips {
				pinnedAddress := net.JoinHostPort(ip.String(), port)
				dial := f.dial
				if dial == nil {
					dial = dialer.DialContext
				}
				conn, dialErr := dial(ctx, network, pinnedAddress)
				if dialErr == nil {
					return conn, nil
				}
				lastErr = dialErr
			}
			return nil, lastErr
		},
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second,
		DisableKeepAlives:     true,
	}
	client := &http.Client{
		Transport:     transport,
		Timeout:       timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", ErrUnsafeRemoteURL
	}
	req.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,image/gif,image/tiff,image/bmp,*/*;q=0.1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch: download remote image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("fetch: remote image returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxBytes {
		return nil, "", fmt.Errorf("%w: max %d bytes", ErrRemoteImageTooLarge, maxBytes)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("fetch: read remote image: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, "", fmt.Errorf("%w: max %d bytes", ErrRemoteImageTooLarge, maxBytes)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func (f RemoteImage) publicIPs(ctx context.Context, host string) ([]netip.Addr, error) {
	if ip, err := netip.ParseAddr(host); err == nil {
		if !isPublicIP(ip) {
			return nil, ErrUnsafeRemoteURL
		}
		return []netip.Addr{ip}, nil
	}
	var ips []netip.Addr
	var err error
	if f.lookupIP != nil {
		ips, err = f.lookupIP(ctx, host)
	} else {
		resolver := f.Resolver
		if resolver == nil {
			resolver = net.DefaultResolver
		}
		ips, err = resolver.LookupNetIP(ctx, "ip", host)
	}
	if err != nil || len(ips) == 0 {
		if err == nil {
			err = errors.New("no addresses")
		}
		return nil, err
	}
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return nil, ErrUnsafeRemoteURL
		}
	}
	return ips, nil
}

func isPublicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	for _, prefix := range nonPublicPrefixes {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}
