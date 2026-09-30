package orchestration

import (
	"os"
	"path/filepath"
	"testing"
)

// The strings below are the four highest-volume boilerplate prefixes observed
// in the live imported corpus, where accepting the first user-role record
// verbatim collapsed 3,992 derived labels into 477 distinct values.
func TestIsInjectedContextTextRejectsHarnessPreambles(t *testing.T) {
	injected := []string{
		"<user_instructions> # AGENTS.md You are an expert software engineer",
		"# AGENTS.md instructions for /home/matthalloran8/Vrooli <INSTRUCTIONS>",
		"<environment_context> <cwd>/home/matthalloran8</cwd>",
		"<local-command-caveat>Caveat: The messages below were generated",
		"",
		"   ",
	}
	for _, text := range injected {
		if !isInjectedContextText(text) {
			t.Errorf("expected injected: %q", text)
		}
	}

	operator := []string{
		"Investigate the disk usage issue and add monitoring",
		"continue",
		"Reply with exactly one word: PROBE_OK",
		"a < b is the failing assertion, please fix it",
	}
	for _, text := range operator {
		if isInjectedContextText(text) {
			t.Errorf("expected operator message: %q", text)
		}
	}
}

// A codex transcript opens with the injected instructions block and carries the
// operator's real request in a later record. The label must come from the
// request, not the preamble.
func TestReadTranscriptLabelEvidenceSkipsInjectedPreamble(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	body := `{"role":"user","content":"<user_instructions> # AGENTS.md You are an expert software engineer"}` + "\n" +
		`{"role":"user","content":"Audit the bash scripts for cross-platform migration"}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	evidence, err := readTranscriptLabelEvidence(file)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.UserPrompt != "Audit the bash scripts for cross-platform migration" {
		t.Fatalf("user prompt = %q", evidence.UserPrompt)
	}
}

// When every user record is injected context there is no derivable label, so
// the evidence stays empty and the caller falls through to the generator lane
// rather than persisting boilerplate.
func TestReadTranscriptLabelEvidenceLeavesPromptEmptyWhenAllInjected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	body := `{"role":"user","content":"<environment_context> <cwd>/tmp</cwd>"}` + "\n" +
		`{"role":"user","content":"# AGENTS.md instructions for /home/matthalloran8/Vrooli"}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	evidence, err := readTranscriptLabelEvidence(file)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.UserPrompt != "" {
		t.Fatalf("user prompt = %q, want empty so the generator lane engages", evidence.UserPrompt)
	}
}
