package prompttrace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTrace_JSONRoundTrip(t *testing.T) {
	original := Trace{
		SkillID:        "research-web",
		Purpose:        "gather sources",
		Variables:      map[string]string{"topic": "AI safety", "depth": "deep"},
		Prompt:         "Research the following topic...",
		PromptRevision: "sha256:abcdef01",
		UsedFallback:   true,
		CapturedAt:     "2026-01-15T10:30:00Z",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Trace
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.SkillID != original.SkillID {
		t.Errorf("SkillID: got %q, want %q", decoded.SkillID, original.SkillID)
	}
	if decoded.Purpose != original.Purpose {
		t.Errorf("Purpose: got %q, want %q", decoded.Purpose, original.Purpose)
	}
	if decoded.Prompt != original.Prompt {
		t.Errorf("Prompt: got %q, want %q", decoded.Prompt, original.Prompt)
	}
	if decoded.PromptRevision != original.PromptRevision {
		t.Errorf("PromptRevision: got %q, want %q", decoded.PromptRevision, original.PromptRevision)
	}
	if decoded.UsedFallback != original.UsedFallback {
		t.Errorf("UsedFallback: got %v, want %v", decoded.UsedFallback, original.UsedFallback)
	}
	if decoded.CapturedAt != original.CapturedAt {
		t.Errorf("CapturedAt: got %q, want %q", decoded.CapturedAt, original.CapturedAt)
	}
	if len(decoded.Variables) != len(original.Variables) {
		t.Fatalf("Variables length: got %d, want %d", len(decoded.Variables), len(original.Variables))
	}
	for k, v := range original.Variables {
		if decoded.Variables[k] != v {
			t.Errorf("Variables[%q]: got %q, want %q", k, decoded.Variables[k], v)
		}
	}
}

func TestTrace_JSONFieldNames(t *testing.T) {
	tr := Trace{
		SkillID:        "s",
		Purpose:        "p",
		Prompt:         "pr",
		PromptRevision: "rev",
		UsedFallback:   false,
		CapturedAt:     "2026-01-01T00:00:00Z",
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}

	wantKeys := []string{"skill_id", "purpose", "prompt", "prompt_revision", "used_fallback", "captured_at"}
	for _, key := range wantKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected JSON key %q, not found in %s", key, string(data))
		}
	}
}

func TestTrace_VariablesOmitEmpty(t *testing.T) {
	tr := Trace{
		SkillID:    "s",
		Purpose:    "p",
		Prompt:     "pr",
		CapturedAt: "2026-01-01T00:00:00Z",
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if strings.Contains(string(data), `"variables"`) {
		t.Errorf("expected variables to be omitted when nil, got %s", string(data))
	}
}

func TestPromptRevision_Deterministic(t *testing.T) {
	prompt := "Hello, world!"
	r1 := PromptRevision(prompt)
	r2 := PromptRevision(prompt)
	if r1 != r2 {
		t.Errorf("PromptRevision not deterministic: %q != %q", r1, r2)
	}
	if !strings.HasPrefix(r1, "sha256:") {
		t.Errorf("expected sha256: prefix, got %q", r1)
	}
	// 8 bytes = 16 hex chars after prefix
	hex := strings.TrimPrefix(r1, "sha256:")
	if len(hex) != 16 {
		t.Errorf("expected 16 hex chars, got %d in %q", len(hex), hex)
	}
}

func TestPromptRevision_DifferentInputsDiffer(t *testing.T) {
	r1 := PromptRevision("prompt A")
	r2 := PromptRevision("prompt B")
	if r1 == r2 {
		t.Error("different prompts should produce different revisions")
	}
}

func TestNowRFC3339_Format(t *testing.T) {
	got := NowRFC3339()
	if _, err := time.Parse(time.RFC3339, got); err != nil {
		t.Errorf("NowRFC3339() = %q, not valid RFC3339: %v", got, err)
	}
	if !strings.HasSuffix(got, "Z") {
		t.Errorf("expected UTC (Z suffix), got %q", got)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "trace.json")

	original := Trace{
		SkillID:    "test-skill",
		Purpose:    "testing",
		Prompt:     "Do something useful.",
		CapturedAt: "2026-03-01T12:00:00Z",
	}

	if err := Save(path, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.SkillID != original.SkillID {
		t.Errorf("SkillID: got %q, want %q", loaded.SkillID, original.SkillID)
	}
	if loaded.CapturedAt != original.CapturedAt {
		t.Errorf("CapturedAt: got %q, want %q", loaded.CapturedAt, original.CapturedAt)
	}
	// Save should populate PromptRevision when empty.
	if loaded.PromptRevision == "" {
		t.Error("expected Save to populate PromptRevision")
	}
	wantRev := PromptRevision(original.Prompt)
	if loaded.PromptRevision != wantRev {
		t.Errorf("PromptRevision: got %q, want %q", loaded.PromptRevision, wantRev)
	}
}

func TestSave_PopulatesCapturedAt(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "trace.json")

	tr := Trace{
		SkillID: "s",
		Purpose: "p",
		Prompt:  "pr",
	}

	if err := Save(path, tr); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.CapturedAt == "" {
		t.Error("expected Save to populate CapturedAt")
	}
	if _, err := time.Parse(time.RFC3339, loaded.CapturedAt); err != nil {
		t.Errorf("CapturedAt not valid RFC3339: %q", loaded.CapturedAt)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/trace.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.json")
	if err := os.WriteFile(path, []byte("{not json}"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestResearchTracePath(t *testing.T) {
	got := ResearchTracePath("/some/item/dir")
	want := "/some/item/dir/.swarm/last-research-prompt-trace.json"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
