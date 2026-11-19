package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type imageToolsURLResolver struct {
	urls  []string
	calls int
	err   error
}

func (r *imageToolsURLResolver) ResolveScenarioURL(_ context.Context, scenario, portKey string) (string, error) {
	r.calls++
	if scenario != "image-tools" || portKey != "API_PORT" {
		return "", fmt.Errorf("unexpected target %s %s", scenario, portKey)
	}
	if r.err != nil {
		return "", r.err
	}
	index := r.calls - 1
	if index >= len(r.urls) {
		index = len(r.urls) - 1
	}
	if index < 0 {
		return "", errors.New("no scenario URL configured")
	}
	return r.urls[index], nil
}

func TestDiscoveryImageToolsHeavyJobSourceReResolvesAfterPortDrift(t *testing.T) {
	resolver := &imageToolsURLResolver{urls: []string{
		"http://127.0.0.1:17850",
		"http://127.0.0.1:17851",
	}}
	source := NewDiscoveryImageToolsHeavyJobSource(resolver, nil)
	base, err := source.resolveBaseURL(context.Background())
	if err != nil || base != "http://127.0.0.1:17850" {
		t.Fatalf("first base URL = %q, %v", base, err)
	}
	base, err = source.resolveBaseURL(context.Background())
	if err != nil || base != "http://127.0.0.1:17851" || resolver.calls != 2 {
		t.Fatalf("base URL after peer restart = %q, %v, resolver calls=%d", base, err, resolver.calls)
	}
}

func TestDiscoveryImageToolsHeavyJobSourcePropagatesResolutionError(t *testing.T) {
	wantErr := errors.New("target is stopped")
	source := NewDiscoveryImageToolsHeavyJobSource(&imageToolsURLResolver{err: wantErr}, nil)
	if _, err := source.resolveBaseURL(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("resolveBaseURL error = %v, want wrapped resolver error", err)
	}
}

func TestDiscoveryImageToolsHeavyJobSourceReadsOwnerTelemetryThroughResolvedURL(t *testing.T) {
	now := time.Date(2026, 11, 15, 19, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != imageToolsHeavyJobTelemetryPath || r.Method != http.MethodGet {
			t.Errorf("request = %s %s, want GET %s", r.Method, r.URL.Path, imageToolsHeavyJobTelemetryPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"observed_at":%q,"complete":true,"jobs":[{"owner_id":"image-tools","job_id":"j-1","operation":"text_to_image","state":"running","started_at":%q,"waiter_count":0,"heavy":true}]}`, now.Format(time.RFC3339Nano), now.Add(-time.Hour).Format(time.RFC3339Nano))
	}))
	defer server.Close()
	resolver := &imageToolsURLResolver{urls: []string{server.URL}}
	source := NewDiscoveryImageToolsHeavyJobSource(resolver, server.Client())
	got, err := source.GetHeavyJobs(context.Background())
	if err != nil {
		t.Fatalf("GetHeavyJobs: %v", err)
	}
	if !got.Complete || !got.ObservedAt.Equal(now) || len(got.Jobs) != 1 || got.Jobs[0].OwnerID != "image-tools" || got.Jobs[0].JobID != "j-1" {
		t.Fatalf("owner telemetry = %+v", got)
	}
	if resolver.calls != 1 {
		t.Fatalf("resolved URL calls=%d, want one resolution per telemetry request", resolver.calls)
	}
}
