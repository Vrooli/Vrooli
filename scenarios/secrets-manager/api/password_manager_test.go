package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"secrets-manager-api/internal/vault"
)

func vaultRequest(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &payload)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-ID", "workspace-a")
	req.Header.Set("X-Actor-ID", "owner-a")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}

func TestPasswordManagerEncryptsPayloadAndRequiresBoundAssurance(t *testing.T) {
	router := newAPIServer(nil, NewLogger("test")).routes()
	if response := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/unlock", nil); response.Code != http.StatusOK {
		t.Fatalf("unlock status = %d: %s", response.Code, response.Body.String())
	}
	create := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/items", createItemInput{
		Type: "login", Name: "Production", Username: "alice", URI: "https://example.test/login",
		Tags: []string{"Prod", "prod"}, Fields: map[string]string{"password": " exact  secret\n"},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", create.Code, create.Body.String())
	}
	var item VaultItem
	if err := json.Unmarshal(create.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if item.ID == "" || item.Revision != 1 || len(item.Tags) != 1 || item.Tags[0] != "prod" {
		t.Fatalf("unexpected metadata: %+v", item)
	}
	if bytes.Contains(create.Body.Bytes(), []byte("exact  secret")) {
		t.Fatal("create response disclosed encrypted field")
	}

	list := vaultRequest(t, router, http.MethodGet, "/api/v1/vaults/vault-a/items", nil)
	if list.Code != http.StatusOK || bytes.Contains(list.Body.Bytes(), []byte("exact  secret")) {
		t.Fatalf("metadata list disclosed secret or failed: %d %s", list.Code, list.Body.String())
	}

	assurance := vaultRequest(t, router, http.MethodPost, "/api/v1/assurance", map[string]string{
		"operation": "reveal:" + item.ID,
	})
	if assurance.Code != http.StatusCreated {
		t.Fatalf("assurance status = %d: %s", assurance.Code, assurance.Body.String())
	}
	var assuranceBody struct {
		Token string `json:"assurance_token"`
	}
	if err := json.Unmarshal(assurance.Body.Bytes(), &assuranceBody); err != nil {
		t.Fatal(err)
	}
	reveal := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/items/"+item.ID+"/reveal", map[string]any{
		"assurance_token": assuranceBody.Token,
		"fields":          []string{"password"},
	})
	var revealBody struct {
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(reveal.Body.Bytes(), &revealBody); err != nil {
		t.Fatal(err)
	}
	if reveal.Code != http.StatusOK || revealBody.Fields["password"] != " exact  secret\n" {
		t.Fatalf("reveal failed: %d %s", reveal.Code, reveal.Body.String())
	}
	if bytes.Contains(reveal.Body.Bytes(), []byte("alice")) {
		t.Fatal("field-scoped reveal disclosed unrelated username")
	}

	replay := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/items/"+item.ID+"/reveal", map[string]any{
		"assurance_token": assuranceBody.Token,
		"fields":          []string{"password"},
	})
	if replay.Code != http.StatusPreconditionRequired {
		t.Fatalf("assurance replay status = %d, want %d: %s", replay.Code, http.StatusPreconditionRequired, replay.Body.String())
	}
}

