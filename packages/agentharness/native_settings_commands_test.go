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
