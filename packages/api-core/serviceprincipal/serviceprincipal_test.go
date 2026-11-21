package serviceprincipal

import (
	"strings"
	"testing"
)

func TestParseProfilesRefusesWideOrAmbiguousProfiles(t *testing.T) {
	if profiles, err := ParseProfiles(""); err != nil || profiles != nil {
		t.Fatal("empty configuration must configure nothing")
	}
	for _, raw := range []string{
		`not json`,
		`[{"name":"agent-manager","resource":"web-console","scopes":["*"]}]`,
		`[{"name":"agent-manager","resource":"web-console","scopes":["web-console:*"]}]`,
		`[{"name":"agent-manager","resource":"","scopes":[]}]`,
		`[{"name":"Agent Manager","resource":"web-console","scopes":[]}]`,
		`[{"name":"agent-manager","resource":"web-console"},{"name":"agent-manager","resource":"other"}]`,
		`[{"name":"agent-manager","resource":"web-console","issuers":[" "]}]`,
	} {
		if _, err := ParseProfiles(raw); err == nil {
			t.Fatalf("accepted invalid profile configuration %q", raw)
		}
	}
}

func TestCredentialIdentityIsTheAddressTheAuthenticatorWrites(t *testing.T) {
	if got := CredentialIdentity(" Research-Desk ", " Web-Console "); got != "scenario-authenticator/service-principal/research-desk/web-console" {
		t.Fatalf("identity = %q", got)
	}
}

func TestParseDeclarationsAcceptsAPrincipalAndRefusesWhatItMayNotAskFor(t *testing.T) {
	good := `{"principals":[{"name":"research-desk","resource":"web-console","audience":"scenario-authenticator:web-console","scopes":["web-console:read","web-console:write"]}]}`
	got, err := ParseDeclarations([]byte(good))
	if err != nil || len(got) != 1 || got[0].Name != "research-desk" || len(got[0].Scopes) != 2 {
		t.Fatalf("declarations = %+v, %v", got, err)
	}
	for name, raw := range map[string]string{
		"issuers are operator authority": `{"principals":[{"name":"a-b","resource":"r","audience":"scenario-authenticator:r","scopes":["r:x"],"issuers":["me"]}]}`,
		"unknown top level":              `{"principals":[],"extra":1}`,
		"bad name":                       `{"principals":[{"name":"A","resource":"r","audience":"scenario-authenticator:r","scopes":["r:x"]}]}`,
		"reserved name":                  `{"principals":[{"name":"ai-gateway","resource":"r","audience":"scenario-authenticator:r","scopes":["r:x"]}]}`,
		"duplicate name":                 `{"principals":[{"name":"a-b","resource":"r","audience":"scenario-authenticator:r","scopes":["r:x"]},{"name":"a-b","resource":"s","audience":"scenario-authenticator:s","scopes":["s:x"]}]}`,
		"no resource":                    `{"principals":[{"name":"a-b","audience":"scenario-authenticator:r","scopes":["r:x"]}]}`,
		"foreign audience":               `{"principals":[{"name":"a-b","resource":"r","audience":"someone-else:r","scopes":["r:x"]}]}`,
		"bare audience prefix":           `{"principals":[{"name":"a-b","resource":"r","audience":"scenario-authenticator:","scopes":["r:x"]}]}`,
		"no scopes":                      `{"principals":[{"name":"a-b","resource":"r","audience":"scenario-authenticator:r","scopes":[]}]}`,
		"wildcard scope":                 `{"principals":[{"name":"a-b","resource":"r","audience":"scenario-authenticator:r","scopes":["r:*"]}]}`,
		"not json":                       `nope`,
	} {
		if _, err := ParseDeclarations([]byte(raw)); err == nil {
			t.Errorf("%s: accepted %s", name, raw)
		} else if strings.TrimSpace(err.Error()) == "" {
			t.Errorf("%s: empty error", name)
		}
	}
}
