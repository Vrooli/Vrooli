package phases

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"

	"github.com/vrooli/api-core/scopecatalog"

	"github.com/google/uuid"
)

func TestGenerateIdentityTokenReplacesRevokedGenerationEvenWithinSameSecond(t *testing.T) {
	now := time.Now()
	run := &domain.Run{ID: uuid.New(), TaskID: uuid.New(), OwnerSubject: "original-owner", OwnerScopes: []string{"agent-manager:read"}}
	in := GenerateIdentityTokenInput{Run: run, Secret: []byte("test-rotation-secret"), Deps: Deps{Clock: func() time.Time { return now }}}
	first := GenerateIdentityToken(context.Background(), in)
	run.IdentityTokenRevokedAt = &now
	second := GenerateIdentityToken(context.Background(), in)
	if first == "" || second == "" || first == second {
		t.Fatal("reissued credential must have a distinct generation even at the same timestamp")
	}
	if run.IdentityTokenRevokedAt != nil || run.IdentityTokenHash != identity.HashToken(second) {
		t.Fatal("new credential did not replace the revoked generation")
	}
	claims, err := identity.VerifyToken(second, in.Secret)
	if err != nil || claims.Subject != "original-owner" || len(claims.Scopes) != 1 || claims.Scopes[0] != "agent-manager:read" {
		t.Fatal("rotation changed authority or invalidated signature")
	}
	run.IdentityTokenRevokedAt = &now
	in.Secret = nil
	if got := GenerateIdentityToken(context.Background(), in); got != "" || run.IdentityTokenRevokedAt == nil {
		t.Fatal("disabled minting must not un-revoke an old credential")
	}
}

func TestGenerateIdentityTokenRequiresOwnerCeilingForDeclaredScopes(t *testing.T) {
	in := GenerateIdentityTokenInput{Run: &domain.Run{ID: uuid.New(), TaskID: uuid.New()}, Profile: &domain.AgentProfile{DeclaredScopes: []string{"agent-manager:supervise"}}, RequestedScopes: []string{"agent-manager:supervise"}, Secret: []byte("test-secret")}
	claims, err := identity.VerifyToken(GenerateIdentityToken(context.Background(), in), in.Secret)
	if err != nil || len(claims.Scopes) != 0 {
		t.Fatal("profile or request declaration became a grant", err)
	}
}

func TestGenerateIdentityToken_SignsWorkflowMetadata(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), TaskID: uuid.New()}
	token := GenerateIdentityToken(context.Background(), GenerateIdentityTokenInput{
		Run: run, Secret: []byte("test-secret"),
		Meta: map[string]string{"workflowExecutionId": "execution-1", "workflowNodeId": "node-a", "workflowAttemptId": "attempt-1"},
	})
	if token == "" {
		t.Fatal("expected token")
	}
	claims, err := identity.VerifyToken(token, []byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if claims.Meta["workflowExecutionId"] != "execution-1" || claims.Meta["workflowNodeId"] != "node-a" || claims.Meta["workflowAttemptId"] != "attempt-1" {
		t.Fatalf("claims metadata = %#v", claims.Meta)
	}
}

func TestGenerateIdentityTokenCarriesExplicitAttenuatedIdentity(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), TaskID: uuid.New(), Subject: []string{"owner@example"}}
	token := GenerateIdentityToken(context.Background(), GenerateIdentityTokenInput{
		Run: run, Secret: []byte("test-secret"),
		AccountScopes:   []string{"vrooli-bridge:read", "vrooli-bridge:dispatch"},
		RequestedScopes: []string{"vrooli-bridge:dispatch"},
	})
	claims, err := identity.VerifyToken(token, []byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "owner@example" {
		t.Fatalf("subject = %q", claims.Subject)
	}
	if len(claims.Scopes) != 1 || claims.Scopes[0] != "vrooli-bridge:dispatch" {
		t.Fatalf("scopes = %#v", claims.Scopes)
	}
}

func TestGenerateIdentityTokenCarriesWorkspaceBinding(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), TaskID: uuid.New()}
	token := GenerateIdentityToken(context.Background(), GenerateIdentityTokenInput{
		Run: run, Secret: []byte("test-secret"), Meta: map[string]string{"workspace_id": "workspace-a"},
	})
	claims, err := identity.VerifyToken(token, []byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if claims.WorkspaceID != "workspace-a" {
		t.Fatalf("workspace = %q, want workspace-a", claims.WorkspaceID)
	}
}

func TestGenerateIdentityTokenPrefersPersistedOwnerIdentity(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), TaskID: uuid.New(), OwnerSubject: "account-42", OwnerScopes: []string{"agent-manager:write"}}
	token := GenerateIdentityToken(context.Background(), GenerateIdentityTokenInput{
		Run: run, Secret: []byte("test-secret"),
	})
	claims, err := identity.VerifyToken(token, []byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "account-42" {
		t.Fatalf("subject = %q", claims.Subject)
	}
	if len(claims.Scopes) != 1 || claims.Scopes[0] != "agent-manager:write" {
		t.Fatalf("scopes = %#v", claims.Scopes)
	}
}

func TestGenerateIdentityTokenMaterializesCatalogWildcardAndOmitsHumanOnly(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), TaskID: uuid.New(), Subject: []string{"owner@example"}}
	token := GenerateIdentityToken(context.Background(), GenerateIdentityTokenInput{
		Run: run, Secret: []byte("test-secret"), AccountScopes: []string{"vrooli-bridge:*"},
		ConcreteScopes: []scopecatalog.Scope{
			{Value: "vrooli-bridge:read", RunEligible: true},
			{Value: "vrooli-bridge:write", RunEligible: false},
		},
	})
	claims, err := identity.VerifyToken(token, []byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if len(claims.Scopes) != 1 || claims.Scopes[0] != "vrooli-bridge:read" {
		t.Fatalf("scopes = %#v", claims.Scopes)
	}
}
