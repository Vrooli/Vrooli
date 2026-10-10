package effortauthority

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type finiteAcceptanceOwnerFixture struct {
	calls  int
	result AcceptanceContract
	fail   bool
}

func (o *finiteAcceptanceOwnerFixture) ReadFiniteAcceptance(context.Context, Policy) (AcceptanceContract, error) {
	o.calls++
	if o.fail {
		return AcceptanceContract{}, ErrRefused
	}
	return o.result, nil
}
func TestFiniteAcceptancePrivateReadExactFullSubjectAndRevocation(t *testing.T) {
	_, _, _, p, _, _ := authorityFixture(t)
	peer := &x509.Certificate{Raw: []byte("fixture enrolled certificate")}
	sum := sha256.Sum256(peer.Raw)
	pin := hex.EncodeToString(sum[:])
	owner := &finiteAcceptanceOwnerFixture{result: AcceptanceContract{Subject: CommissionSubjectFor(p), Actor: p.Owner, AcceptedAt: time.Now().UTC(), Generation: 1}}
	grants := map[string]map[string]AcceptanceContractGrant{pin: {p.ID: {Policy: p}}}
	handler, e := NewAcceptanceContractHandler(owner, grants)
	if e != nil {
		t.Fatal(e)
	}
	delete(grants, pin) // owner startup copied the grant table.
	body := `{"binding":{"policyId":"` + p.ID + `","policyDigest":"` + Digest(p) + `","epoch":1,"owner":"` + p.Owner + `","client":"` + p.Client + `","deadline":"` + p.Deadline.Format(time.RFC3339Nano) + `"}}`
	makeReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, AcceptanceContractEndpoint, strings.NewReader(body))
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{peer}}, PeerCertificates: []*x509.Certificate{peer}}
		return r
	}
	positive := httptest.NewRecorder()
	handler.ServeHTTP(positive, makeReq())
	if positive.Code != 200 {
		t.Fatalf("exact fixture read refused: %d %s", positive.Code, positive.Body.String())
	}
	for _, name := range []string{"no-peer", "wrong-channel", "revoked", "foreign-team", "profile-drift", "unavailable"} {
		t.Run(name, func(t *testing.T) {
			before := owner.calls
			original := owner.result
			owner.fail = false
			r := makeReq()
			switch name {
			case "no-peer":
				r.TLS = nil
			case "wrong-channel":
				r.Header.Set("Authorization", "Bearer must-not-substitute")
			case "revoked":
				owner.result.Revoked = true
			case "foreign-team":
				owner.result.Subject.Team = "other"
			case "profile-drift":
				owner.result.Subject.Profiles = map[string]string{"bounded": Digest("different")}
			case "unavailable":
				owner.fail = true
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != 403 {
				t.Fatalf("refusal %s: %d", name, w.Code)
			}
			if (name == "no-peer" || name == "wrong-channel") && owner.calls != before {
				t.Fatal("identity/channel rejection reached owner")
			}
			owner.result = original
			owner.fail = false
		})
	}
}
