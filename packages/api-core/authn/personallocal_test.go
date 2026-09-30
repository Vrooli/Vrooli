package authn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

func TestPersonalLocalProviderRequiresRuntimeOwnedSessionToken(t *testing.T) {
	provider := PersonalLocalProvider{
		SessionToken: "local-session-token",
		Scopes:       []string{"vrooli-onboarding:write"},
	}
	req := httptest.NewRequest("POST", "http://127.0.0.1/api", nil)
	req.RemoteAddr = "127.0.0.1:44000"
	req.Header.Set("Authorization", "LocalSession local-session-token")

	principal, err := provider.VerifyRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if principal.Source != identity.SourcePersonalLocal || principal.Kind != identity.ActorHuman || principal.Subject == "" || principal.Subject == "osuser:1001" || !principal.Verified || len(principal.Scopes) != 1 {
		t.Fatalf("principal = %#v", principal)
	}
}

func TestPersonalLocalProviderRejectsMissingInvalidRemoteAgentAndForwardedCredential(t *testing.T) {
	provider := PersonalLocalProvider{SessionToken: "local-session-token"}
	local := func() *http.Request {
		req := httptest.NewRequest("POST", "http://127.0.0.1/api", nil)
		req.RemoteAddr = "127.0.0.1:44000"
		return req
	}
	if _, err := provider.VerifyRequest(context.Background(), local()); err == nil {
		t.Fatal("loopback request without a session token unexpectedly authenticated")
	}
	wrong := local()
	wrong.Header.Set("Authorization", "Bearer supervisor-process-token")
	if _, err := provider.VerifyRequest(context.Background(), wrong); err == nil {
		t.Fatal("wrong forwarded credential unexpectedly authenticated")
	}

	remote := httptest.NewRequest("POST", "http://example.test/api", nil)
	remote.RemoteAddr = "192.0.2.10:44000"
	remote.Header.Set("Authorization", "Bearer local-session-token")
	if _, err := provider.VerifyRequest(context.Background(), remote); err == nil {
		t.Fatal("remote request unexpectedly authenticated")
	}

	agent := local()
	agent.Header.Set("Authorization", "Bearer local-session-token")
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-1"})
	if _, err := provider.VerifyRequest(ctx, agent); err == nil {
		t.Fatal("agent request unexpectedly authenticated")
	}

}

func TestPersonalLocalProviderReadsTokenFileWithoutExposingTokenInPrincipal(t *testing.T) {
	provider := PersonalLocalProvider{
		TokenFile: "/run/vrooli/onboarding-token",
		ReadToken: func(path string) ([]byte, error) {
			if path != "/run/vrooli/onboarding-token" {
				t.Fatalf("token path = %q", path)
			}
			return []byte("file-session-token\n"), nil
		},
	}
	req := httptest.NewRequest("POST", "http://127.0.0.1/api", nil)
	req.RemoteAddr = "127.0.0.1:44000"
	req.Header.Set("Authorization", "Bearer file-session-token")
	principal, err := provider.VerifyRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if principal.Subject == "file-session-token" || principal.Subject == "" {
		t.Fatalf("principal leaked or omitted token identity: %#v", principal)
	}
}
