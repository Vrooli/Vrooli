package fetch

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

func TestRemoteImageRejectsURLClassesBeforeDNS(t *testing.T) {
	fetcher := RemoteImage{}
	for _, raw := range []string{
		"file:///etc/passwd",
		"ftp://example.org/image.png",
		"http://user:pass@example.org/image.png",
		"http://localhost/image.png",
		"http://api.local/image.png",
		"http://example.org:8080/image.png",
		"https://example.org:8443/image.png",
	} {
		t.Run(raw, func(t *testing.T) {
			_, _, err := fetcher.Fetch(context.Background(), raw)
			if !errors.Is(err, ErrUnsafeRemoteURL) {
				t.Fatalf("Fetch(%q) error=%v, want ErrUnsafeRemoteURL", raw, err)
			}
		})
	}
}

func TestRemoteImageRejectsNonPublicAddresses(t *testing.T) {
	fetcher := RemoteImage{}
	for _, raw := range []string{
		"http://127.0.0.1/image.png",
		"http://10.1.2.3/image.png",
		"http://172.16.1.1/image.png",
		"http://192.168.1.1/image.png",
		"http://169.254.169.254/latest/meta-data/",
		"http://[::1]/image.png",
		"http://[fd00::1]/image.png",
		"http://[fe80::1]/image.png",
	} {
		t.Run(raw, func(t *testing.T) {
			_, _, err := fetcher.Fetch(context.Background(), raw)
			if !errors.Is(err, ErrUnsafeRemoteURL) {
				t.Fatalf("Fetch(%q) error=%v, want ErrUnsafeRemoteURL", raw, err)
			}
		})
	}
}

func TestRemoteImagePublicIPPolicy(t *testing.T) {
	for _, tc := range []struct {
		ip     string
		wantOK bool
	}{
		{ip: "1.1.1.1", wantOK: true},
		{ip: "2606:4700:4700::1111", wantOK: true},
		{ip: "0.0.0.0", wantOK: false},
		{ip: "100.64.0.1", wantOK: false},
		{ip: "198.18.0.1", wantOK: false},
		{ip: "2001:db8::1", wantOK: false},
	} {
		ip := netip.MustParseAddr(tc.ip)
		if got := isPublicIP(ip); got != tc.wantOK {
			t.Errorf("isPublicIP(%s)=%t want %t", ip, got, tc.wantOK)
		}
	}
}

func TestRemoteImageRejectsMixedPublicAndPrivateDNSAnswers(t *testing.T) {
	fetcher := RemoteImage{lookupIP: func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("10.0.0.5")}, nil
	}}
	_, _, err := fetcher.Fetch(context.Background(), "https://mixed.example/image.png")
	if !errors.Is(err, ErrUnsafeRemoteURL) {
		t.Fatalf("Fetch error=%v, want ErrUnsafeRemoteURL", err)
	}
}

func TestRemoteImageDoesNotFollowRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "images.example" {
			t.Errorf("Host=%q", r.Host)
		}
		http.Redirect(w, r, "http://127.0.0.1/private.png", http.StatusFound)
	}))
	defer server.Close()
	serverAddr := strings.TrimPrefix(server.URL, "http://")
	dials := 0
	fetcher := RemoteImage{
		lookupIP: func(context.Context, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("1.1.1.1")}, nil
		},
		dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			dials++
			return (&net.Dialer{}).DialContext(ctx, network, serverAddr)
		},
	}
	_, _, err := fetcher.Fetch(context.Background(), "http://images.example/image.png")
	if err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("Fetch error=%v, want HTTP 302 response", err)
	}
	if dials != 1 {
		t.Fatalf("dial count=%d, want one request and no redirect", dials)
	}
}

func TestRemoteImageEnforcesExactResponseByteLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("123"))
	}))
	defer server.Close()
	serverAddr := strings.TrimPrefix(server.URL, "http://")
	fetcher := RemoteImage{
		MaxBytes: 3,
		lookupIP: func(context.Context, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("1.1.1.1")}, nil
		},
		dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, serverAddr)
		},
	}
	body, _, err := fetcher.Fetch(context.Background(), "http://images.example/image.png")
	if err != nil || string(body) != "123" {
		t.Fatalf("exact limit result=%q err=%v", body, err)
	}
	fetcher.MaxBytes = 2
	_, _, err = fetcher.Fetch(context.Background(), "http://images.example/image.png")
	if !errors.Is(err, ErrRemoteImageTooLarge) {
		t.Fatalf("over-limit error=%v, want ErrRemoteImageTooLarge", err)
	}
}
