package provenance

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// A request that presents one of this process's own service tokens is the
// service speaking as itself, however it was built: the default transport adds
// no caller identity to it. A request presenting any other bearer (an owner
// credential) still carries the agent, so the receiver keeps the agent's limits.
func TestTheDefaultTransportNeverAddsAnAgentToARequestThatPresentsAServiceToken(t *testing.T) {
	MarkServiceBearer("service-token", time.Now().Add(time.Hour))
	seen := map[string]string{}
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.URL.Path] = r.Header.Get(cliutil.HeaderAgentIdentityToken)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer receiver.Close()

	send := func(r *http.Request, path, bearer string) {
		out, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, receiver.URL+path, nil)
		if bearer != "" {
			out.Header.Set("Authorization", "Bearer "+bearer)
		}
		if response, err := http.DefaultClient.Do(out); err == nil {
			_ = response.Body.Close()
		}
	}
	inbound := Middleware(VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "run-a"}}, nil
	}))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		send(r, "/service", "service-token")
		send(r, "/owner", "owner-credential")
		send(r, "/plain", "")
	}))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "run-a-credential")
	inbound.ServeHTTP(httptest.NewRecorder(), req)

	if got := seen["/service"]; got != "" {
		t.Fatalf("a service-token request carried the agent identity %q", got)
	}
	if seen["/owner"] != "run-a-credential" || seen["/plain"] != "run-a-credential" {
		t.Fatalf("other requests must still carry the agent: owner=%q plain=%q", seen["/owner"], seen["/plain"])
	}
}
