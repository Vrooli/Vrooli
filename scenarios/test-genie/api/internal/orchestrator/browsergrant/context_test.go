package browsergrant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

type verifiedAgent struct{}

func (verifiedAgent) Verify(token string) (*cliutil.VerifyResult, error) {
	return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: token}}, nil
}

func agentContext(t *testing.T, token string) context.Context {
	t.Helper()
	var got context.Context
	handler := provenance.Middleware(verifiedAgent{})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = r.Context() }))
	req := httptest.NewRequest(http.MethodPost, "/start", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, token)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestVerifiedBrowserGrantIsCarriedOnlyInActiveRunContext(t *testing.T) {
	initiator := AttachRequestReceipt(agentContext(t, "original-agent-token"), []string{"opaque-browser-grant"})
	runCtx := provenance.CarryVerifiedIdentity(context.Background(), initiator)
	runCtx = CarryVerifiedReceipt(runCtx, initiator)
	runCtx = AudioToolsSoakContext(runCtx, "opaque-test-genie-lease")

	if got := Receipt(runCtx); got != "opaque-browser-grant" {
		t.Fatalf("request-scoped grant = %q", got)
	}
	headers := AudioToolsSoakHeaders(runCtx)
	if headers.Get(cliutil.HeaderAgentIdentityToken) != "original-agent-token" ||
		headers.Get(HeaderBrowserGrantReceipt) != "opaque-browser-grant" ||
		headers.Get("X-Test-Genie-Lease-Receipt") != "opaque-test-genie-lease" || len(headers) != 3 {
		t.Fatalf("Audio Tools soak headers = %#v", headers)
	}
	if got := Receipt(context.Background()); got != "" {
		t.Fatalf("grant escaped run context: %q", got)
	}
}

func TestGrantDoesNotPropagateWithoutVerifiedIdentityOrExactlyOneReceipt(t *testing.T) {
	for name, tc := range map[string]struct {
		ctx    context.Context
		values []string
	}{
		"unverified identity": {context.Background(), []string{"untrusted"}},
		"missing receipt":     {agentContext(t, "verified-agent"), nil},
		"duplicate receipt":   {agentContext(t, "verified-agent"), []string{"one", "two"}},
		"blank receipt":       {agentContext(t, "verified-agent"), []string{"  "}},
	} {
		t.Run(name, func(t *testing.T) {
			initiator := AttachRequestReceipt(tc.ctx, tc.values)
			runCtx := provenance.CarryVerifiedIdentity(context.Background(), initiator)
			runCtx = CarryVerifiedReceipt(runCtx, initiator)
			runCtx = AudioToolsSoakContext(runCtx, "lease-receipt")
			if got := Receipt(runCtx); got != "" {
				t.Fatalf("unexpected grant propagation: %q", got)
			}
			if headers := AudioToolsSoakHeaders(runCtx); len(headers) != 0 {
				t.Fatalf("unexpected partial bearer headers: %#v", headers)
			}
		})
	}
}
