package config

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"tunnel-manager/internal/manifest"
	"tunnel-manager/internal/testutil/mocks"
)

func newTestAccessClient(doer *mocks.FakeDoer) *cfAccessClient {
	return &cfAccessClient{
		doer:      doer,
		apiToken:  "tok",
		accountID: "acct1",
		baseURL:   "https://api.cloudflare.com/client/v4",
	}
}

// TestPublicBypassDomainScoped proves the host→domain builder only ever yields
// <host>/public and refuses anything carrying a path (the scope ceiling).
func TestPublicBypassDomainScoped(t *testing.T) {
	ok := map[string]string{
		"web-console.example.invalid": "web-console.example.invalid/public",
		"API.example.invalid":         "api.example.invalid/public",
	}
	for in, want := range ok {
		got, err := publicBypassDomain(in)
		if err != nil || got != want {
			t.Errorf("publicBypassDomain(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "  ", "host/public", "host/", "host?x=1", "host#f", "a b"} {
		if _, err := publicBypassDomain(bad); err == nil {
			t.Errorf("publicBypassDomain(%q) should have errored (host must be bare)", bad)
		}
	}
}

// TestAssertPublicBypassDomainRefusals is the hard guardrail: the create path
// must refuse empty/"/"/"/*"/non-/public domains outright.
func TestAssertPublicBypassDomainRefusals(t *testing.T) {
	good := []string{"web-console.example.invalid/public", "x.y.com/public"}
	for _, d := range good {
		if err := assertPublicBypassDomain(d); err != nil {
			t.Errorf("assertPublicBypassDomain(%q) unexpected err: %v", d, err)
		}
	}
	bad := []string{"", "/", "/*", "host", "host/admin", "host/public/secret", "host/publicx", "/public"}
	for _, d := range bad {
		if err := assertPublicBypassDomain(d); err == nil {
			t.Errorf("assertPublicBypassDomain(%q) should have refused", d)
		}
	}
}

// TestAssertBypassDecisionRefusals proves TM only ever writes a bypass decision.
func TestAssertBypassDecisionRefusals(t *testing.T) {
	if err := assertBypassDecision("bypass"); err != nil {
		t.Errorf("bypass should be allowed: %v", err)
	}
	for _, d := range []string{"allow", "block", "non_identity", "", "deny"} {
		if err := assertBypassDecision(d); err == nil {
			t.Errorf("decision %q should have been refused", d)
		}
	}
}

func TestEnsurePublicBypassCreatesWhenAbsent(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`))                                       // list: none ours
	doer.AddResponse(200, []byte(`{"success":true,"result":{"id":"app9","policies":[{"id":"pol9"}]}}`)) // create

	c := newTestAccessClient(doer)
	res, err := c.EnsurePublicBypass(context.Background(), "web-console.example.invalid")
	if err != nil {
		t.Fatalf("EnsurePublicBypass: %v", err)
	}
	if !res.Created || res.AppID != "app9" || res.PolicyID != "pol9" {
		t.Fatalf("got %+v, want Created=true AppID=app9 PolicyID=pol9", res)
	}
	if doer.Calls.Load() != 2 {
		t.Fatalf("expected 2 calls (list+create), got %d", doer.Calls.Load())
	}
	create := doer.Requests[1]
	if create.Method != http.MethodPost {
		t.Errorf("create method = %s, want POST", create.Method)
	}
	if !strings.HasSuffix(create.URL.Path, "/accounts/acct1/access/apps") {
		t.Errorf("create path = %s", create.URL.Path)
	}
	body, _ := io.ReadAll(create.Body)
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal create body: %v", err)
	}
	if payload["type"] != "self_hosted" {
		t.Errorf("type = %v, want self_hosted", payload["type"])
	}
	if payload["domain"] != "web-console.example.invalid/public" {
		t.Errorf("domain = %v, want <host>/public", payload["domain"])
	}
	policies, ok := payload["policies"].([]any)
	if !ok || len(policies) != 1 {
		t.Fatalf("expected exactly one inline policy, got %v", payload["policies"])
	}
	pol := policies[0].(map[string]any)
	if pol["decision"] != "bypass" {
		t.Errorf("policy decision = %v, want bypass", pol["decision"])
	}
	incl, ok := pol["include"].([]any)
	if !ok || len(incl) != 1 {
		t.Fatalf("expected one include rule, got %v", pol["include"])
	}
	if _, hasEveryone := incl[0].(map[string]any)["everyone"]; !hasEveryone {
		t.Errorf("include rule = %v, want everyone{}", incl[0])
	}
}

func TestEnsurePublicBypassIdempotentWhenOursExists(t *testing.T) {
	doer := &mocks.FakeDoer{}
	// An app TM owns (name carries the marker) at the exact <host>/public domain.
	doer.AddResponse(200, []byte(`{"success":true,"result":[
		{"id":"appX","name":"web-console.example.invalid/public `+accessAppMarker+`","domain":"web-console.example.invalid/public","policies":[{"id":"polX"}]}
	]}`))

	c := newTestAccessClient(doer)
	res, err := c.EnsurePublicBypass(context.Background(), "web-console.example.invalid")
	if err != nil {
		t.Fatalf("EnsurePublicBypass: %v", err)
	}
	if res.Created {
		t.Errorf("expected Created=false for pre-existing TM app")
	}
	if res.AppID != "appX" || res.PolicyID != "polX" {
		t.Errorf("got %+v, want AppID=appX PolicyID=polX", res)
	}
	if doer.Calls.Load() != 1 {
		t.Errorf("expected 1 call (list only, no create), got %d", doer.Calls.Load())
	}
}

// TestRemoveNeverClobbersForeignApp is the critical never-clobber guardrail: an
// Access app at the same <host>/public domain but WITHOUT the TM name marker
// (operator-created) is left completely untouched — no DELETE is issued.
func TestRemoveNeverClobbersForeignApp(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true,"result":[
		{"id":"foreign1","name":"operator's own public app","domain":"web-console.example.invalid/public","policies":[{"id":"p"}]}
	]}`))

	c := newTestAccessClient(doer)
	removed, err := c.RemovePublicBypass(context.Background(), "web-console.example.invalid")
	if err != nil {
		t.Fatalf("RemovePublicBypass: %v", err)
	}
	if removed {
		t.Errorf("expected removed=false — a foreign app must never be deleted")
	}
	if doer.Calls.Load() != 1 {
		t.Errorf("expected 1 call (list only, no DELETE), got %d", doer.Calls.Load())
	}
}

