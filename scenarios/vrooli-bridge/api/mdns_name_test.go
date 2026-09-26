package main

import (
	"context"
	"errors"
	"testing"
)

func TestLocalMDNSNameIsUsedOnlyWhenItResolvesToTheRouteAddress(t *testing.T) {
	hostname := func() (string, error) { return "swarminator.lan", nil }
	resolves := func(addrs ...string) func(context.Context, string) ([]string, error) {
		return func(_ context.Context, host string) ([]string, error) {
			if host != "swarminator.local" {
				t.Fatalf("looked up %q, want the short name under .local", host)
			}
			return addrs, nil
		}
	}
	if got := localMDNSName("192.168.1.173", hostname, resolves("172.17.0.1", "192.168.1.173")); got != "swarminator.local" {
		t.Fatalf("got %q, want swarminator.local", got)
	}
	if got := localMDNSName("192.168.1.173", hostname, resolves("172.17.0.1")); got != "" {
		t.Fatalf("a name that does not reach the route address must not be used, got %q", got)
	}
	failing := func(context.Context, string) ([]string, error) { return nil, errors.New("no mdns") }
	if got := localMDNSName("192.168.1.173", hostname, failing); got != "" {
		t.Fatalf("an unresolvable name must not be used, got %q", got)
	}
	if got := localMDNSName("127.0.0.1", hostname, resolves("127.0.0.1")); got != "" {
		t.Fatalf("loopback keeps the address, got %q", got)
	}
}
