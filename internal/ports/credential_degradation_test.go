package ports

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/credentialauthority"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	"github.com/vrooli/vrooli/internal/resources/securestore"
	"github.com/vrooli/vrooli/internal/scenario"
)

// withCredentialStore points the resolver at a store representing a host
// condition that cannot be produced on a real machine.
func withCredentialStore(t *testing.T, store securestore.Store) *credentialauthority.Authority {
	t.Helper()
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	previous := credentialauthority.DefaultAuthority
	credentialauthority.DefaultAuthority = func() (*credentialauthority.Authority, error) { return authority, nil }
	t.Cleanup(func() { credentialauthority.DefaultAuthority = previous })
	return authority
}

type memoryCredentialStore struct{ values map[string]string }

func (s *memoryCredentialStore) Put(service, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}

func (s *memoryCredentialStore) Get(service, key string) (string, error) {
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", securestore.ErrNotFound
	}
	return value, nil
}

func (s *memoryCredentialStore) Delete(service, key string) error {
	delete(s.values, service+"/"+key)
	return nil
}

func openrouterScenario(t *testing.T, root string) scenario.Scenario {
	t.Helper()
	return scenario.Scenario{
		Slug: "credential-degradation",
		Path: root,
		Manifest: scenario.ServiceManifest{
			Dependencies: scenario.Dependencies{
				Resources: map[string]scenario.Dependency{"openrouter": {Enabled: true}},
			},
		},
	}
}

// TestBuildEnvironmentSurvivesAnUnreachableCredentialStore is the regression
// that started this work. On 2026-07-31 a `vrooli scenario start web-console`
// aborted because a root-owned session bus made the operator keyring
// unreachable, even though the same run had already decided the openrouter
// resource was degraded and chosen to continue.
func TestBuildEnvironmentSurvivesAnUnreachableCredentialStore(t *testing.T) {
	root := repoRoot(t)
	home := t.TempDir()
	withCredentialStore(t, securestore.Unavailable(
		"write probe: store secure resource material: exit status 1: secret-tool: Could not connect: Permission denied"))

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	env, err := manager.BuildEnvironment(openrouterScenario(t, home), nil)
	if err != nil {
		t.Fatalf("BuildEnvironment: %v, want no error when the credential store is unreachable", err)
	}
	if _, present := env.EnvVars["OPENROUTER_API_KEY"]; present {
		t.Fatalf("OPENROUTER_API_KEY present as %q, want it omitted rather than injected empty",
			env.EnvVars["OPENROUTER_API_KEY"])
	}
	if len(env.CredentialGaps) == 0 {
		t.Fatal("CredentialGaps is empty; the operator would have no idea a credential is missing")
	}
	if env.CredentialProvider != credentialauthority.ProviderUnavailable {
		t.Fatalf("CredentialProvider = %q, want unavailable", env.CredentialProvider)
	}
	gap := env.CredentialGaps[0]
	if gap.Reason != resourceenv.GapProviderUnavailable {
		t.Fatalf("gap reason = %q, want provider_unavailable rather than an unset-value diagnosis", gap.Reason)
	}
	if !strings.Contains(gap.Remediation, "vrooli credentials doctor") {
		t.Fatalf("remediation = %q, want it to point at the host diagnosis", gap.Remediation)
	}
}

// A host with no secure store at all — every macOS and Windows host before
// this work — must also reach a running scenario.
func TestBuildEnvironmentSurvivesAHostWithNoCredentialBackend(t *testing.T) {
	root := repoRoot(t)
	home := t.TempDir()
	withCredentialStore(t, securestore.Absent("no adapter for this platform"))

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	env, err := manager.BuildEnvironment(openrouterScenario(t, home), nil)
	if err != nil {
		t.Fatalf("BuildEnvironment: %v, want no error on a host with no credential backend", err)
	}
	if env.CredentialProvider != credentialauthority.ProviderAbsent {
		t.Fatalf("CredentialProvider = %q, want absent", env.CredentialProvider)
	}
	if len(env.CredentialGaps) == 0 {
		t.Fatal("CredentialGaps is empty on a backend-less host")
	}
}

// TestStartThenConfigureNeedsNoControlPlaneRestart proves the second half of
// the contract: an operator can start first and provision later. The authority
// is memoized per process, so a cached value here would force a restart.
func TestStartThenConfigureNeedsNoControlPlaneRestart(t *testing.T) {
	root := repoRoot(t)
	home := t.TempDir()
	authority := withCredentialStore(t, &memoryCredentialStore{})

	manager, err := NewManager(root, home)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	item := openrouterScenario(t, home)

	before, err := manager.BuildEnvironment(item, nil)
	if err != nil {
		t.Fatalf("BuildEnvironment before provisioning: %v", err)
	}
	if len(before.CredentialGaps) != 1 || before.CredentialGaps[0].Reason != resourceenv.GapUnconfigured {
		t.Fatalf("CredentialGaps = %+v, want one unconfigured gap", before.CredentialGaps)
	}
	if _, present := before.EnvVars["OPENROUTER_API_KEY"]; present {
		t.Fatal("OPENROUTER_API_KEY was injected before it was provisioned")
	}

	// Same process, same manager, same memoized authority — exactly the state
	// a running control plane is in when the operator provisions.
	if err := authority.Put("vrooli/openrouter", "api-key", "sk-provisioned-after-start"); err != nil {
		t.Fatalf("provision credential: %v", err)
	}

	after, err := manager.BuildEnvironment(item, nil)
	if err != nil {
		t.Fatalf("BuildEnvironment after provisioning: %v", err)
	}
	if len(after.CredentialGaps) != 0 {
		t.Fatalf("CredentialGaps = %+v, want none after provisioning", after.CredentialGaps)
	}
	if after.EnvVars["OPENROUTER_API_KEY"] != "sk-provisioned-after-start" {
		t.Fatalf("OPENROUTER_API_KEY = %q, want the value provisioned after start",
			after.EnvVars["OPENROUTER_API_KEY"])
	}
}
