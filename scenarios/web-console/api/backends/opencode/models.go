// Package opencode holds the OpenCode HTTP integration used by web-console's
// conversation capture: typed models for the `opencode serve` API, a thin
// client (session list, message history, SSE event stream), and a normalizer
// that turns full message history into stable user/assistant emissions.
//
// The HTTP API — not the on-disk SQLite store — is the runtime contract.
// `GET /session/{id}/message` returns full normalized history, so reconciliation
// is idempotent by design; the watcher layers a per-session high-water-mark
// cursor on top so restarts and SSE reconnects do not re-append.
package opencode

// Session is one entry from `GET /session`.
type Session struct {
	ID        string `json:"id"`
	Directory string `json:"directory"`
	Title     string `json:"title"`
	Time      struct {
		Created int64 `json:"created"`
		Updated int64 `json:"updated"`
	} `json:"time"`
}

// MessageInfo is the `info` half of a `GET /session/{id}/message` element.
// Assistant messages carry Time.Completed once finished; a zero Completed means
// the message is still streaming and must not be appended yet.
type MessageInfo struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Role      string `json:"role"`
	Time      struct {
		Created   int64 `json:"created"`
		Completed int64 `json:"completed"`
	} `json:"time"`
}

// Part is one element of a message's `parts` array. Only `text` parts carry
// conversation prose; `step-start`/`step-finish`/tool parts are ignored.
type Part struct {
	ID        string `json:"id"`
	MessageID string `json:"messageID"`
	Type      string `json:"type"`
	Text      string `json:"text"`
}

// MessageWithParts is one element of `GET /session/{id}/message`.
type MessageWithParts struct {
	Info  MessageInfo `json:"info"`
	Parts []Part      `json:"parts"`
}

// Event is one `data:` frame from the `GET /event` SSE stream. Properties is
// left as a decoded map so callers can pull sessionID without a type per event.
type Event struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

// SessionID extracts the affected session id from an event's properties, looking
// at the common shapes: a top-level "sessionID" or a nested "info.id".
func (e Event) SessionID() string {
	if v, ok := e.Properties["sessionID"].(string); ok && v != "" {
		return v
	}
	if info, ok := e.Properties["info"].(map[string]interface{}); ok {
		if v, ok := info["id"].(string); ok {
			return v
		}
	}
	return ""
}