func TestPasswordManagerLocksAndRejectsStaleWrites(t *testing.T) {
	router := newAPIServer(nil, NewLogger("test")).routes()
	if response := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/unlock", nil); response.Code != http.StatusOK {
		t.Fatalf("unlock status = %d: %s", response.Code, response.Body.String())
	}
	create := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/items", map[string]any{
		"type": "password", "name": "Deploy", "fields": map[string]string{"password": "one"},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", create.Code, create.Body.String())
	}
	var item VaultItem
	if err := json.Unmarshal(create.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	// Send the revision-sensitive operation directly so the optimistic lock
	// header is present before routing.
	request := httptest.NewRequest(http.MethodPut, "/api/v1/vaults/vault-a/items/"+item.ID, bytes.NewBufferString(`{"type":"password","name":"Deploy updated","fields":{"password":"two"}}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Workspace-ID", "workspace-a")
	request.Header.Set("X-Actor-ID", "owner-a")
	request.Header.Set("If-Match", "1")
	first := httptest.NewRecorder()
	router.ServeHTTP(first, request)
	if first.Code != http.StatusOK {
		t.Fatalf("first update status = %d: %s", first.Code, first.Body.String())
	}
	history := vaultRequest(t, router, http.MethodGet, "/api/v1/vaults/vault-a/items/"+item.ID+"/history", nil)
	if history.Code != http.StatusOK || !bytes.Contains(history.Body.Bytes(), []byte(`"version":1`)) {
		t.Fatalf("history status = %d: %s", history.Code, history.Body.String())
	}
	stale := httptest.NewRequest(http.MethodPut, "/api/v1/vaults/vault-a/items/"+item.ID, bytes.NewBufferString(`{"type":"password","name":"stale","fields":{"password":"three"}}`))
	stale.Header.Set("Content-Type", "application/json")
	stale.Header.Set("X-Workspace-ID", "workspace-a")
	stale.Header.Set("X-Actor-ID", "owner-a")
	stale.Header.Set("If-Match", "1")
	conflict := httptest.NewRecorder()
	router.ServeHTTP(conflict, stale)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("stale update status = %d, want %d: %s", conflict.Code, http.StatusConflict, conflict.Body.String())
	}

	locked := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/lock", nil)
	if locked.Code != http.StatusOK {
		t.Fatalf("lock status = %d: %s", locked.Code, locked.Body.String())
	}
	blocked := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/items/"+item.ID+"/reveal", map[string]any{})
	if blocked.Code != http.StatusLocked {
		t.Fatalf("locked reveal status = %d, want %d: %s", blocked.Code, http.StatusLocked, blocked.Body.String())
	}
}

func TestPasswordManagerBindsAssuranceToRequestDigest(t *testing.T) {
	router := newAPIServer(nil, NewLogger("test")).routes()
	if response := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/unlock", nil); response.Code != http.StatusOK {
		t.Fatalf("unlock status = %d: %s", response.Code, response.Body.String())
	}
	create := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/items", map[string]any{
		"type": "password", "name": "Digest-bound", "fields": map[string]string{"password": "synthetic"},
	})
	var item VaultItem
	if err := json.Unmarshal(create.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	digest := "reveal:" + item.ID + ":password"
	assurance := vaultRequest(t, router, http.MethodPost, "/api/v1/assurance", map[string]string{
		"operation": "reveal:" + item.ID, "request_digest": digest,
	})
	var token struct {
		Value string `json:"assurance_token"`
	}
	if err := json.Unmarshal(assurance.Body.Bytes(), &token); err != nil {
		t.Fatal(err)
	}
	wrong := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/items/"+item.ID+"/reveal", map[string]any{
		"assurance_token": token.Value, "request_digest": "reveal:" + item.ID + ":notes", "fields": []string{"password"},
	})
	if wrong.Code != http.StatusPreconditionRequired {
		t.Fatalf("mismatched assurance status = %d, want %d: %s", wrong.Code, http.StatusPreconditionRequired, wrong.Body.String())
	}
}

func TestPasswordManagerReportsRecoveryEvidenceWithoutClaimingBackup(t *testing.T) {
	router := newAPIServer(nil, NewLogger("test")).routes()
	response := vaultRequest(t, router, http.MethodGet, "/api/v1/recovery/status", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("recovery status = %d: %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["backup_status"] != "not_configured" || body["restore_status"] != "not_run" || body["recovery_ready"] != false {
		t.Fatalf("recovery status made an unsupported claim: %+v", body)
	}
}

func TestPasswordManagerProjectsVerifiedBackupAndRestoreEvidence(t *testing.T) {
	server := newAPIServer(nil, NewLogger("test"))
	server.handlers.passwordManager.backupEvidence = func(context.Context) recoveryEvidenceSnapshot {
		return recoveryEvidenceSnapshot{
			BackupStatus:  "verified",
			RestoreStatus: "verified",
			RecoveryReady: true,
			Evidence: []recoveryEvidenceReference{{
				Kind:             "durable-backup-coverage",
				ArtifactIdentity: "data-backup-manager/coverage",
				Verified:         true,
			}},
			UpdatedAt: time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC),
		}
	}
	response := vaultRequest(t, server.routes(), http.MethodGet, "/api/v1/recovery/status", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("recovery status = %d: %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["backup_status"] != "verified" || body["restore_status"] != "verified" || body["recovery_ready"] != true {
		t.Fatalf("unexpected verified recovery status: %+v", body)
	}
}

func TestPasswordManagerOwnerEnrollmentIsSingleUse(t *testing.T) {
	t.Setenv("SECRETS_MANAGER_OWNER_BOOTSTRAP_TOKEN", "bootstrap-secret")
	t.Setenv("SECRETS_MANAGER_WORKSPACE_ID", "workspace-a")
	router := newAPIServer(nil, NewLogger("test")).routes()
	status := vaultRequest(t, router, http.MethodGet, "/api/v1/enrollment/status", nil)
	if status.Code != http.StatusOK || bytes.Contains(status.Body.Bytes(), []byte("bootstrap-secret")) {
		t.Fatalf("unexpected enrollment status: %d %s", status.Code, status.Body.String())
	}
	complete := vaultRequest(t, router, http.MethodPost, "/api/v1/enrollment/complete", map[string]string{"bootstrap_token": "bootstrap-secret"})
	if complete.Code != http.StatusCreated || !bytes.Contains(complete.Body.Bytes(), []byte(`"token_once":true`)) {
		t.Fatalf("enrollment status = %d: %s", complete.Code, complete.Body.String())
	}
	if bytes.Contains(complete.Body.Bytes(), []byte("bootstrap-secret")) {
		t.Fatal("enrollment response echoed bootstrap material")
	}
	replay := vaultRequest(t, router, http.MethodPost, "/api/v1/enrollment/complete", map[string]string{"bootstrap_token": "bootstrap-secret"})
	if replay.Code != http.StatusConflict {
		t.Fatalf("enrollment replay status = %d, want %d: %s", replay.Code, http.StatusConflict, replay.Body.String())
	}
}

func TestGeneratePasswordHonorsBoundsAndCharacterClasses(t *testing.T) {
	h := newPasswordManagerHandlers(nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/vault/password/generate", bytes.NewBufferString(`{"length":32,"upper":false,"lower":false,"digits":true,"symbols":false}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	h.generate(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("generate status = %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Password) != 32 {
		t.Fatalf("generated password length = %d", len(body.Password))
	}
	for _, char := range body.Password {
		if char < '2' || char > '9' {
			t.Fatalf("generated password contains disallowed character %q", char)
		}
	}
}

func TestGenerateTOTPCodeUsesRFC6238AndActionableBounds(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	code, err := generateTOTPCode(secret, time.Unix(59, 0), 30, 6, "SHA1")
	if err != nil || code != "287082" {
		t.Fatalf("RFC 6238 TOTP = %q, err=%v; want 287082", code, err)
	}
	uriCode, err := generateTOTPCode("otpauth://totp/example?secret="+secret, time.Unix(59, 0), 30, 6, "")
	if err != nil || uriCode != code {
		t.Fatalf("otpauth URI TOTP = %q, err=%v; want %q", uriCode, err, code)
	}
	for _, test := range []struct {
		name   string
		period int
		digits int
		want   string
	}{
		{name: "period", period: 0, digits: 6, want: "period"},
		{name: "digits", period: 30, digits: 7, want: "digits"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := generateTOTPCode(secret, time.Unix(59, 0), test.period, test.digits, "SHA1"); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want actionable %q error", err, test.want)
			}
		})
	}
}

func TestPasswordManagerGrantSnapshotAndApprovalDigest(t *testing.T) {
	router := newAPIServer(nil, NewLogger("test")).routes()
	grantResponse := vaultRequest(t, router, http.MethodPost, "/api/v1/grants", map[string]any{
		"vault_id": "vault-a", "item_id": "item-a", "principal_type": "agent",
		"principal_id": "agent-a", "selector_mode": "current_snapshot", "operations": []string{"use"},
	})
	if grantResponse.Code != http.StatusCreated {
		t.Fatalf("grant status = %d: %s", grantResponse.Code, grantResponse.Body.String())
	}
	var grant grantRecord
	if err := json.Unmarshal(grantResponse.Body.Bytes(), &grant); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/access-requests", bytes.NewBufferString(`{"grant_id":"`+grant.ID+`","item_id":"item-a","operation":"use"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Workspace-ID", "workspace-a")
	request.Header.Set("X-Actor-ID", "agent-a")
	pending := httptest.NewRecorder()
	router.ServeHTTP(pending, request)
	if pending.Code != http.StatusCreated {
		t.Fatalf("request status = %d: %s", pending.Code, pending.Body.String())
	}
	var access accessRequest
	if err := json.Unmarshal(pending.Body.Bytes(), &access); err != nil {
		t.Fatal(err)
	}
	if access.Status != "pending" || access.RequestDigest == "" {
		t.Fatalf("unexpected pending request: %+v", access)
	}

	badDecision := vaultRequest(t, router, http.MethodPost, "/api/v1/access-requests/"+access.ID+"/approve", map[string]string{"request_digest": "wrong"})
	if badDecision.Code != http.StatusConflict {
		t.Fatalf("stale approval status = %d, want %d: %s", badDecision.Code, http.StatusConflict, badDecision.Body.String())
	}
	goodDecision := vaultRequest(t, router, http.MethodPost, "/api/v1/access-requests/"+access.ID+"/approve", map[string]string{"request_digest": access.RequestDigest})
	if goodDecision.Code != http.StatusOK {
		t.Fatalf("approval status = %d: %s", goodDecision.Code, goodDecision.Body.String())
	}
	secondDecision := vaultRequest(t, router, http.MethodPost, "/api/v1/access-requests/"+access.ID+"/deny", map[string]string{"request_digest": access.RequestDigest})
	if secondDecision.Code != http.StatusConflict {
		t.Fatalf("second decision status = %d, want %d: %s", secondDecision.Code, http.StatusConflict, secondDecision.Body.String())
	}
}

