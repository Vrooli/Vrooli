package env

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/resources/securestore"
	"github.com/vrooli/vrooli/internal/scenario"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
)

// The degradation contract under test: credential resolution returns an error
// only for a manifest defect. Every host condition and every unset value is
// reported as structured data so a scenario can still start.

func credentialManifest(descriptors ...manifestpkg.CredentialDescriptor) manifestpkg.ResourceManifest {
	return manifestpkg.ResourceManifest{
		Name:        "openrouter",
		Credentials: manifestpkg.ResourceCredentials{Descriptors: descriptors},
	}
}

func openrouterDescriptor() manifestpkg.CredentialDescriptor {
	return manifestpkg.CredentialDescriptor{
		LogicalID: "vrooli/openrouter",
		Field:     "api-key",
		Env:       "OPENROUTER_API_KEY",
		Label:     "OpenRouter API Key",
		Required:  true,
		ObtainURL: "https://openrouter.ai/keys",
	}
}

// withAuthority swaps the process authority for the duration of a test. The
// resolver deliberately has one construction path, so tests reach it here
// rather than by threading a store through every signature.
func withAuthority(t *testing.T, store securestore.Store) {
	t.Helper()
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	previous := credentialauthority.DefaultAuthority
	credentialauthority.DefaultAuthority = func() (*credentialauthority.Authority, error) { return authority, nil }
	t.Cleanup(func() { credentialauthority.DefaultAuthority = previous })
}

type memoryStore struct{ values map[string]string }

func (s *memoryStore) Put(service, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}

func (s *memoryStore) Get(service, key string) (string, error) {
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", securestore.ErrNotFound
	}
	return value, nil
}

func (s *memoryStore) Delete(service, key string) error {
	delete(s.values, service+"/"+key)
	return nil
}

func TestResolveCredentialValuesReportsHostConditionsWithoutFailing(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		store    securestore.Store
		reason   CredentialGapReason
		provider credentialauthority.ProviderState
		remedy   string
	}{
		{
			name:     "unreachable provider",
			store:    securestore.Unavailable("keyring session is unreachable"),
			reason:   GapProviderUnavailable,
			provider: credentialauthority.ProviderUnavailable,
			remedy:   "vrooli credentials doctor",
		},
		{
			name:     "absent provider",
			store:    securestore.Absent("no adapter on this platform"),
			reason:   GapProviderAbsent,
			provider: credentialauthority.ProviderAbsent,
			remedy:   "vrooli credentials doctor",
		},
		{
			name:     "unset required value on a working provider",
			store:    &memoryStore{},
			reason:   GapUnconfigured,
			provider: credentialauthority.ProviderAvailable,
			remedy:   "vrooli credentials provision --identity vrooli/openrouter --field api-key",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			withAuthority(t, testCase.store)

			resolution, err := ResolveCredentialValues(credentialManifest(openrouterDescriptor()))
			if err != nil {
				t.Fatalf("ResolveCredentialValues() error = %v, want nil for a runtime credential condition", err)
			}
			if len(resolution.Values) != 0 {
				t.Fatalf("resolved values = %v, want none", resolution.Values)
			}
			if len(resolution.Missing) != 1 {
				t.Fatalf("Missing = %+v, want exactly the one declared descriptor", resolution.Missing)
			}
			gap := resolution.Missing[0]
			if gap.Env != "OPENROUTER_API_KEY" || gap.LogicalID != "vrooli/openrouter" || gap.Field != "api-key" {
				t.Fatalf("gap does not name its descriptor: %+v", gap)
			}
			if !gap.Required {
				t.Fatal("gap lost the manifest required flag consumers rank by")
			}
			if gap.Reason != testCase.reason {
				t.Fatalf("gap reason = %q, want %q", gap.Reason, testCase.reason)
			}
			if !strings.Contains(gap.Remediation, testCase.remedy) {
				t.Fatalf("remediation = %q, want it to name %q", gap.Remediation, testCase.remedy)
			}
			if resolution.Provider != testCase.provider {
				t.Fatalf("provider state = %q, want %q", resolution.Provider, testCase.provider)
			}
		})
	}
}

