package vroolicli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/resources"
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

// useNoLiveVaultInstances keeps a credential test off this host's broker state.
// The inventory legitimately includes live managed instances, but a test that
// reads them is a test whose result depends on the machine it runs on.
func useNoLiveVaultInstances(t *testing.T) {
	t.Helper()
	previous := liveVaultUnsealKeyEntries
	liveVaultUnsealKeyEntries = func() []resources.VaultUnsealKeyEntry { return nil }
	t.Cleanup(func() { liveVaultUnsealKeyEntries = previous })
}

func useNoLiveKopiaRepositories(t *testing.T) {
	t.Helper()
	previous := liveKopiaRepositoryEntries
	liveKopiaRepositoryEntries = func() []resources.KopiaRepositoryEntry { return nil }
	t.Cleanup(func() { liveKopiaRepositoryEntries = previous })
}

func useNoLiveCredentialInstances(t *testing.T) {
	useNoLiveVaultInstances(t)
	useNoLiveKopiaRepositories(t)
}

// TestCredentialsDoctorDistinguishesEveryProviderCondition is the reason doctor
// exists: an operator seeing "Could not connect: Permission denied" has no way
// to know whether to repair a session, install a backend, or set a value.
func TestCredentialsDoctorDistinguishesEveryProviderCondition(t *testing.T) {
	useNoLiveCredentialInstances(t)
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

func TestCredentialsDoctorJSONContractIncludesRecoveryFields(t *testing.T) {
	useNoLiveCredentialInstances(t)
	withDoctorAuthority(t, &doctorTestStore{})
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(runCredentials(t, credentialFixtureRoot(t), "doctor", "--format", "json")), &raw); err != nil {
		t.Fatal(err)
	}
	assertJSONKeys := func(name string, value map[string]json.RawMessage, want ...string) {
		t.Helper()
		if len(value) != len(want) {
			t.Fatalf("%s keys = %v, want exactly %v", name, sortedJSONKeys(value), want)
		}
		for _, key := range want {
			if _, ok := value[key]; !ok {
				t.Fatalf("%s missing key %q; got %v", name, key, sortedJSONKeys(value))
			}
		}
	}
	assertJSONKeys("doctor", raw, "credentials", "provider", "recovery")
	var recovery map[string]json.RawMessage
	if err := json.Unmarshal(raw["recovery"], &recovery); err != nil {
		t.Fatal(err)
	}
	assertJSONKeys("recovery", recovery, "entry_count", "exported_at", "path", "receipt_exists", "uncovered")
	for _, key := range []string{"receipt_exists", "entry_count", "path", "uncovered"} {
		if len(recovery[key]) == 0 {
			t.Fatalf("recovery.%s is empty", key)
		}
	}
}

func sortedJSONKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
	if strings.TrimSpace(diagnosis.Fix) == "" {
		t.Fatal("a bare Permission denied reached the operator with no explanation")
	}

	// Fix must name a condition that is blocking *now*. SessionRepair records a
	// correction Vrooli already made successfully, so reusing it as the fix
	// hands the operator work that is both already done and unrelated to what
	// is currently broken.
	if repair := strings.TrimSpace(diagnosis.SessionRepair); repair != "" && strings.TrimSpace(diagnosis.Fix) == repair {
		t.Fatalf("diagnosis fix restates an already-applied session repair instead of the blocking condition: %q", diagnosis.Fix)
	}

	if diagnosis.SessionRepair != "" {
		// Vrooli corrected the session itself, so the uid mismatch is no longer
		// what stops the read. Whatever failed next owns the remedy, and naming
		// the mismatch here would send the operator after a solved problem.
		t.Logf("session was repaired; the blocking condition is downstream and owns the fix: %q", diagnosis.Fix)
		return
	}
	if !strings.Contains(diagnosis.Fix, "XDG_RUNTIME_DIR") || !strings.Contains(diagnosis.Fix, "uid") {
		t.Fatalf("diagnosis fix = %q, want a named uid/session mismatch when the session was not repaired", diagnosis.Fix)
	}
}

// The remedy for a condition must come from the layer that detected it. A
// half-loaded Secret Service collection is cleared by a fresh login and by
// nothing else, so that is what an operator must be told — not the session
// correction that already succeeded, and not a bare transport error.
func TestDiagnosisFixNamesTheDetectedConditionRatherThanAnAppliedRepair(t *testing.T) {
	diagnosis := securestore.Diagnose()
	if diagnosis.Available {
		t.Skip("this host's credential store is reachable, so there is no blocking condition to attribute")
	}
	if strings.TrimSpace(diagnosis.Fix) == "" {
		t.Fatalf("condition %q reached the operator with no fix at all", diagnosis.Condition)
	}
	if repair := strings.TrimSpace(diagnosis.SessionRepair); repair != "" && strings.TrimSpace(diagnosis.Fix) == repair {
		t.Fatalf("fix restates the already-applied session repair: %q", diagnosis.Fix)
	}
	// The write fix is assembled from two clauses; an empty second clause used
	// to leave it ending in a dangling semicolon.
	if strings.HasSuffix(strings.TrimSpace(diagnosis.WriteFix), ";") {
		t.Fatalf("write fix ends mid-sentence: %q", diagnosis.WriteFix)
	}
}

// Neither read-only command may ever print a stored value. They exist to be
// pasted into bug reports and shared terminals.
func TestCredentialsReadOnlyCommandsNeverPrintAValue(t *testing.T) {
	useNoLiveCredentialInstances(t)
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
	} {
		output := runCredentials(t, root, args...)
		if strings.Contains(output, provisionedTestValue) {
			t.Fatalf("credentials %s printed a stored value:\n%s", strings.Join(args, " "), output)
		}
	}
}

func TestCredentialsBootstrapHelpListsOnlyTheFloor(t *testing.T) {
	var out bytes.Buffer
	if err := (&App{}).runCredentialsCommand(&CommandContext{Stdout: &out, Stderr: &out}, []string{"--help"}); err != nil {
		t.Fatal(err)
	}
	help := out.String()
	for _, command := range []string{"doctor", "provision", "status", "store", "keyring", "recovery"} {
		if !strings.Contains(help, "vrooli credentials "+command) {
			t.Fatalf("help does not list floor command %q:\n%s", command, help)
		}
	}
	for _, moved := range []string{"vrooli credentials list", "vrooli credentials delete"} {
		if strings.Contains(help, moved) {
			t.Fatalf("help still lists moved command %q:\n%s", moved, help)
		}
	}
}
