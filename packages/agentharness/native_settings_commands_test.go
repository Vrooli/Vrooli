package agentharness

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func jsonCodecConfig(t *testing.T, settingsPath, docPath string) NativeSettingsConfig {
	t.Helper()
	return NativeSettingsConfig{
		Runner:       "codex",
		DocumentPath: docPath,
		SettingsPath: func() (string, error) { return settingsPath, nil },
		Decode: func(data []byte) (map[string]any, error) {
			out := map[string]any{}
			dec := json.NewDecoder(bytes.NewReader(data))
			dec.UseNumber()
			if err := dec.Decode(&out); err != nil {
				return nil, err
			}
			return nativeValue(out).(map[string]any), nil
		},
		Encode:        func(doc map[string]any) ([]byte, error) { return json.Marshal(doc) },
		WriterVersion: "test",
		Stdout:        &bytes.Buffer{},
		Stderr:        &bytes.Buffer{},
	}
}

func TestEnsureNativeSettingsThroughSymlinkedSessionHome(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "native-settings.json")
	if err := os.WriteFile(docPath, []byte(nativeDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	shared := filepath.Join(dir, "shared", "config.json")
	if err := os.MkdirAll(filepath.Dir(shared), 0o700); err != nil {
		t.Fatal(err)
	}
	original := `{"model":"gpt","features":{"multi_agent_v2":{"multi_agent_mode_hint_text":"mine"}}}`
	if err := os.WriteFile(shared, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	session := filepath.Join(dir, "session", "config.json")
	if err := os.MkdirAll(filepath.Dir(session), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(shared, session); err != nil {
		t.Fatal(err)
	}

	report, err := EnsureNativeSettings(jsonCodecConfig(t, session, docPath))
	if err != nil {
		t.Fatal(err)
	}
	if !report.Applied || report.SettingsPath != shared {
		t.Fatalf("report = %+v, want applied to the symlink target", report)
	}
	if info, err := os.Lstat(session); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("session symlink replaced by a file")
	}
	data, _ := os.ReadFile(shared)
	if !strings.Contains(string(data), `"max_concurrent_threads_per_session":8`) || !strings.Contains(string(data), `"mine"`) || !strings.Contains(string(data), `"model":"gpt"`) {
		t.Fatalf("shared config = %s", data)
	}
	if info, _ := os.Stat(shared); info.Mode().Perm() != 0o600 {
		t.Fatalf("mode changed to %o", info.Mode().Perm())
	}
	backup, _ := os.ReadFile(report.Backup)
	if string(backup) != original {
		t.Fatalf("backup = %q", backup)
	}
	if len(report.Conflicts()) != 1 || report.Conflicts()[0].ID != "hint" {
		t.Fatalf("conflicts = %+v", report.Conflicts())
	}

	again, err := EnsureNativeSettings(jsonCodecConfig(t, session, docPath))
	if err != nil {
		t.Fatal(err)
	}
	if again.Mutates() || again.Backup != "" {
		t.Fatalf("second ensure rewrote the file: %+v", again)
	}
}

func TestNativeSettingsPlanCommandIsReadOnly(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "native-settings.json")
	_ = os.WriteFile(docPath, []byte(nativeDoc), 0o644)
	settings := filepath.Join(dir, "config.json")
	cfg := jsonCodecConfig(t, settings, docPath)
	out := &bytes.Buffer{}
	cfg.Stdout = out
	group := NativeSettingsCommands(cfg)
	for _, cmd := range group.Subcommands {
		if cmd.Name == "plan" {
			if err := cmd.Run([]string{"--json"}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := os.Stat(settings); !os.IsNotExist(err) {
		t.Fatal("plan created the settings file")
	}
	if _, err := os.Stat(NativeSettingsStatePath(settings)); !os.IsNotExist(err) {
		t.Fatal("plan wrote state")
	}
	if !strings.Contains(out.String(), `"action": "add"`) {
		t.Fatalf("plan output = %s", out)
	}
}

func TestNativeSettingsRejectsDocumentForAnotherRunner(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "native-settings.json")
	_ = os.WriteFile(docPath, []byte(strings.Replace(nativeDoc, `"runner": "codex"`, `"runner": "grok"`, 1)), 0o644)
	if _, err := EnsureNativeSettings(jsonCodecConfig(t, filepath.Join(dir, "c.json"), docPath)); err == nil {
		t.Fatal("document for another runner accepted")
	}
}

func TestVerifiedVersionNotes(t *testing.T) {
	plan := NativeSettingsPlan{Changes: []NativeSettingChange{{ID: "slots", verified: "0.166.0"}}}
	if notes := verifiedVersionNotes(plan, "codex-cli 0.167.1"); len(notes) != 0 {
		t.Fatalf("newer version noted: %v", notes)
	}
	if notes := verifiedVersionNotes(plan, "codex-cli 0.161.0"); len(notes) != 1 || !strings.Contains(notes[0], "older") {
		t.Fatalf("older version notes = %v", notes)
	}
}

func TestNativeSettingsKeepsTheOriginalConfigAcrossRewrites(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "native-settings.json")
	_ = os.WriteFile(docPath, []byte(nativeDoc), 0o644)
	settings := filepath.Join(dir, "config.json")
	original := `{"model":"pristine"}`
	_ = os.WriteFile(settings, []byte(original), 0o600)
	cfg := jsonCodecConfig(t, settings, docPath)
	if _, err := EnsureNativeSettings(cfg); err != nil {
		t.Fatal(err)
	}
	// Change the declaration so a second write happens.
	_ = os.WriteFile(docPath, []byte(strings.Replace(nativeDoc, `"value": 8`, `"value": 9`, 1)), 0o644)
	report, err := EnsureNativeSettings(cfg)
	if err != nil || !report.Mutates() {
		t.Fatalf("second ensure = %+v, %v", report, err)
	}
	if got, _ := os.ReadFile(settings + ".vrooli-native-settings.orig"); string(got) != original {
		t.Fatalf(".orig = %q, want the pre-Vrooli config", got)
	}
	if got, _ := os.ReadFile(report.Backup); string(got) == original {
		t.Fatal(".bak should hold the latest pre-image, not the original")
	}
}

// A setting declared with value_from follows the resource's model policy: the
// value is written once, a policy edit propagates, a user-chosen value is a
// conflict, a matching live value is adopted, and removing the policy value
// retires the setting. The declaration never repeats the value.
func TestNativeSettingsValueFromFollowsThePolicy(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "native-settings.json")
	policyPath := filepath.Join(dir, "model-policy.json")
	doc := `{"schema_version":"vrooli.agent-native-settings/v1","runner":"codex","settings":[
	  {"id":"model","path":["model"],"value_from":"interactive_default.model","rationale":"r"},
	  {"id":"effort","path":["model_reasoning_effort"],"value_from":"interactive_default.effort","rationale":"r"}]}`
	writePolicy := func(body string) {
		t.Helper()
		if err := os.WriteFile(policyPath, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.WriteFile(docPath, []byte(doc), 0o644)
	settings := filepath.Join(dir, "config.json")
	cfg := jsonCodecConfig(t, settings, docPath)
	read := func() map[string]any {
		t.Helper()
		data, _ := os.ReadFile(settings)
		out := map[string]any{}
		_ = json.Unmarshal(data, &out)
		return out
	}

	writePolicy(`{"interactive_default":{"model":"sol","effort":"high"}}`)
	if report, err := EnsureNativeSettings(cfg); err != nil || !report.Applied {
		t.Fatalf("first ensure = %+v, %v", report, err)
	}
	if got := read(); got["model"] != "sol" || got["model_reasoning_effort"] != "high" {
		t.Fatalf("config = %v", got)
	}

	writePolicy(`{"interactive_default":{"model":"terra","effort":"high"}}`)
	if _, err := EnsureNativeSettings(cfg); err != nil {
		t.Fatal(err)
	}
	if got := read(); got["model"] != "terra" || got["model_reasoning_effort"] != "high" {
		t.Fatalf("policy edit did not propagate: %v", got)
	}

	_ = os.WriteFile(settings, []byte(`{"model":"mine","model_reasoning_effort":"high"}`), 0o600)
	writePolicy(`{"interactive_default":{"model":"luna","effort":"high"}}`)
	report, err := EnsureNativeSettings(cfg)
	if err != nil || len(report.Conflicts()) != 1 || report.Conflicts()[0].ID != "model" {
		t.Fatalf("user-chosen model must conflict: %+v, %v", report, err)
	}
	if read()["model"] != "mine" {
		t.Fatal("user-chosen model overwritten")
	}

	_ = os.WriteFile(settings, []byte(`{"model":"luna"}`), 0o600)
	report, err = EnsureNativeSettings(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range report.Changes {
		if change.ID == "model" && change.Action != NativeSettingAdopt {
			t.Fatalf("matching live value must be adopted, got %s", change.Action)
		}
	}

	writePolicy(`{}`)
	if _, err := EnsureNativeSettings(cfg); err != nil {
		t.Fatal(err)
	}
	if got := read(); got["model"] != nil || got["model_reasoning_effort"] != nil {
		t.Fatalf("values the policy no longer defines must be retired: %v", got)
	}
}

func TestNativeSettingsNeedsExactlyOneValueSource(t *testing.T) {
	for _, setting := range []string{
		`{"id":"a","path":["a"],"rationale":"r"}`,
		`{"id":"a","path":["a"],"value":1,"value_from":"x.y","rationale":"r"}`,
	} {
		doc := `{"schema_version":"vrooli.agent-native-settings/v1","runner":"codex","settings":[` + setting + `]}`
		if _, err := ParseNativeSettingsDocument([]byte(doc)); err == nil {
			t.Fatalf("accepted %s", setting)
		}
	}
}

// A model policy that cannot be used must never fail ensure, write a bad value,
// retire a value Vrooli wrote earlier, or touch the rest of the native file.
// The settings that do not follow the policy still apply.
func TestNativeSettingsBrokenPolicyHoldsPolicySettingsAndAppliesTheRest(t *testing.T) {
	doc := `{"schema_version":"vrooli.agent-native-settings/v1","runner":"codex","settings":[
	  {"id":"model","path":["model"],"value_from":"interactive_default.model","rationale":"r"},
	  {"id":"slots","path":["features","slots"],"value":8,"rationale":"r"}]}`
	broken := map[string]func(dir string) error{
		"invalid json": func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "model-policy.json"), []byte(`{"interactive_default":`), 0o644)
		},
		"empty file": func(dir string) error { return os.WriteFile(filepath.Join(dir, "model-policy.json"), nil, 0o644) },
		"not an object": func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "model-policy.json"), []byte(`["sol"]`), 0o644)
		},
		"oversized": func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "model-policy.json"), bytes.Repeat([]byte(" "), maxPolicyBytes+1), 0o644)
		},
		"missing":   func(string) error { return nil },
		"directory": func(dir string) error { return os.Mkdir(filepath.Join(dir, "model-policy.json"), 0o755) },
	}
	for name, breakPolicy := range broken {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			docPath, settings := filepath.Join(dir, "native-settings.json"), filepath.Join(dir, "config.json")
			if err := os.WriteFile(docPath, []byte(doc), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg := jsonCodecConfig(t, settings, docPath)
			if err := breakPolicy(dir); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(settings, []byte(`{"model":"sol","extra":true}`), 0o600); err != nil {
				t.Fatal(err)
			}
			report, err := EnsureNativeSettings(cfg)
			if err != nil {
				t.Fatalf("a broken policy failed ensure: %v", err)
			}
			got := actions(report.NativeSettingsPlan)
			if got["model"] != NativeSettingHeld || got["slots"] != NativeSettingAdd {
				t.Fatalf("actions = %v, want model held and slots added", got)
			}
			if len(report.Notes) == 0 || !strings.Contains(report.Notes[0], "model policy unusable") {
				t.Fatalf("the policy problem is not surfaced: %v", report.Notes)
			}
			data, _ := os.ReadFile(settings)
			live := map[string]any{}
			if err := json.Unmarshal(data, &live); err != nil || live["model"] != "sol" || live["extra"] != true || live["features"] == nil {
				t.Fatalf("native file after broken policy = %s (%v)", data, err)
			}
		})
	}
}
