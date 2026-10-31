package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRecoveryActivationRejectsAnOldCapabilityAfterRestoredRecordSurvives(t *testing.T) {
	manager := newPasswordManager(nil)
	token := "old-use-token"
	digest := sha256.Sum256([]byte(token))
	old := brokerSession{
		ID: "restored-session", TokenDigest: hex.EncodeToString(digest[:]), Workspace: "workspace-a", Actor: "agent-a",
		GrantID: "restored-grant", ItemID: "item-a", Origin: "https://example.test", Status: "active",
		RecoveryEpoch: 1, Expires: time.Now().UTC().Add(time.Hour),
	}
	manager.brokers[old.ID] = old

	newEpoch, err := manager.activateRecovery(context.Background(), "workspace-a", "owner-a", 1)
	if err != nil {
		t.Fatal(err)
	}
	if newEpoch != 2 {
		t.Fatalf("new recovery epoch = %d, want 2", newEpoch)
	}

	// Simulate a restored metadata row that was not rewritten by activation.
	// The epoch check remains authoritative even if its status was copied as
	// active from the old database.
	old.Status = "active"
	manager.brokers[old.ID] = old
	if _, err := manager.brokerSessionFor(context.Background(), "workspace-a", "agent-a", token); err == nil {
		t.Fatal("capability from the previous recovery epoch was accepted")
	}
}

