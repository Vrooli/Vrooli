package effortauthority

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type profileReadSpy struct {
	calls int
	out   ProfileContract
	err   error
}

func (s *profileReadSpy) ReadFiniteProfileContract(context.Context, string) (ProfileContract, error) {
	s.calls++
	return s.out, s.err
}

// Synthetic verified TLS state exercises the HTTP handler contract only. It
// creates no certificate, listener, peer enrollment or host-isolation evidence.
func syntheticProfileRequest(binding Binding) *http.Request {
	raw, _ := json.Marshal(profileCheckRequest{Binding: binding, Key: "bounded"})
	r := httptest.NewRequest(http.MethodPost, ProfileContractEndpoint, bytes.NewReader(raw))
	cert := &x509.Certificate{Raw: []byte("synthetic-unit-peer")}
	r.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}, VerifiedChains: [][]*x509.Certificate{{cert}}}
	return r
}
func profileHandlerFixture(t *testing.T) (*ProfileContractHandler, *profileReadSpy, Binding, map[string]map[string]ProfileContractGrant) {
	t.Helper()
	b := Binding{PolicyID: "approved-policy", PolicyDigest: Digest("exact approved policy"), Epoch: 1, Owner: "owner", Client: "client", Deadline: time.Now().UTC().Add(time.Hour)}
	pin := sha256.Sum256([]byte("synthetic-unit-peer"))
	peers := map[string]map[string]ProfileContractGrant{hex.EncodeToString(pin[:]): {b.PolicyID: {Binding: b, Profiles: map[string]string{"bounded": "owner-profile-id"}}}}
	spy := &profileReadSpy{out: ProfileContract{ID: "owner-profile-id", Key: "bounded", Digest: Digest("complete owner profile")}}
	h, e := NewProfileContractHandler(spy, peers)
	if e != nil {
		t.Fatal(e)
	}
	return h, spy, b, peers
}
func TestReadinessProfileContractExactPeerAndImmutableGrants(t *testing.T) {
	h, spy, b, peers := profileHandlerFixture(t)
	// Mutation of offered installation maps must not expand the captured grant.
	for pin, grants := range peers {
		g := grants[b.PolicyID]
		g.Profiles["bounded"] = "changed"
		g.Profiles["foreign"] = "foreign"
		g.Binding.Owner = "changed"
		grants[b.PolicyID] = g
		delete(peers, pin)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, syntheticProfileRequest(b))
	if w.Code != 200 || spy.calls != 1 {
		t.Fatalf("valid exact peer refused: code=%d reads=%d body=%s", w.Code, spy.calls, w.Body.String())
	}
	var got ProfileContract
	if json.Unmarshal(w.Body.Bytes(), &got) != nil || got != spy.out {
		t.Fatal("owner contract changed", got)
	}
}
func TestReadinessProfileContractRefusesBeforeOwnerRead(t *testing.T) {
	cases := []string{"absent-TLS", "unverified-TLS", "absent-peer", "wrong-peer", "method", "path", "query", "authorization", "empty-authorization", "cookie", "run-channel", "effort-channel", "policy", "epoch", "owner", "client", "digest", "deadline-binding", "profile", "malformed", "unknown-field", "trailing", "oversized", "expired-grant"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			h, spy, b, _ := profileHandlerFixture(t)
			r := syntheticProfileRequest(b)
			switch name {
			case "absent-TLS":
				r.TLS = nil
			case "unverified-TLS":
				r.TLS.VerifiedChains = nil
			case "absent-peer":
				r.TLS.PeerCertificates = nil
			case "wrong-peer":
				r.TLS.PeerCertificates[0] = &x509.Certificate{Raw: []byte("foreign")}
			case "method":
				r.Method = "GET"
			case "path":
				r.URL.Path = "/api/v1/profiles"
			case "query":
				r.URL.RawQuery = "other=true"
			case "authorization":
				r.Header.Set("Authorization", "Bearer fixture")
			case "empty-authorization":
				r.Header["Authorization"] = []string{""}
			case "cookie":
				r.Header.Set("Cookie", "fixture=value")
			case "run-channel":
				r.Header.Set("X-Agent-Identity-Token", "fixture")
			case "effort-channel":
				r.Header.Set(Header, "fixture")
			case "policy":
				b.PolicyID = "foreign"
				r = syntheticProfileRequest(b)
			case "epoch":
				b.Epoch++
				r = syntheticProfileRequest(b)
			case "owner":
				b.Owner = "foreign"
				r = syntheticProfileRequest(b)
			case "client":
				b.Client = "foreign"
				r = syntheticProfileRequest(b)
			case "digest":
				b.PolicyDigest = Digest("foreign")
				r = syntheticProfileRequest(b)
			case "deadline-binding":
				b.Deadline = b.Deadline.Add(time.Minute)
				r = syntheticProfileRequest(b)
			case "profile":
				r.Body = http.NoBody
				raw, _ := json.Marshal(profileCheckRequest{Binding: b, Key: "foreign"})
				r.Body = ioReadCloser(raw)
			case "malformed":
				r.Body = ioReadCloser([]byte("{"))
			case "unknown-field":
				raw, _ := json.Marshal(profileCheckRequest{Binding: b, Key: "bounded"})
				r.Body = ioReadCloser(append(raw[:len(raw)-1], []byte(",\"unknown\":1}")...))
			case "trailing":
				raw, _ := json.Marshal(profileCheckRequest{Binding: b, Key: "bounded"})
				r.Body = ioReadCloser(append(raw, []byte(" {}")...))
			case "oversized":
				r.Body = ioReadCloser([]byte(strings.Repeat(" ", 8193) + "{}"))
			case "expired-grant":
				b.Deadline = time.Now().Add(-time.Second)
				for pin, grants := range h.peers {
					g := grants[b.PolicyID]
					g.Binding = b
					h.peers[pin][b.PolicyID] = g
				}
				r = syntheticProfileRequest(b)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 403 || spy.calls != 0 {
				t.Fatalf("refusal effects: code=%d reads=%d body=%s", w.Code, spy.calls, w.Body.String())
			}
		})
	}
}

