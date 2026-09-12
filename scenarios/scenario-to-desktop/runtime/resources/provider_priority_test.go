package resources

import (
	"context"
	"errors"
	"testing"
	"time"
)

type priorityResolver struct {
	name    string
	called  *[]string
	binding SharedServiceBinding
	err     error
}

func (r priorityResolver) ResolveSharedService(_ context.Context, _ Item) (SharedServiceBinding, error) {
	*r.called = append(*r.called, r.name)
	return r.binding, r.err
}

func TestPrioritySharedServiceResolverPrefersLocalVrooliThenDesktopPeer(t *testing.T) {
	called := []string{}
	expires := time.Now().Add(time.Minute)
	peer := priorityResolver{name: "peer", called: &called, binding: SharedServiceBinding{Endpoint: "http://127.0.0.1:48002", ExpiresAt: expires}}
	local := priorityResolver{name: "local", called: &called, err: errors.New("tier1 unavailable")}
	resolver, err := NewPrioritySharedServiceResolver(
		SharedServiceCandidate{Tier: SharedProviderTierDesktopPeer, Resolver: peer},
		SharedServiceCandidate{Tier: SharedProviderTierLocalVrooli, Resolver: local},
	)
	if err != nil {
		t.Fatal(err)
	}

	binding, err := resolver.ResolveSharedService(context.Background(), Item{Resource: "vault"})
	if err != nil {
		t.Fatalf("ResolveSharedService: %v", err)
	}
	if got, want := called, []string{"local", "peer"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("provider order = %v, want %v", got, want)
	}
	if binding.Provider != string(SharedProviderTierDesktopPeer) {
		t.Fatalf("provider = %q, want %q", binding.Provider, SharedProviderTierDesktopPeer)
	}
}

func TestPrioritySharedServiceResolverStopsAtLocalProvider(t *testing.T) {
	called := []string{}
	expires := time.Now().Add(time.Minute)
	local := priorityResolver{name: "local", called: &called, binding: SharedServiceBinding{Endpoint: "http://127.0.0.1:48001", ExpiresAt: expires}}
	peer := priorityResolver{name: "peer", called: &called, binding: SharedServiceBinding{Endpoint: "http://127.0.0.1:48002", ExpiresAt: expires}}
	resolver, err := NewPrioritySharedServiceResolver(
		SharedServiceCandidate{Tier: SharedProviderTierDesktopPeer, Resolver: peer},
		SharedServiceCandidate{Tier: SharedProviderTierLocalVrooli, Resolver: local},
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := resolver.ResolveSharedService(context.Background(), Item{Resource: "postgres"}); err != nil {
		t.Fatal(err)
	}
	if len(called) != 1 || called[0] != "local" {
		t.Fatalf("lower-priority provider was consulted: %v", called)
	}
}

func TestPrioritySharedServiceResolverRejectsDuplicateOrUnknownTiers(t *testing.T) {
	resolver := priorityResolver{called: &[]string{}}
	if _, err := NewPrioritySharedServiceResolver(
		SharedServiceCandidate{Tier: SharedProviderTierLocalVrooli, Resolver: resolver},
		SharedServiceCandidate{Tier: SharedProviderTierLocalVrooli, Resolver: resolver},
	); err == nil {
		t.Fatal("duplicate provider tier was accepted")
	}
	if _, err := NewPrioritySharedServiceResolver(SharedServiceCandidate{Tier: "unknown", Resolver: resolver}); err == nil {
		t.Fatal("unknown provider tier was accepted")
	}
}
