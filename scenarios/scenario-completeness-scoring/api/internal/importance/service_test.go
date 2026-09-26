package importance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type staticResolver map[string]string

func (r staticResolver) ResolveScenarioURLDefault(_ context.Context, scenario string) (string, error) {
	return r[scenario], nil
}

func TestFetchCombinesCentralityAndRecency(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/dep/api/v1/graph/centrality", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"nodes": [{
				"scenario": "web-search",
				"direct_reverse_dependency_count": 2,
				"transitive_reverse_dependency_count": 5,
				"required_reverse_dependency_count": 3,
				"required_edge_weighted_score": 8,
				"distance_to_core_seed": 1,
				"nearest_core_seed": "test-genie"
			}]
		}`))
	})
	mux.HandleFunc("/swarm/api/v1/operations", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"activities": [
				{"owner_type": "scenario", "owner_name": "web-search"},
				{"owner_type": "scenario", "owner_name": "other"}
			],
			"recently_finished": [
				{"owner_kind": "scenario", "owner_name": "web-search"}
			]
		}`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	svc := New(Config{
		HTTPClient: server.Client(),
		Resolver: staticResolver{
			"scenario-dependency-analyzer": server.URL + "/dep",
			"swarm-manager":                server.URL + "/swarm",
		},
		Timeout: time.Second,
	})
	got := svc.Fetch(context.Background(), "web-search", true)
	if got == nil {
		t.Fatal("Fetch returned nil")
	}
	if !got.SystemRequired {
		t.Fatal("system_required not preserved")
	}
	if got.Score < defaultSystemRequiredFloor {
		t.Fatalf("score = %v, want floor >= %v", got.Score, defaultSystemRequiredFloor)
	}
	if got.Signals.TransitiveReverseDependencyCount != 5 || got.Signals.RecentActivityCount != 2 {
		t.Fatalf("signals = %+v", got.Signals)
	}
	if got.Signals.NearestCoreSeed != "test-genie" {
		t.Fatalf("nearest core = %q", got.Signals.NearestCoreSeed)
	}
	if len(got.Degraded) != 0 {
		t.Fatalf("unexpected degraded notes: %+v", got.Degraded)
	}
}

func TestFetchOmittedWhenEverySourceMisses(t *testing.T) {
	svc := New(Config{
		Resolver: staticResolver{},
		Timeout:  10 * time.Millisecond,
	})
	if got := svc.Fetch(context.Background(), "web-search", false); got != nil {
		t.Fatalf("Fetch = %+v, want nil", got)
	}
}

func TestFetchPartialSourceUsesNeutralAndDegradedNote(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/dep/api/v1/graph/centrality", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"nodes": [{
				"scenario": "web-search",
				"required_edge_weighted_score": 2,
				"distance_to_core_seed": -1
			}]
		}`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	svc := New(Config{
		HTTPClient: server.Client(),
		Resolver: staticResolver{
			"scenario-dependency-analyzer": server.URL + "/dep",
		},
		Timeout: time.Second,
	})
	got := svc.Fetch(context.Background(), "web-search", false)
	if got == nil {
		t.Fatal("Fetch returned nil")
	}
	if got.Components.Recency != defaultNeutralScore {
		t.Fatalf("recency component = %v, want neutral", got.Components.Recency)
	}
	if len(got.Degraded) != 1 {
		t.Fatalf("degraded = %+v, want one recency note", got.Degraded)
	}
}
