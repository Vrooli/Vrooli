package permissions

import (
	"os"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// A config as earlier adapters left it: user comments and order around the
// retired table, which the migration must keep.
const legacyConfig = `model = "gpt-6.1-sol"
# Render inline for tmux scrollback.
[tui]
fullscreen_transcript = false

[vrooli]
[vrooli.permissions]
bash_deny = ['rm -rf /']

# Vrooli long-effort settings.
[features.multi_agent_v2]
max_concurrent_threads_per_session = 8
`

const legacyConfigStripped = `model = "gpt-6.1-sol"
# Render inline for tmux scrollback.
[tui]
fullscreen_transcript = false

# Vrooli long-effort settings.
[features.multi_agent_v2]
max_concurrent_threads_per_session = 8
`

func TestStripLegacyTableKeepsEveryOtherByte(t *testing.T) {
	cases := map[string]struct{ in, want string }{
		"tables between others": {legacyConfig, legacyConfigStripped},
		"inline root value":     {"vrooli = { permissions = { bash_deny = ['x'] } }\nmodel = 'm'\n", "model = 'm'\n"},
		"dotted root keys":      {"model = 'm'\nvrooli.permissions.bash_deny = ['x']\nvrooli.execution_active = true\n[tui]\na = 1\n", "model = 'm'\n[tui]\na = 1\n"},
		"comments inside and execution subtables": {
			"model = 'm'\n\n[vrooli]\nexecution_active = true\n# inside\n[vrooli.execution.filesystem]\nworkspace = \"write\"\n\n[vrooli.permissions]\nbash_ask = [\n  'a',\n  'b',\n]\n",
			"model = 'm'\n",
		},
		"quoted key and trailing comment": {"[\"vrooli\".permissions] # old\nbash_allow = ['ls']\n[other]\nvrooli = 'not ours'\n", "[other]\nvrooli = 'not ours'\n"},
		"absent":                          {"[other]\nvrooli = 'nested is native data'\n", "[other]\nvrooli = 'nested is native data'\n"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := stripLegacyTable([]byte(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestLoadReadsALegacyTableUntilItMoves(t *testing.T) {
	a := newTestAdapter(t)
	if err := os.WriteFile(a.SettingsPath, []byte(legacyConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := a.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !p.LegacyTable || strings.Join(p.BashDeny, ",") != "rm -rf /" {
		t.Fatalf("legacy intent not visible before migration: %+v", p)
	}
	report, err := a.MigrateLegacyTable()
	if err != nil || !strings.Contains(report, "moved the legacy [vrooli] table") {
		t.Fatalf("migrate = %q, %v", report, err)
	}
	if got, _ := os.ReadFile(a.SettingsPath); string(got) != legacyConfigStripped {
		t.Fatalf("migrated config:\n%s", got)
	}
	if got, _ := os.ReadFile(a.SettingsPath + ".vrooli-permissions.orig"); string(got) != legacyConfig {
		t.Fatalf("pre-migration backup missing: %q", got)
	}
	after, err := a.Load()
	if err != nil || after.LegacyTable || Fingerprint(after) != Fingerprint(p) {
		t.Fatalf("migration changed the intent: before %+v after %+v (%v)", p, after, err)
	}
	if report, err = a.MigrateLegacyTable(); err != nil || report != "" {
		t.Fatalf("second migration = %q, %v; want nothing to do", report, err)
	}
}

func TestMigrationKeepsAnExistingSidecarAuthoritative(t *testing.T) {
	a := newTestAdapter(t)
	if err := a.Save(Policy{BashDeny: []string{"current"}}); err != nil {
		t.Fatal(err)
	}
	// An older binary re-adds the table after the move.
	if err := os.WriteFile(a.SettingsPath, []byte(legacyConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := a.MigrateLegacyTable()
	if err != nil || !strings.Contains(report, "already holds the intent") {
		t.Fatalf("migrate = %q, %v", report, err)
	}
	p, err := a.Load()
	if err != nil || strings.Join(p.BashDeny, ",") != "current" || p.LegacyTable {
		t.Fatalf("sidecar lost to the legacy table: %+v %v", p, err)
	}
}

func TestRuleWriteStripsTheLegacyTable(t *testing.T) {
	a := newTestAdapter(t)
	if err := os.WriteFile(a.SettingsPath, []byte(legacyConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := a.Save(Policy{BashDeny: []string{"rm -rf /", "git stash *"}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(a.SettingsPath); string(got) != legacyConfigStripped {
		t.Fatalf("config after rule write:\n%s", got)
	}
	if intent := readIntent(t, a); len(intent.BashDeny) != 2 {
		t.Fatalf("intent after rule write: %+v", intent)
	}
}

func TestLegacyExecutionIntentMovesWithItsOwnership(t *testing.T) {
	a := newTestAdapter(t)
	e := testExecution()
	doc := map[string]any{"model": "m"}
	if err := projectExecution(doc, e, true, false); err != nil {
		t.Fatal(err)
	}
	doc[legacyTableKey] = map[string]any{"execution": e, "execution_active": true}
	raw, err := toml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(a.SettingsPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := a.Load()
	if err != nil || before.Execution == nil || !before.ExecutionActive || before.NativeFingerprint != DesiredExecutionFingerprint(e, true) {
		t.Fatalf("legacy execution intent unreadable: %+v %v", before, err)
	}
	if _, err = a.MigrateLegacyTable(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mustRead(t, a.SettingsPath)), "execution_active") {
		t.Fatal("legacy execution keys left in config")
	}
	after, err := a.Load()
	if err != nil || Fingerprint(after) != Fingerprint(before) || !after.ExecutionActive {
		t.Fatalf("execution intent changed in the move: %+v %v", after, err)
	}
	// The moved intent still owns the reserved native profile.
	if err = a.Save(Policy{Execution: e, ExecutionActive: true}); err != nil {
		t.Fatalf("owned profile refused after migration: %v", err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
