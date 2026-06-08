// Pins the contract that swarm-manager spawn helpers populate
// CreateRunRequest.ConversationId explicitly per Decision D7 of the
// agent-sandbox auditability contract. Spawn surfaces should never rely on
// agent-manager's mint-on-empty fallback.

package agentmanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
)

// captureCreateRunSurface stands up a fake agent-manager that captures the
// CreateRunRequest body sent by the spawn helper.
func captureCreateRunSurface(t *testing.T) (*httptest.Server, *capturedCreateRun) {
	t.Helper()
	cap := &capturedCreateRun{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/profiles/reconcile-scenario", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"scenario":"swarm-manager","results":[{"profileKey":"swarm-manager/default","profileId":"00000000-0000-0000-0000-000000000099","status":"PROFILE_RECONCILE_STATUS_CREATED"}],"created":1}`))
	})
	mux.HandleFunc("/api/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"task":{"id":"task-1","title":"t","scopePath":"."}}`))
	})
	mux.HandleFunc("/api/v1/runs", func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		var req apipb.CreateRunRequest
		if err := protojson.Unmarshal(body, &req); err != nil {
			t.Errorf("decode CreateRunRequest: %v", err)
		}
		cap.req = &req
		_, _ = w.Write([]byte(`{"run":{"id":"run-1","taskId":"task-1","status":"RUN_STATUS_PENDING"}}`))
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, cap
}

type capturedCreateRun struct {
	req *apipb.CreateRunRequest
}

func newSpawnTestService(t *testing.T, baseURL string) *AgentService {
	t.Helper()
	// The lifecycle always injects the scenario identity into a running
	// swarm-manager process; the variant-aware research tag (buildResearchTag)
	// resolves its namespace from it. Mirror that here so spawn tests exercise
	// the live path instead of failing loud on a missing namespace.
	t.Setenv("VROOLI_SCENARIO", "swarm-manager")
	svc := NewAgentService(AgentServiceConfig{
		ProfileName: "swarm-manager",
		ProfileKey:  "swarm-manager/default",
		Enabled:     true,
	})
	svc.client = NewHTTPClientWithResolver(func(_ context.Context) (string, error) {
		return baseURL, nil
	}, nil)
	return svc
}

func TestSpawnResearch_PopulatesConversationId(t *testing.T) {
	srv, cap := captureCreateRunSurface(t)
	svc := newSpawnTestService(t, srv.URL)

	_, err := svc.SpawnResearch(context.Background(), ResearchSpawnRequest{
		IdeaName:    "demo",
		Prompt:      "go",
		ScopePath:   ".",
		ProjectRoot: "/tmp/p",
	})
	if err != nil {
		t.Fatalf("SpawnResearch: %v", err)
	}
	if cap.req == nil {
		t.Fatal("CreateRunRequest was not captured")
	}
	if cap.req.ConversationId == nil || strings.TrimSpace(*cap.req.ConversationId) == "" {
		t.Fatal("expected SpawnResearch to populate CreateRunRequest.ConversationId")
	}
}

func TestSpawnBacklog_PopulatesConversationId(t *testing.T) {
	srv, cap := captureCreateRunSurface(t)
	svc := newSpawnTestService(t, srv.URL)

	_, err := svc.SpawnBacklog(context.Background(), BacklogSpawnRequest{
		Kind:        "execute",
		Name:        "demo",
		Prompt:      "do",
		ScopePath:   ".",
		ProjectRoot: "/tmp/p",
	})
	if err != nil {
		t.Fatalf("SpawnBacklog: %v", err)
	}
	if cap.req == nil || cap.req.ConversationId == nil || *cap.req.ConversationId == "" {
		t.Fatal("expected SpawnBacklog to populate CreateRunRequest.ConversationId")
	}
}

func TestSpawnInitiative_PopulatesConversationId(t *testing.T) {
	srv, cap := captureCreateRunSurface(t)
	svc := newSpawnTestService(t, srv.URL)

	_, err := svc.SpawnInitiative(context.Background(), InitiativeSpawnRequest{
		Name:        "demo",
		Purpose:     "review",
		Prompt:      "do",
		ScopePath:   ".",
		ProjectRoot: "/tmp/p",
		RoundNumber: 1,
	})
	if err != nil {
		t.Fatalf("SpawnInitiative: %v", err)
	}
	if cap.req == nil || cap.req.ConversationId == nil || *cap.req.ConversationId == "" {
		t.Fatal("expected SpawnInitiative to populate CreateRunRequest.ConversationId")
	}
}

// Two consecutive spawns should produce two distinct ConversationIds; each
// top-level spawn from swarm-manager is conceptually a fresh conversation.
func TestFreshConversationID_Unique(t *testing.T) {
	a := freshConversationID()
	b := freshConversationID()
	if a == nil || b == nil {
		t.Fatal("freshConversationID returned nil")
	}
	if *a == *b {
		t.Fatalf("expected distinct ConversationIds, got %q twice", *a)
	}
}
