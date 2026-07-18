package agentmanager

import (
	"fmt"
	"strings"
)

func buildSessionTag(kind, sessionID string) string {
	kind = strings.TrimSpace(kind)
	sessionID = strings.TrimSpace(sessionID)
	if kind == "" {
		kind = "session"
	}
	tag := fmt.Sprintf("swarm-manager:session:%s", kind)
	if sessionID != "" {
		tag = fmt.Sprintf("%s:%s", tag, sessionID)
	}
	return tag
}

func buildSessionTitle(kind, sessionID string) string {
	label := strings.TrimSpace(sessionID)
	if label == "" {
		label = "session"
	}
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "meta_orchestration":
		return "Meta-orchestration: " + label
	case "operating_mode_authoring":
		return "Operating mode authoring: " + label
	case "swarm_operations":
		return "Swarm operations: " + label
	default:
		return "Agent session: " + label
	}
}

// maxTaskDescriptionLen is the agent-manager limit for task descriptions (64KB).
const maxTaskDescriptionLen = 65536

// truncateDescription ensures the description fits within agent-manager's
// limit. The full prompt is still sent via the CreateRunRequest.Prompt field,
// so the agent receives the complete text regardless of truncation here.
func truncateDescription(desc string) string {
	if len(desc) <= maxTaskDescriptionLen {
		return desc
	}
	const suffix = "\n\n[truncated — full prompt provided via run request]"
	return desc[:maxTaskDescriptionLen-len(suffix)] + suffix
}
