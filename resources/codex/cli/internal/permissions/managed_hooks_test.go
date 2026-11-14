package permissions

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const modelHookDeclaration = `{"schema_version":"vrooli.agent-managed-hooks/v1","runner":"codex","hooks":[
 {"id":"vrooli-model-hook","event":"PreToolUse","matcher":"spawn_agent","program":"vrooli-policy-runner","args":["model-hook","--runner","codex"]}]}`

func declared(t *testing.T, body string) ManagedHooksDocument {
	t.Helper()
	path := filepath.Join(t.TempDir(), "managed-hooks.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	doc, err := LoadManagedHooks(path)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func hookAdapter(t *testing.T) *Adapter {
	t.Helper()
	t.Setenv(policyRunnerEnv, "")
	t.Setenv("HOME", t.TempDir())
	a := newTestAdapter(t)
	a.HookPath = filepath.Join(filepath.Dir(a.SettingsPath), "hooks.json")
	return a
}

func TestTheShippedDeclarationIsValid(t *testing.T) {
	doc, err := LoadManagedHooks(filepath.Join("..", "..", "..", "managed-hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Hooks) == 0 || doc.Hooks[0].ID != "vrooli-model-hook" || doc.Hooks[0].Matcher != "spawn_agent" {
		t.Fatalf("declaration = %+v", doc.Hooks)
	}
}

func TestLoadManagedHooksRejectsInvalidDeclarations(t *testing.T) {
	cases := map[string]string{
		"wrong schema":      `{"schema_version":"x","hooks":[]}`,
		"missing matcher":   `{"schema_version":"` + ManagedHooksSchema + `","hooks":[{"id":"a","event":"PreToolUse","program":"p"}]}`,
		"duplicate ids":     `{"schema_version":"` + ManagedHooksSchema + `","hooks":[{"id":"a","event":"E","matcher":"m","program":"p"},{"id":"a","event":"E","matcher":"m","program":"p"}]}`,
		"program with path": `{"schema_version":"` + ManagedHooksSchema + `","hooks":[{"id":"a","event":"E","matcher":"m","program":"/bin/p"}]}`,
	}
	for name, body := range cases {
		path := filepath.Join(t.TempDir(), "d.json")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadManagedHooks(path); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestApplyAddsOnlyTheDeclaredHookAndKeepsOperatorHooks(t *testing.T) {
	a := hookAdapter(t)
	operator := `{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"/usr/bin/mine"}]}],"Stop":[{"hooks":[{"type":"command","command":"/usr/bin/bye"}]}]}}`
	if err := os.WriteFile(a.HookPath, []byte(operator), 0o600); err != nil {
		t.Fatal(err)
	}
	doc := declared(t, modelHookDeclaration)
	before, err := a.ManagedHookStatuses(doc)
	if err != nil || before[0].State != HookMissing {
		t.Fatalf("before = %+v, %v", before, err)
	}
	if _, err := a.ApplyManagedHooks(doc); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(a.HookPath)
	var got struct {
		Hooks map[string][]struct {
			Matcher string
			Hooks   []map[string]any
		}
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("%v: %s", err, raw)
	}
	home, _ := os.UserHomeDir()
	wantCommand := filepath.Join(home, ".vrooli", "bin", "vrooli-policy-runner") + " model-hook --runner codex"
	pre := got.Hooks["PreToolUse"]
	if len(pre) != 2 || pre[0].Hooks[0]["command"] != "/usr/bin/mine" || len(got.Hooks["Stop"]) != 1 {
		t.Fatalf("operator hooks changed: %s", raw)
	}
	added := pre[1]
	if added.Matcher != "spawn_agent" || added.Hooks[0]["command"] != wantCommand || added.Hooks[0]["_id"] != "vrooli-model-hook" || added.Hooks[0]["managedBy"] != "vrooli" || added.Hooks[0]["timeout"] != float64(defaultHookTimeoutSeconds) {
		t.Fatalf("added entry = %+v, want command %q", added, wantCommand)
	}
	if after, _ := a.ManagedHookStatuses(doc); after[0].State != HookOK {
		t.Fatalf("after = %+v", after)
	}
	// A second apply is a no-op.
	if _, err := a.ApplyManagedHooks(doc); err != nil {
		t.Fatal(err)
	}
	if again, _ := os.ReadFile(a.HookPath); string(again) != string(raw) {
		t.Fatalf("second apply rewrote hooks.json:\n%s\n%s", raw, again)
	}
}

func TestApplyReplacesAHookWhoseMatcherOrCommandChanged(t *testing.T) {
	a := hookAdapter(t)
	stale := `{"hooks":{"PreToolUse":[{"matcher":"old_name","hooks":[{"type":"command","command":"vrooli-policy-runner model-hook --runner codex","_id":"vrooli-model-hook","managedBy":"vrooli"}]}]}}`
	if err := os.WriteFile(a.HookPath, []byte(stale), 0o600); err != nil {
		t.Fatal(err)
	}
	doc := declared(t, modelHookDeclaration)
	if before, _ := a.ManagedHookStatuses(doc); before[0].State != HookDiffers {
		t.Fatalf("before = %+v", before)
	}
	if _, err := a.ApplyManagedHooks(doc); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(a.HookPath)
	if strings.Contains(string(raw), "old_name") || strings.Count(string(raw), "vrooli-model-hook") != 1 {
		t.Fatalf("stale entry survived: %s", raw)
	}
	if after, _ := a.ManagedHookStatuses(doc); after[0].State != HookOK {
		t.Fatalf("after = %+v", after)
	}
}

func TestPolicyRunnerCommandHonoursTheOverrideAndQuotesPaths(t *testing.T) {
	t.Setenv(policyRunnerEnv, "my-runner")
	if got := PolicyRunnerCommand(); got != "my-runner" {
		t.Fatalf("override = %q", got)
	}
	t.Setenv(policyRunnerEnv, "")
	t.Setenv("HOME", "/home/a b")
	if got := PolicyRunnerCommand(); got != "'/home/a b/.vrooli/bin/vrooli-policy-runner'" {
		t.Fatalf("quoted path = %q", got)
	}
}

// Codex 0.167 sends tool_name "Bash" for a shell command and "apply_patch" for
// an edit; a matcher that names neither never fires (verified in a throwaway home).
func TestPermissionHookMatcherCoversTheToolNamesCodexSends(t *testing.T) {
	matcher := regexp.MustCompile(permissionHookMatcher)
	for _, tool := range []string{"Bash", "apply_patch", "mcp__server__tool"} {
		if !matcher.MatchString(tool) {
			t.Errorf("matcher %q does not match Codex tool %q", permissionHookMatcher, tool)
		}
	}
	if matcher.MatchString("spawn_agent") {
		t.Errorf("deny-rule matcher %q also matches spawn_agent", permissionHookMatcher)
	}
}

func TestListHookTrustReadsCodexsOwnVerdict(t *testing.T) {
	reply := `{"id":2,"result":{"data":[{"cwd":"/x","hooks":[` +
		`{"command":"/r model-hook --runner codex","sourcePath":"/h/hooks.json","trustStatus":"untrusted"},` +
		`{"command":"/other","sourcePath":"/proj/.codex/config.toml","trustStatus":"trusted"}]}]}}` + "\n"
	server := `{"id":1,"result":{}}` + "\n" + `{"method":"noise"}` + "\n" + reply
	trust, err := listHookTrust(io.Discard, strings.NewReader(server), "/h")
	if err != nil {
		t.Fatal(err)
	}
	if len(trust) != 1 || TrustOf(trust, "/r model-hook --runner codex") != TrustUntrusted || TrustOf(trust, "/absent") != TrustUnknown {
		t.Fatalf("trust = %+v", trust)
	}
	if _, err := listHookTrust(io.Discard, strings.NewReader(`{"id":1,"result":{}}`+"\n"), "/h"); err == nil {
		t.Fatal("a server that never answers hooks/list must be an error, not a verdict")
	}
}

func TestTrustAdviceNamesTheOperatorStep(t *testing.T) {
	for status, want := range map[string]string{TrustUntrusted: "/hooks", TrustModified: "/hooks", TrustUnknown: "/hooks", TrustTrusted: "Codex will run it"} {
		if got := TrustAdvice(status); !strings.Contains(got, want) {
			t.Errorf("%s advice %q lacks %q", status, got, want)
		}
	}
}
