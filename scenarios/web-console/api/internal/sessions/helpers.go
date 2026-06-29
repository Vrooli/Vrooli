package sessions

import (
	"web-console/internal/backend"
	"web-console/internal/sessionstore"
	"web-console/session"
)

// FromSession converts an internal Session to the cached Response shape.
func FromSession(s *session.Session) Response {
	return Response{
		ID:              s.ID,
		Shell:           s.Shell,
		CreatedAt:       s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Cols:            int(s.Cols),
		Rows:            int(s.Rows),
		Backend:         s.Backend,
		SurvivesRestart: s.Backend == backend.Persistent,
		Policy:          s.GetPolicy(),
		Busy:            s.HasChildProcess(),
		Recovered:       s.Recovered(),
	}
}

// NormalizeAgentType maps a free-form string to a closed-set Agent value.
// Unrecognized inputs return AgentNone so a future client that learns about
// a new agent kind can roll forward without breaking older API builds.
func NormalizeAgentType(s string) sessionstore.Agent {
	switch sessionstore.Agent(s) {
	case sessionstore.AgentCodex:
		return sessionstore.AgentCodex
	case sessionstore.AgentClaude:
		return sessionstore.AgentClaude
	case sessionstore.AgentOpenCode:
		return sessionstore.AgentOpenCode
	case sessionstore.AgentGrok:
		return sessionstore.AgentGrok
	case sessionstore.AgentNone:
		return sessionstore.AgentNone
	default:
		return sessionstore.AgentNone
	}
}

// Recoverability decides whether a stored session row can be recovered and,
// if not, why.
func Recoverability(m sessionstore.Metadata) (bool, string) {
	switch m.AgentType {
	case sessionstore.AgentNone:
		return false, "no agent identity recorded"
	case sessionstore.AgentClaude:
		if m.AgentSessionID == "" {
			return false, "claude session id is required (resuming the wrong project is unsafe)"
		}
		return true, ""
	case sessionstore.AgentCodex:
		return true, ""
	case sessionstore.AgentOpenCode:
		if m.AgentSessionID == "" {
			return false, "opencode session id is required (resuming the wrong project is unsafe)"
		}
		return true, ""
	case sessionstore.AgentGrok:
		if m.AgentSessionID == "" {
			return false, "grok session id is required (resuming the wrong project is unsafe)"
		}
		return true, ""
	default:
		return false, "unknown agent type: " + string(m.AgentType)
	}
}

// BuildResumeCommand returns the literal string to paste into the new pane's
// stdin. Includes a trailing newline so it executes immediately. Never
// returns the empty string.
func BuildResumeCommand(m sessionstore.Metadata) string {
	switch m.AgentType {
	case sessionstore.AgentCodex:
		if m.AgentSessionID != "" {
			return "codex --yolo resume " + m.AgentSessionID + "\n"
		}
		return "codex --yolo resume --last\n"
	case sessionstore.AgentClaude:
		return "claude --resume " + m.AgentSessionID + " --dangerously-skip-permissions\n"
	case sessionstore.AgentOpenCode:
		if m.AgentSessionID != "" {
			return "opencode --session " + m.AgentSessionID + "\n"
		}
		return "echo 'opencode session id missing; nothing to resume'\n"
	case sessionstore.AgentGrok:
		if m.AgentSessionID != "" {
			return "grok --resume " + m.AgentSessionID + "\n"
		}
		return "echo 'grok session id missing; nothing to resume'\n"
	}
	return "echo 'no agent identity recorded; nothing to resume'\n"
}