type profileBody struct{ *bytes.Reader }

func (profileBody) Close() error          { return nil }
func ioReadCloser(raw []byte) profileBody { return profileBody{bytes.NewReader(raw)} }

func TestReadinessProfileContractOwnerMismatchRefuses(t *testing.T) {
	for _, name := range []string{"error", "key", "id", "digest"} {
		t.Run(name, func(t *testing.T) {
			h, spy, b, _ := profileHandlerFixture(t)
			switch name {
			case "error":
				spy.err = errors.New("owner unavailable")
			case "key":
				spy.out.Key = "foreign"
			case "id":
				spy.out.ID = "foreign"
			case "digest":
				spy.out.Digest = "invalid"
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, syntheticProfileRequest(b))
			if w.Code != 403 || spy.calls != 1 {
				t.Fatalf("code=%d reads=%d", w.Code, spy.calls)
			}
		})
	}
}

func TestReadinessProfileContractInstallationCheckHasNoOwnerEffects(t *testing.T) {
	for _, name := range []string{"exact", "policy", "digest", "epoch", "owner", "client", "deadline", "profile"} {
		t.Run(name, func(t *testing.T) {
			p := readinessInstallationFixture().Policy
			b := Binding{PolicyID: p.ID, PolicyDigest: Digest(p), Epoch: p.Epoch, Owner: p.Owner, Client: p.Client, Deadline: p.Deadline}
			profiles := map[string]string{"bounded": "owner-profile-id"}
			switch name {
			case "policy":
				b.PolicyID = "foreign"
			case "digest":
				b.PolicyDigest = Digest("foreign")
			case "epoch":
				b.Epoch++
			case "owner":
				b.Owner = "foreign"
			case "client":
				b.Client = "foreign"
			case "deadline":
				b.Deadline = b.Deadline.Add(time.Second)
			case "profile":
				profiles["foreign"] = "foreign"
			}
			spy := &profileReadSpy{}
			h, e := NewProfileContractHandler(spy, map[string]map[string]ProfileContractGrant{Digest("synthetic-pin"): {b.PolicyID: {Binding: b, Profiles: profiles}}})
			if e != nil {
				t.Fatal(e)
			}
			e = h.CheckInstallation(p)
			if (name == "exact") != (e == nil) {
				t.Fatalf("check result %s: %v", name, e)
			}
			if spy.calls != 0 {
				t.Fatal("startup validation read owner state")
			}
		})
	}
}

func TestReadinessPrivateProfileStartupBindsActualOwnerAndCompleteDigest(t *testing.T) {
	p := readinessInstallationFixture().Policy
	spy := &profileReadSpy{out: ProfileContract{ID: "profile-id", Key: "bounded", Digest: p.Profiles["bounded"]}}
	peers := map[string]map[string]ProfileContractGrant{Digest("peer"): {p.ID: {Binding: binding(p), Profiles: map[string]string{"bounded": "profile-id"}}}}
	h, e := NewProfileContractHandler(spy, peers)
	if e != nil {
		t.Fatal(e)
	}
	foreign := &profileReadSpy{out: spy.out}
	if h.QualifyOwnerContracts(context.Background(), p, foreign) == nil || spy.calls != 0 || foreign.calls != 0 {
		t.Fatal("foreign owner read/admitted")
	}
	if e = h.QualifyOwnerContracts(context.Background(), p, spy); e != nil || spy.calls != 1 {
		t.Fatal("exact owner refused", e)
	}
	spy.out.Digest = Digest("changed profile")
	if h.QualifyOwnerContracts(context.Background(), p, spy) == nil || spy.calls != 2 {
		t.Fatal("owner drift admitted")
	}
}
