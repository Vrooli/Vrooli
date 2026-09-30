package policygate

import (
	"context"
	"net/http/httptest"
	"os/user"
	"testing"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

func TestPersonalLocalProviderBindsLoopbackToCurrentOSUser(t *testing.T) {
	provider := PersonalLocalProvider{currentUser: func() (*user.User, error) {
		return &user.User{Uid: "1001", Username: "operator"}, nil
	}}
	req := httptest.NewRequest("GET", "http://127.0.0.1/api", nil)
	req.RemoteAddr = "127.0.0.1:44000"

	principal, err := provider.VerifyRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if principal.Source != identity.SourcePersonalLocal || principal.Kind != identity.ActorHuman || principal.Subject != "osuser:1001" || !principal.Verified {
		t.Fatalf("principal = %#v", principal)
	}
}

func TestPersonalLocalProviderRejectsRemoteAndAgentRequests(t *testing.T) {
	provider := PersonalLocalProvider{currentUser: func() (*user.User, error) {
		return &user.User{Uid: "1001"}, nil
	}}
	remote := httptest.NewRequest("GET", "http://example.test/api", nil)
	remote.RemoteAddr = "192.0.2.10:44000"
	_, err := provider.VerifyRequest(context.Background(), remote)
	if err == nil {
		t.Fatal("remote request unexpectedly authenticated")
	}

	agent := httptest.NewRequest("GET", "http://127.0.0.1/api", nil)
	agent.RemoteAddr = "127.0.0.1:44000"
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-1"})
	_, err = provider.VerifyRequest(ctx, agent)
	if err == nil {
		t.Fatal("agent request unexpectedly authenticated")
	}
}
