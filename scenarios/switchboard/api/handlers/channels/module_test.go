package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/switchboard/v1/channels"
	"switchboard/internal/agents"
	channelcore "switchboard/internal/channels"
	"switchboard/internal/gates"
)

func TestGateRESTPersistsAndEnforcesOwner(t *testing.T) {
	dir := t.TempDir()
	descriptor := `{"kind":"channel","schemaVersion":1,"id":"fixture","displayName":"Fixture","transport":"fixture","supports":{"text":true},"limits":{"maxTextBytes":100},"setup":{"friction":0},"cost":"free"}`
	if err := os.WriteFile(filepath.Join(dir, "fixture.json"), []byte(descriptor), 0o644); err != nil {
		t.Fatal(err)
	}
	registry, err := channelcore.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	db := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(gates.Schema), apidb.SchemaProviderFunc(agents.Schema)); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(ModuleDeps{Registry: registry, DB: db, Gates: gates.NewStore(db, nil)}).Mount(router)

	reqBody, _ := json.Marshal(map[string]any{"thread_id": "thread-1", "owner_id": "owner-1", "scope": "files.write", "withheld": "private file", "unblock": "owner approval", "ttl_seconds": 60})
	unauthenticated := httptest.NewRequest(http.MethodPost, "/api/v1/gates", bytes.NewReader(reqBody))
	unauthenticatedResp := httptest.NewRecorder()
	router.ServeHTTP(unauthenticatedResp, unauthenticated)
	if unauthenticatedResp.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated raise status=%d body=%s", unauthenticatedResp.Code, unauthenticatedResp.Body.String())
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gates", bytes.NewReader(reqBody))
	req.Header.Set("X-Vrooli-Identity-Subject", "owner-1")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("raise status=%d body=%s", resp.Code, resp.Body.String())
	}
	var gate gates.Gate
	if err := json.Unmarshal(resp.Body.Bytes(), &gate); err != nil {
		t.Fatal(err)
	}

	answerBody, _ := json.Marshal(map[string]any{"actor_id": "owner-1", "grant": true})
	answer := httptest.NewRequest(http.MethodPost, "/api/v1/gates/"+gate.ID+"/answer", bytes.NewReader(answerBody))
	answer.Header.Set("X-Vrooli-Identity-Subject", "viewer-1")
	answerResp := httptest.NewRecorder()
	router.ServeHTTP(answerResp, answer)
	if answerResp.Code != http.StatusForbidden {
		t.Fatalf("non-owner answer status=%d body=%s", answerResp.Code, answerResp.Body.String())
	}

	answerBody, _ = json.Marshal(map[string]any{"actor_id": "owner-1", "grant": true})
	answer = httptest.NewRequest(http.MethodPost, "/api/v1/gates/"+gate.ID+"/answer", bytes.NewReader(answerBody))
	answer.Header.Set("X-Vrooli-Identity-Subject", "owner-1")
	answerResp = httptest.NewRecorder()
	router.ServeHTTP(answerResp, answer)
	if answerResp.Code != http.StatusOK {
		t.Fatalf("owner answer status=%d body=%s", answerResp.Code, answerResp.Body.String())
	}

	svc := &service{deps: ModuleDeps{DB: db}}
	_, err = svc.CreateBinding(context.Background(), connect.NewRequest(&channelv1.CreateBindingRequest{AgentId: "agent-1", ChannelId: "in-app", Address: "browser"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unauthenticated binding error code=%s, want unauthenticated", connect.CodeOf(err))
	}
	bound := connect.NewRequest(&channelv1.CreateBindingRequest{AgentId: "agent-1", ChannelId: "in-app", Address: "browser"})
	bound.Header().Set("X-Vrooli-Identity-Subject", "owner-1")
	_, err = svc.CreateBinding(context.Background(), bound)
	if err != nil {
		t.Fatalf("authenticated binding: %v", err)
	}
}