func TestResolveCredentialValuesResolvesAProvisionedValue(t *testing.T) {
	store := &memoryStore{}
	withAuthority(t, store)
	authority, err := credentialauthority.DefaultAuthority()
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.Put("vrooli/openrouter", "api-key", "sk-test"); err != nil {
		t.Fatal(err)
	}

	resolution, err := ResolveCredentialValues(credentialManifest(openrouterDescriptor()))
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Values["OPENROUTER_API_KEY"] != "sk-test" {
		t.Fatalf("Values = %v, want the provisioned value", resolution.Values)
	}
	if len(resolution.Missing) != 0 {
		t.Fatalf("Missing = %+v, want none", resolution.Missing)
	}
	if resolution.Provider != credentialauthority.ProviderAvailable {
		t.Fatalf("provider state = %q, want available", resolution.Provider)
	}
}

// An optional credential is reported exactly like a required one. The
// difference is how a consumer ranks it, not whether it is survivable — that
// distinction is what used to abort the start.
func TestResolveCredentialValuesReportsOptionalGapsToo(t *testing.T) {
	withAuthority(t, &memoryStore{})
	descriptor := openrouterDescriptor()
	descriptor.Required = false

	resolution, err := ResolveCredentialValues(credentialManifest(descriptor))
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Missing) != 1 || resolution.Missing[0].Required {
		t.Fatalf("Missing = %+v, want one optional gap", resolution.Missing)
	}
}

func TestResolveCredentialValuesErrorsOnlyForManifestDefects(t *testing.T) {
	withAuthority(t, &memoryStore{})
	for _, testCase := range []struct {
		name        string
		descriptors []manifestpkg.CredentialDescriptor
		wantMessage string
	}{
		{
			name:        "unparsable logical id",
			descriptors: []manifestpkg.CredentialDescriptor{{LogicalID: "openrouter", Env: "OPENROUTER_API_KEY"}},
			wantMessage: "namespaced",
		},
		{
			name:        "empty env name",
			descriptors: []manifestpkg.CredentialDescriptor{{LogicalID: "vrooli/openrouter", Env: "  "}},
			wantMessage: "empty env name",
		},
		{
			name: "two descriptors fight over one env name",
			descriptors: []manifestpkg.CredentialDescriptor{
				{LogicalID: "vrooli/openrouter", Field: "api-key", Env: "OPENROUTER_API_KEY"},
				{LogicalID: "vrooli/openrouter-backup", Field: "api-key", Env: "OPENROUTER_API_KEY"},
			},
			wantMessage: "twice",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ResolveCredentialValues(credentialManifest(testCase.descriptors...))
			if err == nil {
				t.Fatal("ResolveCredentialValues() = nil error, want a manifest defect")
			}
			if !strings.Contains(err.Error(), testCase.wantMessage) {
				t.Fatalf("error = %v, want it to mention %q", err, testCase.wantMessage)
			}
		})
	}
}

// ResolveCredentialGaps is the presence path. It must answer the same question
// as ResolveCredentialValues without ever materializing a value.
func TestResolveCredentialGapsNeverMaterializesValues(t *testing.T) {
	store := &memoryStore{}
	withAuthority(t, store)
	authority, err := credentialauthority.DefaultAuthority()
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.Put("vrooli/openrouter", "api-key", "sk-test"); err != nil {
		t.Fatal(err)
	}

	gaps, err := ResolveCredentialGaps(credentialManifest(openrouterDescriptor()))
	if err != nil {
		t.Fatal(err)
	}
	if len(gaps.Missing) != 0 {
		t.Fatalf("Missing = %+v, want none for a provisioned credential", gaps.Missing)
	}
	if len(gaps.Values) != 0 {
		t.Fatalf("presence check materialized values: %v", gaps.Values)
	}

	encoded, err := json.Marshal(gaps)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "sk-test") {
		t.Fatalf("credential gap report leaked a value: %s", encoded)
	}
}

