package vroolicli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
)

const provisionedTestValue = "sk-doctor-must-never-print-this"

type doctorTestStore struct{ values map[string]string }

func (s *doctorTestStore) Put(service, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}

func (s *doctorTestStore) Get(service, key string) (string, error) {
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", securestore.ErrNotFound
	}
	return value, nil
}

func (s *doctorTestStore) Delete(service, key string) error {
	delete(s.values, service+"/"+key)
	return nil
}

func withDoctorAuthority(t *testing.T, store securestore.Store) *credentialauthority.Authority {
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

// credentialFixtureRoot writes two resources: one whose credential is
// provisioned and one whose credential is not, so both output branches are
// exercised in the same run.
func credentialFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeCredentialFixture(t, root, "openrouter", "vrooli/openrouter", "api-key", "OPENROUTER_API_KEY", true)
	writeCredentialFixture(t, root, "elevenlabs", "vrooli/elevenlabs", "api-key", "ELEVENLABS_API_KEY", false)
	return root
}

func writeCredentialFixture(t *testing.T, root, name, logicalID, field, envName string, required bool) {
	t.Helper()
	manifest := map[string]any{
		"name":             name,
		"driver":           "cloud-api",
		"endpoint":         "https://" + name + ".invalid",
		"portability_tier": "full",
		"cli": map[string]any{
			"enabled":      true,
			"command":      "resource-" + name,
			"adapter":      map[string]any{"kind": "go_module", "module_dir": "cli"},
			"source_build": map[string]any{"kind": "go_module"},
			"invoke":       map[string]any{"kind": "installed_command", "command": "resource-" + name},
			"freshness":    map[string]any{"inputs": []string{"cli/**", "resource.json"}},
		},
		"credentials": map[string]any{
			"descriptors": []any{map[string]any{
				"logical_id": logicalID,
				"field":      field,
				"env":        envName,
				"required":   required,
				"label":      name + " key",
				"obtain_url": "https://" + name + ".invalid/keys",
			}},
		},
	}
	path := filepath.Join(root, "resources", name, "resource.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}

func runCredentials(t *testing.T, root string, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	ctx := &CommandContext{Root: root, Stdout: &out, Stderr: &out}
	app := &App{}
	if err := app.runCredentialsCommand(ctx, args); err != nil {
		t.Fatalf("credentials %s: %v", strings.Join(args, " "), err)
	}
	return out.String()
}

// TestCredentialsDoctorDistinguishesEveryProviderCondition is the reason doctor
// exists: an operator seeing "Could not connect: Permission denied" has no way
// to know whether to repair a session, install a backend, or set a value.
func TestCredentialsDoctorDistinguishesEveryProviderCondition(t *testing.T) {
	root := credentialFixtureRoot(t)

	t.Run("unset value names the provision command", func(t *testing.T) {
		withDoctorAuthority(t, &doctorTestStore{})
		output := runCredentials(t, root, "doctor")
		if !strings.Contains(output, "vrooli credentials provision --identity vrooli/openrouter --field api-key") {
			t.Fatalf("doctor did not name the provision command:\n%s", output)
		}
		if !strings.Contains(output, "OPENROUTER_API_KEY") || !strings.Contains(output, "ELEVENLABS_API_KEY") {
			t.Fatalf("doctor did not name every declared credential:\n%s", output)
		}
	})

	t.Run("every credential resolved reports a clean host", func(t *testing.T) {
		authority := withDoctorAuthority(t, &doctorTestStore{})
		if err := authority.Put("vrooli/openrouter", "api-key", provisionedTestValue); err != nil {
			t.Fatal(err)
		}
		if err := authority.Put("vrooli/elevenlabs", "api-key", provisionedTestValue); err != nil {
			t.Fatal(err)
		}
		output := runCredentials(t, root, "doctor")
		if !strings.Contains(output, "Every declared credential resolves on this host") {
			t.Fatalf("doctor did not report a clean host:\n%s", output)
		}
	})
}

// TestCredentialsDoctorReportsTheUidMismatchByName reproduces the incident
// host: a shell running as one uid while XDG_RUNTIME_DIR names another uid's
// runtime directory.
func TestCredentialsDoctorReportsTheUidMismatchByName(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("the uid-mismatch diagnosis needs a non-root uid to disagree with a root-owned runtime dir")
	}
	foreign := "/run/user/0"
	info, err := os.Stat(foreign)
	if err != nil {
		t.Skipf("no foreign runtime directory to point at on this host: %v", err)
	}
	if !info.IsDir() {
		t.Skipf("%s is not a directory", foreign)
	}

	t.Setenv("XDG_RUNTIME_DIR", foreign)
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+foreign+"/bus")

	diagnosis := securestore.Diagnose()
	if diagnosis.Available {
		t.Skip("this host reaches its secure store even with a foreign runtime dir")
	}
	if !strings.Contains(diagnosis.Fix, "XDG_RUNTIME_DIR") || !strings.Contains(diagnosis.Fix, "uid") {
		t.Fatalf("diagnosis fix = %q, want a named uid/session mismatch", diagnosis.Fix)
	}
	if strings.TrimSpace(diagnosis.Fix) == "" {
		t.Fatal("a bare Permission denied reached the operator with no explanation")
	}
}

