package sessions

import (
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/cli-core/agenttranscript"
	"web-console/internal/backend"
	"web-console/internal/sessionstore"
	"web-console/session"
)

// ResolveLineage follows recovered_into to the newest known row. Corrupt
// cycles are bounded and return the last safe row instead of looping.
func ResolveLineage(start sessionstore.Metadata, byID map[string]sessionstore.Metadata) sessionstore.Metadata {
	current := start
	visited := map[string]struct{}{current.ID: {}}
	for current.RecoveredInto != "" {
		next, ok := byID[current.RecoveredInto]
		if !ok {
			break
		}
		if _, seen := visited[next.ID]; seen {
			break
		}
		visited[next.ID] = struct{}{}
		current = next
		if len(visited) > len(byID)+1 {
			break
		}
	}
	return current
}

// FromSession converts an internal Session to the cached Response shape.
func FromSession(s *session.Session) Response {
	return Response{
		ID:              s.ID,
		Shell:           s.Shell,
		CreatedAt:       s.CreatedAt.UTC().Format(time.RFC3339Nano),
		Cols:            int(s.Cols),
		Rows:            int(s.Rows),
		Backend:         s.Backend,
		SurvivesRestart: s.Backend == backend.Persistent || s.Backend == backend.Remote,
		Policy:          s.GetPolicy(),
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

// NormalizeOrigin maps a free-form origin string to a closed-set Origin value.
// The empty string (an origin-less create) and any unrecognized input normalize
// to OriginProgrammatic: every first-party UI client sets 'ui' explicitly, so a
// create that arrives without an origin can only have come from a programmatic
// caller.
func NormalizeOrigin(s string) sessionstore.Origin {
	switch sessionstore.Origin(s) {
	case sessionstore.OriginUI:
		return sessionstore.OriginUI
	case sessionstore.OriginRemote:
		return sessionstore.OriginRemote
	case sessionstore.OriginProgrammatic:
		return sessionstore.OriginProgrammatic
	default:
		return sessionstore.OriginProgrammatic
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
		format, ok := agenttranscript.FormatFor(agenttranscript.HarnessCodex)
		if !ok {
			return "echo 'shared Codex resume format missing'\n"
		}
		flags := codexResumeFlags(m.LaunchCommand)
		if m.AgentSessionID != "" {
			return "codex --yolo " + format.ResumeCommand + " " + flags + m.AgentSessionID + "\n"
		}
		return "codex --yolo " + format.ResumeCommand + " " + flags + "--last\n"
	case sessionstore.AgentClaude:
		format, ok := agenttranscript.FormatFor(agenttranscript.HarnessClaude)
		if !ok {
			return "echo 'shared Claude resume format missing'\n"
		}
		return "claude " + format.ResumeFlag + " " + m.AgentSessionID + " --dangerously-skip-permissions\n"
	case sessionstore.AgentOpenCode:
		format, ok := agenttranscript.FormatFor("opencode")
		if !ok {
			return "echo 'shared OpenCode resume format missing'\n"
		}
		if m.AgentSessionID != "" {
			return "opencode " + format.ResumeFlag + " " + m.AgentSessionID + "\n"
		}
		return "echo 'opencode session id missing; nothing to resume'\n"
	case sessionstore.AgentGrok:
		format, ok := agenttranscript.FormatFor("grok")
		if !ok {
			return "echo 'shared Grok resume format missing'\n"
		}
		if m.AgentSessionID != "" {
			return "grok " + format.ResumeFlag + " " + m.AgentSessionID + "\n"
		}
		return "echo 'grok session id missing; nothing to resume'\n"
	}
	return "echo 'no agent identity recorded; nothing to resume'\n"
}

// codexFlagValue is a model, profile or reasoning-effort value safe to paste
// into a shell unquoted.
var codexFlagValue = regexp.MustCompile(`^[A-Za-z0-9._:/-]+$`)

// codexResumeFlags carries the model, profile and reasoning effort the session
// was launched with into its resume command. Without them `codex resume` runs
// the thread on the config default model, so a reopened lead would silently
// change model. Each flag ends with a space; unsafe values are dropped.
func codexResumeFlags(launch string) string {
	fields := strings.Fields(launch)
	var model, profile, effort string
	for i := 0; i < len(fields); i++ {
		token := strings.TrimRight(fields[i], ";")
		next := ""
		if i+1 < len(fields) {
			next = strings.TrimRight(fields[i+1], ";")
		}
		switch {
		case (token == "-m" || token == "--model") && model == "":
			model, i = next, i+1
		case strings.HasPrefix(token, "--model=") && model == "":
			model = strings.TrimPrefix(token, "--model=")
		case (token == "-p" || token == "--profile") && profile == "":
			profile, i = next, i+1
		case (token == "-c" || token == "--config") && strings.HasPrefix(next, "model_reasoning_effort=") && effort == "":
			effort, i = strings.TrimPrefix(next, "model_reasoning_effort="), i+1
		}
	}
	var out strings.Builder
	if codexFlagValue.MatchString(model) {
		out.WriteString("-m " + model + " ")
	}
	if codexFlagValue.MatchString(profile) {
		out.WriteString("-p " + profile + " ")
	}
	if codexFlagValue.MatchString(effort) {
		out.WriteString("-c model_reasoning_effort=" + effort + " ")
	}
	return out.String()
}
