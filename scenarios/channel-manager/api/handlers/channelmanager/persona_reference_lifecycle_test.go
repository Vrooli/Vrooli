package channelmanager

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	core "channel-manager/internal/channelmanager"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

func TestPersonaReferenceCorrectionSurvivesRetirementAndReload(t *testing.T) {
	platforms := []core.Platform{{ID: "x", DailyCeiling: 2, ActionKinds: []string{"engage"}, Formats: testFormats()}}
	service, err := core.New(platforms, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.CreateIdentity(core.Identity{
		ID: "artist-world-trial", PlatformID: "x", Purpose: "persona-actor",
		PersonaRef: "asset-studio://ira-v1", EnvironmentRef: "env",
		CredentialRef: "authority://artist-world-trial", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(core.Schema()); err != nil {
		t.Fatal(err)
	}
	store := core.NewStore(db)
	router := mux.NewRouter()
	Module(service, store).Mount(router)
	call := func(method, path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		raw, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		request := httptest.NewRequest(method, path, bytes.NewReader(raw))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	update := call(http.MethodPut, "/api/v1/channel-manager/identities/artist-world-trial", map[string]any{
		"handle": "@ira", "display_label": "Ira", "purpose": "persona-actor",
		"persona_ref": "asset-studio://ira-v2", "environment_ref": "env",
		"lifecycle": "active", "automation_mode": "manual",
	})
	if update.Code != http.StatusOK {
		t.Fatalf("correct persona reference: status=%d body=%s", update.Code, update.Body.String())
	}
	retire := call(http.MethodPost, "/api/v1/channel-manager/identities/artist-world-trial/retire", map[string]any{})
	if retire.Code != http.StatusOK {
		t.Fatalf("retire identity: status=%d body=%s", retire.Code, retire.Body.String())
	}

	reloaded, err := core.New(platforms, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Load(context.Background(), reloaded); err != nil {
		t.Fatal(err)
	}
	identity := reloaded.Identities["artist-world-trial"]
	if identity == nil || identity.PersonaRef != "asset-studio://ira-v2" || identity.Status != "retired" || identity.Lifecycle != "retired" {
		t.Fatalf("reloaded identity=%+v; want corrected reference and retired lifecycle", identity)
	}
}

func TestRetiredIdentityCannotEnqueueActions(t *testing.T) {
	platforms := []core.Platform{{ID: "x", DailyCeiling: 2, ActionKinds: []string{"publish"}, Formats: testFormats()}}
	service, err := core.New(platforms, nil)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(core.Schema()); err != nil {
		t.Fatal(err)
	}
	store := core.NewStore(db)
	router := mux.NewRouter()
	Module(service, store).Mount(router)
	call := func(method, path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		raw, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		request := httptest.NewRequest(method, path, bytes.NewReader(raw))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	create := call(http.MethodPost, "/api/v1/channel-manager/identities", core.Identity{
		ID: "retired-queue-trial", PlatformID: "x", Purpose: "persona-actor",
		EnvironmentRef: "env", CredentialRef: "authority://retired-queue-trial",
		Status: "active", Lifecycle: "active",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create identity: status=%d body=%s", create.Code, create.Body.String())
	}
	retire := call(http.MethodPost, "/api/v1/channel-manager/identities/retired-queue-trial/retire", map[string]any{})
	if retire.Code != http.StatusOK {
		t.Fatalf("retire identity: status=%d body=%s", retire.Code, retire.Body.String())
	}
	queued := call(http.MethodPost, "/api/v1/channel-manager/actions", map[string]any{
		"identity_id": "retired-queue-trial", "kind": "publish", "idempotency_key": "retired-queue-trial-publish",
	})
	if queued.Code != http.StatusConflict {
		t.Fatalf("enqueue after retirement: status=%d body=%s", queued.Code, queued.Body.String())
	}

	reloaded, err := core.New(platforms, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Load(context.Background(), reloaded); err != nil {
		t.Fatal(err)
	}
	identity := reloaded.Identities["retired-queue-trial"]
	if identity == nil || identity.Status != "retired" || identity.Lifecycle != "retired" || len(reloaded.Actions) != 0 {
		t.Fatalf("reloaded identity=%+v actions=%d; want retired identity and no actions", identity, len(reloaded.Actions))
	}
}