func TestRemovePublicBypassDeletesOurApp(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true,"result":[
		{"id":"appDel","name":"web-console.example.invalid/public `+accessAppMarker+`","domain":"web-console.example.invalid/public","policies":[{"id":"polDel"}]}
	]}`))
	doer.AddResponse(200, []byte(`{"success":true}`)) // delete

	c := newTestAccessClient(doer)
	removed, err := c.RemovePublicBypass(context.Background(), "web-console.example.invalid")
	if err != nil {
		t.Fatalf("RemovePublicBypass: %v", err)
	}
	if !removed {
		t.Errorf("expected removed=true")
	}
	del := doer.Requests[1]
	if del.Method != http.MethodDelete {
		t.Errorf("delete method = %s, want DELETE", del.Method)
	}
	if !strings.HasSuffix(del.URL.Path, "/accounts/acct1/access/apps/appDel") {
		t.Errorf("delete path = %s", del.URL.Path)
	}
}

func TestRemovePublicBypassIdempotentWhenAbsent(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true,"result":[]}`)) // none

	c := newTestAccessClient(doer)
	removed, err := c.RemovePublicBypass(context.Background(), "gone.example.invalid")
	if err != nil {
		t.Fatalf("RemovePublicBypass: %v", err)
	}
	if removed {
		t.Errorf("expected removed=false")
	}
	if doer.Calls.Load() != 1 {
		t.Errorf("expected 1 call (list only, no DELETE), got %d", doer.Calls.Load())
	}
}

