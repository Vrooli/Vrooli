package backend

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// PromptHash identifies a prompt for an answer: its kind, text, and options.
// The selection is left out — moving it does not make a different prompt — so
// an answer is refused only when the agent has started asking something else.
func PromptHash(p *PendingPrompt) string {
	if p == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(p.Kind)
	b.WriteByte(0)
	b.WriteString(p.Text)
	for _, option := range p.Options {
		b.WriteByte(0)
		b.WriteString(option.Key)
		b.WriteByte(1)
		b.WriteString(option.Label)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}
