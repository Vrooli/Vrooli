package nodereach

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

// A Bridge call runs under the owner's credential. When an agent run's request
// led to it, the Bridge must see that run too, so it keeps the agent as the
// actor and intersects its grant with the owner's authority (authn.Middleware).
// Stripping the run here, as a service-token client does, would let delegated
// work run with the owner's full authority.
func TestABridgeCallMadeForAnAgentRunCarriesThatRunBesideTheOwnerCredential(t *testing.T) {
	var agent, bearer string
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		agent, bearer = req.Header.Get(cliutil.HeaderAgentIdentityToken), req.Header.Get("Authorization")
		http.NotFound(w, req)
	}))
	defer bridge.Close()
	client := New(Config{BridgeURL: bridge.URL, Token: "owner-credential"})

	inbound := provenance.Middleware(provenance.VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "run-a"}}, nil
	}))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = client.List(r.Context(), time.Second)
	}))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "run-a-credential")
	inbound.ServeHTTP(httptest.NewRecorder(), req)

	if bearer == "" {
		t.Fatal("the call did not reach the Bridge with the owner credential")
	}
	if agent != "run-a-credential" {
		t.Fatalf("the Bridge must see the agent run beside the owner credential, got %q", agent)
	}
}
