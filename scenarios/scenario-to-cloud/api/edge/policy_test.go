package edge

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var typed *apierrors.Error
	if !errors.As(err, &typed) {
		t.Fatalf("error %v is not typed", err)
	}
	return typed.Code
}

func declaredInputs() PolicyInputs {
	return PolicyInputs{
		DeploymentID: "dep-1", ScenarioID: "app", Environment: "staging", Domain: "App.Example.Test",
		Listeners: []domain.ClosureListener{
			{ID: "app/ui", Owner: "scenario:app", PortName: "ui", Visibility: domain.EdgeVisibilityPublicViaEdge},
			{ID: "app/api", Owner: "scenario:app", PortName: "api", Visibility: domain.EdgeVisibilityPrivate},
			{ID: "app/metrics", Owner: "scenario:app", PortName: "metrics", Visibility: ""},
			{ID: "store/server", Owner: "resource:store", PortName: "server", Visibility: domain.EdgeVisibilityPublicViaEdge},
			{ID: "postgres/server", Owner: "resource:postgres", PortName: "server", Visibility: ""},
		},
		Ports:      domain.ManifestPorts{"ui": 3000, "api": 3001, "metrics": 9100, "debug": 6060, "server": 5432},
		TargetHost: "203.0.113.10",
		ACMEEmail:  "ops@example.test",
	}
}

// [REQ:STC-P0-036] P14-O01/P14-O04: only the scenario's own public_via_edge
// listener is routed; database, metrics, management, resource-owned and
// undeclared ports are private with a stated reason.
func TestDeriveRoutesOnlyDeclaredPublicScenarioListeners(t *testing.T) {
	spec, err := Derive(declaredInputs())
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if spec.Domain != "app.example.test" || len(spec.Routes) != 1 || spec.Routes[0] != (domain.EdgeRoute{Host: "app.example.test", UpstreamPort: 3000, ListenerID: "app/ui"}) {
		t.Fatalf("routes = %+v (domain %s)", spec.Routes, spec.Domain)
	}
	reasons := map[string]string{}
	for _, l := range spec.PrivateListeners {
		reasons[l.ID] = l.Reason
	}
	want := map[string]string{
		"app/api": ReasonDeclaredPrivate, "app/metrics": ReasonMetrics, "app/debug": ReasonUndeclared,
		"store/server": ReasonResourceOwned, "postgres/server": ReasonDatabase,
		"management/ssh": ReasonManagement, "management/bridge": ReasonManagement,
	}
	for id, reason := range want {
		if reasons[id] != reason {
			t.Fatalf("private listener %s reason = %q want %q (all: %v)", id, reasons[id], reason, reasons)
		}
	}
	for _, l := range spec.PrivateListeners {
		for _, r := range spec.Routes {
			if l.Port != 0 && l.Port == r.UpstreamPort {
				t.Fatalf("private listener %s shares port with a route", l.ID)
			}
		}
	}
	if strings.Join(intsToStrings(spec.FirewallAllow), ",") != "80,443" {
		t.Fatalf("firewall allow = %v", spec.FirewallAllow)
	}
	if len(spec.IPPolicy.IPv4) != 1 || spec.IPPolicy.IPv4[0] != "203.0.113.10" || len(spec.IPPolicy.IPv6) != 0 {
		t.Fatalf("ip policy = %+v", spec.IPPolicy)
	}
	if spec.ACMEEnvironment != domain.ACMEEnvironmentStaging {
		t.Fatalf("non-production environment must default to staging ACME, got %s", spec.ACMEEnvironment)
	}
	if !strings.HasPrefix(spec.Digest, "sha256:") {
		t.Fatalf("digest = %q", spec.Digest)
	}
	again, _ := Derive(declaredInputs())
	if again.Digest != spec.Digest {
		t.Fatal("digest is not deterministic")
	}
	changed := declaredInputs()
	changed.Ports["ui"] = 3100
	if other, _ := Derive(changed); other.Digest == spec.Digest {
		t.Fatal("a different upstream must change the digest")
	}
}

func intsToStrings(in []int) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, strconv.Itoa(v))
	}
	return out
}

// A legacy deployment without declarations exposes only its ui port, so the
// previous whole-file behaviour is preserved and nothing extra is opened.
func TestDeriveLegacyManifestExposesOnlyUI(t *testing.T) {
	spec, err := Derive(PolicyInputs{DeploymentID: "dep", ScenarioID: "legacy", Environment: "production", Domain: "legacy.example.test", Ports: domain.ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002}, TargetHost: "vps.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Routes) != 1 || spec.Routes[0].UpstreamPort != 3000 {
		t.Fatalf("routes = %+v", spec.Routes)
	}
	ids := map[string]bool{}
	for _, l := range spec.PrivateListeners {
		ids[l.ID] = true
	}
	if !ids["legacy/api"] || !ids["legacy/ws"] || !ids["management/ssh"] || !ids["management/bridge"] {
		t.Fatalf("private listeners = %+v", spec.PrivateListeners)
	}
	if spec.ACMEEnvironment != domain.ACMEEnvironmentProduction {
		t.Fatalf("the operator's production environment is its own ACME authority, got %s", spec.ACMEEnvironment)
	}
	if len(spec.IPPolicy.IPv4) != 0 {
		t.Fatalf("a name locator must not seed the IP policy: %+v", spec.IPPolicy)
	}
}

