// unit_owner_http_test.go qualifies synthetic verified TLS and selector-only refusals.
package nativeisolation

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type refuseBody struct{}

func (refuseBody) Read([]byte) (int, error) { panic("unauthenticated request body read") }
func (refuseBody) Close() error             { return nil }
func syntheticUnitRequest(selector, operation string) *http.Request {
	raw, _ := json.Marshal(unitControl{operation, selector})
	r := httptest.NewRequest("POST", UnitOwnerEndpoint, strings.NewReader(string(raw)))
	cert := &x509.Certificate{Raw: []byte("synthetic-native-peer")}
	r.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}, VerifiedChains: [][]*x509.Certificate{{cert}}}
	return r
}
func TestFixedUnitOwnerHTTPVerifiedSelectorReadAndZeroEffectRejections(t *testing.T) {
	d := &fixtureDriver{terminal: true}
	o, _, r, _ := unitOwnerFixture(t, d)
	plan, e := o.PrepareNative(context.Background(), r)
	if e != nil {
		t.Fatal(e)
	}
	o.units[plan.selector].started = true
	d.shows = 2
	h, e := NewFixedUnitOwnerHandler(o)
	if e != nil {
		t.Fatal(e)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, syntheticUnitRequest(plan.selector, "read"))
	if w.Code != 200 || d.shows != 3 {
		t.Fatal("exact verified selector read refused", w.Code)
	}
	for _, name := range []string{"absent-TLS", "unverified-TLS", "wrong-peer", "authorization", "blank-authorization", "cookie", "run-proof", "effort-proof", "foreign-selector", "wrong-role", "native-start", "command", "malformed", "path", "query", "oversized"} {
		t.Run(name, func(t *testing.T) {
			before := d.started + d.shows + len(d.signals)
			request := syntheticUnitRequest(plan.selector, "read")
			switch name {
			case "absent-TLS":
				request.TLS = nil
				request.Body = refuseBody{}
			case "unverified-TLS":
				request.TLS.VerifiedChains = nil
				request.Body = refuseBody{}
			case "wrong-peer":
				request.TLS.PeerCertificates[0] = &x509.Certificate{Raw: []byte("foreign")}
				request.Body = refuseBody{}
			case "authorization":
				request.Header.Set("Authorization", "Bearer fixture")
				request.Body = refuseBody{}
			case "blank-authorization":
				request.Header["Authorization"] = []string{""}
				request.Body = refuseBody{}
			case "cookie":
				request.Header.Set("Cookie", "fixture=value")
				request.Body = refuseBody{}
			case "run-proof":
				request.Header.Set("X-Agent-Identity-Token", "fixture")
				request.Body = refuseBody{}
			case "effort-proof":
				request.Header.Set(effortauthority.Header, "fixture")
				request.Body = refuseBody{}
			case "foreign-selector":
				request = syntheticUnitRequest(digest("foreign"), "read")
			case "wrong-role":
				request.TLS.PeerCertificates[0] = &x509.Certificate{Raw: []byte("synthetic-bootstrap-peer")}
			case "native-start":
				request = syntheticUnitRequest(plan.selector, "start")
			case "command":
				request.Body = io.NopCloser(strings.NewReader(`{"selector":"` + plan.selector + `","operation":"read","command":"sh"}`))
			case "malformed":
				request.Body = io.NopCloser(strings.NewReader("{"))
			case "oversized":
				request.Body = io.NopCloser(strings.NewReader(strings.Repeat(" ", 4097)))
			case "path":
				request.URL.Path = "/api/v1/runs"
				request.Body = refuseBody{}
			case "query":
				request.URL.RawQuery = "unit=foreign"
				request.Body = refuseBody{}
			}
			recorder := httptest.NewRecorder()
			h.ServeHTTP(recorder, request)
			if recorder.Code != 403 || before != d.started+d.shows+len(d.signals) {
				t.Fatalf("refusal effects: %s status%d", name, recorder.Code)
			}
		})
	}
}