func TestRecoveryActivationQuarantinesRestoredGrants(t *testing.T) {
	manager := newPasswordManager(nil)
	manager.grants["grant-a"] = grantRecord{ID: "grant-a", WorkspaceID: "workspace-a", Status: "active"}
	if _, err := manager.activateRecovery(context.Background(), "workspace-a", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if got := manager.grants["grant-a"].Status; got != "recovery_review" {
		t.Fatalf("restored grant status = %q, want recovery_review", got)
	}
}

func TestRecoveryActivationAllowsNewAuthorizedUseAfterDenyingOldSession(t *testing.T) {
	manager, item, oldGrant := credentialUseFixture(t)
	executor := &recordingBrowserExecutor{}
	manager.SetTrustedBrowserExecutor(executor)
	oldSession, err := manager.createBrowserSession(context.Background(), "workspace-a", "agent-a", browserUseInput{
		GrantID: oldGrant.ID, ItemID: item.ID, Origin: "https://login.example.test", Account: item.Username,
		DocumentID: "old-document", Allowed: []string{"login"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.activateRecovery(context.Background(), "workspace-a", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.browserAction(context.Background(), "workspace-a", "agent-a", oldSession.ID, browserActionInput{
		Action: "fill", Origin: oldSession.Origin, DocumentID: oldSession.DocumentID,
	}); err == nil {
		t.Fatal("old browser capability was accepted after recovery activation")
	}

	newGrant, err := manager.createGrant(context.Background(), "workspace-a", "owner-a", createGrantInput{
		VaultID: oldGrant.VaultID, ItemID: item.ID, PrincipalType: "agent", PrincipalID: "agent-a",
		SelectorMode: "current_snapshot", Members: []string{"agent-a"}, Operations: []string{"inject"},
		Target: "https://login.example.test", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	newSession, err := manager.createBrowserSession(context.Background(), "workspace-a", "agent-a", browserUseInput{
		GrantID: newGrant.ID, ItemID: item.ID, Origin: "https://login.example.test", Account: item.Username,
		DocumentID: "new-document", Allowed: []string{"login"},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := manager.browserAction(context.Background(), "workspace-a", "agent-a", newSession.ID, browserActionInput{
		Action: "fill", Origin: newSession.Origin, DocumentID: newSession.DocumentID,
	})
	if err != nil || result.Status != "active" || result.OperationClass != "trusted_fill" || len(executor.filled) == 0 {
		t.Fatalf("new authorized browser use = %+v, err=%v", result, err)
	}
}

func TestRecoveryActivationDurablyInvalidatesRestoredCapabilities(t *testing.T) {
	db := openWorkspaceSecurityDB(t)
	ctx := context.Background()
	expires := time.Now().UTC().Add(time.Hour)

	if _, err := db.ExecContext(ctx, `INSERT INTO pm_broker_sessions
		(id,token_digest,workspace_id,actor_id,run_id,grant_id,item_id,target_origin,resolved_ips,expires_at,status,one_use,use_count,recovery_epoch)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'active',FALSE,0,1)`,
		"session-fixture", "digest-fixture", "workspace-fixture", "agent-fixture", "run-fixture", "grant-fixture", "item-fixture", "https://fixture.example", "", expires); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pm_grants
		(id,workspace_id,vault_id,principal_type,principal_id,status,created_by)
		VALUES($1,$2,$3,$4,$5,'active',$6)`,
		"grant-fixture", "workspace-fixture", "vault-fixture", "agent", "agent-fixture", "owner-fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pm_assurance_tokens
		(digest,workspace_id,actor_id,operation,request_digest,expires_at)
		VALUES($1,$2,$3,$4,$5,$6)`,
		"assurance-digest-fixture", "workspace-fixture", "owner-fixture", "recovery.activate", "request-digest-fixture", expires); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pm_export_handles
		(id,workspace_id,vault_id,format,capability_digest,encrypted_payload,expires_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)`,
		"export-fixture", "workspace-fixture", "vault-fixture", "fixture", "capability-digest-fixture", "synthetic-payload", expires); err != nil {
		t.Fatal(err)
	}

	manager := newPasswordManager(db)
	epoch, err := manager.activateRecovery(ctx, "workspace-fixture", "owner-fixture", 1)
	if err != nil {
		t.Fatal(err)
	}
	if epoch != 2 {
		t.Fatalf("recovery epoch = %d, want 2", epoch)
	}
	assertFixtureState := func(wantEpoch int64) {
		t.Helper()
		var gotEpoch int64
		var sessionStatus, grantStatus string
		var assuranceConsumed, exportUsed bool
		if err := db.QueryRowContext(ctx, `SELECT epoch FROM pm_recovery_state WHERE workspace_id=$1`, "workspace-fixture").Scan(&gotEpoch); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, `SELECT status FROM pm_broker_sessions WHERE id=$1`, "session-fixture").Scan(&sessionStatus); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, `SELECT status FROM pm_grants WHERE id=$1`, "grant-fixture").Scan(&grantStatus); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, `SELECT consumed_at IS NOT NULL FROM pm_assurance_tokens WHERE digest=$1`, "assurance-digest-fixture").Scan(&assuranceConsumed); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, `SELECT used_at IS NOT NULL FROM pm_export_handles WHERE id=$1`, "export-fixture").Scan(&exportUsed); err != nil {
			t.Fatal(err)
		}
		if gotEpoch != wantEpoch || sessionStatus != "recovery_invalidated" || grantStatus != "recovery_review" || !assuranceConsumed || !exportUsed {
			t.Fatalf("durable recovery state: epoch=%d session=%q grant=%q assuranceConsumed=%t exportUsed=%t", gotEpoch, sessionStatus, grantStatus, assuranceConsumed, exportUsed)
		}
	}
	assertFixtureState(2)

	if _, err := manager.activateRecovery(ctx, "workspace-fixture", "owner-fixture", 1); !errors.Is(err, errRecoveryEpochConflict) {
		t.Fatalf("stale expected epoch error = %v, want %v", err, errRecoveryEpochConflict)
	}
	assertFixtureState(2)
}

func TestRecoveryActivationEpochDeniesInProcessBrowserAndSSHCapabilities(t *testing.T) {
	db := openWorkspaceSecurityDB(t)
	manager := newPasswordManager(db)
	ctx := context.Background()
	workspace, vault, owner, agent := "workspace-fixture", "vault-fixture", "owner-fixture", "agent-fixture"
	if err := manager.lock(ctx, workspace, vault, false); err != nil {
		t.Fatal(err)
	}

	login, err := manager.createItem(ctx, workspace, vault, owner, "", createItemInput{
		Type: "login", Name: "Synthetic browser fixture", Username: "fixture@example.test",
		Fields: map[string]string{"password": "synthetic-browser-value"},
	})
	if err != nil {
		t.Fatal(err)
	}
	loginGrant, err := manager.createGrant(ctx, workspace, owner, createGrantInput{
		VaultID: vault, ItemID: login.ID, PrincipalType: "agent", PrincipalID: agent,
		SelectorMode: "current_snapshot", Members: []string{agent}, Operations: []string{"inject"},
		Target: "https://login.example.test", ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	executor := &recordingBrowserExecutor{}
	manager.SetTrustedBrowserExecutor(executor)
	browser, err := manager.createBrowserSession(ctx, workspace, agent, browserUseInput{
		GrantID: loginGrant.ID, ItemID: login.ID, Origin: "https://login.example.test", Account: login.Username,
		DocumentID: "document-fixture", Allowed: []string{"login"}, DurationSeconds: 300,
	})
	if err != nil {
		t.Fatal(err)
	}

	sshItem, err := manager.createItem(ctx, workspace, vault, owner, "", createItemInput{
		Type: "ssh", Name: "Synthetic SSH fixture",
		Fields: map[string]string{"private_key": "synthetic-private-key-marker", "principal": "fixture-user"},
	})
	if err != nil {
		t.Fatal(err)
	}
	sshGrant, err := manager.createGrant(ctx, workspace, owner, createGrantInput{
		VaultID: vault, ItemID: sshItem.ID, PrincipalType: "agent", PrincipalID: agent,
		SelectorMode: "current_snapshot", Members: []string{agent}, Operations: []string{"sign"},
		Target: "fixture.example.test:22", ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	sshSession, err := manager.createSSHSigner(ctx, workspace, agent, sshSignerInput{
		GrantID: sshGrant.ID, ItemID: sshItem.ID, Destination: "fixture.example.test:22", Principal: "fixture-user",
		DurationSeconds: 300,
	})
	if err != nil {
		t.Fatal(err)
	}
	if browser.RecoveryEpoch != 1 || sshSession.RecoveryEpoch != 1 {
		t.Fatalf("fixture epochs browser=%d ssh=%d, want 1", browser.RecoveryEpoch, sshSession.RecoveryEpoch)
	}

	epoch, err := manager.activateRecovery(ctx, workspace, owner, 1)
	if err != nil || epoch != 2 {
		t.Fatalf("activate recovery epoch=%d err=%v, want epoch 2", epoch, err)
	}
	if _, err := manager.browserAction(ctx, workspace, agent, browser.ID, browserActionInput{
		Action: "fill", Origin: browser.Origin, DocumentID: browser.DocumentID,
	}); !errors.Is(err, errAccessDenied) {
		t.Fatalf("old browser capability error=%v, want access denied", err)
	}
	if len(executor.filled) != 0 || executor.closed != 1 {
		t.Fatalf("browser executor after activation: filled=%v closed=%d, want no fill and one close", executor.filled, executor.closed)
	}
	manager.mu.RLock()
	browserStatus := manager.browsers[browser.ID].Status
	manager.mu.RUnlock()
	if browserStatus != "recovery_invalidated" {
		t.Fatalf("old browser session status=%q, want recovery_invalidated", browserStatus)
	}

	if _, err := manager.signSSH(ctx, workspace, agent, sshSession.ID, sshSignInput{Challenge: "Y2hhbGxlbmdl"}); !errors.Is(err, errAccessDenied) {
		t.Fatalf("old SSH capability error=%v, want access denied", err)
	}
}

func TestRecoveryActivationHTTPRequiresAssuranceAndKeepsResponseMetadataOnly(t *testing.T) {
	db := openWorkspaceSecurityDB(t)
	t.Setenv("SECRETS_MANAGER_WORKSPACE_ID", "workspace-http-fixture")
	t.Setenv("SECRETS_MANAGER_ALLOWED_WORKSPACES", "")
	workspace := "workspace-http-fixture"
	ownerToken, viewerToken := "owner-http-fixture-token", "viewer-http-fixture-token"
	ctx := context.Background()
	if err := initializeDesktopSchema(ctx, db); err != nil {
		t.Fatalf("initialize production SQLite schema: %v", err)
	}
	seedWorkspacePrincipal(t, db, workspace, "owner-http-fixture", "owner", ownerToken, "active")
	seedWorkspacePrincipal(t, db, workspace, "viewer-http-fixture", "viewer", viewerToken, "active")
	if _, err := db.ExecContext(ctx, `INSERT INTO pm_recovery_state(workspace_id,epoch) VALUES($1,1)`, workspace); err != nil {
		t.Fatal(err)
	}
	marker := "synthetic-capability-marker-phase12"
	if _, err := db.ExecContext(ctx, `INSERT INTO pm_broker_sessions
		(id,token_digest,workspace_id,actor_id,run_id,grant_id,item_id,target_origin,resolved_ips,expires_at,status,one_use,use_count,recovery_epoch)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'active',FALSE,0,1)`,
		"session-http-fixture", marker, workspace, "agent-fixture", "run-fixture", "grant-http-fixture", "item-http-fixture", "https://fixture.example", "", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pm_grants
		(id,workspace_id,vault_id,principal_type,principal_id,status,created_by)
		VALUES($1,$2,$3,$4,$5,'active',$6)`,
		"grant-http-fixture", workspace, "vault-http-fixture", "agent", "agent-fixture", "owner-http-fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pm_export_handles
		(id,workspace_id,vault_id,format,capability_digest,encrypted_payload,expires_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)`,
		"export-http-fixture", workspace, "vault-http-fixture", "fixture", "synthetic-digest", marker, time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	router := newAPIServer(db, NewLogger("recovery-http-test")).routes()
	activate := func(token, assurance string) *httptest.ResponseRecorder {
		t.Helper()
		return productionWorkspaceRequest(t, router, http.MethodPost, "/api/v1/recovery/activate", workspace, token, map[string]any{
			"expected_epoch": 1, "assurance_token": assurance,
		})
	}
	assertNoActivation := func() {
		t.Helper()
		var epoch int
		var sessionStatus, grantStatus string
		if err := db.QueryRowContext(ctx, `SELECT epoch FROM pm_recovery_state WHERE workspace_id=$1`, workspace).Scan(&epoch); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, `SELECT status FROM pm_broker_sessions WHERE id=$1`, "session-http-fixture").Scan(&sessionStatus); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRowContext(ctx, `SELECT status FROM pm_grants WHERE id=$1`, "grant-http-fixture").Scan(&grantStatus); err != nil {
			t.Fatal(err)
		}
		if epoch != 1 || sessionStatus != "active" || grantStatus != "active" {
			t.Fatalf("denied activation changed state: epoch=%d session=%q grant=%q", epoch, sessionStatus, grantStatus)
		}
	}
	for _, tc := range []struct{ name, token string }{
		{name: "unauthenticated"},
		{name: "viewer", token: viewerToken},
		{name: "owner without assurance", token: ownerToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := activate(tc.token, "")
			if response.Code == http.StatusOK {
				t.Fatalf("activation unexpectedly succeeded: %s", response.Body.String())
			}
			assertNoActivation()
		})
	}

	issueAssurance := func(requestDigest string) string {
		t.Helper()
		response := productionWorkspaceRequest(t, router, http.MethodPost, "/api/v1/assurance", workspace, ownerToken, map[string]string{
			"operation": "recovery:activate", "request_digest": requestDigest,
		})
		if response.Code != http.StatusCreated {
			t.Fatalf("issue assurance status=%d body=%s", response.Code, response.Body.String())
		}
		var body struct {
			Token string `json:"assurance_token"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Token == "" {
			t.Fatalf("assurance response missing token: body=%s err=%v", response.Body.String(), err)
		}
		return body.Token
	}
	wrongScope := issueAssurance("2")
	if response := activate(ownerToken, wrongScope); response.Code == http.StatusOK {
		t.Fatalf("wrong-scope assurance activated recovery: %s", response.Body.String())
	}
	assertNoActivation()
	assurance := issueAssurance("1")
	response := activate(ownerToken, assurance)
	if response.Code != http.StatusOK {
		t.Fatalf("authorized activation status=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), marker) || strings.Contains(response.Body.String(), assurance) {
		t.Fatalf("activation response disclosed a capability marker or assurance token: %s", response.Body.String())
	}
	var auditDetail string
	if err := db.QueryRowContext(ctx, `SELECT detail FROM pm_audit_events WHERE workspace_id=$1 AND action='recovery.activate'`, workspace).Scan(&auditDetail); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(auditDetail, marker) || strings.Contains(auditDetail, assurance) {
		t.Fatalf("recovery audit disclosed a capability marker or assurance token: %s", auditDetail)
	}
	var epoch int
	if err := db.QueryRowContext(ctx, `SELECT epoch FROM pm_recovery_state WHERE workspace_id=$1`, workspace).Scan(&epoch); err != nil || epoch != 2 {
		t.Fatalf("authorized recovery epoch=%d err=%v, want 2", epoch, err)
	}
}