func TestResolveCredentialGapsReportsUnsetAndUnreachableSeparately(t *testing.T) {
	withAuthority(t, &memoryStore{})
	gaps, err := ResolveCredentialGaps(credentialManifest(openrouterDescriptor()))
	if err != nil {
		t.Fatal(err)
	}
	if len(gaps.Missing) != 1 || gaps.Missing[0].Reason != GapUnconfigured {
		t.Fatalf("Missing = %+v, want one unconfigured gap", gaps.Missing)
	}

	withAuthority(t, securestore.Unavailable("keyring session unreachable"))
	gaps, err = ResolveCredentialGaps(credentialManifest(openrouterDescriptor()))
	if err != nil {
		t.Fatal(err)
	}
	if len(gaps.Missing) != 1 || gaps.Missing[0].Reason != GapProviderUnavailable {
		t.Fatalf("Missing = %+v, want one provider-unavailable gap", gaps.Missing)
	}
}

// A resource with several descriptors must not spawn one doomed backend call
// per descriptor when the provider itself is down, and must still report every
// descriptor as missing.
func TestProviderOutageReportsEveryDescriptorOnce(t *testing.T) {
	withAuthority(t, securestore.Unavailable("keyring session unreachable"))
	first := openrouterDescriptor()
	second := openrouterDescriptor()
	second.Field = "org-id"
	second.Env = "OPENROUTER_ORG_ID"

	resolution, err := ResolveCredentialValues(credentialManifest(first, second))
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Missing) != 2 {
		t.Fatalf("Missing = %+v, want both descriptors", resolution.Missing)
	}
	for _, gap := range resolution.Missing {
		if gap.Reason != GapProviderUnavailable {
			t.Fatalf("gap %s reason = %q, want provider_unavailable", gap.Env, gap.Reason)
		}
	}
}

