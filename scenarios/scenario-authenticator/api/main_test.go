package main

import (
	"context"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
	"time"

	"scenario-authenticator/internal/realm"

	"github.com/vrooli/api-core/trustposture"
)

// TestBreakGlassIssuersConform proves the authenticated scenario issuer and
// the operator passphrase issuer produce credentials accepted by the same real
// verifier with the same audience, target, and scope semantics.
func TestBreakGlassIssuersConform(t *testing.T) {
	now := time.Unix(10_000, 0).UTC()
	target := "host-a"
	audience := realm.DefaultAudience
	scopes := []string{"vrooli:uninstall"}

	authDir := t.TempDir()
	authPaths := trustposture.KeyPaths{Dir: authDir, Private: filepath.Join(authDir, "private.key"), Public: filepath.Join(authDir, "public.key"), Metadata: filepath.Join(authDir, "provisioning.json")}
	authIssuer := breakGlassProvisioner{paths: authPaths, available: true, ttl: 10 * time.Minute, target: target}
	if err := authIssuer.Provision(context.TODO(), "operator-1", "default", scopes, now); err != nil {
		t.Fatal(err)
	}
	authToken, _, err := authIssuer.Issue(context.TODO(), "operator-1", "default", scopes, now)
	if err != nil {
		t.Fatal(err)
	}

	operatorDir := t.TempDir()
	operatorPaths := trustposture.KeyPaths{Dir: operatorDir, Private: filepath.Join(operatorDir, "private.key"), WrappedPrivate: filepath.Join(operatorDir, "private.key"), Public: filepath.Join(operatorDir, "public.key"), Metadata: filepath.Join(operatorDir, "provisioning.json")}
	if err := trustposture.ProvisionWrapped(operatorPaths, "correct horse", "operator-1", audience, target, scopes, now); err != nil {
		t.Fatal(err)
	}
	operatorToken, err := trustposture.IssueFromWrappedProvision(operatorPaths, "correct horse", audience, target, scopes, now, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	authPublic, err := os.ReadFile(authPaths.Public)
	if err != nil {
		t.Fatal(err)
	}
	authClaims, err := trustposture.Verify(ed25519.PublicKey(authPublic), authToken, audience, target, now)
	if err != nil {
		t.Fatal(err)
	}
	operatorPublic, err := os.ReadFile(operatorPaths.Public)
	if err != nil {
		t.Fatal(err)
	}
	operatorClaims, err := trustposture.Verify(ed25519.PublicKey(operatorPublic), operatorToken, audience, target, now)
	if err != nil {
		t.Fatal(err)
	}
	if authClaims.Subject != operatorClaims.Subject || authClaims.Audience != operatorClaims.Audience || authClaims.Target != operatorClaims.Target || len(authClaims.Scopes) != len(operatorClaims.Scopes) {
		t.Fatalf("issuer claim shape diverged: auth=%+v operator=%+v", authClaims, operatorClaims)
	}
}
