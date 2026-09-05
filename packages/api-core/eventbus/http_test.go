package eventbus

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/provenance"
)

type hijackingRecorder struct{ *httptest.ResponseRecorder }

func (h hijackingRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	server, client := net.Pipe()
	_ = client.Close()
	return server, bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server)), nil
}

func TestVerifiedCorrelationUsesOnlyVerifiedProvenance(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := VerifiedCorrelation(request); got.RunID != "" {
		t.Fatalf("unverified correlation = %+v", got)
	}
	request = request.WithContext(provenance.NewContext(request.Context(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-1", TaskID: "task-1", ProfileKey: "agent-1"}))
	if got := VerifiedCorrelation(request); got.RunID != "run-1" {
		t.Fatalf("verified correlation = %+v", got)
	}
	request.Header.Set("X-Agent-Identity-Token", "verified-token")
	if got := VerifiedIdentityToken(request); got != "verified-token" {
		t.Fatalf("verified token = %q", got)
	}
	if got := VerifiedSubjectID(request); got != "agent-1" {
		t.Fatalf("verified subject = %q", got)
	}
}

func TestMiddlewareDoesNotAlterBusinessResponseWithoutMatchingPolicy(t *testing.T) {
	h := Middleware(MiddlewareConfig{Source: "a", Target: "b"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) }))
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/x", nil))
	if r.Code != http.StatusCreated {
		t.Fatalf("status=%d", r.Code)
	}
}

func TestMiddlewarePreservesResponseWriterHijacker(t *testing.T) {
	var supportsHijacking bool
	h := Middleware(MiddlewareConfig{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, supportsHijacking = w.(http.Hijacker)
	}))
	h.ServeHTTP(hijackingRecorder{httptest.NewRecorder()}, httptest.NewRequest(http.MethodGet, "/terminal", nil))
	if !supportsHijacking {
		t.Fatal("eventbus middleware removed http.Hijacker required by WebSocket handlers")
	}
}
