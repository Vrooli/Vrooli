package codecs

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"

	"github.com/google/uuid"
)

// Event identity is deterministic, so the store treats a re-read line as a
// replay only if parsing it again yields byte-identical payloads: a payload
// carrying time.Now() or a random value would hit replay_event_conflict.
func TestTranscriptParsersAreDeterministicPerLine(t *testing.T) {
	newParsers := map[string]func() runner.TranscriptParser{
		"claude": func() runner.TranscriptParser { return NewClaudeForTest().NewTranscriptParser() },
		"codex":  func() runner.TranscriptParser { return NewCodexForTest().NewTranscriptParser() },
		"grok":   func() runner.TranscriptParser { return NewGrokForTest().NewTranscriptParser() },
	}
	files, err := filepath.Glob(filepath.Join("testdata", "*.jsonl"))
	if err != nil || len(files) == 0 {
		t.Fatalf("fixtures: %v (%d)", err, len(files))
	}
	runID := uuid.New()
	total := 0
	for _, file := range files {
		kind := strings.SplitN(filepath.Base(file), "_", 2)[0]
		newParser, ok := newParsers[kind]
		if !ok {
			continue
		}
		f, err := os.Open(file)
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1<<20), 16<<20)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text()+"\n")
		}
		f.Close()
		payloads := func() []string {
			parser := newParser()
			var out []string
			var offset int64
			for _, line := range lines {
				events := parser.ParseTranscriptLine(runID, line).Events
				runner.AssignTranscriptEventIDs(runID, file, offset, line, events)
				offset += int64(len(line))
				for _, event := range events {
					if event == nil {
						continue
					}
					data, err := json.Marshal(event.Data)
					if err != nil {
						t.Fatal(err)
					}
					out = append(out, string(event.EventType)+" "+string(data))
				}
			}
			return out
		}
		first, second := payloads(), payloads()
		total += len(first)
		if len(first) != len(second) {
			t.Fatalf("%s: event counts differ between parses: %d vs %d", file, len(first), len(second))
		}
		for i := range first {
			if first[i] != second[i] {
				t.Fatalf("%s event %d payload differs between parses:\n%s\n%s", file, i, first[i], second[i])
			}
		}
	}
	if total == 0 {
		t.Fatal("no fixture produced events")
	}
}
