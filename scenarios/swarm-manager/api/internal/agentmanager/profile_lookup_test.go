package agentmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInitializeResolvesExistingOwnerProfileWithReadOnlyGet(t *testing.T) {
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
			t.Errorf("profile initialization must not POST to Agent Manager: %s", r.URL.Path)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/profiles" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("limit") != "128" || r.URL.Query().Get("offset") != "" {
			t.Errorf("profile read bounds = %v, want limit=128 and no offset", r.URL.Query())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"profiles":[{"id":"default-id","profileKey":"swarm-manager/default","ownerScenario":"swarm-manager","sourcePath":".vrooli/agent-manager/default.json","sourceHash":"same","lastAppliedHash":"same"},{"id":"analysis-id","profileKey":"swarm-manager/analysis","ownerScenario":"swarm-manager"}],"total":2}`)
	}))
	defer server.Close()

	svc := NewAgentService(AgentServiceConfig{ProfileKey: "swarm-manager/default", Enabled: true, Timeout: time.Second})
	svc.client = NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	if err := svc.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if got := svc.GetProfileID(); got != "default-id" {
		t.Fatalf("resolved profile ID = %q, want default-id", got)
	}
	if _, err := svc.profileRefFor("swarm-manager/analysis"); err != nil {
		t.Fatalf("same-owner profile key was not retained for run admission: %v", err)
	}
	if posts != 0 {
		t.Fatalf("profile initialization made %d POST requests", posts)
	}
}

func TestInitializeDoesNotAcceptForeignOwnerOrCreateMissingProfile(t *testing.T) {
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
			t.Errorf("profile initialization must not POST to Agent Manager: %s", r.URL.Path)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"profiles":[{"id":"foreign-id","profileKey":"swarm-manager/default","ownerScenario":"another-scenario"}],"total":1}`)
	}))
	defer server.Close()

	svc := NewAgentService(AgentServiceConfig{ProfileKey: "swarm-manager/default", Enabled: true, Timeout: time.Second})
	svc.client = NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	err := svc.Initialize(context.Background())
	if err == nil || !strings.Contains(err.Error(), "required profile") || !strings.Contains(err.Error(), "declaration owner") {
		t.Fatalf("Initialize error = %v, want explicit unresolved registration routed to declaration owner", err)
	}
	if posts != 0 {
		t.Fatalf("unresolved profile read made %d POST requests", posts)
	}
}



func TestInitializeRejectsBlankIDForOwnedProfileWithoutPosting(t *testing.T) {
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
			t.Errorf("profile initialization must not POST to Agent Manager: %s", r.URL.Path)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"profiles":[{"id":"","profileKey":"swarm-manager/default","ownerScenario":"swarm-manager"}],"total":1}`)
	}))
	defer server.Close()

	svc := NewAgentService(AgentServiceConfig{ProfileKey: "swarm-manager/default", Enabled: true, Timeout: time.Second})
	svc.client = NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	err := svc.Initialize(context.Background())
	if err == nil || !strings.Contains(err.Error(), "invalid profile identity") {
		t.Fatalf("Initialize error = %v, want invalid owner profile identity", err)
	}
	if posts != 0 {
		t.Fatalf("invalid owner profile read made %d POST requests", posts)
	}
}

func TestInitializeWarnsWhenResolvedOwnerProfileHasDeclarationDrift(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/profiles" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"profiles":[{"id":"default-id","profileKey":"swarm-manager/default","ownerScenario":"swarm-manager","sourcePath":".vrooli/agent-manager/default.json","sourceHash":"declared","lastAppliedHash":"applied","localOverride":true}],"total":1}`)
	}))
	defer server.Close()

	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	defer slog.SetDefault(previousLogger)

	svc := NewAgentService(AgentServiceConfig{ProfileKey: "swarm-manager/default", Enabled: true, Timeout: time.Second})
	svc.client = NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	if err := svc.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize should use the resolved owner profile while leaving reconciliation to its owner: %v", err)
	}
	if !strings.Contains(output.String(), "agent-manager profile has declaration drift metadata") || !strings.Contains(output.String(), "swarm-manager/default") {
		t.Fatalf("drift diagnostic missing from logs: %s", output.String())
	}
}

func TestInitializeAcceptsRequiredOwnerMatchWithPartialBoundedCoverage(t *testing.T) {
	const pageSize = 128
	const pageCount = 8
	allProfiles := make([]map[string]string, pageSize*pageCount)
	allProfiles[0] = map[string]string{"id": "default-id", "profileKey": "swarm-manager/default", "ownerScenario": "swarm-manager"}
	for i := 1; i < len(allProfiles); i++ {
		allProfiles[i] = map[string]string{"id": fmt.Sprintf("id-%d", i), "profileKey": fmt.Sprintf("swarm-manager/profile-%d", i), "ownerScenario": "swarm-manager"}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/profiles" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotFound)
			return
		}
		offset := 0
		if raw := r.URL.Query().Get("offset"); raw != "" {
			if _, err := fmt.Sscanf(raw, "%d", &offset); err != nil {
				t.Errorf("invalid offset %q: %v", raw, err)
			}
		}
		end := offset + pageSize
		if end > len(allProfiles) {
			end = len(allProfiles)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"profiles": allProfiles[offset:end], "total": pageSize})
	}))
	defer server.Close()

	svc := NewAgentService(AgentServiceConfig{ProfileKey: "swarm-manager/default", Enabled: true, Timeout: time.Second})
	svc.client = NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	if err := svc.Initialize(context.Background()); err != nil {
		t.Fatalf("known required owner/key match should resolve despite bounded global coverage: %v", err)
	}
	if got := svc.GetProfileID(); got != "default-id" {
		t.Fatalf("resolved profile ID = %q, want default-id", got)
	}
	if _, err := svc.profileRefFor("swarm-manager/profile-1024"); err == nil {
		t.Fatal("profile key outside observed pages should not be admitted from partial coverage")
	}
}