func TestEffectiveExposure(t *testing.T) {
	cases := []struct {
		override manifest.PublicExposure
		global   bool
		want     bool
	}{
		{"", false, false}, // inherit + global off
		{"", true, true},   // inherit + global on
		{manifest.PublicExposureInherit, true, true},     // explicit inherit + global on
		{manifest.PublicExposureEnabled, false, true},    // enabled wins over global off
		{manifest.PublicExposureDisabled, true, false},   // disabled wins over global on
		{manifest.PublicExposure("garbage"), true, true}, // unknown normalizes to inherit
	}
	for _, c := range cases {
		if got := effectiveExposure(c.override, c.global); got != c.want {
			t.Errorf("effectiveExposure(%q, %v) = %v, want %v", c.override, c.global, got, c.want)
		}
	}
}

func TestPlanAccessMatrix(t *testing.T) {
	desired := []DesiredEntry{
		{Hostname: "a.example.invalid", PublicExposure: manifest.PublicExposureInherit},
		{Hostname: "b.example.invalid", PublicExposure: manifest.PublicExposureEnabled},
		{Hostname: "c.example.invalid", PublicExposure: manifest.PublicExposureDisabled},
		{Hostname: "d.example.invalid", PublicExposure: manifest.PublicExposureInherit},
	}

	// Global OFF: only b (explicitly enabled) is ensured. Ledger holds c, d, and
	// an orphan whose route is gone — none are ensured, so all three are removed.
	off := planAccess(desired, nil, nil,
		[]string{"c.example.invalid", "d.example.invalid", "orphan.example.invalid"}, false)
	if got := strings.Join(off.Ensure, ","); got != "b.example.invalid" {
		t.Errorf("global-off ensure = %q, want only b", got)
	}
	if got := strings.Join(off.Remove, ","); got != "c.example.invalid,d.example.invalid,orphan.example.invalid" {
		t.Errorf("global-off remove = %q, want c,d,orphan", got)
	}

	// Global ON: a, b, d ensured (inherit->on, enabled); c stays off (disabled
	// wins). Empty ledger -> nothing to remove.
	on := planAccess(desired, nil, nil, nil, true)
	if got := strings.Join(on.Ensure, ","); got != "a.example.invalid,b.example.invalid,d.example.invalid" {
		t.Errorf("global-on ensure = %q, want a,b,d", got)
	}
	if len(on.Remove) != 0 {
		t.Errorf("global-on remove = %v, want empty", on.Remove)
	}

	// Ignored + pruned hosts are never ensured even when global on / enabled.
	ig := planAccess(desired,
		map[string]bool{"a.example.invalid": true},
		map[string]bool{"b.example.invalid": true},
		nil, true)
	if got := strings.Join(ig.Ensure, ","); got != "d.example.invalid" {
		t.Errorf("ignore+prune ensure = %q, want only d", got)
	}
}

func TestNewCFAccessClientNilWithoutCreds(t *testing.T) {
	if NewCFAccessClient(&mocks.FakeDoer{}, CFConfig{APIToken: "x"}) != nil {
		t.Error("expected nil Access client when account id absent")
	}
	if NewCFAccessClient(&mocks.FakeDoer{}, CFConfig{AccountID: "a"}) != nil {
		t.Error("expected nil Access client when token absent")
	}
	if NewCFAccessClient(&mocks.FakeDoer{}, CFConfig{APIToken: "x", AccountID: "a"}) == nil {
		t.Error("expected a client when token+account present")
	}
}

