package agentmanager

import (
	"fmt"
	"strings"

	"github.com/vrooli/api-core/storage"
)

// buildResearchTag composes the agent-manager run tag for an idea-research run.
// It is variant-aware (Baseline Modes P5): under the live instance the tag is
// "swarm-manager:idea:<idea>:research" (byte-identical to the pre-adoption
// form), while a shadow instance tags its runs "swarm-manager_shadow:idea:...",
// so a shadow's research runs are never confused with live's when enumerated by
// tag. storage.RedisKey is the right helper because the key interleaves a
// dynamic token (the idea name) mid-string. It only errors outside the
// variant-aware lifecycle (no identity env), which is necessarily live; the
// caller propagates that rather than silently aliasing onto live.
func buildResearchTag(ideaName string) (string, error) {
	ideaName = strings.TrimSpace(ideaName)
	if ideaName == "" {
		return storage.RedisKey("idea", "research")
	}
	return storage.RedisKey("idea", ideaName, "research")
}

func buildResearchTitle(mode, ideaName string) string {
	label := strings.TrimSpace(ideaName)
	if label == "" {
		label = "idea"
	}
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "clarify":
		return "Clarify idea: " + label
	case "suggest":
		return "Suggest improvements: " + label
	case "enhance":
		return "Enhance idea: " + label
	default:
		return "Research idea: " + label
	}
}

func buildInitiativeTag(name, purpose string, round int) string {
	name = strings.TrimSpace(name)
	purpose = strings.TrimSpace(purpose)
	if name == "" {
		name = "initiative"
	}
	tag := fmt.Sprintf("swarm-manager:initiative:%s", name)
	if purpose != "" {
		tag = fmt.Sprintf("%s:%s", tag, purpose)
	}
	if round > 0 {
		tag = fmt.Sprintf("%s:round-%03d", tag, round)
	}
	return tag
}

func buildInitiativeTitle(name, purpose string, round int) string {
	label := strings.TrimSpace(name)
	if label == "" {
		label = "initiative"
	}
	switch strings.ToLower(strings.TrimSpace(purpose)) {
	case "feedback":
		if round > 0 {
			return fmt.Sprintf("Feedback round %d: %s", round, label)
		}
		return "Feedback: " + label
	case "feedback_continue":
		if round > 0 {
			return fmt.Sprintf("Feedback round %d (continue): %s", round, label)
		}
		return "Feedback continue: " + label
	case "review":
		return "Review: " + label
	}
	return "Initiative: " + label
}

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

func buildBacklogTag(kind, name, purpose string) string {
	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	purpose = strings.TrimSpace(purpose)
	if kind == "" {
		kind = "backlog"
	}
	tag := fmt.Sprintf("swarm-manager:backlog:%s", kind)
	if name != "" {
		tag = fmt.Sprintf("%s:%s", tag, name)
	}
	if purpose != "" {
		tag = fmt.Sprintf("%s:%s", tag, purpose)
	}
	return tag
}

func buildBacklogTitle(kind, name, purpose string) string {
	label := strings.TrimSpace(name)
	if label == "" {
		label = "backlog item"
	}
	if strings.TrimSpace(purpose) != "" {
		return fmt.Sprintf("%s: %s", capitalizeLabel(strings.TrimSpace(purpose)), label)
	}
	if strings.TrimSpace(kind) != "" {
		return fmt.Sprintf("Backlog %s: %s", strings.TrimSpace(kind), label)
	}
	return "Backlog item: " + label
}

func capitalizeLabel(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
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
