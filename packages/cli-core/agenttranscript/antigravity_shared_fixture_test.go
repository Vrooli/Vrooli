package agenttranscript

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"
)

func TestAntigravitySharedConsumerFixtureUsesNativeDialect(t *testing.T) {
	file, err := os.Open(filepath.Join("testdata", "antigravity-shared-consumer.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var messages []Message
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		messages = append(messages, ParseMessages("antigravity", scanner.Bytes())...)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 {
		t.Fatalf("shared Antigravity fixture decoded to %d messages, want user, assistant, and tool result", len(messages))
	}
	for i, message := range messages {
		if message.Text == "" || message.OccurredAt.IsZero() {
			t.Fatalf("message %d lacks shared codec text or source time: %+v", i, message)
		}
	}
}
