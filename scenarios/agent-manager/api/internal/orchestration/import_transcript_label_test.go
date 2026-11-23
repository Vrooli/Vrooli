package orchestration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"agent-manager/internal/domain"
	"agent-manager/internal/structuredresult"
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
		if !domain.IsInjectedContextText(text) {
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
		if domain.IsInjectedContextText(text) {
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

// A Codex sub-agent transcript opens with tens of kilobytes of setup before
// the parent's task and the agent's first reply. The title excerpt must carry
// those two and none of the setup, tool output or injected instructions.
func TestTitleExcerptKeepsConversationAndDropsSetup(t *testing.T) {
	setup := strings.Repeat("base instructions ", 6000)
	toolOutput := strings.Repeat("tool output line ", 3000)
	reply := "I'll inspect the duplex implementation and its open-problem notes first."
	records := []map[string]any{
		{"type": "session_meta", "payload": map[string]any{"id": "s1", "base_instructions": map[string]any{"text": setup}}},
		{"type": "response_item", "payload": map[string]any{"type": "message", "role": "developer", "content": []any{map[string]any{"type": "input_text", "text": "You are a sub-agent. " + setup}}}},
		{"type": "response_item", "payload": map[string]any{"type": "message", "role": "user", "content": []any{map[string]any{"type": "input_text", "text": "# AGENTS.md instructions for /home/u/Vrooli\n\n<INSTRUCTIONS>" + setup}}}},
		{"type": "response_item", "payload": map[string]any{"type": "agent_message", "author": "/root", "content": []any{
			map[string]any{"type": "input_text", "text": "Message Type: NEW_TASK\nTask name: /root/duplex_completion_slice\nSender: /root\nPayload:\n"},
			map[string]any{"type": "encrypted_content", "encrypted_content": "gAAAAB" + strings.Repeat("x", 4000)},
		}}},
		{"type": "response_item", "payload": map[string]any{"type": "reasoning", "summary": []any{}, "encrypted_content": strings.Repeat("y", 3000)}},
		{"type": "response_item", "payload": map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": reply}}}},
		{"type": "event_msg", "payload": map[string]any{"type": "agent_message", "message": reply}},
		{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "output": toolOutput}},
	}
	evidence := readEvidence(t, records)

	if evidence.UserPrompt != "" || evidence.HarnessTitle != "" {
		t.Fatalf("this transcript has no operator prompt or harness title, got %+v", evidence)
	}
	want := "Agent: Message Type: NEW_TASK Task name: /root/duplex_completion_slice Sender: /root Payload:\nAgent: " + reply + "\n"
	if evidence.Source != want {
		t.Fatalf("excerpt =\n%s\nwant\n%s", evidence.Source, want)
	}
}

// However long the conversation, the excerpt stays within its byte budget and
// never splits a multi-byte character.
func TestTitleExcerptStaysWithinBudget(t *testing.T) {
	var records []map[string]any
	for i := 0; i < 200; i++ {
		text := fmt.Sprintf("reply %d ", i) + strings.Repeat("é", 1000)
		records = append(records, map[string]any{"role": "assistant", "content": text})
	}
	evidence := readEvidence(t, records)
	if len(evidence.Source) > titleSourceBytes {
		t.Fatalf("excerpt is %d bytes, budget %d", len(evidence.Source), titleSourceBytes)
	}
	if !utf8.ValidString(evidence.Source) {
		t.Fatal("excerpt split a multi-byte character")
	}
	if !strings.HasPrefix(evidence.Source, "Agent: reply 0 ") || strings.Count(evidence.Source, "\n") < 2 {
		t.Fatalf("excerpt should hold the first replies, got %q…", evidence.Source[:40])
	}
}

type recordingLabelGenerator struct {
	requests []structuredresult.ExtractRequest
}

func (g *recordingLabelGenerator) Extract(_ context.Context, req structuredresult.ExtractRequest) (structuredresult.ExtractResponse, error) {
	g.requests = append(g.requests, req)
	return structuredresult.ExtractResponse{Candidate: json.RawMessage(`"Close duplex completion gaps"`)}, nil
}

// The generator reads the excerpt as built. With no conversation text it is
// not called and the run keeps a placeholder that marks it for regeneration.
func TestResolveImportedLabelSendsExcerptOnlyWhenThereIsConversation(t *testing.T) {
	generator := &recordingLabelGenerator{}
	o := &Orchestrator{labelGenerator: generator}
	req := ImportTranscriptRequest{SourceSessionID: "s1"}

	label, source := o.resolveImportedLabel(context.Background(), req, domain.RunnerTypeCodex, transcriptLabelEvidence{Source: " \n"})
	if len(generator.requests) != 0 || source != domain.RunLabelSourcePlaceholder || label != "codex session s1" {
		t.Fatalf("empty excerpt: label %q source %q after %d generator calls", label, source, len(generator.requests))
	}

	excerpt := "Agent: Task name: /root/duplex_completion_slice\n"
	label, source = o.resolveImportedLabel(context.Background(), req, domain.RunnerTypeCodex, transcriptLabelEvidence{Source: excerpt})
	if source != domain.RunLabelSourceGenerated || label != "Close duplex completion gaps" {
		t.Fatalf("label %q source %q", label, source)
	}
	if len(generator.requests) != 1 || generator.requests[0].Source != excerpt {
		t.Fatalf("generator requests = %+v", generator.requests)
	}
}

func readEvidence(t *testing.T, records []map[string]any) transcriptLabelEvidence {
	t.Helper()
	var body bytes.Buffer
	for _, record := range records {
		line, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		body.Write(line)
		body.WriteByte('\n')
	}
	evidence, err := readTranscriptLabelEvidence(&body)
	if err != nil {
		t.Fatal(err)
	}
	return evidence
}
