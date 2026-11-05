package codecs

import (
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestCodexDefaultParserNewUserTurnInvalidatesAcceptedGoal(t *testing.T) {
	accepted, err := os.ReadFile(filepath.Join("testdata", "codex_goal_accepted.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for name, record := range map[string]string{
		"event_message":    `{"type":"event_msg","payload":{"type":"user_message","message":"private fixture user content"}}`,
		"response_message": `{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"private fixture user content"}]}}`,
		"operator_message": `{"type":"response_item","payload":{"type":"message","role":"operator","content":[{"type":"input_text","text":"private fixture user content"}]}}`,
		"attachment_only":  `{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_image","image_url":"fixture:image"}]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "transcript.jsonl")
			if err := os.WriteFile(path, append(append([]byte{}, accepted...), []byte(record+"\n")...), 0600); err != nil {
				t.Fatal(err)
			}
			runID := uuid.New()
			// The prefix really has accepted completion. This is not a fixture
			// that passes merely because it never parsed a terminal at all.
			prefix := NewCodexForTest().NewTranscriptParser()
			_, terminal, err := runner.Consume(t.Context(), runner.ConsumeArgs{RunID: runID, Transcript: path, EndAt: int64(len(accepted)), ParseFn: prefix.ParseTranscriptLine})
			if err != nil || terminal == nil || terminal.TerminalReason != "goal_complete" {
				t.Fatalf("accepted native prefix did not qualify: terminal=%+v err=%v", terminal, err)
			}
			// Production defaults to no user-content retention. Boundary
			// handling must not require enabling it or emitting user content.
			parser := NewCodexForTest().NewTranscriptParser()
			userEvents := 0
			cursor, terminal, err := runner.Consume(t.Context(), runner.ConsumeArgs{
				RunID: runID, Transcript: path, ParseFn: parser.ParseTranscriptLine,
				OnEvents: func(events []*domain.RunEvent) {
					for _, event := range events {
						if message, ok := event.Data.(*domain.MessageEventData); ok && message.Role == "user" {
							userEvents++
						}
					}
				},
			})
			if err != nil || cursor != int64(len(accepted)+len(record)+1) {
				t.Fatalf("default replay did not consume the new turn: cursor=%d err=%v", cursor, err)
			}
			if userEvents != 0 {
				t.Fatal("boundary handling exposed suppressed user content")
			}
			if terminal != nil {
				t.Fatalf("historical accepted goal finalized a later user turn: %+v", terminal)
			}
		})
	}
}
