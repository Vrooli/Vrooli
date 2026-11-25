package opencode

import (
	"strings"

	"github.com/vrooli/cli-core/agenttranscript"
)

// Cursor is the per-session high-water mark persisted between reconciliations.
// User and assistant are tracked separately because they advance on different
// signals (user message creation vs. assistant completion). Both are
// monotonic, so re-running Normalize over full history never re-emits.
type Cursor struct {
	LastUserCreated        int64 `json:"lastUserCreated"`
	LastAssistantCompleted int64 `json:"lastAssistantCompleted"`
}

// Emission is one normalized conversation event to append.
type Emission struct {
	Role      string // "user" or "assistant"
	Text      string
	MessageID string
	TurnID    string
	PartID    string
}

// messageText joins prose parts and renders tool-call parts as selectable
// assistant text, dropping empty and lifecycle-only parts.
func messageText(parts []Part) string {
	shared := make([]agenttranscript.TextPart, 0, len(parts))
	for _, p := range parts {
		if p.Type == "tool" {
			payload := p.Input
			if len(payload) == 0 {
				payload = p.State
			}
			shared = append(shared, agenttranscript.TextPart{Type: "text", Text: agenttranscript.FormatToolEvent(p.Tool, "call", p.CallID, payload)})
		} else {
			shared = append(shared, agenttranscript.TextPart{Type: p.Type, Text: p.Text})
		}
	}
	return agenttranscript.JoinTextParts(shared)
}

// Normalize converts full message history into the emissions newer than cur and
// returns the advanced cursor. It is idempotent: passing the returned cursor
// back over the same (or a superset) history yields no further emissions.
//
// User messages emit on creation; assistant messages emit only once complete
// (Time.Completed != 0) so partial streaming text is never appended.
func Normalize(messages []MessageWithParts, cur Cursor) ([]Emission, Cursor) {
	var out []Emission
	for _, m := range messages {
		text := messageText(m.Parts)
		if text == "" {
			continue
		}
		switch m.Info.Role {
		case "user":
			if m.Info.Time.Created > cur.LastUserCreated {
				out = append(out, Emission{Role: "user", Text: text, MessageID: m.Info.ID, TurnID: m.Info.ID, PartID: firstTextPartID(m.Parts)})
				cur.LastUserCreated = m.Info.Time.Created
			}
		case "assistant":
			if m.Info.Time.Completed == 0 {
				continue // still streaming; wait for completion
			}
			if m.Info.Time.Completed > cur.LastAssistantCompleted {
				turnID := m.Info.ParentID
				if turnID == "" {
					turnID = m.Info.ID
				}
				out = append(out, Emission{Role: "assistant", Text: text, MessageID: m.Info.ID, TurnID: turnID, PartID: firstTextPartID(m.Parts)})
				cur.LastAssistantCompleted = m.Info.Time.Completed
			}
		}
	}
	return out, cur
}

func firstTextPartID(parts []Part) string {
	for _, p := range parts {
		if p.Type == "text" && strings.TrimSpace(p.Text) != "" {
			return p.ID
		}
	}
	return ""
}