// A manifest default keeps standing in when the credential does not resolve.
// Deleting the variable would hand postgres an empty password rather than no
// password, which is strictly worse than the pre-credential behavior.
func TestUnresolvedCredentialKeepsManifestDefaultAndWarns(t *testing.T) {
	withAuthority(t, &memoryStore{})
	root := t.TempDir()
	writeResourceManifest(t, root, "postgres", map[string]any{
		"name":     "postgres",
		"driver":   "cloud-api",
		"endpoint": "https://postgres.invalid",
		"credentials": map[string]any{
			"descriptors": []any{map[string]any{
				"logical_id": "vrooli/postgres",
				"field":      "password",
				"env":        "POSTGRES_PASSWORD",
				"required":   true,
			}},
		},
		"runtime":             map[string]any{"env": map[string]string{"POSTGRES_PASSWORD": "vrooli"}},
		"environment_exports": map[string]any{"from_runtime_env": []string{"POSTGRES_PASSWORD"}},
	})

	report, err := ResolveResource(root, t.TempDir(), "postgres", ResolveOptions{ScenarioName: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Values["POSTGRES_PASSWORD"] != "vrooli" {
		t.Fatalf("POSTGRES_PASSWORD = %q, want the manifest default rather than an empty value",
			report.Values["POSTGRES_PASSWORD"])
	}
	if len(report.MissingCredentials) != 1 {
		t.Fatalf("MissingCredentials = %+v, want the gap still reported", report.MissingCredentials)
	}
	if !strings.Contains(strings.Join(report.Warnings, "\n"), "manifest default") {
		t.Fatalf("warnings = %v, want one naming the manifest default stand-in", report.Warnings)
	}
}

// A credential that has no other source is omitted rather than injected empty,
// so a consumer checking presence never sees a configured-but-blank value.
func TestUnresolvedCredentialWithNoOtherSourceIsOmitted(t *testing.T) {
	withAuthority(t, &memoryStore{})
	root := t.TempDir()
	writeResourceManifest(t, root, "openrouter", map[string]any{
		"name":     "openrouter",
		"driver":   "cloud-api",
		"endpoint": "https://openrouter.ai/api/v1",
		"credentials": map[string]any{
			"descriptors": []any{map[string]any{
				"logical_id": "vrooli/openrouter",
				"field":      "api-key",
				"env":        "OPENROUTER_API_KEY",
				"required":   true,
			}},
		},
	})

	report, err := ResolveResource(root, t.TempDir(), "openrouter", ResolveOptions{ScenarioName: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if _, present := report.Values["OPENROUTER_API_KEY"]; present {
		t.Fatalf("OPENROUTER_API_KEY present as %q, want it omitted entirely",
			report.Values["OPENROUTER_API_KEY"])
	}
	if len(report.MissingCredentials) != 1 {
		t.Fatalf("MissingCredentials = %+v, want one gap", report.MissingCredentials)
	}
}

func TestResolveScenarioAllowsOneSharedCredentialButRejectsDifferentCredentialsForOneEnv(t *testing.T) {
	withAuthority(t, &memoryStore{})
	root := t.TempDir()
	writeResourceManifest(t, root, "first-runner", map[string]any{
		"name": "first-runner", "driver": "cloud-api", "endpoint": "https://first.invalid",
		"credentials": map[string]any{"descriptors": []any{map[string]any{
			"logical_id": "vrooli/openai", "field": "api-key", "env": "OPENAI_API_KEY",
		}}},
	})
	writeResourceManifest(t, root, "second-runner", map[string]any{
		"name": "second-runner", "driver": "cloud-api", "endpoint": "https://second.invalid",
		"credentials": map[string]any{"descriptors": []any{map[string]any{
			"logical_id": "vrooli/openai", "field": "api-key", "env": "OPENAI_API_KEY",
		}}},
	})
	service := scenario.ServiceManifest{Dependencies: scenario.Dependencies{Resources: map[string]scenario.Dependency{
		"first-runner":  {Enabled: true},
		"second-runner": {Enabled: true},
	}}}
	if _, err := ResolveScenario(root, t.TempDir(), "demo", "", service); err != nil {
		t.Fatalf("ResolveScenario() error = %v, want shared credential to be accepted", err)
	}

	writeResourceManifest(t, root, "second-runner", map[string]any{
		"name": "second-runner", "driver": "cloud-api", "endpoint": "https://second.invalid",
		"credentials": map[string]any{"descriptors": []any{map[string]any{
			"logical_id": "vrooli/second-runner", "field": "api-key", "env": "OPENAI_API_KEY",
		}}},
	})
	if _, err := ResolveScenario(root, t.TempDir(), "demo", "", service); err == nil || !strings.Contains(err.Error(), "credential env collision") {
		t.Fatalf("ResolveScenario() error = %v, want collision for distinct credentials", err)
	}
}

func writeResourceManifest(t *testing.T, root, name string, manifest map[string]any) {
	t.Helper()
	path := manifestpkg.DefaultPath(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// Every resource manifest must declare a CLI; these fixtures care only
	// about credentials, so the block is minimal but real.
	if _, declared := manifest["portability_tier"]; !declared {
		manifest["portability_tier"] = "full"
	}
	if _, declared := manifest["cli"]; !declared {
		manifest["cli"] = map[string]any{
			"enabled":      true,
			"command":      "resource-" + name,
			"adapter":      map[string]any{"kind": "go_module", "module_dir": "cli"},
			"source_build": map[string]any{"kind": "go_module"},
			"invoke":       map[string]any{"kind": "installed_command", "command": "resource-" + name},
			"freshness":    map[string]any{"inputs": []string{"cli/**", "resource.json"}},
		}
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}
