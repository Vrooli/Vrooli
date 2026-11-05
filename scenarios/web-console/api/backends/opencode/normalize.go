package opencode

import "strings"

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

// messageText joins a message's text parts, dropping empty parts and the
// non-prose part kinds (step-start, step-finish, tool-*).
func messageText(parts []Part) string {
	var pieces []string
	for _, p := range parts {
		if p.Type != "text" {
			continue
		}
		if t := strings.TrimSpace(p.Text); t != "" {
			pieces = append(pieces, t)
		}
	}
	return strings.Join(pieces, "\n")
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
