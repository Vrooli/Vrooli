package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func withHostFactsProbe(t *testing.T, probe func(context.Context) (hostFactsResponse, error)) {
	t.Helper()
	hostFactsMu.Lock()
	originalProbe := hostFactsProbe
	originalCached := cachedHostFacts
	originalCachedAt := cachedHostFactsAt
	hostFactsProbe = probe
	cachedHostFacts = hostFactsResponse{}
	cachedHostFactsAt = time.Time{}
	hostFactsMu.Unlock()
	t.Cleanup(func() {
		hostFactsMu.Lock()
		hostFactsProbe = originalProbe
		cachedHostFacts = originalCached
		cachedHostFactsAt = originalCachedAt
		hostFactsMu.Unlock()
	})
}

func TestV2HostFactsHealthyPathIsCached(t *testing.T) {
	calls := 0
	withHostFactsProbe(t, func(context.Context) (hostFactsResponse, error) {
		calls++
		cpu := 8
		return hostFactsResponse{Available: true, CPUCount: &cpu, Platform: "linux/amd64"}, nil
	})

	server := NewServer()
	for range 2 {
		response := doPost(t, server, "/vrooli.vrooli_onboarding.v1.host.HostService/GetHostFacts", `{"target":"local"}`)
		if response.Code != http.StatusOK || response.Body.String() == "" {
			t.Fatalf("host facts response = %d %q", response.Code, response.Body.String())
		}
	}
	if calls != 1 {
		t.Fatalf("probe calls = %d, want one cached probe", calls)
	}
}

func TestV2HostFactsTimeoutDegradesWithHTTP200(t *testing.T) {
	withHostFactsProbe(t, func(ctx context.Context) (hostFactsResponse, error) {
		<-ctx.Done()
		return hostFactsResponse{}, ctx.Err()
	})

	started := time.Now()
	response := doPost(t, NewServer(), "/vrooli.vrooli_onboarding.v1.host.HostService/GetHostFacts", `{"target":"local"}`)
	if response.Code != http.StatusOK || response.Body.String() == "" {
		t.Fatalf("timeout response = %d %q", response.Code, response.Body.String())
	}
	if elapsed := time.Since(started); elapsed > 2500*time.Millisecond {
		t.Fatalf("timeout took %s", elapsed)
	}
}

func TestV2HostFactsPartialProbeKeepsAvailableFields(t *testing.T) {
	withHostFactsProbe(t, func(context.Context) (hostFactsResponse, error) {
		memory := uint64(123)
		return hostFactsResponse{Available: true, MemoryTotalBytes: &memory}, context.DeadlineExceeded
	})

	response := doPost(t, NewServer(), "/vrooli.vrooli_onboarding.v1.host.HostService/GetHostFacts", `{"target":"local"}`)
	if response.Code != http.StatusOK || response.Body.String() == "" {
		t.Fatalf("partial response = %d %q", response.Code, response.Body.String())
	}
	if got := response.Body.String(); !containsAll(got, `"available":true`, `"memoryTotalBytes":"123"`) {
		t.Fatalf("partial response lost fields: %s", got)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !contains(value, part) {
			return false
		}
	}
	return true
}

func contains(value, part string) bool {
	for len(value) >= len(part) {
		if value[:len(part)] == part {
			return true
		}
		value = value[1:]
	}
	return false
}
