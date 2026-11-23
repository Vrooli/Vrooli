package orchestration

import (
	"fmt"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/agenttranscript"
)

const geminiDocumentParserVersion = 1

type geminiDocumentParser struct{ source string }

func (p *geminiDocumentParser) ParseTranscriptLine(uuid.UUID, string) runner.TranscriptParseResult {
	return runner.TranscriptParseResult{}
}

func (p *geminiDocumentParser) TranscriptVersion() int { return geminiDocumentParserVersion }

func (p *geminiDocumentParser) ParseTranscriptDocument(runID uuid.UUID, document []byte) runner.TranscriptParseResult {
	conversation, ok := agenttranscript.ParseGeminiConversationDocument(document)
	if !ok || conversation.SessionID == "" {
		return runner.TranscriptParseResult{Err: fmt.Errorf("Gemini CLI archive has no native sessionId")}
	}
	events := make([]*domain.RunEvent, 0, len(conversation.Messages))
	for index, message := range conversation.Messages {
		event := domain.NewMessageEvent(runID, message.Role, message.Text)
		if !message.OccurredAt.IsZero() {
			event.Timestamp = message.OccurredAt
		}
		event.ID = runner.TranscriptEventID(runID, p.source, int64(index), 0, message.Text)
		events = append(events, event)
	}
	return runner.TranscriptParseResult{Events: events, SessionID: conversation.SessionID}
}

var _ runner.TranscriptParser = (*geminiDocumentParser)(nil)
var _ runner.TranscriptDocumentParser = (*geminiDocumentParser)(nil)
var _ runner.TranscriptVersioner = (*geminiDocumentParser)(nil)
