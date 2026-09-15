package config

import (
	"context"
	"strings"
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
	doer.AddResponse(200, []byte(`{"success":true}`))                                                 // token verify
	doer.AddResponse(200, []byte(`{"success":true}`))                                                 // account
	doer.AddResponse(200, []byte(`{"success":true}`))                                                 // tunnel
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"zone1"}]}`))                       // zone lookup
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`))                                     // dns records
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`))                                     // access apps (scope probe)
	doer.AddResponse(200, []byte(`{"success":true,"result":{"auth_domain":"team.example.invalid"}}`)) // access organization

	v := newTestVerifier(doer)
	// accessRequired=true: the capability is on, so the Access-scope check
	// counts toward Ready and must be OK like the rest.
	got, err := v.Verify(context.Background(), CFConfig{APIToken: "t", AccountID: "a", TunnelID: "tun"}, []string{"example.invalid"}, true)
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
	if len(got.Checks) != 7 {
		t.Errorf("expected 7 checks, got %d", len(got.Checks))
	}
}

func TestVerifyMissingTokenSkipsNetwork(t *testing.T) {
	doer := &mocks.FakeDoer{} // no responses queued: any HTTP call would error
	v := newTestVerifier(doer)
	got, err := v.Verify(context.Background(), CFConfig{AccountID: "a", TunnelID: "tun"}, []string{"example.invalid"}, false)
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
	got, _ := v.Verify(context.Background(), CFConfig{APIToken: "bad", AccountID: "a", TunnelID: "tun"}, []string{"example.invalid"}, false)
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
	doer.AddResponse(200, []byte(`{"success":true}`))                                                 // token
	doer.AddResponse(200, []byte(`{"success":true}`))                                                 // account
	doer.AddResponse(200, []byte(`{"success":true}`))                                                 // tunnel
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"zone1"}]}`))                       // zone lookup ok
	doer.AddResponse(403, []byte(`{"success":false}`))                                                // dns records forbidden
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`))                                     // access apps (scope probe)
	doer.AddResponse(200, []byte(`{"success":true,"result":{"auth_domain":"team.example.invalid"}}`)) // access organization

	v := newTestVerifier(doer)
	got, _ := v.Verify(context.Background(), CFConfig{APIToken: "t", AccountID: "a", TunnelID: "tun"}, []string{"example.invalid"}, false)
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
	doer.AddResponse(200, []byte(`{"success":true}`))                                                 // token
	doer.AddResponse(200, []byte(`{"success":true}`))                                                 // account
	doer.AddResponse(200, []byte(`{"success":true}`))                                                 // tunnel
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`))                                     // zone lookup: empty
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`))                                     // access apps (scope probe)
	doer.AddResponse(200, []byte(`{"success":true,"result":{"auth_domain":"team.example.invalid"}}`)) // access organization

	v := newTestVerifier(doer)
	got, _ := v.Verify(context.Background(), CFConfig{APIToken: "t", AccountID: "a", TunnelID: "tun"}, []string{"unknown.example"}, false)
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
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`))                                     // access apps (scope probe)
	doer.AddResponse(200, []byte(`{"success":true,"result":{"auth_domain":"team.example.invalid"}}`)) // access organization

	v := newTestVerifier(doer)
	got, _ := v.Verify(context.Background(), CFConfig{APIToken: "t", AccountID: "a", TunnelID: "tun"}, []string{"example.invalid"}, false)
	tun, _ := checkByName(got.Checks, CheckNameTunnel)
	if tun.State != CheckInsufficientScope {
		t.Errorf("tunnel state = %s, want insufficient_scope", tun.State)
	}
}

func TestVerifyAccessOrganizationInsufficientScopeIsActionable(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true}`))                       // token
	doer.AddResponse(200, []byte(`{"success":true}`))                       // account
	doer.AddResponse(200, []byte(`{"success":true}`))                       // tunnel
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"id":"z"}]}`)) // zone lookup
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`))           // dns records
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`))           // access apps
	doer.AddResponse(403, []byte(`{"success":false}`))                      // access organization

	got, err := newTestVerifier(doer).Verify(context.Background(), CFConfig{APIToken: "t", AccountID: "a", TunnelID: "tun"}, []string{"example.invalid"}, false)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	org, ok := checkByName(got.Checks, CheckNameAccessOrganization)
	if !ok || org.State != CheckInsufficientScope {
		t.Fatalf("organization check = %+v, want insufficient_scope", org)
	}
	if !strings.Contains(org.Detail, "HTTP 403") || !strings.Contains(org.Remediation, "VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN") {
		t.Fatalf("organization check lacks actionable safe detail: %+v", org)
	}
	if !got.Ready {
		t.Fatal("core remote readiness should remain ready when only optional gated-auth metadata fails")
	}
	if got.AllChecksOK {
		t.Fatal("all checks must be false when organization metadata is forbidden")
	}
	capability, ok := capabilityByName(got.Capabilities, CapabilityGatedUIAuthentication)
	if !ok || capability.Ready || capability.Required {
		t.Fatalf("access capability = %+v, want not-ready optional capability", capability)
	}
}

func capabilityByName(capabilities []CredentialCapability, name string) (CredentialCapability, bool) {
	for _, capability := range capabilities {
		if capability.Name == name {
			return capability, true
		}
	}
	return CredentialCapability{}, false
}
