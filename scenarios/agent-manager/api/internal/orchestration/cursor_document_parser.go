package orchestration

import (
	"fmt"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/agenttranscript"
)

const cursorDocumentParserVersion = 1

type cursorDocumentParser struct {
	source     string
	composerID string
}

func (p *cursorDocumentParser) ParseTranscriptLine(uuid.UUID, string) runner.TranscriptParseResult {
	return runner.TranscriptParseResult{}
}

func (p *cursorDocumentParser) TranscriptVersion() int { return cursorDocumentParserVersion }

func (p *cursorDocumentParser) ParseTranscriptDocument(runID uuid.UUID, document []byte) runner.TranscriptParseResult {
	snapshot, err := agenttranscript.DecodeCursorConversationSnapshot(document)
	if err != nil {
		return runner.TranscriptParseResult{Err: fmt.Errorf("decode Cursor composer snapshot: %w", err)}
	}
	sessionID, messages, ok := agenttranscript.ParseCursorConversationForID(snapshot.Composer, snapshot.Bubbles, p.composerID)
	if !ok {
		return runner.TranscriptParseResult{Err: fmt.Errorf("Cursor snapshot composer identity mismatch")}
	}
	events := make([]*domain.RunEvent, 0, len(messages))
	for index, message := range messages {
		event := domain.NewMessageEvent(runID, message.Role, message.Text)
		if !message.OccurredAt.IsZero() {
			event.Timestamp = message.OccurredAt
		}
		event.ID = runner.TranscriptEventID(runID, p.source, int64(index), 0, message.Text)
		events = append(events, event)
	}
	return runner.TranscriptParseResult{Events: events, SessionID: sessionID}
}

var _ runner.TranscriptParser = (*cursorDocumentParser)(nil)
var _ runner.TranscriptDocumentParser = (*cursorDocumentParser)(nil)
var _ runner.TranscriptVersioner = (*cursorDocumentParser)(nil)
