package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/evidence"
	"swarm-manager/internal/identity"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/cli-core/cliutil"
	_ "modernc.org/sqlite"
)

type evidenceOwnerIndex struct{ owners []evidence.Owner }

func (i evidenceOwnerIndex) LookupOwners(context.Context, string) ([]evidence.Owner, error) {
	return i.owners, nil
}

func TestEvidenceInvocationMiddlewareRecordsOnlyVerifiedSuccessfulCLIRequests(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := evidence.NewStore(database.NewFromPrimary(db))
	if err := store.InitSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	service := evidence.NewService(store, evidence.RunOwnerResolver{
		Sessions:       evidenceOwnerIndex{owners: []evidence.Owner{{Kind: evidence.OwnerAgentSession, ID: "session-1"}}},
		OperatingModes: evidenceOwnerIndex{},
	})
	server := &Server{evidenceSvc: service}
	handler := server.evidenceInvocationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog", nil)
	req.Header.Set(cliutil.HeaderInvocationScenario, "swarm-manager")
	req.Header.Set(cliutil.HeaderInvocationCommand, "backlog create")
	req.Header.Set(cliutil.HeaderInvocationID, "cli-invocation-1")
	req = req.WithContext(identity.NewContext(req.Context(), identity.Provenance{Type: identity.TypeAgent, RunID: "run-1"}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
	var observations, links int
	if err := db.QueryRow(`SELECT COUNT(*) FROM evidence_observations`).Scan(&observations); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM evidence_links`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if observations != 1 || links != 1 {
		t.Fatalf("observations=%d links=%d", observations, links)
	}

	failure := server.evidenceInvocationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	failure.ServeHTTP(httptest.NewRecorder(), req)
	if err := db.QueryRow(`SELECT COUNT(*) FROM evidence_observations`).Scan(&observations); err != nil {
		t.Fatal(err)
	}
	if observations != 1 {
		t.Fatalf("failed request must not create evidence, got %d observations", observations)
	}
}
