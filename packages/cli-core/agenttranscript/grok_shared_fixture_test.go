package agenttranscript

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"
)

func TestGrokSharedConsumerFixtureUsesNativeACPRecords(t *testing.T) {
	file, err := os.Open(filepath.Join("testdata", "grok-shared-consumer.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	wantRoles := []string{"user", "assistant"}
	scanner := bufio.NewScanner(file)
	var messages []Message
	for scanner.Scan() {
		messages = append(messages, ParseMessages(HarnessGrok, scanner.Bytes())...)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(messages) != len(wantRoles) {
		t.Fatalf("shared Grok ACP fixture decoded to %d messages, want %d", len(messages), len(wantRoles))
	}
	for index, message := range messages {
		if message.Role != wantRoles[index] || message.SessionID != "grok-shared-synthetic-session" || message.Text == "" {
			t.Fatalf("shared Grok message %d = %+v", index, message)
		}
	}
}
