package codecs

import (
	"strings"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestCodexToolOutputOmitsInlineImageBytes(t *testing.T) {
	line := `{"timestamp":"2026-10-28T10:00:00Z","type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"c1","output":[{"type":"input_image","detail":"high","image_url":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUg=="}]}}`
	events := NewCodexForTest().NewTranscriptParser().ParseTranscriptLine(uuid.New(), line).Events
	for _, event := range events {
		if result, ok := event.Data.(*domain.ToolResultEventData); ok {
			stored := result.RawOutputJSON
			for _, block := range result.OutputBlocks {
				stored += block.RawJSON
			}
			if strings.Contains(stored, "iVBORw0KGgo") || !strings.Contains(stored, "data:image/png;base64,<omitted>") {
				t.Fatalf("stored tool output kept image bytes: %s", stored)
			}
			return
		}
	}
	t.Fatalf("no tool result parsed from %d events", len(events))
}