func TestDeriveRefusesUnroutableDeclarations(t *testing.T) {
	noPublic := declaredInputs()
	noPublic.Listeners = noPublic.Listeners[1:]
	if _, err := Derive(noPublic); codeOf(t, err) != CodeNoPublicListener {
		t.Fatalf("no public listener: %v", err)
	}
	noPort := declaredInputs()
	delete(noPort.Ports, "ui")
	if _, err := Derive(noPort); codeOf(t, err) != CodeListenerPortUnknown {
		t.Fatalf("missing port: %v", err)
	}
	management := declaredInputs()
	management.Ports["ui"] = ManagementBridgePort
	if _, err := Derive(management); codeOf(t, err) != CodeListenerPortUnknown {
		t.Fatalf("management port routed: %v", err)
	}
	for _, bad := range []string{"", "https://app.example.test", "app.example.test/path", "localhost", "203.0.113.10", "*.example.test", "app.example.test:443", "-bad.example.test", "app.example.123"} {
		in := declaredInputs()
		in.Domain = bad
		if _, err := Derive(in); codeOf(t, err) != CodeDomainInvalid {
			t.Fatalf("domain %q accepted: %v", bad, err)
		}
	}
}

// [REQ:STC-P0-036] Step 6/7: staging is the default for repeated tests;
// production issuance outside the operator's production environment needs
// the EXT-04 authority and is refused otherwise.
func TestSelectACMEStagingByDefaultProductionOnlyWithAuthority(t *testing.T) {
	cases := []struct {
		name string
		in   ACMEInputs
		want string
		code string
	}{
		{"staging env default", ACMEInputs{Environment: "staging"}, domain.ACMEEnvironmentStaging, ""},
		{"production env default", ACMEInputs{Environment: "production"}, domain.ACMEEnvironmentProduction, ""},
		{"explicit staging", ACMEInputs{Requested: "staging", Environment: "production"}, domain.ACMEEnvironmentStaging, ""},
		{"production requested without authority", ACMEInputs{Requested: "production", Environment: "fixture"}, "", CodeACMEProductionUnauthorized},
		{"production requested allowed without budget", ACMEInputs{Requested: "production", Environment: "fixture", Authority: ACMEAuthority{ProductionAllowed: true}}, "", CodeACMEProductionUnauthorized},
		{"production requested with authority", ACMEInputs{Requested: "production", Environment: "fixture", Authority: ACMEAuthority{ProductionAllowed: true, Budget: 2}}, domain.ACMEEnvironmentProduction, ""},
		{"unknown value", ACMEInputs{Requested: "prod"}, "", CodeACMEEnvironmentInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SelectACME(tc.in)
			if tc.code != "" {
				if codeOf(t, err) != tc.code {
					t.Fatalf("code = %v", err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %q err=%v", got, err)
			}
		})
	}
	authority := ACMEAuthorityFromEnvironment(func(key string) (string, bool) {
		return map[string]string{"VROOLI_ACME_PRODUCTION_ALLOWED": "true", "VROOLI_ACME_PRODUCTION_BUDGET": "3"}[key], true
	})
	if !authority.ProductionAllowed || authority.Budget != 3 {
		t.Fatalf("authority = %+v", authority)
	}
}

// The rendered snippet is owner-scoped: site blocks for the routes only,
// site-level ACME directory, provider token by environment placeholder.
func TestRenderSnippetNeverInlinesSecretsOrGlobalOptions(t *testing.T) {
	in := declaredInputs()
	in.DNSProvider = &domain.EdgeDNSProvider{Provider: "cloudflare", Descriptor: domain.CredentialDescriptor{LogicalID: "vrooli/app", Field: "cloudflare-api-token"}, EnvVar: "CLOUDFLARE_API_TOKEN"}
	spec, err := Derive(in)
	if err != nil {
		t.Fatal(err)
	}
	snippet := RenderSnippet(spec)
	for _, want := range []string{"app.example.test {", "tls ops@example.test {", "ca " + ACMEStagingCA, "dns cloudflare {env.CLOUDFLARE_API_TOKEN}", "reverse_proxy 127.0.0.1:3000"} {
		if !strings.Contains(snippet, want) {
			t.Fatalf("snippet lacks %q:\n%s", want, snippet)
		}
	}
	for _, forbid := range []string{"\n{\n", "import ", "acme_ca", "5432", "9100", "18767"} {
		if strings.Contains(snippet, forbid) {
			t.Fatalf("snippet contains %q:\n%s", forbid, snippet)
		}
	}
	raw, err := TargetSpecJSON(spec)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "canary") || !strings.Contains(string(raw), `"schema_version":1`) {
		t.Fatalf("target spec = %s", raw)
	}
	if SnippetPath("dep-1") != "/etc/caddy/conf.d/vrooli-dep-1.caddy" {
		t.Fatalf("snippet path = %s", SnippetPath("dep-1"))
	}
}