// Neither read-only command may ever print a stored value. They exist to be
// pasted into bug reports and shared terminals.
func TestCredentialsReadOnlyCommandsNeverPrintAValue(t *testing.T) {
	root := credentialFixtureRoot(t)
	authority := withDoctorAuthority(t, &doctorTestStore{})
	if err := authority.Put("vrooli/openrouter", "api-key", provisionedTestValue); err != nil {
		t.Fatal(err)
	}
	if err := authority.Put("vrooli/elevenlabs", "api-key", provisionedTestValue); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"doctor"},
		{"doctor", "--format", "json"},
		{"list"},
		{"list", "--format", "json"},
	} {
		output := runCredentials(t, root, args...)
		if strings.Contains(output, provisionedTestValue) {
			t.Fatalf("credentials %s printed a stored value:\n%s", strings.Join(args, " "), output)
		}
	}
}

func TestCredentialsListNamesEveryDeclaredCredentialAndItsState(t *testing.T) {
	root := credentialFixtureRoot(t)
	authority := withDoctorAuthority(t, &doctorTestStore{})
	if err := authority.Put("vrooli/openrouter", "api-key", provisionedTestValue); err != nil {
		t.Fatal(err)
	}

	var entries []credentialEntry
	var out bytes.Buffer
	ctx := &CommandContext{Root: root, Stdout: &out, Stderr: &out}
	if err := credentialsList(ctx, []string{"--format", "json"}); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatalf("decode credentials list: %v\n%s", err, out.String())
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %+v, want both declared credentials", entries)
	}
	byEnv := map[string]credentialEntry{}
	for _, entry := range entries {
		byEnv[entry.Env] = entry
	}
	if !byEnv["OPENROUTER_API_KEY"].Configured || byEnv["OPENROUTER_API_KEY"].State != "configured" {
		t.Fatalf("provisioned credential reported as %+v", byEnv["OPENROUTER_API_KEY"])
	}
	if !byEnv["OPENROUTER_API_KEY"].Required {
		t.Fatal("required flag lost")
	}
	if byEnv["ELEVENLABS_API_KEY"].Configured || byEnv["ELEVENLABS_API_KEY"].State != string(resourceenv.GapUnconfigured) {
		t.Fatalf("unset credential reported as %+v", byEnv["ELEVENLABS_API_KEY"])
	}
}

// A provider outage must not read as "the operator never set this".
func TestCredentialsListSeparatesOutageFromUnsetValue(t *testing.T) {
	root := credentialFixtureRoot(t)
	withDoctorAuthority(t, securestore.Unavailable("keyring session unreachable"))

	var out bytes.Buffer
	ctx := &CommandContext{Root: root, Stdout: &out, Stderr: &out}
	if err := credentialsList(ctx, []string{"--format", "json"}); err != nil {
		t.Fatal(err)
	}
	var entries []credentialEntry
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.State != string(resourceenv.GapProviderUnavailable) {
			t.Fatalf("entry %s state = %q, want provider_unavailable", entry.Env, entry.State)
		}
	}
}

// Deleting is the deprovision half of the surface: a leaked or rotated key has
// to be revocable through the same documented interface that wrote it.
func TestCredentialsDeleteRemovesAValueAndReportsHonestly(t *testing.T) {
	authority := withDoctorAuthority(t, &doctorTestStore{})
	if err := authority.Put("vrooli/openrouter", "api-key", provisionedTestValue); err != nil {
		t.Fatal(err)
	}

	output := runCredentials(t, t.TempDir(), "delete", "--identity", "vrooli/openrouter", "--field", "api-key", "--yes")
	if !strings.Contains(output, "removed") {
		t.Fatalf("delete output = %q, want it to confirm removal", output)
	}
	if authority.Status("vrooli/openrouter", "api-key").Configured {
		t.Fatal("credential still configured after delete")
	}

	// Deleting again is not an error, but must not claim it removed something.
	output = runCredentials(t, t.TempDir(), "delete", "--identity", "vrooli/openrouter", "--field", "api-key", "--yes")
	if !strings.Contains(output, "was not configured") {
		t.Fatalf("second delete output = %q, want it to say nothing was there", output)
	}
}

func TestCredentialsDeleteRefusesWithoutExplicitConfirmation(t *testing.T) {
	authority := withDoctorAuthority(t, &doctorTestStore{})
	if err := authority.Put("vrooli/openrouter", "api-key", provisionedTestValue); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	ctx := &CommandContext{Root: t.TempDir(), Stdout: &out, Stderr: &out}
	app := &App{}
	err := app.runCredentialsCommand(ctx, []string{"delete", "--identity", "vrooli/openrouter", "--field", "api-key"})
	if err == nil {
		t.Fatal("delete without --yes succeeded; an unrecoverable removal must be explicit")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("refusal = %v, want it to name the required flag", err)
	}
	if !authority.Status("vrooli/openrouter", "api-key").Configured {
		t.Fatal("refused delete still removed the credential")
	}
}

// A provider outage must not be reported as a successful removal — the operator
// would believe a leaked key was revoked when it is still in the store.
func TestCredentialsDeleteRefusesWhileTheProviderIsDown(t *testing.T) {
	withDoctorAuthority(t, securestore.Unavailable("keyring session unreachable"))

	var out bytes.Buffer
	ctx := &CommandContext{Root: t.TempDir(), Stdout: &out, Stderr: &out}
	app := &App{}
	err := app.runCredentialsCommand(ctx, []string{"delete", "--identity", "vrooli/openrouter", "--field", "api-key", "--yes"})
	if err == nil {
		t.Fatal("delete succeeded while the credential store was unreachable")
	}
	if !strings.Contains(err.Error(), "credentials doctor") {
		t.Fatalf("error = %v, want it to point at the host diagnosis", err)
	}
}
