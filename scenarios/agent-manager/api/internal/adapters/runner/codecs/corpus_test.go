package codecs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestReplayCorpusParsesWithoutSecrets(t *testing.T) {
	cases := []struct {
		name     string
		newCodec func(t *testing.T) Codec
	}{
		{"claude-stdout.jsonl", func(t *testing.T) Codec { t.Helper(); return NewClaudeForTest() }},
		{"claude-ondisk.jsonl", func(t *testing.T) Codec { t.Helper(); return NewClaudeForTest() }},
		{"codex-stdout.jsonl", func(t *testing.T) Codec { t.Helper(); return NewCodexForTest() }},
		{"codex-rollout.jsonl", func(t *testing.T) Codec { t.Helper(); return NewCodexForTest() }},
		{"grok-stdout.jsonl", func(t *testing.T) Codec { t.Helper(); return NewGrokForTest() }},
		{"grok-updates.jsonl", func(t *testing.T) Codec { t.Helper(); return NewGrokForTest() }},
		{"opencode-stdout.jsonl", func(t *testing.T) Codec { t.Helper(); return NewOpenCodeForTest() }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join("testdata", "corpus", tc.name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			lower := strings.ToLower(string(raw))
			for _, forbidden := range []string{"/home/", "api_key=", "authorization=", "bearer "} {
				if strings.Contains(lower, forbidden) {
					t.Fatalf("corpus contains unredacted %q", forbidden)
				}
			}
			parser := tc.newCodec(t).NewTranscriptParser()
			messages := 0
			for i, line := range strings.Split(string(raw), "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				result := parser.ParseTranscriptLine(uuid.New(), line)
				if result.Err != nil {
					t.Fatalf("line %d: %v", i+1, result.Err)
				}
				for _, event := range result.Events {
					if event != nil && event.EventType == domain.EventTypeMessage {
						messages++
					}
				}
			}
			if messages == 0 {
				t.Fatal("corpus replay produced no message event")
			}
		})
	}
}
