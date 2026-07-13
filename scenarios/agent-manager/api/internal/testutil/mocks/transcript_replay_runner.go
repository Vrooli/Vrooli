package mocks

import (
	"strings"

	adapterrunner "agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

var (
	_ adapterrunner.TranscriptParser        = (*TranscriptReplayRunner)(nil)
	_ adapterrunner.TranscriptParserFactory = (*TranscriptReplayRunner)(nil)
)

// TranscriptReplayRunner is a runner.MockRunner with a compact transcript
// grammar for recovery tests:
//   - session:<id> records a session ID
//   - message:<text> emits an assistant message
//   - done:<text> emits a successful terminal summary
//   - fail:<text> emits a failed terminal state
//   - any other non-empty line emits an info log event
type TranscriptReplayRunner struct {
	*adapterrunner.MockRunner
}

func NewTranscriptReplayRunner(rt domain.RunnerType) *TranscriptReplayRunner {
	return &TranscriptReplayRunner{MockRunner: adapterrunner.NewMockRunner(rt)}
}

// NewTranscriptParser returns a fresh parser for one logical transcript. The
// replay grammar is stateless, so the same value serves every consumption.
func (r *TranscriptReplayRunner) NewTranscriptParser() adapterrunner.TranscriptParser {
	return r
}

func (r *TranscriptReplayRunner) ParseTranscriptLine(runID uuid.UUID, line string) adapterrunner.TranscriptParseResult {
	line = strings.TrimSpace(line)
	switch {
	case line == "":
		return adapterrunner.TranscriptParseResult{}
	case strings.HasPrefix(line, "session:"):
		return adapterrunner.TranscriptParseResult{SessionID: strings.TrimPrefix(line, "session:")}
	case strings.HasPrefix(line, "message:"):
		return adapterrunner.TranscriptParseResult{
			Events: []*domain.RunEvent{
				domain.NewMessageEvent(runID, "assistant", strings.TrimPrefix(line, "message:")),
			},
		}
	case strings.HasPrefix(line, "done:"):
		return adapterrunner.TranscriptParseResult{
			Terminal: &adapterrunner.TranscriptTerminal{
				Success:  true,
				ExitCode: 0,
				Summary: &domain.RunSummary{
					Description: strings.TrimPrefix(line, "done:"),
				},
			},
		}
	case strings.HasPrefix(line, "fail:"):
		return adapterrunner.TranscriptParseResult{
			Terminal: &adapterrunner.TranscriptTerminal{
				Success:      false,
				ExitCode:     1,
				ErrorMessage: strings.TrimPrefix(line, "fail:"),
			},
		}
	default:
		return adapterrunner.TranscriptParseResult{
			Events: []*domain.RunEvent{
				domain.NewLogEvent(runID, "info", line),
			},
		}
	}
}
