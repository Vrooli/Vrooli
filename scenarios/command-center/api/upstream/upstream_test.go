package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDescriptorClientFetchesHTTPJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/stats" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"value":42}`))
	}))
	defer srv.Close()

	client := NewDescriptorClient(Descriptor{ID: "fixture", Transport: TransportHTTPJSON, ResolveBase: func() string { return srv.URL }})
	body, err := client.Fetch(context.Background(), "/stats")
	if err != nil || string(body) != `{"value":42}` {
		t.Fatalf("body=%s err=%v", body, err)
	}
}

func TestDescriptorClientFetchesConnectProjection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/example.v1.Service/Get" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["projection"] != "ledger" || body["window_days"] != float64(30) {
			t.Fatalf("projection body=%#v", body)
		}
		_, _ = w.Write([]byte(`{"observed_at":"2026-09-17T00:00:00Z","value":7}`))
	}))
	defer srv.Close()

	client := NewDescriptorClient(Descriptor{ID: "fixture", Transport: TransportConnect, ResolveBase: func() string { return srv.URL }})
	if _, err := client.Fetch(context.Background(), "/example.v1.Service/Get?projection=ledger&window_days=30"); err != nil {
		t.Fatal(err)
	}
}

func TestDescriptorClientUsesGenericHealthAndAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/health" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer descriptor-token" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	client := NewDescriptorClient(Descriptor{
		ID: "fixture", Transport: TransportConnect, ResolveBase: func() string { return srv.URL },
		Auth: func(req *http.Request) { req.Header.Set("Authorization", "Bearer descriptor-token") },
	})
	body, err := client.Fetch(context.Background(), "/health")
	if err != nil || string(body) != `{"status":"ok"}` {
		t.Fatalf("body=%s err=%v", body, err)
	}
}

func TestDescriptorClientMapsUnavailableSourcesToGapMode(t *testing.T) {
	client := NewDescriptorClient(Descriptor{ID: "missing", ResolveBase: func() string { return "http://127.0.0.1:1" }})
	_, err := client.Fetch(context.Background(), "/stats")
	if !errors.Is(err, ErrNotAvailable) {
		t.Fatalf("expected ErrNotAvailable, got %v", err)
	}
}

func TestDescriptorClientFeatureProbeUsesDeclaredProjection(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := NewDescriptorClient(Descriptor{
		ID: "fixture", ResolveBase: func() string { return srv.URL }, Paths: []string{"/probe"},
		Features: map[string]string{"throughput": ""},
	})
	probe, ok := client.(FeatureProbe)
	if !ok {
		t.Fatal("descriptor client does not expose FeatureProbe")
	}
	features, reasons := probe.ProbeFeatures(context.Background())
	if calls != 1 || features["throughput"] != "compatible" || reasons["throughput"] == "" {
		t.Fatalf("calls=%d features=%#v reasons=%#v", calls, features, reasons)
	}
}

func TestLegacyConstructorNamesAreDescriptorBacked(t *testing.T) {
	clients := []Client{NewSwarm(""), NewVrooli(""), NewLPBS("", "")}
	want := []string{"swarm", "vrooli", "lpbs"}
	for i, client := range clients {
		if client.Name() != want[i] {
			t.Fatalf("client %d name=%q want %q", i, client.Name(), want[i])
		}
		if _, ok := client.(*descriptorClient); !ok {
			t.Fatalf("client %q is %T, want descriptorClient", client.Name(), client)
		}
	}
}

func TestLPBSRemoteProfileRelaysProductionProcedure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer local-service" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/api/v1/admin/remote-profiles":
			if r.Method != http.MethodGet {
				t.Fatalf("profile method=%s", r.Method)
			}
			_, _ = w.Write([]byte(`{"profiles":[{"id":12,"tag":"prod"}]}`))
		case "/api/v1/admin/remote-profiles/12/proxy":
			if r.Method != http.MethodPost {
				t.Fatalf("proxy method=%s", r.Method)
			}
			var request remoteProfileProxyRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Path != "/landing_page_business_suite.v1.AdminRevenueService/GetRevenueSummary" || request.Method != http.MethodPost {
				t.Fatalf("proxy request=%#v", request)
			}
			_, _ = w.Write([]byte(`{"total_revenue":12}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewLPBSRemoteProfileResolved(func() string { return srv.URL }, func() string { return "local-service" }, "prod")
	body, err := client.Fetch(context.Background(), "/api/v1/admin/dashboard/revenue")
	if err != nil || string(body) != `{"total_revenue":12}` {
		t.Fatalf("body=%s err=%v", body, err)
	}
}
