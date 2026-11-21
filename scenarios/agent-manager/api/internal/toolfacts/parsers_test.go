package toolfacts_test

import (
	"fmt"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/toolfacts"
)

// codecParsers resolves the real per-harness transcript parsers, the same
// codecs the importer and the Transcript Indexer read with.
func codecParsers(harness string) (runner.TranscriptParser, error) {
	var parser runner.TranscriptParser
	switch harness {
	case "codex":
		parser = codecs.NewCodexForTest().NewTranscriptParser()
	case "claude-code":
		parser = codecs.NewClaudeForTest().NewTranscriptParser()
	case "opencode":
		parser = codecs.NewOpenCodeForTest().NewTranscriptParser()
	case "grok":
		parser = codecs.NewGrokForTest().NewTranscriptParser()
	default:
		return nil, fmt.Errorf("no codec for %q", harness)
	}
	if setter, ok := parser.(runner.TranscriptRetentionSetter); ok {
		setter.SetTranscriptRetention(true)
	}
	return parser, nil
}

var _ toolfacts.ParserResolver = codecParsers
