package chatauth

import (
	"context"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"net/http"
	"portal/internal/integrations/agentmanager"
	"testing"
	"time"
)

type auth01Verifier struct{}

func (auth01Verifier) Verify(_ context.Context, token string) (identity.Principal, error) {
	return identity.Principal{Kind: identity.ActorHuman, Subject: "fixture", Verified: true, Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

var _ authn.TokenVerifier = auth01Verifier{}

func TestAuth01PortalAuthenticatedIngressRetainsVerifiedProof(t *testing.T) {
	i := &interceptor{validator: auth01Verifier{}, now: time.Now}
	h := http.Header{"Authorization": []string{"Bearer fixture-owner"}}
	ctx, cancel, err := i.authorize(context.Background(), h)
	defer cancel()
	if err != nil || agentmanager.RequireCreateCaller(ctx) != nil {
		t.Fatalf("verified ingress lost%v", err)
	}
	h.Set("X-Agent-Identity-Token", "fixture-run")
	ctx, cancel, err = i.authorize(context.Background(), h)
	defer cancel()
	if err != nil || agentmanager.RequireCreateCaller(ctx) == nil {
		t.Fatal("unsupported offered run channel ignored")
	}
	ctx, cancel, err = i.authorize(context.Background(), http.Header{})
	defer cancel()
	if err != nil || agentmanager.RequireCreateCaller(ctx) == nil {
		t.Fatal("guest became creator")
	}
}