func TestPasswordManagerExpiredAccessRequestBecomesTerminal(t *testing.T) {
	manager := newPasswordManager(nil)
	ctx := context.Background()
	grant, err := manager.createGrant(ctx, "workspace-a", "owner-a", createGrantInput{
		VaultID: "vault-a", ItemID: "item-a", PrincipalType: "agent", PrincipalID: "agent-a",
		Members: []string{"agent-a"}, Operations: []string{"use"}, ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := manager.createAccessRequest(ctx, "workspace-a", "agent-a", createAccessRequestInput{GrantID: grant.ID, ItemID: "item-a", Operation: "use", DurationSec: 1})
	if err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	expired := manager.requests[request.ID]
	expired.ExpiresAt = time.Now().Add(-time.Second)
	manager.requests[request.ID] = expired
	manager.mu.Unlock()
	if _, err := manager.decideAccessRequest(ctx, "workspace-a", "owner-a", request.ID, request.RequestDigest, "approved", ""); !errors.Is(err, errAccessDenied) {
		t.Fatalf("expired decision error = %v, want access denied", err)
	}
	manager.mu.RLock()
	status := manager.requests[request.ID].Status
	manager.mu.RUnlock()
	if status != "expired" {
		t.Fatalf("expired request status = %q, want expired", status)
	}
}

func TestAccessRequestRetryIdentityAndWaitResume(t *testing.T) {
	manager := newPasswordManager(nil)
	ctx := context.Background()
	grant, err := manager.createGrant(ctx, "workspace-a", "owner-a", createGrantInput{
		VaultID: "vault-a", PrincipalType: "agent", PrincipalID: "agent-a", Members: []string{"agent-a"},
		Operations: []string{"use"}, ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := manager.createAccessRequest(ctx, "workspace-a", "agent-a", createAccessRequestInput{GrantID: grant.ID, ItemID: "item-a", Operation: "use", IdempotencyKey: "retry-1"})
	if err != nil {
		t.Fatal(err)
	}
	retried, err := manager.createAccessRequest(ctx, "workspace-a", "agent-a", createAccessRequestInput{GrantID: grant.ID, ItemID: "item-a", Operation: "use", IdempotencyKey: "retry-1"})
	if err != nil || retried.ID != request.ID {
		t.Fatalf("idempotent request retry = %+v, err=%v; want original %s", retried, err, request.ID)
	}
	if _, err := manager.createAccessRequest(ctx, "workspace-a", "agent-a", createAccessRequestInput{GrantID: grant.ID, ItemID: "item-b", Operation: "use", IdempotencyKey: "retry-1"}); !errors.Is(err, errItemConflict) {
		t.Fatalf("idempotency scope mismatch = %v, want conflict", err)
	}

	result := make(chan struct {
		request  accessRequest
		timedOut bool
		err      error
	}, 1)
	go func() {
		resumed, timedOut, waitErr := manager.waitForAccessRequest(ctx, "workspace-a", request.ID, 2*time.Second)
		result <- struct {
			request  accessRequest
			timedOut bool
			err      error
		}{resumed, timedOut, waitErr}
	}()
	time.Sleep(20 * time.Millisecond)
	if _, err := manager.decideAccessRequest(ctx, "workspace-a", "owner-a", request.ID, request.RequestDigest, "approved", "reviewed"); err != nil {
		t.Fatal(err)
	}
	select {
	case waited := <-result:
		if waited.err != nil || waited.timedOut || waited.request.Status != "approved" {
			t.Fatalf("wait result = %+v", waited)
		}
	case <-time.After(time.Second):
		t.Fatal("wait did not resume after authority decision")
	}
	denied, err := manager.createAccessRequest(ctx, "workspace-a", "agent-a", createAccessRequestInput{GrantID: grant.ID, ItemID: "item-a", Operation: "use", IdempotencyKey: "deny-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.decideAccessRequest(ctx, "workspace-a", "owner-a", denied.ID, denied.RequestDigest, "denied", "needs review"); err != nil {
		t.Fatal(err)
	}
	terminal, timedOut, err := manager.waitForAccessRequest(ctx, "workspace-a", denied.ID, time.Second)
	if err != nil || timedOut || terminal.Status != "denied" {
		t.Fatalf("denied wait result = %+v, timed_out=%v, err=%v", terminal, timedOut, err)
	}
}

func TestAccessRequestIdempotencySurvivesManagerRestart(t *testing.T) {
	db := openWorkspaceSecurityDB(t)
	ctx := context.Background()
	first := newPasswordManager(db)
	grant, err := first.createGrant(ctx, "workspace-a", "owner-a", createGrantInput{
		VaultID: "vault-a", PrincipalType: "agent", PrincipalID: "agent-a", Members: []string{"agent-a"},
		Operations: []string{"use"}, ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	original, err := first.createAccessRequest(ctx, "workspace-a", "agent-a", createAccessRequestInput{GrantID: grant.ID, ItemID: "item-a", Operation: "use", IdempotencyKey: "restart-1"})
	if err != nil {
		t.Fatal(err)
	}
	second := newPasswordManager(db)
	restarted, err := second.createAccessRequest(ctx, "workspace-a", "agent-a", createAccessRequestInput{GrantID: grant.ID, ItemID: "item-a", Operation: "use", IdempotencyKey: "restart-1"})
	if err != nil || restarted.ID != original.ID || restarted.Status != "pending" {
		t.Fatalf("restart request = %+v, err=%v; want %+v", restarted, err, original)
	}
}

func TestMachinePrincipalEnrollmentIsBoundAndRevocable(t *testing.T) {
	manager := newPasswordManager(nil)
	handlers := &passwordManagerHandlers{manager: manager}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/machine-principals", strings.NewReader(`{"machine_id":"runner-1","label":"CI runner"}`))
	setOwnerAuthContext(request, ownerAuthContext{WorkspaceID: "workspace-a", PrincipalID: "owner-a", Role: "owner"})
	response := httptest.NewRecorder()
	handlers.enrollMachinePrincipal(response, request)
	if response.Code != http.StatusCreated || bytes.Contains(response.Body.Bytes(), []byte("token_digest")) {
		t.Fatalf("machine enrollment = %d %s", response.Code, response.Body.String())
	}
	var enrolled struct {
		ID    string `json:"id"`
		Token string `json:"enrollment_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &enrolled); err != nil {
		t.Fatal(err)
	}
	if enrolled.ID == "" || enrolled.Token == "" {
		t.Fatalf("enrollment omitted stable id or one-time token: %+v", enrolled)
	}
	auth, err := manager.AuthenticateMachinePrincipal(context.Background(), "workspace-a", enrolled.Token)
	if err != nil || auth.PrincipalID != enrolled.ID || auth.Role != "machine" {
		t.Fatalf("machine authentication = %+v, %v", auth, err)
	}
	if _, err := manager.AuthenticateMachinePrincipal(context.Background(), "workspace-b", enrolled.Token); !errors.Is(err, errAccessDenied) {
		t.Fatalf("cross-workspace machine authentication = %v, want access denied", err)
	}

	revoke := httptest.NewRequest(http.MethodDelete, "/api/v1/machine-principals/"+enrolled.ID, nil)
	revoke = mux.SetURLVars(revoke, map[string]string{"id": enrolled.ID})
	setOwnerAuthContext(revoke, ownerAuthContext{WorkspaceID: "workspace-a", PrincipalID: "owner-a", Role: "owner"})
	revoked := httptest.NewRecorder()
	handlers.revokeMachinePrincipal(revoked, revoke)
	if revoked.Code != http.StatusOK {
		t.Fatalf("machine revoke = %d %s", revoked.Code, revoked.Body.String())
	}
	if _, err := manager.AuthenticateMachinePrincipal(context.Background(), "workspace-a", enrolled.Token); !errors.Is(err, errAccessDenied) {
		t.Fatalf("revoked machine authentication = %v, want access denied", err)
	}
}

func TestPasswordManagerPersistsEncryptedItemsInSQLite(t *testing.T) {
	t.Setenv("SECRETS_MANAGER_VAULT_KEY", base64.RawStdEncoding.EncodeToString([]byte("01234567890123456789012345678901")))
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: "file:" + filepath.Join(t.TempDir(), "vault.sqlite"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), vault.Schema()); err != nil {
		t.Fatal(err)
	}
	manager := newPasswordManager(db)
	if _, err := manager.ensureVault(context.Background(), "workspace-a", "personal"); err != nil {
		t.Fatal(err)
	}
	if err := manager.lock(context.Background(), "workspace-a", "personal", false); err != nil {
		t.Fatal(err)
	}
	item, err := manager.createItem(context.Background(), "workspace-a", "personal", "owner-a", "sql-retry", createItemInput{Type: "password", Name: "SQLite", Fields: map[string]string{"password": "  keep whitespace  "}})
	if err != nil {
		t.Fatal(err)
	}
	retried, err := manager.createItem(context.Background(), "workspace-a", "personal", "owner-a", "sql-retry", createItemInput{Type: "password", Name: "must-not-duplicate", Fields: map[string]string{"password": "different"}})
	if err != nil {
		t.Fatal(err)
	}
	if retried.ID != item.ID {
		t.Fatalf("idempotent retry created a second item: first=%s retry=%s", item.ID, retried.ID)
	}
	if _, err := manager.listItems(context.Background(), "workspace-a", "personal", "SQLite", false, 1, 10); err != nil {
		t.Fatal(err)
	}
	token, _, err := manager.issueAssurance(context.Background(), "workspace-a", "owner-a", "reveal:"+item.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	fields, err := manager.reveal(context.Background(), "workspace-a", "personal", item.ID, "owner-a", token, "", []string{"password"})
	if err != nil {
		t.Fatal(err)
	}
	if fields["password"] != "  keep whitespace  " {
		t.Fatalf("persisted field changed: %q", fields["password"])
	}
	source, err := manager.createSource(context.Background(), "workspace-a", "owner-a", sourceInput{Kind: "bitwarden_vault", Label: "Bitwarden", Endpoint: "https://vault.example", BootstrapRef: "owner-ref"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.listSources(context.Background(), "workspace-a"); err != nil {
		t.Fatalf("list persisted sources: %v", err)
	}
	if err := manager.bindSource(context.Background(), "workspace-a", "owner-a", "personal", item.ID, sourceBinding{SourceID: source.ID, ExternalRef: "bw-item"}); err != nil {
		t.Fatalf("bind persisted source: %v", err)
	}
}

func TestBrokerRejectsUnapprovedOriginsAndRedactsCredentialEchoes(t *testing.T) {
	if _, err := validateBrokerOrigin("http://127.0.0.1:8080", false); err == nil {
		t.Fatal("expected private origin to require explicit policy")
	}
	if _, ips, err := resolveBrokerOrigin("http://127.0.0.1:8080", true); err != nil || len(ips) != 1 || ips[0] != "127.0.0.1" {
		t.Fatalf("internal origin was not pinned: ips=%v err=%v", ips, err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", "synthetic-request")
		_, _ = w.Write([]byte(`{"received":"Bearer broker-secret"}`))
	}))
	defer server.Close()

	router := newAPIServer(nil, NewLogger("test")).routes()
	if response := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/unlock", nil); response.Code != http.StatusOK {
		t.Fatalf("unlock status = %d: %s", response.Code, response.Body.String())
	}
	create := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/items", map[string]any{
		"type": "api_credential", "name": "Broker fixture", "fields": map[string]string{"token": "broker-secret"},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", create.Code, create.Body.String())
	}
	var item VaultItem
	if err := json.Unmarshal(create.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	grant := vaultRequest(t, router, http.MethodPost, "/api/v1/grants", map[string]any{
		"vault_id": "vault-a", "item_id": item.ID, "principal_type": "agent", "principal_id": "owner-a", "operations": []string{"use"}, "target": server.URL,
	})
	if grant.Code != http.StatusCreated {
		t.Fatalf("grant status = %d: %s", grant.Code, grant.Body.String())
	}
	var grantBody grantRecord
	if err := json.Unmarshal(grant.Body.Bytes(), &grantBody); err != nil {
		t.Fatal(err)
	}
	session := vaultRequest(t, router, http.MethodPost, "/api/v1/broker/sessions", map[string]any{
		"grant_id": grantBody.ID, "item_id": item.ID, "target_origin": server.URL, "allow_internal": true,
	})
	if session.Code != http.StatusCreated {
		t.Fatalf("session status = %d: %s", session.Code, session.Body.String())
	}
	var sessionBody struct {
		ID    string `json:"session_id"`
		Token string `json:"session_token"`
	}
	if err := json.Unmarshal(session.Body.Bytes(), &sessionBody); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/broker/sessions/"+sessionBody.ID+"/operations", bytes.NewBufferString(`{"method":"GET","path":"/health"}`))
	request.Header.Set("Authorization", "Bearer test")
	request.Header.Set("X-Workspace-ID", "workspace-a")
	request.Header.Set("X-Actor-ID", "owner-a")
	request.Header.Set("X-Broker-Session", sessionBody.Token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("broker operation status = %d: %s", response.Code, response.Body.String())
	}
	if bytes.Contains(response.Body.Bytes(), []byte("broker-secret")) || !bytes.Contains(response.Body.Bytes(), []byte("[REDACTED]")) {
		t.Fatalf("broker response was not redacted: %s", response.Body.String())
	}
	locked := vaultRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/lock", nil)
	if locked.Code != http.StatusOK {
		t.Fatalf("lock status = %d: %s", locked.Code, locked.Body.String())
	}
	lockedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/broker/sessions/"+sessionBody.ID+"/operations", bytes.NewBufferString(`{"method":"GET","path":"/health"}`))
	lockedRequest.Header.Set("Authorization", "Bearer test")
	lockedRequest.Header.Set("X-Workspace-ID", "workspace-a")
	lockedRequest.Header.Set("X-Actor-ID", "owner-a")
	lockedRequest.Header.Set("X-Broker-Session", sessionBody.Token)
	lockedResponse := httptest.NewRecorder()
	router.ServeHTTP(lockedResponse, lockedRequest)
	if lockedResponse.Code != http.StatusLocked {
		t.Fatalf("locked broker operation status = %d, want %d: %s", lockedResponse.Code, http.StatusLocked, lockedResponse.Body.String())
	}
	revoke := vaultRequest(t, router, http.MethodPost, "/api/v1/broker/sessions/"+sessionBody.ID+"/revoke", nil)
	if revoke.Code != http.StatusOK {
		t.Fatalf("broker revoke status = %d: %s", revoke.Code, revoke.Body.String())
	}
	requestAfterRevoke := httptest.NewRequest(http.MethodPost, "/api/v1/broker/sessions/"+sessionBody.ID+"/operations", bytes.NewBufferString(`{"method":"GET","path":"/health"}`))
	requestAfterRevoke.Header.Set("Authorization", "Bearer test")
	requestAfterRevoke.Header.Set("X-Workspace-ID", "workspace-a")
	requestAfterRevoke.Header.Set("X-Actor-ID", "owner-a")
	requestAfterRevoke.Header.Set("X-Broker-Session", sessionBody.Token)
	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, requestAfterRevoke)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("revoked broker operation status = %d, want %d: %s", denied.Code, http.StatusForbidden, denied.Body.String())
	}
}

func TestBrokerBindsTargetAndSupportsOneUseSessions(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	manager := newPasswordManager(nil)
	ctx := context.Background()
	item, err := manager.createItem(ctx, "local", "personal", "owner-a", "broker-one-use", createItemInput{
		Type: "api_credential", Name: "One use", Fields: map[string]string{"token": "one-use-secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	grant, err := manager.createGrant(ctx, "local", "owner-a", createGrantInput{
		VaultID: "personal", ItemID: item.ID, PrincipalType: "agent", PrincipalID: "agent-a", Members: []string{"agent-a"},
		Operations: []string{"use"}, Target: target.URL, ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.createBrokerSession(ctx, "local", "agent-a", brokerSessionInput{GrantID: grant.ID, ItemID: item.ID, TargetOrigin: "http://127.0.0.1:9", AllowInternal: true}); !errors.Is(err, errAccessDenied) {
		t.Fatalf("target mismatch error = %v, want access denied", err)
	}
	session, token, err := manager.createBrokerSession(ctx, "local", "agent-a", brokerSessionInput{GrantID: grant.ID, ItemID: item.ID, TargetOrigin: target.URL, AllowInternal: true, OneUse: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.brokerOperation(ctx, "local", "agent-a", session.ID, token, brokerOperationInput{Method: http.MethodGet, Path: "/health"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.brokerOperation(ctx, "local", "agent-a", session.ID, token, brokerOperationInput{Method: http.MethodGet, Path: "/health"}); !errors.Is(err, errAccessDenied) {
		t.Fatalf("one-use replay error = %v, want access denied", err)
	}
	if _, err := manager.brokerOperation(ctx, "local", "agent-a", session.ID, token, brokerOperationInput{Method: http.MethodPost, Path: "/health"}); err == nil {
		t.Fatal("write method was accepted by the read-only broker")
	}
	if _, err := manager.brokerOperation(ctx, "local", "agent-a", session.ID, token, brokerOperationInput{Method: http.MethodGet, Path: "/health", Headers: map[string]string{"X-Forwarded-For": "127.0.0.1"}}); err == nil {
		t.Fatal("unapproved forwarding header was accepted")
	}
}

func TestBrokerRejectsCrossOriginRedirectAndReplayOutsidePrincipal(t *testing.T) {
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer other.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusFound)
	}))
	defer redirect.Close()
	manager := newPasswordManager(nil)
	ctx := context.Background()
	item, err := manager.createItem(ctx, "local", "personal", "owner-a", "broker-redirect", createItemInput{
		Type: "api_credential", Name: "Redirect", Fields: map[string]string{"token": "redirect-secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	grant, err := manager.createGrant(ctx, "local", "owner-a", createGrantInput{
		VaultID: "personal", ItemID: item.ID, PrincipalType: "agent", PrincipalID: "agent-a", Members: []string{"agent-a"},
		Operations: []string{"use"}, Target: redirect.URL, ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	session, token, err := manager.createBrokerSession(ctx, "local", "agent-a", brokerSessionInput{GrantID: grant.ID, ItemID: item.ID, TargetOrigin: redirect.URL, AllowInternal: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.brokerOperation(ctx, "local", "agent-b", session.ID, token, brokerOperationInput{Method: http.MethodGet, Path: "/redirect"}); !errors.Is(err, errAccessDenied) {
		t.Fatalf("cross-principal replay error = %v, want access denied", err)
	}
	if _, err := manager.brokerOperation(ctx, "local", "agent-a", session.ID, token, brokerOperationInput{Method: http.MethodGet, Path: "/redirect"}); err == nil {
		t.Fatal("cross-origin redirect was accepted")
	}
	manager.mu.Lock()
	expired := manager.brokers[session.ID]
	expired.Expires = time.Now().Add(-time.Second)
	manager.brokers[session.ID] = expired
	manager.mu.Unlock()
	if _, err := manager.brokerOperation(ctx, "local", "agent-a", session.ID, token, brokerOperationInput{Method: http.MethodGet, Path: "/health"}); !errors.Is(err, errAccessDenied) {
		t.Fatalf("expired replay error = %v, want access denied", err)
	}
}

func TestPasswordManagerExportHandlesAndNativeImport(t *testing.T) {
	manager := newPasswordManager(nil)
	ctx := context.Background()
	if err := manager.lock(ctx, "workspace-a", "vault-a", false); err != nil {
		t.Fatal(err)
	}
	item, err := manager.createItem(ctx, "workspace-a", "vault-a", "owner-a", "", createItemInput{Type: "password", Name: "Portable", Fields: map[string]string{"password": "keep exact"}})
	if err != nil {
		t.Fatal(err)
	}
	handle, token, err := manager.exportBundle(ctx, "workspace-a", "owner-a", "vault-a", exportInput{Format: "native", ItemIDs: []string{item.ID}}, "")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := manager.redeemExport(ctx, "workspace-a", handle.ID, token)
	if err != nil || len(bundle.Items) != 1 || bundle.Items[0].Ciphertext == "" {
		t.Fatalf("native redemption failed: %v %+v", err, bundle)
	}
	if _, err := manager.redeemExport(ctx, "workspace-a", handle.ID, token); !errors.Is(err, errAccessDenied) {
		t.Fatalf("expected one-time redemption denial, got %v", err)
	}

	if _, _, err := manager.exportBundle(ctx, "workspace-a", "owner-a", "vault-a", exportInput{Format: "plaintext"}, ""); !errors.Is(err, errAssuranceRequired) {
		t.Fatalf("plaintext export without assurance = %v", err)
	}
	assurance, _, err := manager.issueAssurance(ctx, "workspace-a", "owner-a", "export:vault-a", "")
	if err != nil {
		t.Fatal(err)
	}
	plainHandle, plainToken, err := manager.exportBundle(ctx, "workspace-a", "owner-a", "vault-a", exportInput{Format: "plaintext"}, assurance)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := manager.redeemExport(ctx, "workspace-a", plainHandle.ID, plainToken)
	if err != nil || plain.Items[0].Fields["password"] != "keep exact" {
		t.Fatalf("plaintext export failed: %v %+v", err, plain)
	}

	if err := manager.lock(ctx, "workspace-a", "vault-b", false); err != nil {
		t.Fatal(err)
	}
	result, err := manager.importNative(ctx, "workspace-a", "vault-b", "owner-a", bundle, "rename")
	if err != nil || result["created"] != 1 {
		t.Fatalf("native import failed: %v %+v", err, result)
	}
}

func TestPasswordManagerSourcesRequireExplicitAuthorityAndBinding(t *testing.T) {
	manager := newPasswordManager(nil)
	ctx := context.Background()
	if _, err := manager.createSource(ctx, "workspace-a", "owner-a", sourceInput{Kind: "onepassword", Label: "1Password"}); err == nil {
		t.Fatal("expected external source without bootstrap reference to fail")
	}
	if _, err := manager.createSource(ctx, "workspace-a", "owner-a", sourceInput{Kind: "onepassword", Label: "1Password", Endpoint: "http://127.0.0.1:8080", BootstrapRef: "secret-ref"}); err == nil {
		t.Fatal("expected non-HTTPS source endpoint to fail")
	}
	source, err := manager.createSource(ctx, "workspace-a", "owner-a", sourceInput{Kind: "onepassword", Label: "1Password", Endpoint: "https://vault.example", BootstrapRef: "secret-ref"})
	if err != nil {
		t.Fatal(err)
	}
	if source.Status != "unverified" || !source.Capabilities.Read || source.Capabilities.Write {
		t.Fatalf("unexpected external source capabilities: %+v", source)
	}
	public := sourcePublic(source)
	if _, exposed := public["bootstrap_ref"]; exposed {
		t.Fatal("source serialization disclosed the bootstrap reference")
	}
	health, err := manager.sourceHealth(ctx, "workspace-a", source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != "unavailable" || !strings.Contains(health.LastError, "no fallback source") {
		t.Fatalf("external source health made an unsupported claim: %+v", health)
	}
	if err := manager.lock(ctx, "workspace-a", "vault-a", false); err != nil {
		t.Fatal(err)
	}
	item, err := manager.createItem(ctx, "workspace-a", "vault-a", "owner-a", "", createItemInput{Type: "login", Name: "External", Fields: map[string]string{"username": "alice"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.bindSource(ctx, "workspace-a", "owner-a", "vault-a", item.ID, sourceBinding{SourceID: source.ID, ExternalRef: "item-123"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.getSource(ctx, "other-workspace", source.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-workspace source lookup = %v", err)
	}
}

func productionWorkspaceRequest(t *testing.T, router http.Handler, method, path, workspace, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &payload)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Workspace-ID", workspace)
	req.Header.Set("X-Actor-ID", "request-header-must-not-override-token")
	if method == http.MethodPut {
		req.Header.Set("If-Match", "1")
	}
	// Simulate the authenticated public edge. The handler must still load the
	// role and active status from the workspace tables instead of trusting the
	// request headers.
	setOwnerAuthContext(req, ownerAuthContext{WorkspaceID: workspace, PrincipalID: strings.TrimSuffix(token, "-token"), AuthenticatedAt: time.Now().UTC()})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func seedWorkspacePrincipal(t *testing.T, db *database.RoutedDB, workspace, principal, role, token, status string) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `INSERT INTO pm_workspace_enrollments(workspace_id,status,enrolled_by) VALUES($1,'enrolled',$2) ON CONFLICT (workspace_id) DO NOTHING`, workspace, principal); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO pm_workspace_members(id,workspace_id,principal_id,role,status) VALUES($1,$2,$3,$4,$5)`, uuid.NewString(), workspace, principal, role, status); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO pm_owner_tokens(token_digest,workspace_id,principal_id,role,status) VALUES($1,$2,$3,$4,$5)`, ownerTokenDigest(token), workspace, principal, role, status); err != nil {
		t.Fatal(err)
	}
}

func openWorkspaceSecurityDB(t *testing.T) *database.RoutedDB {
	t.Helper()
	t.Setenv("SECRETS_MANAGER_VAULT_KEY", base64.RawStdEncoding.EncodeToString([]byte("01234567890123456789012345678901")))
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: "file:" + filepath.Join(t.TempDir(), "workspace-security.sqlite"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), vault.Schema()); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestPasswordManagerProductionWorkspaceIsolationAndRoleCapabilities(t *testing.T) {
	db := openWorkspaceSecurityDB(t)
	t.Setenv("SECRETS_MANAGER_WORKSPACE_ID", "workspace-a")
	t.Setenv("SECRETS_MANAGER_ALLOWED_WORKSPACES", "workspace-b")
	seedWorkspacePrincipal(t, db, "workspace-a", "owner-a", "owner", "owner-a-token", "active")
	seedWorkspacePrincipal(t, db, "workspace-b", "owner-b", "owner", "owner-b-token", "active")
	seedWorkspacePrincipal(t, db, "workspace-a", "viewer-a", "viewer", "viewer-a-token", "active")
	seedWorkspacePrincipal(t, db, "workspace-a", "removed-a", "member", "removed-a-token", "revoked")
	router := newAPIServer(db, NewLogger("test")).routes()

	if response := productionWorkspaceRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/unlock", "workspace-a", "owner-a-token", nil); response.Code != http.StatusOK {
		t.Fatalf("owner unlock status = %d: %s", response.Code, response.Body.String())
	}
	created := productionWorkspaceRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/items", "workspace-a", "owner-a-token", map[string]any{
		"type": "login", "name": "workspace-a item", "fields": map[string]string{"password": "workspace-a secret"},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("owner create status = %d: %s", created.Code, created.Body.String())
	}
	var item VaultItem
	if err := json.Unmarshal(created.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}

	list := productionWorkspaceRequest(t, router, http.MethodGet, "/api/v1/vaults/vault-a/items", "workspace-b", "owner-b-token", nil)
	if list.Code != http.StatusOK || strings.Contains(list.Body.String(), item.ID) || strings.Contains(list.Body.String(), "workspace-a item") {
		t.Fatalf("cross-workspace list disclosed item: %d %s", list.Code, list.Body.String())
	}
	for _, route := range []string{
		"/api/v1/vaults/vault-a/items/" + item.ID,
		"/api/v1/vaults/vault-a/items/" + item.ID + "/history",
	} {
		response := productionWorkspaceRequest(t, router, http.MethodGet, route, "workspace-b", "owner-b-token", nil)
		if response.Code != http.StatusNotFound {
			t.Fatalf("cross-workspace detail/activity %s status = %d, want %d: %s", route, response.Code, http.StatusNotFound, response.Body.String())
		}
	}
	update := productionWorkspaceRequest(t, router, http.MethodPut, "/api/v1/vaults/vault-a/items/"+item.ID, "workspace-b", "owner-b-token", map[string]any{
		"type": "login", "name": "cross-workspace edit", "fields": map[string]string{"password": "must not write"},
	})
	if update.Code != http.StatusForbidden {
		t.Fatalf("cross-workspace edit status = %d, want %d: %s", update.Code, http.StatusForbidden, update.Body.String())
	}
	assurance := productionWorkspaceRequest(t, router, http.MethodPost, "/api/v1/assurance", "workspace-b", "owner-b-token", map[string]string{"operation": "reveal:" + item.ID})
	if assurance.Code != http.StatusCreated {
		t.Fatalf("cross-workspace assurance setup status = %d: %s", assurance.Code, assurance.Body.String())
	}
	var assuranceBody struct {
		Token string `json:"assurance_token"`
	}
	if err := json.Unmarshal(assurance.Body.Bytes(), &assuranceBody); err != nil {
		t.Fatal(err)
	}
	reveal := productionWorkspaceRequest(t, router, http.MethodPost, "/api/v1/vaults/vault-a/items/"+item.ID+"/reveal", "workspace-b", "owner-b-token", map[string]any{"assurance_token": assuranceBody.Token, "fields": []string{"password"}})
	if reveal.Code != http.StatusForbidden {
		t.Fatalf("cross-workspace reveal status = %d, want %d: %s", reveal.Code, http.StatusForbidden, reveal.Body.String())
	}

	viewerList := productionWorkspaceRequest(t, router, http.MethodGet, "/api/v1/vaults/vault-a/items", "workspace-a", "viewer-a-token", nil)
	if viewerList.Code != http.StatusOK {
		t.Fatalf("viewer metadata status = %d: %s", viewerList.Code, viewerList.Body.String())
	}
	viewerUpdate := productionWorkspaceRequest(t, router, http.MethodPut, "/api/v1/vaults/vault-a/items/"+item.ID, "workspace-a", "viewer-a-token", map[string]any{"type": "login", "name": "viewer edit", "fields": map[string]string{"password": "must not write"}})
	if viewerUpdate.Code != http.StatusForbidden {
		t.Fatalf("viewer edit status = %d, want %d: %s", viewerUpdate.Code, http.StatusForbidden, viewerUpdate.Body.String())
	}
	removed := productionWorkspaceRequest(t, router, http.MethodGet, "/api/v1/vaults/vault-a/items", "workspace-a", "removed-a-token", nil)
	if removed.Code != http.StatusForbidden {
		t.Fatalf("revoked member status = %d, want %d: %s", removed.Code, http.StatusForbidden, removed.Body.String())
	}
}

func TestPasswordManagerEnrollmentPersistsSingleOwnerInProductionDB(t *testing.T) {
	db := openWorkspaceSecurityDB(t)
	t.Setenv("SECRETS_MANAGER_WORKSPACE_ID", "workspace-enroll")
	t.Setenv("SECRETS_MANAGER_OWNER_BOOTSTRAP_TOKEN", "bootstrap-once")
	router := newAPIServer(db, NewLogger("test")).routes()
	first := productionWorkspaceRequest(t, router, http.MethodPost, "/api/v1/enrollment/complete", "workspace-enroll", "unused", map[string]string{"bootstrap_token": "bootstrap-once"})
	if first.Code != http.StatusCreated || strings.Contains(first.Body.String(), "bootstrap-once") {
		t.Fatalf("first enrollment status = %d: %s", first.Code, first.Body.String())
	}
	second := productionWorkspaceRequest(t, router, http.MethodPost, "/api/v1/enrollment/complete", "workspace-enroll", "unused", map[string]string{"bootstrap_token": "bootstrap-once"})
	if second.Code != http.StatusConflict {
		t.Fatalf("second enrollment status = %d, want %d: %s", second.Code, http.StatusConflict, second.Body.String())
	}
	var owners int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM pm_workspace_members WHERE workspace_id=$1 AND role='owner'`, "workspace-enroll").Scan(&owners); err != nil {
		t.Fatal(err)
	}
	if owners != 1 {
		t.Fatalf("owner count = %d, want 1", owners)
	}
}

func TestPasswordManagerMemberRemovalRevokesProtectedRequests(t *testing.T) {
	db := openWorkspaceSecurityDB(t)
	t.Setenv("SECRETS_MANAGER_WORKSPACE_ID", "workspace-a")
	seedWorkspacePrincipal(t, db, "workspace-a", "admin-a", "admin", "admin-a-token", "active")
	seedWorkspacePrincipal(t, db, "workspace-a", "member-a", "member", "member-a-token", "active")
	router := newAPIServer(db, NewLogger("test")).routes()
	removed := productionWorkspaceRequest(t, router, http.MethodDelete, "/api/v1/members/member-a", "workspace-a", "admin-a-token", nil)
	if removed.Code != http.StatusOK {
		t.Fatalf("member removal status = %d: %s", removed.Code, removed.Body.String())
	}
	protected := productionWorkspaceRequest(t, router, http.MethodGet, "/api/v1/audit", "workspace-a", "member-a-token", nil)
	if protected.Code != http.StatusForbidden {
		t.Fatalf("removed member protected status = %d, want %d: %s", protected.Code, http.StatusForbidden, protected.Body.String())
	}
}

func TestAuditEventsAreIdempotentAndIntegrityChecked(t *testing.T) {
	manager := newPasswordManager(nil)
	event := auditEvent{WorkspaceID: "workspace-a", ActorID: "actor-a", Action: "credential_use.browser.fill", EventKey: "run-1:fill", Outcome: "success", Detail: "metadata only", CreatedAt: time.Unix(100, 0).UTC()}
	manager.recordAudit(context.Background(), event)
	manager.recordAudit(context.Background(), event)
	events, err := manager.listAudit(context.Background(), "workspace-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].IntegrityStatus != "verified" {
		t.Fatalf("events = %#v, want one verified event", events)
	}
	manager.mu.Lock()
	manager.events[0].Detail = "changed after write"
	manager.mu.Unlock()
	events, err = manager.listAudit(context.Background(), "workspace-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].IntegrityStatus != "tampered" {
		t.Fatalf("events = %#v, want one tampered event", events)
	}
}

func TestAuditExportDeclaresMetadataOnlyAndIntegrity(t *testing.T) {
	db := openWorkspaceSecurityDB(t)
	t.Setenv("SECRETS_MANAGER_WORKSPACE_ID", "workspace-a")
	t.Setenv("SECRETS_MANAGER_ALLOWED_WORKSPACES", "workspace-a")
	seedWorkspacePrincipal(t, db, "workspace-a", "owner-a", "owner", "owner-a-token", "active")
	server := newAPIServer(db, NewLogger("test"))
	server.handlers.passwordManager.manager.recordAudit(context.Background(), auditEvent{
		WorkspaceID: "workspace-a", ActorID: "owner-a", Action: "credential_use.ssh.sign", EventKey: "sign-1", Outcome: "failure", Detail: "destination rejected",
	})
	response := productionWorkspaceRequest(t, server.routes(), http.MethodGet, "/api/v1/audit/export", "workspace-a", "owner-a-token", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("audit export status = %d: %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["payload_policy"] != "metadata_only" || body["integrity"] != "verified" {
		t.Fatalf("audit export metadata = %#v", body)
	}
	if strings.Contains(response.Body.String(), "private_key") || strings.Contains(response.Body.String(), "password") {
		t.Fatalf("audit export contains credential field name: %s", response.Body.String())
	}
}
