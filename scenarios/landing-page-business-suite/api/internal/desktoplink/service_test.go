package desktoplink

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestServiceIssueRedeemIsScopedOneUseAndAuditableByRepository(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	verifier := "desktop-code-verifier"
	challenge := pkceChallenge(verifier)
	expires := time.Now().UTC().Add(time.Minute)
	rawCode, authorization, err := service.Issue(context.Background(), AuthorizationRequest{
		LPBSUserID: "lpbs-user-1", BusinessAccountID: "lpbs-user-1", InstallationID: "install-1",
		Resource: "git-control-tower", Audience: "scenario:git-control-tower", Scopes: []string{"git-control-tower:read"},
		CodeChallenge: challenge, RedirectURI: "http://127.0.0.1:43120/callback", ExpiresAt: expires,
	})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if rawCode == "" || authorization.CodeHash == rawCode {
		t.Fatal("Issue() must return a one-use raw code while persisting only its hash")
	}

	if _, err := service.Redeem(context.Background(), rawCode, "wrong", "local-principal-1", "install-1", "git-control-tower"); err == nil {
		t.Fatal("Redeem() accepted the wrong PKCE verifier")
	}
	link, err := service.Redeem(context.Background(), rawCode, verifier, "local-principal-1", "install-1", "git-control-tower")
	if err != nil {
		t.Fatalf("Redeem() error = %v", err)
	}
	if link.LPBSUserID != "lpbs-user-1" || link.LocalPrincipal != "local-principal-1" || link.BusinessAccountID != "lpbs-user-1" || len(link.Scopes) != 1 {
		t.Fatalf("Redeem() link = %+v", link)
	}
	if _, err := service.Redeem(context.Background(), rawCode, verifier, "local-principal-1", "install-1", "git-control-tower"); err == nil {
		t.Fatal("Redeem() allowed authorization code reuse")
	}
	if err := service.RevokeForLocal(context.Background(), "local-principal-1", "install-1", "git-control-tower", "local-principal-1"); err != nil {
		t.Fatalf("RevokeForLocal() error = %v", err)
	}
}

func TestServiceRejectsUnboundedOrNonLoopbackRequests(t *testing.T) {
	service := NewService(NewMemoryRepository())
	base := AuthorizationRequest{
		LPBSUserID: "lpbs-user-1", BusinessAccountID: "lpbs-user-1", InstallationID: "install-1",
		Resource: "demo", Audience: "scenario:demo", Scopes: []string{"demo:read"},
		CodeChallenge: pkceChallenge("verifier"), RedirectURI: "http://127.0.0.1:43120/callback",
	}
	for name, mutate := range map[string]func(*AuthorizationRequest){
		"non-loopback redirect": func(req *AuthorizationRequest) { req.RedirectURI = "https://example.com/callback" },
		"duplicate scopes":      func(req *AuthorizationRequest) { req.Scopes = []string{"demo:read", "demo:read"} },
		"missing audience":      func(req *AuthorizationRequest) { req.Audience = "" },
		"cross-resource audience": func(req *AuthorizationRequest) {
			req.Audience = "scenario:other"
		},
		"cross-resource scope": func(req *AuthorizationRequest) {
			req.Scopes = []string{"other:read"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			req := base
			mutate(&req)
			if _, _, err := service.Issue(context.Background(), req); err == nil {
				t.Fatal("Issue() accepted invalid request")
			}
		})
	}
}

func pkceChallenge(verifier string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(verifier)))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
