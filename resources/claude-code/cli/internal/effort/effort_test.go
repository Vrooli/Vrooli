package effort

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompactContextNamesTheRereadAndStance(t *testing.T) {
	out := &bytes.Buffer{}
	group := Commands(out)
	if err := group.Subcommands[0].Run(nil); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"re-read", "stance", "verbatim", "do not narrow"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("context lacks %q: %s", want, out)
		}
	}
}

func TestEnsureStepIsIdempotentAndPreservesSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"model":"opus","hooks":{"SessionStart":[{"matcher":"startup","hooks":[{"type":"command","command":"user"}]}]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	step := EnsureStep(func() string { return path }, func() string { return "/opt/vrooli bin/resource-claude-code" })
	for i := 0; i < 2; i++ {
		if err := step.Run(io.Discard, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	// Claude Code strips the broker's markers when it rewrites settings.
	raw, _ := os.ReadFile(path)
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	stripMarkers(doc)
	stripped, _ := json.Marshal(doc)
	_ = os.WriteFile(path, stripped, 0o600)
	if err := step.Run(io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	pretty, _ := os.ReadFile(path)
	compact := &bytes.Buffer{}
	if err := json.Compact(compact, pretty); err != nil {
		t.Fatal(err)
	}
	raw = compact.Bytes()
	if n := strings.Count(string(raw), "effort compact-context"); n != 1 {
		t.Fatalf("hook appears %d times: %s", n, raw)
	}
	for _, want := range []string{`"model":"opus"`, `"command":"user"`, `"matcher":"compact"`, `'/opt/vrooli bin/resource-claude-code' effort compact-context`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("settings lack %s: %s", want, raw)
		}
	}
}

func stripMarkers(v any) {
	switch typed := v.(type) {
	case map[string]any:
		delete(typed, "_id")
		delete(typed, "managedBy")
		for _, item := range typed {
			stripMarkers(item)
		}
	case []any:
		for _, item := range typed {
			stripMarkers(item)
		}
	}
}
