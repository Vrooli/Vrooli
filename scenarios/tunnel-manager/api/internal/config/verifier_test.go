package config

import (
	"context"
	"testing"

	"tunnel-manager/internal/testutil/mocks"
)

// newTestVerifier builds a cfVerifier bound to a FakeDoer and the production
// base URL (the FakeDoer ignores the URL and serves responses in call order).
func newTestVerifier(doer *mocks.FakeDoer) *cfVerifier {
	return &cfVerifier{doer: doer, baseURL: "https://api.cloudflare.com/client/v4"}
}

func checkByName(checks []CredentialCheck, name string) (CredentialCheck, bool) {
	for _, c := range checks {
		if c.Name == name {
			return c, true
		}
	}
	return CredentialCheck{}, false
}

func TestVerifyHappyPathAllOK(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true}`))                           // token verify
	doer.AddResponse(200, []byte(`{"success":true}`))                           // account
	doer.AddResponse(200, []byte(`{"success":true}`))                           // tunnel
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"zone1"}]}`)) // zone lookup
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`))               // dns records

	v := newTestVerifier(doer)
	got, err := v.Verify(context.Background(), CFConfig{APIToken: "t", AccountID: "a", TunnelID: "tun"}, []string{"itsagitime.com"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !got.Ready {
		t.Fatalf("expected Ready=true, checks=%+v", got.Checks)
	}
	for _, c := range got.Checks {
		if c.State != CheckOK {
			t.Errorf("check %s = %s, want ok", c.Name, c.State)
		}
	}
	if len(got.Checks) != 5 {
		t.Errorf("expected 5 checks, got %d", len(got.Checks))
	}
}

func TestVerifyMissingTokenSkipsNetwork(t *testing.T) {
	doer := &mocks.FakeDoer{} // no responses queued: any HTTP call would error
	v := newTestVerifier(doer)
	got, err := v.Verify(context.Background(), CFConfig{AccountID: "a", TunnelID: "tun"}, []string{"itsagitime.com"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.Ready {
		t.Fatalf("expected Ready=false")
	}
	if doer.Calls.Load() != 0 {
		t.Fatalf("expected zero HTTP calls when token absent, got %d", doer.Calls.Load())
	}
	tok, _ := checkByName(got.Checks, CheckNameToken)
	if tok.State != CheckMissing {
		t.Errorf("token state = %s, want missing", tok.State)
	}
	acct, _ := checkByName(got.Checks, CheckNameAccount)
	if acct.State != CheckUnspecified {
		t.Errorf("account state = %s, want unspecified (skipped)", acct.State)
	}
}

func TestVerifyExpiredTokenInvalid(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(401, []byte(`{"success":false}`)) // token verify rejected
	v := newTestVerifier(doer)
	got, _ := v.Verify(context.Background(), CFConfig{APIToken: "bad", AccountID: "a", TunnelID: "tun"}, []string{"itsagitime.com"})
	tok, _ := checkByName(got.Checks, CheckNameToken)
	if tok.State != CheckInvalid {
		t.Errorf("token state = %s, want invalid", tok.State)
	}
	// Downstream checks skip without making calls (only 1 call made).
	if doer.Calls.Load() != 1 {
		t.Errorf("expected 1 HTTP call (token only), got %d", doer.Calls.Load())
	}
}

func TestVerifyDNSScopeInsufficient(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true}`))                           // token
	doer.AddResponse(200, []byte(`{"success":true}`))                           // account
	doer.AddResponse(200, []byte(`{"success":true}`))                           // tunnel
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"zone1"}]}`)) // zone lookup ok
	doer.AddResponse(403, []byte(`{"success":false}`))                          // dns records forbidden

	v := newTestVerifier(doer)
	got, _ := v.Verify(context.Background(), CFConfig{APIToken: "t", AccountID: "a", TunnelID: "tun"}, []string{"itsagitime.com"})
	if got.Ready {
		t.Fatalf("expected Ready=false on missing DNS scope")
	}
	dns, ok := checkByName(got.Checks, CheckNameDNSScope)
	if !ok || dns.State != CheckInsufficientScope {
		t.Errorf("dns scope state = %s, want insufficient_scope", dns.State)
	}
	if dns.Remediation == "" {
		t.Error("expected a remediation for insufficient DNS scope")
	}
}

func TestVerifyZoneNotFoundInvalid(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true}`))             // token
	doer.AddResponse(200, []byte(`{"success":true}`))             // account
	doer.AddResponse(200, []byte(`{"success":true}`))             // tunnel
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`)) // zone lookup: empty

	v := newTestVerifier(doer)
	got, _ := v.Verify(context.Background(), CFConfig{APIToken: "t", AccountID: "a", TunnelID: "tun"}, []string{"unknown.example"})
	lookup, _ := checkByName(got.Checks, CheckNameZoneLookup)
	if lookup.State != CheckInvalid {
		t.Errorf("zone lookup state = %s, want invalid", lookup.State)
	}
	dns, _ := checkByName(got.Checks, CheckNameDNSScope)
	if dns.State != CheckUnspecified {
		t.Errorf("dns state = %s, want unspecified (zone unresolved)", dns.State)
	}
}

func TestVerifyTunnelInsufficientScope(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true}`))  // token
	doer.AddResponse(200, []byte(`{"success":true}`))  // account
	doer.AddResponse(403, []byte(`{"success":false}`)) // tunnel forbidden
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"z"}]}`))
	doer.AddResponse(200, []byte(`{"success":true}`))

	v := newTestVerifier(doer)
	got, _ := v.Verify(context.Background(), CFConfig{APIToken: "t", AccountID: "a", TunnelID: "tun"}, []string{"itsagitime.com"})
	tun, _ := checkByName(got.Checks, CheckNameTunnel)
	if tun.State != CheckInsufficientScope {
		t.Errorf("tunnel state = %s, want insufficient_scope", tun.State)
	}
}
