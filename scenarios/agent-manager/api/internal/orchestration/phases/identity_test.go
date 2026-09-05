package phases

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"

	"github.com/vrooli/api-core/scopecatalog"

	"github.com/google/uuid"
)

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
