package resolver

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestConfigureRequestUsesInjectedEnvironmentAndFlagOverrides(t *testing.T) {
	env := map[string]string{
		"ADGUARD_HOME_BASE_URL":       "http://adguard.local",
		"ADGUARD_HOME_URL":            "http://fallback.local",
		"ADGUARD_HOME_USERNAME":       "service-user",
		"ADGUARD_HOME_CREDENTIAL_REF": "vrooli/adguard-home",
	}
	h := handlers{Getenv: func(key string) string { return env[key] }}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "base-url"}, {Name: "username"}, {Name: "credential-ref"}, {Name: "dry-run", Bool: true},
		}},
		Flags:     map[string]string{"username": "operator"},
		BoolFlags: map[string]bool{"dry-run": true},
	})

	got := h.configureRequest(ctx)
	if got.GetBaseUrl() != "http://adguard.local" || got.GetUsername() != "operator" || got.GetCredentialRef() != "vrooli/adguard-home" || !got.GetDryRun() {
		t.Fatalf("configure request = %+v", got)
	}
}
