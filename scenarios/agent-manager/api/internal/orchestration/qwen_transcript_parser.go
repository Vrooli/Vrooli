package orchestration

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/agenttranscript"
)

const qwenTranscriptParserVersion = 1

type qwenTranscriptParser struct{}

func (qwenTranscriptParser) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	messages := agenttranscript.ParseQwenMessages([]byte(line))
	result := runner.TranscriptParseResult{}
	for _, message := range messages {
		event := domain.NewMessageEvent(runID, message.Role, message.Text)
		if !message.OccurredAt.IsZero() {
			event.Timestamp = message.OccurredAt
		}
		event.ID = uuid.Nil // runner.Consume assigns the stable source-offset identity.
		result.Events = append(result.Events, event)
		if message.SessionID != "" && result.SessionID == "" {
			result.SessionID = message.SessionID
		}
	}
	return result
}

func (qwenTranscriptParser) TranscriptVersion() int { return qwenTranscriptParserVersion }

var _ runner.TranscriptParser = qwenTranscriptParser{}
var _ runner.TranscriptVersioner = qwenTranscriptParser{}
