package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	agentconfig "agent-manager/internal/config"
	aisearch "github.com/vrooli/ai-go/search"
	searchregister "github.com/vrooli/searchregister-go"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

func TestDatabaseConfigUsesGovernedPoolLevers(t *testing.T) {
	config := databaseConfigFromLevers("file:test.db", agentconfig.StorageLevers{
		MaxOpenConns: 17, MaxIdleConns: 4, ConnMaxLifetime: 7 * time.Minute,
	})
	if config.MaxOpenConns != 17 || config.MaxIdleConns != 4 || config.ConnMaxLifetime != 7*time.Minute {
		t.Fatalf("database pool ignored storage levers: %+v", config)
	}
}

// TestNewServerBuildsAndShutsDownTheRealCompositionRoot is a lightweight
// composition-root proof: it uses a disposable SQLite store and a bounded
// empty project root, but otherwise exercises the same graph construction,
// recovery ordering, middleware, and cleanup path as production startup.
func TestNewServerBuildsAndShutsDownTheRealCompositionRoot(t *testing.T) {
	t.Setenv("AM_SQLITE_PATH", filepath.Join(t.TempDir(), "agent-manager.db"))
	t.Setenv("UPLOAD_DIR", t.TempDir())
	t.Setenv("PROJECT_ROOT", t.TempDir())
	server, err := NewServer()
	if err != nil {
		t.Fatalf("build server: %v", err)
	}
	if server.Router() == nil || server.orchestrator == nil || server.reconciler == nil || server.awaitRegistry == nil || server.workflowNudger == nil {
		t.Fatalf("incomplete service graph: %+v", server)
	}
	t.Cleanup(func() {
		if err := server.Cleanup(); err != nil {
			t.Errorf("cleanup server: %v", err)
		}
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	server.Router().ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestEnvOrEmptyReflectsConfiguredAndAbsentValues(t *testing.T) {
	t.Setenv("AM_TEST_ENV", "configured")
	if got := envOrEmpty("AM_TEST_ENV"); got != "configured" {
		t.Fatalf("configured env=%q", got)
	}
	if got := envOrEmpty("AM_UNSET_TEST_ENV"); got != "" {
		t.Fatalf("unset env=%q", got)
	}
}

func TestConversationSearchDescriptorConformsToLiveProviderContract(t *testing.T) {
	file, err := aisearch.LoadSearchFile("../.vrooli/search.json")
	if err != nil {
		t.Fatalf("load committed search descriptor: %v", err)
	}
	descriptors, err := searchregister.Descriptors(file)
	if err != nil {
		t.Fatalf("map committed search descriptor: %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("descriptor count=%d, want 1", len(descriptors))
	}
	if err := searchregister.ValidateRegistration(file.Providers[0]); err != nil {
		t.Fatalf("descriptor would be rejected during live self-registration: %v", err)
	}
	d := descriptors[0]
	if d.GetProviderId() != "agent-manager.runs" || d.GetLifecycle() != registryv1.Lifecycle_LIFECYCLE_PRODUCTION || d.GetScope() != registryv1.Scope_SCOPE_PROJECT {
		t.Fatalf("unexpected provider identity/state/scope: %+v", d)
	}
	if d.GetType() != "run" || d.GetEndpoint().GetHttpJson().GetScenarioId() != "agent-manager" || d.GetStatusEndpoint() == nil || d.GetReindexEndpoint() == nil || d.GetConfigEndpoint() == nil {
		t.Fatalf("incomplete live provider contract: %+v", d)
	}
	if got := d.GetResultMapping().GetIdField(); got != "stableHitId" {
		t.Fatalf("id mapping=%q, want native stableHitId", got)
	}
	mapping := d.GetResultMapping()
	if got := mapping.GetMetadataFields()["run_id"]; got != "runId" {
		t.Fatalf("run identity mapping=%q, want runId", got)
	}
	if mapping.GetRankEvidenceField() != "rankEvidence" || mapping.GetCoverageField() != "coverage" || mapping.GetDegradationsField() != "degradations" || mapping.GetNextCursorField() != "nextPageCursor" {
		t.Fatalf("incomplete native evidence mappings: %+v", mapping)
	}
}

func TestSearchRegistrationStartsBeforeListener(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	registerAt := strings.Index(text, "srv.startSearchRegistration(")
	serveAt := strings.Index(text, "server.Run(")
	if registerAt < 0 || serveAt < 0 || registerAt >= serveAt {
		t.Fatalf("registration must start before listener: register=%d serve=%d", registerAt, serveAt)
	}
}

func TestSearchControlTokensRejectEmptyAndRoundTripMintedToken(t *testing.T) {
	tokens := newSearchControlTokens()
	tokens.set("agent-manager.runs", "")
	if got := tokens.get("agent-manager.runs"); got != "" {
		t.Fatalf("empty token persisted: %q", got)
	}
	tokens.set("agent-manager.runs", "minted-token")
	if got := tokens.get("agent-manager.runs"); got != "minted-token" {
		t.Fatalf("token=%q", got)
	}
}
