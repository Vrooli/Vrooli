package planmodel

import "strings"

const (
	noContextPrefix      = "NO_CONTEXT:"
	noSkillContextPrefix = "NO_SKILL_CONTEXT:"
	noCodeRefsPrefix     = "NO_CODE_REFS:"
)

// AuthoredPhaseNoteReason is the canned placeholder reason the authoring
// projection once stamped on free-form phase notes. It carries no information,
// so the renderer omits it rather than emitting a boilerplate Reason line.
const AuthoredPhaseNoteReason = "Authored phase note."

// RelevantContextItemFromSetupLine converts a legacy/free-form setup line into
// the typed execution-context contract. It is intentionally model-owned so
// import, authoring projection, validation, and rendering classify setup items
// the same way.
func RelevantContextItemFromSetupLine(line string, scope RelevantContextScope, phaseID, reason string) RelevantContextItem {
	line = cleanSetupLine(line)
	item := RelevantContextItem{
		Kind:         RelevantContextNote,
		Scope:        scope,
		PhaseID:      phaseID,
		Label:        line,
		Reason:       strings.TrimSpace(reason),
		Instruction:  "Load or inspect this context before implementation work.",
		Target:       line,
		Required:     true,
		RepeatPolicy: defaultRelevantContextRepeatPolicy(scope),
		Source:       RelevantContextSourceMigrated,
		Status:       RelevantContextStatusReady,
	}
	lower := strings.ToLower(line)
	switch {
	case strings.HasPrefix(lower, "prompt-manager skill read "):
		item.Kind = RelevantContextSkill
		item.Command = line
		item.Argv = strings.Fields(line)
		item.Target = strings.TrimSpace(strings.TrimPrefix(line, "prompt-manager skill read "))
		item.Label = item.Target
		item.Instruction = "Load this internal skill before implementation."
	case strings.HasPrefix(lower, "search-hub "):
		item.Kind = RelevantContextSearch
		item.Command = line
		item.Argv = strings.Fields(line)
		item.Target = line
		item.Instruction = "Run this discovery search before implementation."
	case strings.HasPrefix(lower, "sed "):
		item.Kind = RelevantContextDoc
		item.Command = line
		item.Argv = strings.Fields(line)
		item.Target = lastField(line)
		item.Label = item.Target
		item.Instruction = "Read this document before implementation."
	case strings.HasPrefix(lower, "cli:"):
		item.Kind = RelevantContextCommand
		item.Command = strings.TrimSpace(line[len("cli:"):])
		item.Argv = strings.Fields(item.Command)
		item.Target = ""
		item.Label = item.Command
		item.Instruction = "Run this command before implementation."
	case strings.HasPrefix(lower, "docs/") || strings.HasPrefix(lower, "scenarios/") || strings.HasPrefix(lower, "packages/") || strings.HasSuffix(lower, ".md"):
		item.Kind = RelevantContextDoc
		item.Instruction = "Read this document before implementation."
	case strings.HasPrefix(lower, "[req:") || strings.HasPrefix(lower, "req:") || strings.Contains(lower, "requirements/"):
		item.Kind = RelevantContextReqRef
		item.Target = targetFromReferenceLikeLabel(line)
		item.Label = item.Target
		item.Instruction = "Inspect this requirement before implementation."
	case strings.HasPrefix(lower, "[code:") || strings.HasPrefix(lower, "code:"):
		item.Kind = RelevantContextCodeRef
		item.Target = targetFromReferenceLikeLabel(line)
		item.Label = item.Target
		item.Instruction = "Inspect this code reference before implementation."
	default:
		item.Instruction = line
		item.Target = ""
	}
	return item
}

// HasGlobalContextOrNoContextReason reports whether plan-wide setup has been
// explicitly addressed by concrete context or by a NO_CONTEXT reason.
func HasGlobalContextOrNoContextReason(items []RelevantContextItem) bool {
	for _, item := range items {
		if item.Scope != "" && item.Scope != RelevantContextScopeGlobal {
			continue
		}
		if IsNoContextItem(item) || contextItemHasPayload(item) {
			return true
		}
	}
	return false
}

