package agentapp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// Every agent manifest advertises supports_ensure, and `vrooli setup` calls
// `<resource> ensure`; the command must exist for each harness.
func TestEveryAgentResourceHasEnsure(t *testing.T) {
	for _, name := range []string{"codex", "claude-code", "opencode", "grok", "antigravity"} {
		if !strings.Contains(helpText(t, testBuild(t, Spec{Name: name})), "ensure ") {
			t.Errorf("%s has no ensure command", name)
		}
	}
}

func TestEnsureRunsEveryStepAndReportsFailures(t *testing.T) {
	ran := []string{}
	spec := Spec{Name: "grok", EnsureSteps: []EnsureStep{
		{Name: "first", Run: func(io.Writer, io.Writer) error { ran = append(ran, "first"); return errors.New("broken") }},
		{Name: "second", Run: func(io.Writer, io.Writer) error { ran = append(ran, "second"); return nil }},
	}}
	out := &bytes.Buffer{}
	err := runEnsure(spec, "", []string{"--json"}, out, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "first: broken") {
		t.Fatalf("err = %v", err)
	}
	if strings.Join(ran, ",") != "first,second" {
		t.Fatalf("ran = %v; a failed step must not hide later steps", ran)
	}
	var result ensureResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("json: %v (%s)", err, out)
	}
	if result.Status != "failed" || len(result.Steps) != 2 || result.Steps[1].Status != "ok" {
		t.Fatalf("result = %+v", result)
	}
}

func TestEnsureAcceptsTheLifecycleConfigContract(t *testing.T) {
	cfg := base64.StdEncoding.EncodeToString([]byte(`{"model":"x"}`))
	out := &bytes.Buffer{}
	if err := runEnsure(Spec{Name: "grok"}, "", []string{"--config-base64", cfg}, out, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "not interpreted") || !strings.Contains(out.String(), "nothing declared") {
		t.Fatalf("output = %s", out)
	}
	if err := runEnsure(Spec{Name: "grok"}, "", []string{"--config-base64", "%%%"}, io.Discard, io.Discard); err == nil {
		t.Fatal("malformed config accepted")
	}
}

func TestEnsureAppliesNativeSettings(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, "config.json")
	doc := filepath.Join(dir, "native-settings.json")
	if err := os.WriteFile(doc, []byte(`{"schema_version":"vrooli.agent-native-settings/v1","runner":"grok","settings":[
	  {"id":"k","path":["a","b"],"value":3,"rationale":"r"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := Spec{Name: "grok", NativeSettings: &NativeSettings{
		DocumentPath: doc,
		SettingsPath: func() (string, error) { return settings, nil },
		Decode: func(b []byte) (map[string]any, error) {
			m := map[string]any{}
			return m, json.Unmarshal(b, &m)
		},
		Encode: func(m map[string]any) ([]byte, error) { return json.Marshal(m) },
	}}
	if err := runEnsure(spec, "", nil, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(settings)
	if string(data) != `{"a":{"b":3}}` {
		t.Fatalf("settings = %s", data)
	}
	if !strings.Contains(helpText(t, testBuild(t, spec)), "settings ") {
		t.Fatal("settings group missing when NativeSettings is declared")
	}
	if strings.Contains(helpText(t, testBuild(t, Spec{Name: "grok"})), "settings ") {
		t.Fatal("settings group present without NativeSettings")
	}
}

func helpText(t *testing.T, app *cliapp.ResourceApp) string {
	t.Helper()
	var help strings.Builder
	if err := app.CLI.RunWithWriters([]string{"--help"}, &help, &help); err != nil {
		t.Fatal(err)
	}
	return help.String()
}
