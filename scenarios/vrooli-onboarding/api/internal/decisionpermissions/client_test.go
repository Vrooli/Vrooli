package decisionpermissions

import (
	"net/http"
	"testing"

	"connectrpc.com/connect"
	conversationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

func TestIncomingOwnerProofForwardsOnlyNarrowAuthenticationHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer owner-token")
	headers.Set("Cf-Access-Jwt-Assertion", "access-jwt")
	headers.Set("Cookie", "other=value; notification_hub_owner_token=hub-cookie")
	headers.Set("X-Forwarded-For", "attacker-controlled")
	ctx := WithIncomingOwnerProof(t.Context(), headers)
	proof, ok := ctx.Value(ownerProofKey{}).(ownerProof)
	if !ok || proof.invalid || proof.authorization != "Bearer owner-token" || proof.accessJWT != "access-jwt" || proof.cookie != "notification_hub_owner_token=hub-cookie" {
		t.Fatalf("unexpected forwarded proof: ok=%t proof=%+v", ok, proof)
	}

	connectRequest := withOwnerProof(proof, connect.NewRequest(&conversationsv1.ListPendingDecisionReferencesRequest{}))
	if connectRequest.Header().Get("Authorization") != "Bearer owner-token" || connectRequest.Header().Get("Cf-Access-Jwt-Assertion") != "access-jwt" || connectRequest.Header().Get("Cookie") != "notification_hub_owner_token=hub-cookie" || connectRequest.Header().Get("X-Forwarded-For") != "" {
		t.Fatalf("unexpected request headers: %v", connectRequest.Header())
	}
}

func TestIncomingOwnerProofRejectsDuplicateAuthorization(t *testing.T) {
	headers := http.Header{"Authorization": []string{"Bearer one", "Bearer two"}}
	ctx := WithIncomingOwnerProof(t.Context(), headers)
	proof, ok := ctx.Value(ownerProofKey{}).(ownerProof)
	if !ok || !proof.invalid {
		t.Fatalf("duplicate Authorization was not rejected: ok=%t proof=%+v", ok, proof)
	}
}