// HasGlobalSkillContextOrNoSkillReason reports whether internal skill setup was
// made explicit: a global skill item, NO_SKILL_CONTEXT, or the stronger
// NO_CONTEXT reason.
func HasGlobalSkillContextOrNoSkillReason(items []RelevantContextItem) bool {
	for _, item := range items {
		if item.Scope != "" && item.Scope != RelevantContextScopeGlobal {
			continue
		}
		if item.Kind == RelevantContextSkill && contextItemHasPayload(item) {
			return true
		}
		if IsNoSkillContextItem(item) || IsNoContextItem(item) {
			return true
		}
	}
	return false
}

// HasPhaseContextOrNoContextReason reports whether phase setup has been
// explicitly addressed by concrete context, legacy required reading, or a
// NO_CONTEXT reason.
func HasPhaseContextOrNoContextReason(phase Phase) bool {
	if len(nonBlankStrings(phase.RequiredReading)) > 0 {
		return true
	}
	for _, item := range phase.RelevantContext {
		if IsNoContextItem(item) || contextItemHasPayload(item) {
			return true
		}
	}
	for _, reminder := range phase.Reminders {
		if hasPrefixFold(reminder, noContextPrefix) {
			return true
		}
	}
	return false
}

// HasPlanReferenceOrNoCodeReason reports whether plan-level connected
// references are present or an explicit no-code/operator-only escape exists.
func HasPlanReferenceOrNoCodeReason(p Plan) bool {
	if len(p.References) > 0 {
		return true
	}
	return NoCodeRefsReason(p.Constraints) != "" || strings.TrimSpace(p.ChangeBoundary.OperatorOnlyReason) != ""
}

// HasPhaseReferenceOrNoCodeReason reports whether a phase has connected
// references or an explicit no-code reason preserved in authored reminders.
func HasPhaseReferenceOrNoCodeReason(phase Phase) bool {
	if len(phase.References) > 0 {
		return true
	}
	for _, reminder := range phase.Reminders {
		if NoCodeRefsReason(reminder) != "" || strings.HasPrefix(strings.ToLower(strings.TrimSpace(reminder)), "no connected code references:") {
			return true
		}
	}
	return false
}

// IsNoContextItem reports whether a context item is the typed preservation of a
// NO_CONTEXT reason rather than a normal setup action.
func IsNoContextItem(item RelevantContextItem) bool {
	return contextItemHasPrefix(item, noContextPrefix)
}

// IsNoSkillContextItem reports whether a context item is the typed preservation
// of a NO_SKILL_CONTEXT reason.
func IsNoSkillContextItem(item RelevantContextItem) bool {
	return contextItemHasPrefix(item, noSkillContextPrefix)
}

func NoCodeRefsReason(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if hasPrefixFold(line, noCodeRefsPrefix) {
			return strings.TrimSpace(line[len(noCodeRefsPrefix):])
		}
	}
	return ""
}

func cleanSetupLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "`")
	return strings.TrimSpace(line)
}

func contextItemHasPrefix(item RelevantContextItem, prefix string) bool {
	for _, value := range []string{item.Label, item.Reason, item.Instruction, item.Target, item.Command} {
		if hasPrefixFold(value, prefix) {
			return true
		}
	}
	return false
}

func hasPrefixFold(value, prefix string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(value)), prefix)
}

func contextItemHasPayload(item RelevantContextItem) bool {
	return strings.TrimSpace(item.Label) != "" ||
		strings.TrimSpace(item.Target) != "" ||
		strings.TrimSpace(item.Command) != "" ||
		strings.TrimSpace(item.Instruction) != "" ||
		len(item.Argv) > 0
}

func nonBlankStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}
