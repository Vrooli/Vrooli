package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	continuityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/continuity/continuity_v1connect"
	"web-console/internal/continuity"
)

// Health must observe continuity catalog links without reading retained event
// bodies. The deliberate failing text projection detects any content audit;
// the explicit integrity operation must still read it and report the failure.
func TestHealthDoesNotReadConversationBodiesButExplicitIntegrityDoes(t *testing.T) {
	db := setupTestDB(t)
	for _, statement := range []string{
		`DROP TABLE conversation_events`,
		`CREATE VIEW conversation_events AS SELECT 'trap' AS id, 'test' AS session_id, 1 AS sequence, 'user' AS role, '2026-11-05T00:00:00Z' AS created_at, json_extract('invalid JSON', '$') AS text`,
	} {
		if _, err := db.ExecContext(context.Background(), statement); err != nil {
			t.Fatal(err)
		}
	}
	srv := newFakeTestServer()
	srv.db = database.NewFromPrimary(db)
	srv.setupRoutes()
	for _, path := range []string{"/health", "/api/v1/health"} {
		response := httptest.NewRecorder()
		srv.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s read conversation bodies: status=%d body=%s", path, response.Code, response.Body.String())
		}
		var report health.Response
		if err := json.Unmarshal(response.Body.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		if report.Functional == nil || !report.Functional.Healthy || report.Status != health.StatusHealthy {
			t.Fatalf("%s performed a conversation content audit: %s", path, response.Body.String())
		}
	}
	// The monetization journey probes and the Archive drawer's warning ask
	// whether the continuity store answers, not for a conservation receipt.
	for _, operation := range []string{"provider_observation", "communication_operation"} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/internal/monetization/journey?operation="+operation, nil)
		request.RemoteAddr = "127.0.0.1:1234"
		response := httptest.NewRecorder()
		srv.router.ServeHTTP(response, request)
		if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "unsupported:") {
			t.Fatalf("journey %s read conversation bodies: status=%d body=%s", operation, response.Code, response.Body.String())
		}
	}
	request := httptest.NewRequest(http.MethodPost, continuityconnect.ContinuityServiceHealthProcedure, strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	srv.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "{") {
		t.Fatalf("Health RPC read conversation bodies: status=%d body=%s", response.Code, response.Body.String())
	}
	if _, err := srv.Integrity(context.Background()); err == nil {
		t.Fatal("explicit integrity audit no longer checks conversation content")
	}
}

func TestContinuityHealthObservesCatalogChangesAndErrorsImmediately(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	srv := &Server{db: database.NewFromPrimary(db)}
	if _, err := db.ExecContext(ctx, `INSERT INTO conversation_sessions(session_id,created_at,updated_at) VALUES ('uncataloged','2026-11-05T00:00:00Z','2026-11-05T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	first, err := srv.observeContinuityHealth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first.UncatalogedConversations != 1 {
		t.Fatalf("uncataloged count=%d", first.UncatalogedConversations)
	}
	if err := continuity.NewSQLCatalogStore(db).Upsert(ctx, continuity.CatalogRecord{SessionID: "uncataloged", LifecycleState: continuity.StateRecoverable, SourceFingerprint: "fixture-health-fingerprint"}); err != nil {
		t.Fatal(err)
	}
	second, err := srv.observeContinuityHealth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if second.UncatalogedConversations != 0 || second.OrphanConversations != 1 {
		t.Fatalf("cataloged observation=%+v", second)
	}
	if _, err := db.ExecContext(ctx, `DROP TABLE conversation_catalog`); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.observeContinuityHealth(ctx); err == nil || !strings.Contains(err.Error(), "conversation_catalog") {
		t.Fatalf("missing catalog was treated as healthy: %v", err)
	}
}
