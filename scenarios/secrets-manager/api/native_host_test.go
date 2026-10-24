package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func nativeHostRequest(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Workspace-ID", "workspace-a")
	request.Header.Set("X-Actor-ID", "owner-a")
	request.Header.Set("X-Secrets-Manager-Native-Host-Token", "native-host-test-token")
	router.ServeHTTP(response, request)
	return response
}

func TestNativeHostAuthorityBindsGrantOriginRevisionAndRevocation(t *testing.T) {
	t.Setenv("SECRETS_MANAGER_NATIVE_HOST_TOKEN", "native-host-test-token")
	router := newAPIServer(nil, NewLogger("native-host-test")).routes()
	if response := nativeHostRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/unlock", nil); response.Code != http.StatusOK {
		t.Fatalf("vault unlock status = %d: %s", response.Code, response.Body.String())
	}

	create := nativeHostRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/items", createItemInput{
		Type: "login", Name: "Native host fixture", Username: "alice", URI: "https://example.test/login",
		Fields: map[string]string{"password": "native-secret"},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", create.Code, create.Body.String())
	}
	var item VaultItem
	if err := json.Unmarshal(create.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}

	grant := nativeHostRequest(t, router, http.MethodPost, "/api/v1/grants", map[string]any{
		"vault_id": "vault-a", "item_id": item.ID, "principal_type": "human", "principal_id": "owner-a",
		"selector_mode": "current_snapshot", "members": []string{"owner-a"}, "operations": []string{"use"},
		"target": "https://example.test", "expires_at": time.Now().UTC().Add(time.Hour),
	})
	if grant.Code != http.StatusCreated {
		t.Fatalf("grant status = %d: %s", grant.Code, grant.Body.String())
	}
	var grantBody grantRecord
	if err := json.Unmarshal(grant.Body.Bytes(), &grantBody); err != nil {
		t.Fatal(err)
	}

	scope := map[string]any{
		"extension_id": "native-host-fixture", "workspace_id": "workspace-a", "grant_id": grantBody.ID,
		"origin": "https://example.test", "tab_id": "tab-1", "frame_id": "frame-1", "document_id": "doc-1",
		"item_id": item.ID, "item_revision": item.Revision, "field": "password",
	}
	enroll := nativeHostRequest(t, router, http.MethodPost, "/api/v1/native-host/enroll", map[string]any{
		"extension_id": "native-host-fixture", "origin": "https://example.test",
	})
	if enroll.Code != http.StatusOK {
		t.Fatalf("enroll status = %d: %s", enroll.Code, enroll.Body.String())
	}
	metadata := nativeHostRequest(t, router, http.MethodPost, "/api/v1/native-host/metadata", map[string]any{
		"extension_id": "native-host-fixture", "workspace_id": "workspace-a", "origin": "https://example.test",
		"tab_id": "tab-1", "frame_id": "frame-1", "document_id": "doc-1",
	})
	if metadata.Code != http.StatusOK || !strings.Contains(metadata.Body.String(), item.ID) || strings.Contains(metadata.Body.String(), "native-secret") {
		t.Fatalf("metadata status = %d: %s", metadata.Code, metadata.Body.String())
	}
	save := nativeHostRequest(t, router, http.MethodPost, "/api/v1/native-host/save", map[string]any{
		"extension_id": "native-host-fixture", "workspace_id": "workspace-a", "vault_id": "vault-a",
		"origin": "https://example.test", "tab_id": "tab-1", "frame_id": "frame-1", "document_id": "doc-1",
		"name": "Saved from browser", "username": "alice", "uri": "https://example.test/login", "password": "save-secret",
	})
	if save.Code != http.StatusCreated || strings.Contains(save.Body.String(), "save-secret") {
		t.Fatalf("save status = %d: %s", save.Code, save.Body.String())
	}
	var saved nativeHostMetadataRecord
	if err := json.Unmarshal(save.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	update := nativeHostRequest(t, router, http.MethodPost, "/api/v1/native-host/update", map[string]any{
		"extension_id": "native-host-fixture", "workspace_id": "workspace-a", "vault_id": "vault-a",
		"origin": "https://example.test", "tab_id": "tab-1", "frame_id": "frame-1", "document_id": "doc-1",
		"item_id": saved.ItemID, "item_revision": saved.Revision, "name": "Updated from browser", "username": "alice",
		"uri": "https://example.test/login", "password": "updated-secret",
	})
	if update.Code != http.StatusOK || strings.Contains(update.Body.String(), "updated-secret") {
		t.Fatalf("update status = %d: %s", update.Code, update.Body.String())
	}
	staleUpdate := cloneNativeScope(map[string]any{
		"extension_id": "native-host-fixture", "workspace_id": "workspace-a", "vault_id": "vault-a",
		"origin": "https://example.test", "tab_id": "tab-1", "frame_id": "frame-1", "document_id": "doc-1",
		"item_id": saved.ItemID, "item_revision": saved.Revision, "name": "Stale browser update", "username": "alice",
		"uri": "https://example.test/login", "password": "stale-secret",
	})
	if response := nativeHostRequest(t, router, http.MethodPost, "/api/v1/native-host/update", staleUpdate); response.Code != http.StatusConflict {
		t.Fatalf("stale update status = %d: %s", response.Code, response.Body.String())
	}
	unlock := nativeHostRequest(t, router, http.MethodPost, "/api/v1/native-host/unlock", scope)
	if unlock.Code != http.StatusOK {
		t.Fatalf("unlock status = %d: %s", unlock.Code, unlock.Body.String())
	}
	fill := nativeHostRequest(t, router, http.MethodPost, "/api/v1/native-host/fill", scope)
	if fill.Code != http.StatusOK || !strings.Contains(fill.Body.String(), "native-secret") {
		t.Fatalf("fill status = %d: %s", fill.Code, fill.Body.String())
	}

	wrongOrigin := cloneNativeScope(scope)
	wrongOrigin["origin"] = "https://other.example"
	if response := nativeHostRequest(t, router, http.MethodPost, "/api/v1/native-host/fill", wrongOrigin); response.Code != http.StatusForbidden {
		t.Fatalf("wrong-origin status = %d: %s", response.Code, response.Body.String())
	}
	stale := cloneNativeScope(scope)
	stale["item_revision"] = item.Revision + 1
	if response := nativeHostRequest(t, router, http.MethodPost, "/api/v1/native-host/fill", stale); response.Code != http.StatusConflict {
		t.Fatalf("stale-revision status = %d: %s", response.Code, response.Body.String())
	}
	revoke := nativeHostRequest(t, router, http.MethodPost, "/api/v1/native-host/revoke", map[string]any{
		"extension_id": "native-host-fixture", "origin": "https://example.test",
	})
	if revoke.Code != http.StatusOK {
		t.Fatalf("revoke status = %d: %s", revoke.Code, revoke.Body.String())
	}
	if response := nativeHostRequest(t, router, http.MethodPost, "/api/v1/native-host/fill", scope); response.Code != http.StatusForbidden {
		t.Fatalf("fill after revoke status = %d: %s", response.Code, response.Body.String())
	}
}

func cloneNativeScope(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
