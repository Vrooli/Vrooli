package authz

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

func TestApprovedAgentCanPrepareDeploymentButCannotApproveReleaseOrAttestEvidence(t *testing.T) {
	p := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Owner: "owner", Verified: true, Scopes: []string{WriteCapability, DestructiveCapability}, Grant: &identity.OperationGrant{ID: "grant", Approver: "owner", Operations: []string{"deployment.manage"}, Targets: []string{"deployment-manager:*"}, ExpiresAt: time.Now().Add(time.Hour)}}
	ctx := identity.WithPrincipal(context.Background(), p)
	if err := RequireWrite(ctx); err != nil {
		t.Fatal(err)
	}
	if err := RequireDestructive(ctx); err != nil {
		t.Fatal(err)
	}
	if err := RequireReleaseDecision(ctx); err == nil {
		t.Fatal("deployment operation grant approved a release")
	}
	if err := RequireReadinessEvidence(ctx); err == nil {
		t.Fatal("deployment operation grant attested producer evidence")
	}
}

func TestRequireWriteNeedsHumanAndCapability(t *testing.T) {
	base := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true}
	if err := RequireWrite(identity.WithPrincipal(context.Background(), base)); err == nil {
		t.Fatal("write authorization accepted a principal without the write capability")
	}
	base.Scopes = []string{WriteCapability}
	if err := RequireWrite(identity.WithPrincipal(context.Background(), base)); err != nil {
		t.Fatalf("write authorization rejected the declared capability: %v", err)
	}
	base.Kind = identity.ActorAgent
	if err := RequireWrite(identity.WithPrincipal(context.Background(), base)); err == nil {
		t.Fatal("write authorization accepted an agent for a human-only mutation")
	}
}

func TestRequireReadNeedsVerifiedPrincipalAndCapability(t *testing.T) {
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{ReadCapability}}
	if err := RequireRead(identity.WithPrincipal(context.Background(), principal)); err != nil {
		t.Fatalf("RequireRead() = %v", err)
	}
	principal.Scopes = nil
	if err := RequireRead(identity.WithPrincipal(context.Background(), principal)); err == nil {
		t.Fatal("read authorization accepted a principal without read capability")
	}
}

func TestRequireClientUpdateReceiptNeedsServiceAndCapability(t *testing.T) {
	principal := identity.Principal{Kind: identity.ActorService, Subject: "scenario-to-desktop", Verified: true, Scopes: []string{ClientReceiptCapability}}
	if _, err := authn.RequireService(identity.WithPrincipal(context.Background(), principal)); err != nil {
		t.Fatal(err)
	}
	if err := RequireClientUpdateReceipt(identity.WithPrincipal(context.Background(), principal)); err != nil {
		t.Fatalf("RequireClientUpdateReceipt() = %v", err)
	}
	human := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{ClientReceiptCapability}}
	if err := RequireClientUpdateReceipt(identity.WithPrincipal(context.Background(), human)); err == nil {
		t.Fatal("human principal unexpectedly admitted to owner receipt route")
	}
}

func TestRequireProductionReleasePreparationKeepsServiceAndCapabilityBoundaries(t *testing.T) {
	service := identity.Principal{Kind: identity.ActorService, Subject: "service:deployment-manager:preparation", Verified: true, Audience: "scenario-authenticator:deployment-manager", Scopes: []string{WriteCapability}}
	if err := RequireProductionReleasePreparation(identity.WithPrincipal(context.Background(), service)); err != nil {
		t.Fatalf("prepared service token rejected: %v", err)
	}
	service.Scopes = []string{"deployment-manager:read"}
	if err := RequireProductionReleasePreparation(identity.WithPrincipal(context.Background(), service)); err == nil {
		t.Fatal("service without write capability passed release preparation")
	}
	service.Scopes = []string{WriteCapability}
	service.Kind = identity.ActorHuman
	if err := RequireProductionReleasePreparation(identity.WithPrincipal(context.Background(), service)); err == nil {
		t.Fatal("human principal passed service-only release preparation")
	}
}

func TestRequireReadinessEvidenceNeedsServiceAndCapability(t *testing.T) {
	principal := identity.Principal{Kind: identity.ActorService, Subject: "scenario-to-desktop", Verified: true, Scopes: []string{ReadinessEvidenceCapability}}
	if err := RequireReadinessEvidence(identity.WithPrincipal(context.Background(), principal)); err != nil {
		t.Fatalf("RequireReadinessEvidence() = %v", err)
	}
	principal.Scopes = nil
	if err := RequireReadinessEvidence(identity.WithPrincipal(context.Background(), principal)); err == nil {
		t.Fatal("readiness evidence authorization accepted a service without its capability")
	}
	principal.Kind = identity.ActorHuman
	principal.Scopes = []string{ReadinessEvidenceCapability}
	if err := RequireReadinessEvidence(identity.WithPrincipal(context.Background(), principal)); err == nil {
		t.Fatal("readiness evidence authorization accepted a human principal")
	}
}

func TestRequireReadinessEvidenceForBindingRequiresDeclaredOwner(t *testing.T) {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorService, Subject: "scenario-to-desktop", Verified: true, Scopes: []string{ReadinessEvidenceCapability}})
	if err := RequireReadinessEvidenceForBinding(ctx, "scenario-to-desktop.evidence.readiness"); err != nil {
		t.Fatalf("declared owner was rejected: %v", err)
	}
	if err := RequireReadinessEvidenceForBinding(ctx, "test-genie.runs.report"); err == nil {
		t.Fatal("owner service was allowed to report another owner's binding")
	}
}

func TestRequireEvidenceForRampBindsAuthenticatedProducer(t *testing.T) {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorService, Subject: "scenario-to-desktop", Verified: true, Scopes: []string{ReadinessEvidenceCapability}})
	if err := RequireEvidenceForRamp(ctx, "scenario-to-desktop"); err != nil {
		t.Fatalf("matching ramp was rejected: %v", err)
	}
	if err := RequireEvidenceForRamp(ctx, "scenario-to-android"); err == nil {
		t.Fatal("service was allowed to report another ramp's evidence")
	}
}

func TestRequireDestructiveDoesNotAcceptWriteAlone(t *testing.T) {
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{WriteCapability}}
	if err := RequireDestructive(identity.WithPrincipal(context.Background(), principal)); err == nil {
		t.Fatal("destructive authorization accepted the write capability alone")
	}
	principal.Scopes = append(principal.Scopes, DestructiveCapability)
	if err := RequireDestructive(identity.WithPrincipal(context.Background(), principal)); err != nil {
		t.Fatalf("destructive authorization rejected the declared capability: %v", err)
	}
}
