package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"strings"
	"testing"
	"time"
)

type recordingBrowserExecutor struct {
	filled  map[string]string
	cleared []string
	closed  int
}

func (e *recordingBrowserExecutor) Fill(_ context.Context, _ browserSession, fields map[string]string) ([]string, error) {
	e.filled = make(map[string]string, len(fields))
	for key, value := range fields {
		e.filled[key] = value
	}
	return []string{"username", "password", "totp"}, nil
}

func (e *recordingBrowserExecutor) Clear(_ context.Context, _ browserSession, fields []string) error {
	e.cleared = append([]string(nil), fields...)
	return nil
}

func (e *recordingBrowserExecutor) Close(context.Context, browserSession) error {
	e.closed++
	return nil
}

func credentialUseFixture(t *testing.T) (*PasswordManager, VaultItem, grantRecord) {
	t.Helper()
	ctx := context.Background()
	manager := newPasswordManager(nil)
	if err := manager.lock(ctx, "workspace-a", "vault-a", false); err != nil {
		t.Fatal(err)
	}
	item, err := manager.createItem(ctx, "workspace-a", "vault-a", "owner-a", "", createItemInput{
		Type: "login", Name: "Fixture login", Username: "alice@example.test",
		Fields: map[string]string{"password": "browser-password", "totp": "JBSWY3DPEHPK3PXP"},
	})
	if err != nil {
		t.Fatal(err)
	}
	grant, err := manager.createGrant(ctx, "workspace-a", "owner-a", createGrantInput{
		VaultID: "vault-a", ItemID: item.ID, PrincipalType: "agent", PrincipalID: "agent-a",
		SelectorMode: "current_snapshot", Members: []string{"agent-a"}, Operations: []string{"inject", "sign"},
		Target: "https://login.example.test", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	return manager, item, grant
}

func TestBrowserCredentialUseBindsOriginAndDocumentAndClearsFailedFields(t *testing.T) {
	manager, item, grant := credentialUseFixture(t)
	executor := &recordingBrowserExecutor{}
	manager.SetTrustedBrowserExecutor(executor)
	ctx := context.Background()
	session, err := manager.createBrowserSession(ctx, "workspace-a", "agent-a", browserUseInput{
		GrantID: grant.ID, ItemID: item.ID, Origin: "https://login.example.test/", Account: item.Username,
		DocumentID: "document-1", Allowed: []string{"login"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.browserAction(ctx, "workspace-a", "agent-a", session.ID, browserActionInput{
		Action: "navigate", Origin: session.Origin, NavigationURL: "https://attacker.example.test", DocumentID: session.DocumentID,
	}); !errors.Is(err, errBrowserAuthorizationRequired) {
		t.Fatalf("cross-origin navigation error = %v, want reauthorization", err)
	}
	if _, err := manager.browserAction(ctx, "workspace-a", "agent-a", session.ID, browserActionInput{Action: "fill", Origin: session.Origin, DocumentID: "wrong-document"}); !errors.Is(err, errBrowserDocumentMismatch) {
		t.Fatalf("document race error = %v, want document mismatch", err)
	}
	if _, err := manager.browserAction(ctx, "workspace-a", "agent-a", session.ID, browserActionInput{Action: "fill", Origin: session.Origin, DocumentID: session.DocumentID}); err != nil {
		t.Fatal(err)
	}
	failed, err := manager.browserAction(ctx, "workspace-a", "agent-a", session.ID, browserActionInput{Action: "submit", Origin: session.Origin, DocumentID: session.DocumentID})
	if err != nil {
		t.Fatal(err)
	}
	if failed.OperationClass != "login_failed_fields_cleared" || len(executor.cleared) == 0 {
		t.Fatalf("failed login result = %+v, cleared = %+v", failed, executor.cleared)
	}
	encoded, _ := json.Marshal(failed)
	if strings.Contains(string(encoded), "browser-password") || strings.Contains(string(encoded), "JBSWY3DPEHPK3PXP") {
		t.Fatalf("browser result leaked a secret: %s", encoded)
	}
}

func TestBrowserCredentialUseDeniesCookieAndSecretEvaluationAndRevokedHandle(t *testing.T) {
	manager, item, grant := credentialUseFixture(t)
	manager.SetTrustedBrowserExecutor(&recordingBrowserExecutor{})
	ctx := context.Background()
	session, err := manager.createBrowserSession(ctx, "workspace-a", "agent-a", browserUseInput{
		GrantID: grant.ID, ItemID: item.ID, Origin: "https://login.example.test", Account: item.Username,
		DocumentID: "document-1", Allowed: []string{"login"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"cookies", "evaluate", "secret_field", "dom"} {
		if _, err := manager.browserAction(ctx, "workspace-a", "agent-a", session.ID, browserActionInput{Action: action, Origin: session.Origin, DocumentID: session.DocumentID}); err == nil || !strings.Contains(err.Error(), "unsupported browser operation") {
			t.Fatalf("action %q error = %v, want unsupported", action, err)
		}
	}
	manager.terminateCredentialUse(ctx, "workspace-a", grant.ID, "", "grant_revoked")
	if _, err := manager.browserAction(ctx, "workspace-a", "agent-a", session.ID, browserActionInput{Action: "fill", Origin: session.Origin, DocumentID: session.DocumentID}); !errors.Is(err, errAccessDenied) {
		t.Fatalf("revoked browser handle error = %v, want access denied", err)
	}
}

func TestCredentialUseRunTerminationRevokesAllRunBoundHandles(t *testing.T) {
	manager, item, grant := credentialUseFixture(t)
	executor := &recordingBrowserExecutor{}
	manager.SetTrustedBrowserExecutor(executor)
	ctx := context.Background()
	const runID = "run-credential-cleanup"
	browser, err := manager.createBrowserSession(ctx, "workspace-a", "agent-a", browserUseInput{
		GrantID: grant.ID, ItemID: item.ID, RunID: runID, Origin: "https://login.example.test", Account: item.Username,
		DocumentID: "document-1", Allowed: []string{"login"},
	})
	if err != nil {
		t.Fatal(err)
	}
	manager.terminateCredentialUseForRun(ctx, "workspace-a", runID, "run_terminal")
	if _, err := manager.browserAction(ctx, "workspace-a", "agent-a", browser.ID, browserActionInput{Action: "fill", Origin: browser.Origin, DocumentID: browser.DocumentID}); !errors.Is(err, errAccessDenied) {
		t.Fatalf("terminal browser handle error = %v, want access denied", err)
	}
	if executor.closed != 1 {
		t.Fatalf("closed executor calls = %d, want 1", executor.closed)
	}
}

func TestBrowserCredentialUseReportsUnrestrictedExposureWithoutFallingBack(t *testing.T) {
	manager, item, grant := credentialUseFixture(t)
	_, err := manager.createBrowserSession(context.Background(), "workspace-a", "agent-a", browserUseInput{
		GrantID: grant.ID, ItemID: item.ID, Origin: "https://login.example.test", Account: item.Username,
		DocumentID: "document-1", Exposure: string(browserExposureUnrestricted), Allowed: []string{"login"},
	})
	if err == nil || !strings.Contains(err.Error(), "unrestricted") {
		t.Fatalf("unrestricted exposure error = %v, want explicit broader exposure class", err)
	}
}

func TestSSHCredentialUseSignsWithoutReturningPrivateKey(t *testing.T) {
	ctx := context.Background()
	manager := newPasswordManager(nil)
	if err := manager.lock(ctx, "workspace-a", "vault-a", false); err != nil {
		t.Fatal(err)
	}
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(private)})
	item, err := manager.createItem(ctx, "workspace-a", "vault-a", "owner-a", "", createItemInput{
		Type: "ssh", Name: "Fixture SSH", Fields: map[string]string{"private_key": string(privatePEM), "principal": "alice"},
	})
	if err != nil {
		t.Fatal(err)
	}
	grant, err := manager.createGrant(ctx, "workspace-a", "owner-a", createGrantInput{
		VaultID: "vault-a", ItemID: item.ID, PrincipalType: "agent", PrincipalID: "agent-a", SelectorMode: "current_snapshot",
		Members: []string{"agent-a"}, Operations: []string{"sign"}, Target: "fixture.example.test:22", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	signer, err := manager.createSSHSigner(ctx, "workspace-a", "agent-a", sshSignerInput{GrantID: grant.ID, ItemID: item.ID, Destination: "fixture.example.test:22", Principal: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	challenge := []byte("server-session-challenge")
	result, err := manager.signSSH(ctx, "workspace-a", "agent-a", signer.ID, sshSignInput{Challenge: base64.RawURLEncoding.EncodeToString(challenge)})
	if err != nil {
		t.Fatal(err)
	}
	if result.Signature == "" || result.Algorithm == "" {
		t.Fatalf("empty SSH result: %+v", result)
	}
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), string(privatePEM)) || strings.Contains(string(encoded), "PRIVATE KEY") {
		t.Fatalf("SSH result leaked private key: %s", encoded)
	}
	manager.terminateCredentialUse(ctx, "workspace-a", grant.ID, "", "grant_revoked")
	if _, err := manager.signSSH(ctx, "workspace-a", "agent-a", signer.ID, sshSignInput{Challenge: base64.RawURLEncoding.EncodeToString(challenge)}); !errors.Is(err, errAccessDenied) {
		t.Fatalf("revoked SSH signer error = %v, want access denied", err)
	}
}
