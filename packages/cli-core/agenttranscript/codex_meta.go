package agenttranscript

import "encoding/json"

// CodexRolloutSessionMeta is the shared payload shape of the native
// session_meta record. Embedding it in a larger runner payload keeps these
// fields defined once without coupling runner lifecycle interpretation here.
type CodexRolloutSessionMeta struct {
	ID           string          `json:"id"`
	SessionID    string          `json:"session_id"`
	CWD          string          `json:"cwd"`
	ForkedFromID string          `json:"forked_from_id"`
	ThreadSource string          `json:"thread_source"`
	Source       json.RawMessage `json:"source"`
}

// ParseCodexRolloutSessionMeta reads the session_meta envelope and its shared
// payload fields from one native rollout line.
func ParseCodexRolloutSessionMeta(line []byte) (CodexRolloutSessionMeta, bool) {
	var record struct {
		Type    string                  `json:"type"`
		Payload CodexRolloutSessionMeta `json:"payload"`
	}
	if err := json.Unmarshal(line, &record); err != nil || record.Type != "session_meta" {
		return CodexRolloutSessionMeta{}, false
	}
	return record.Payload, true
}