func TestResolveRuntimeBindingReadsPrimaryAppAndOrganization(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true,"result":[
		{"id":"public","domain":"web-console.example.invalid/public","aud":"ignored"},
		{"id":"primary","domain":"web-console.example.invalid","aud":"aud-primary"}
	]}`))
	doer.AddResponse(200, []byte(`{"success":true,"result":{"auth_domain":"team.example.com"}}`))

	binding, err := newTestAccessClient(doer).ResolveRuntimeBinding(context.Background(), "web-console.example.invalid")
	if err != nil {
		t.Fatalf("ResolveRuntimeBinding: %v", err)
	}
	if binding.TeamDomain != "https://team.example.com" || binding.Audience != "aud-primary" || binding.RecoveryURL != binding.TeamDomain {
		t.Fatalf("binding = %+v", binding)
	}
	if doer.Calls.Load() != 2 {
		t.Fatalf("expected apps + organization reads, got %d calls", doer.Calls.Load())
	}
	for _, req := range doer.Requests {
		if req.Method != http.MethodGet {
			t.Fatalf("metadata discovery issued %s, want GET", req.Method)
		}
	}
}

func TestResolveRuntimeBindingUsesCentralTeamDomainWhenOrganizationReadIsForbidden(t *testing.T) {
	t.Setenv("VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN", "https://central.example.com")
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"domain":"web.example.invalid/*","aud":"aud-primary"}]}`))
	doer.AddResponse(403, []byte(`{"success":false,"errors":[{"code":10000,"message":"Authentication error"}]}`))

	binding, err := newTestAccessClient(doer).ResolveRuntimeBinding(context.Background(), "web.example.invalid")
	if err != nil {
		t.Fatalf("ResolveRuntimeBinding: %v", err)
	}
	if binding.TeamDomain != "https://central.example.com" || binding.Audience != "aud-primary" {
		t.Fatalf("binding = %+v", binding)
	}
}

func TestResolveRuntimeBindingExplainsForbiddenOrganizationMetadata(t *testing.T) {
	t.Setenv("VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN", "")
	doer := &mocks.FakeDoer{}
	doer.AddResponse(200, []byte(`{"success":true,"result":[{"domain":"web.example.invalid/*","aud":"aud-primary"}]}`))
	doer.AddResponse(403, []byte(`{"success":false,"errors":[{"code":10000,"message":"Authentication error"}]}`))

	_, err := newTestAccessClient(doer).ResolveRuntimeBinding(context.Background(), "web.example.invalid")
	if err == nil || !strings.Contains(err.Error(), "rejected (HTTP 403)") || !strings.Contains(err.Error(), "VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN") {
		t.Fatalf("error = %v, want a safe forbidden-metadata explanation", err)
	}
}

func TestResolveRuntimeBindingFailsForAmbiguousOrIncompletePrimaryApp(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"ambiguous", `{"success":true,"result":[{"domain":"web.example.invalid","aud":"a"},{"domain":"web.example.invalid","aud":"b"}]}`, "multiple primary"},
		{"missing audience", `{"success":true,"result":[{"domain":"web.example.invalid"}]}`, "no audience"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doer := &mocks.FakeDoer{}
			doer.AddResponse(200, []byte(tc.body))
			_, err := newTestAccessClient(doer).ResolveRuntimeBinding(context.Background(), "web.example.invalid")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
			if doer.Calls.Load() != 1 {
				t.Fatalf("incomplete primary metadata should not read organization; calls=%d", doer.Calls.Load())
			}
		})
	}
}

func TestAccessPrimaryDomainRankPrefersSpecificCoverage(t *testing.T) {
	cases := []struct {
		domain string
		rank   int
	}{
		{"web.example.invalid", 3},
		{"web.example.invalid/*", 2},
		{"*.example.invalid", 1},
		{"web.example.invalid/public", 0},
		{"other.example.invalid/*", 0},
	}
	for _, tc := range cases {
		if got := accessPrimaryDomainRank(tc.domain, "web.example.invalid"); got != tc.rank {
			t.Errorf("accessPrimaryDomainRank(%q) = %d, want %d", tc.domain, got, tc.rank)
		}
	}
}
