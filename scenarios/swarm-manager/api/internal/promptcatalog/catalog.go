// Package promptcatalog defines the canonical prompt inventory used by
// swarm-manager runtime flows and the Prompt Center.
//
// DOC: docs/concepts/ARCHITECTURE.md#physical-structure
// DOC: docs/internal/SEAMS.md#prompt-catalog-boundary
package promptcatalog

import (
	"strconv"
	"strings"
)

type Group string

const (
	GroupCapture   Group = "capture"
	GroupBacklog   Group = "backlog"
	GroupExecution Group = "execution"
	GroupArchive   Group = "archive"
	GroupSupport   Group = "support"
)

type UsageType string

const (
	UsageDirectRuntime    UsageType = "direct_runtime"
	UsageGeneratedRuntime UsageType = "generated_runtime"
	UsageSupportReference UsageType = "support_reference"
)

type SourceType string

const (
	SourceSkill     SourceType = "skill"
	SourceGenerated SourceType = "generated"
)

// Entry describes one prompt path or one support/reference skill that affects
// swarm-manager prompt behavior.
type Entry struct {
	ID                string     `json:"id"`
	Title             string     `json:"title"`
	Group             Group      `json:"group"`
	UsageType         UsageType  `json:"usage_type"`
	SourceType        SourceType `json:"source_type"`
	Trigger           string     `json:"trigger"`
	SkillID           string     `json:"skill_id,omitempty"`
	Builder           string     `json:"builder,omitempty"`
	BacklogKinds      []string   `json:"backlog_kinds,omitempty"`
	Modes             []string   `json:"modes,omitempty"`
	Operations        []string   `json:"operations,omitempty"`
	Purpose           string     `json:"purpose"`
	OutputPaths       []string   `json:"output_paths,omitempty"`
	VariableKeys      []string   `json:"variable_keys,omitempty"`
	ReferenceSkillIDs []string   `json:"reference_skill_ids,omitempty"`
}

var entries = []Entry{
	{
		ID:           "capture-classify",
		Title:        "Capture Classification",
		Group:        GroupCapture,
		UsageType:    UsageDirectRuntime,
		SourceType:   SourceSkill,
		Trigger:      "Capture classify action",
		SkillID:      "swarm-manager-classify-capture",
		Purpose:      "Analyze raw capture text and classify it into suggested backlog items.",
		OutputPaths:  []string{"classification.json", "capture.json"},
		VariableKeys: []string{"CAPTURE_ID", "CAPTURE_TEXT"},
	},
	{
		ID:           "backlog-workshop",
		Title:        "Backlog Workshop",
		Group:        GroupBacklog,
		UsageType:    UsageDirectRuntime,
		SourceType:   SourceSkill,
		Trigger:      "Backlog workshop round",
		SkillID:      "swarm-manager-workshop",
		BacklogKinds: []string{"idea", "fix", "execute", "chore"},
		Modes:        []string{"workshop"},
		Purpose:      "Run one workshop round for non-research backlog items and update plan.md.",
		OutputPaths:  []string{"workshop/round-NNN.json", "plan.md"},
		VariableKeys: []string{"ITEM_DESCRIPTION", "ITEM_FOLDER", "ITEM_KIND", "ITEM_NAME", "ITEM_PRIORITY", "ITEM_TAGS", "ITEM_TITLE", "ROUND_NUMBER"},
		ReferenceSkillIDs: []string{
			"swarm-manager-backlog-tools",
			"implementation-plan-authoring",
			"plan-skill-discovery",
		},
	},
	{
		ID:           "backlog-workshop-research",
		Title:        "Backlog Workshop (Research)",
		Group:        GroupBacklog,
		UsageType:    UsageDirectRuntime,
		SourceType:   SourceSkill,
		Trigger:      "Backlog workshop round",
		SkillID:      "swarm-manager-workshop-research",
		BacklogKinds: []string{"research"},
		Modes:        []string{"workshop"},
		Purpose:      "Run one workshop round for research backlog items and update conclusion.md.",
		OutputPaths:  []string{"workshop/round-NNN.json", "conclusion.md"},
		VariableKeys: []string{"ITEM_DESCRIPTION", "ITEM_FOLDER", "ITEM_NAME", "ITEM_PRIORITY", "ITEM_TAGS", "ITEM_TITLE", "ROUND_NUMBER"},
		ReferenceSkillIDs: []string{
			"swarm-manager-backlog-tools",
			"research-conclusion-authoring",
		},
	},
	{
		ID:           "backlog-initialize",
		Title:        "Backlog Initialize",
		Group:        GroupBacklog,
		UsageType:    UsageDirectRuntime,
		SourceType:   SourceSkill,
		Trigger:      "Backlog initialize action",
		SkillID:      "swarm-manager-initialize-backlog",
		BacklogKinds: []string{"idea", "research", "fix", "execute", "chore"},
		Modes:        []string{"initialize"},
		Purpose:      "Bootstrap a backlog item with its first deliverable draft and first workshop round.",
		OutputPaths:  []string{"workshop/round-001.json", "plan.md", "conclusion.md"},
		VariableKeys: []string{"ITEM_DESCRIPTION", "ITEM_FOLDER", "ITEM_KIND", "ITEM_NAME", "ITEM_PRIORITY", "ITEM_TAGS", "ITEM_TITLE"},
		ReferenceSkillIDs: []string{
			"swarm-manager-backlog-tools",
			"implementation-plan-authoring",
			"swarm-manager-processing-guidance",
		},
	},
	{
		ID:           "backlog-finalize",
		Title:        "Backlog Finalize",
		Group:        GroupBacklog,
		UsageType:    UsageDirectRuntime,
		SourceType:   SourceSkill,
		Trigger:      "Backlog finalize action",
		SkillID:      "swarm-manager-workshop-finalize",
		BacklogKinds: []string{"idea", "fix", "execute", "chore"},
		Modes:        []string{"finalize"},
		Purpose:      "Fold the latest workshop answers into plan.md and write a finalize round with no new decisions.",
		OutputPaths:  []string{"workshop/round-NNN.json", "plan.md"},
		VariableKeys: []string{"ITEM_DESCRIPTION", "ITEM_FOLDER", "ITEM_KIND", "ITEM_NAME", "ITEM_PRIORITY", "ITEM_TAGS", "ITEM_TITLE", "ROUND_NUMBER"},
		ReferenceSkillIDs: []string{
			"swarm-manager-backlog-tools",
			"implementation-plan-authoring",
			"plan-skill-discovery",
		},
	},
	{
		ID:           "backlog-finalize-research",
		Title:        "Backlog Finalize (Research)",
		Group:        GroupBacklog,
		UsageType:    UsageDirectRuntime,
		SourceType:   SourceSkill,
		Trigger:      "Backlog finalize action",
		SkillID:      "swarm-manager-workshop-research-finalize",
		BacklogKinds: []string{"research"},
		Modes:        []string{"finalize"},
		Purpose:      "Fold the latest workshop answers into conclusion.md and write a finalize round with no new decisions.",
		OutputPaths:  []string{"workshop/round-NNN.json", "conclusion.md"},
		VariableKeys: []string{"ITEM_DESCRIPTION", "ITEM_FOLDER", "ITEM_NAME", "ITEM_PRIORITY", "ITEM_TAGS", "ITEM_TITLE", "ROUND_NUMBER"},
		ReferenceSkillIDs: []string{
			"swarm-manager-backlog-tools",
			"research-conclusion-authoring",
		},
	},
	{
		ID:           "backlog-clarify",
		Title:        "Backlog Decision Clarification",
		Group:        GroupBacklog,
		UsageType:    UsageDirectRuntime,
		SourceType:   SourceSkill,
		Trigger:      "Decision clarification request",
		SkillID:      "swarm-manager-workshop-clarify",
		BacklogKinds: []string{"idea", "research", "fix", "execute", "chore"},
		Modes:        []string{"clarify"},
		Purpose:      "Clarify a workshop decision item and assess impact on the workshop round.",
		OutputPaths:  []string{},
		VariableKeys: []string{"CLARIFICATION_HISTORY", "DECISION_CONTEXT", "DECISION_OPTIONS", "DECISION_TOPIC", "ITEM_DESCRIPTION", "ITEM_KIND", "ITEM_NAME", "ITEM_TITLE", "USER_QUESTION", "WORKSHOP_HISTORY"},
		ReferenceSkillIDs: []string{
			"swarm-manager-backlog-tools",
		},
	},
	{
		ID:          "execution-process",
		Title:       "Execution Process Prompt",
		Group:       GroupExecution,
		UsageType:   UsageGeneratedRuntime,
		SourceType:  SourceGenerated,
		Trigger:     "Execution start / retry",
		Builder:     "execution.buildExecutionPrompt",
		Operations:  []string{"generator", "improver"},
		Purpose:     "Build the runtime execution prompt from the backlog deliverable (plan.md or conclusion.md).",
		OutputPaths: []string{"plan.md", "conclusion.md"},
	},
	{
		ID:          "execution-fixup",
		Title:       "Execution Fixup Prompt",
		Group:       GroupExecution,
		UsageType:   UsageGeneratedRuntime,
		SourceType:  SourceGenerated,
		Trigger:     "Automatic or manual fixup",
		Builder:     "execution.buildExecutionPrompt",
		Operations:  []string{"fixup"},
		Purpose:     "Build a fixup prompt by combining the deliverable with review feedback.",
		OutputPaths: []string{"plan.md", "conclusion.md"},
	},
	{
		ID:          "execution-followup",
		Title:       "Execution Follow-up Prompt",
		Group:       GroupExecution,
		UsageType:   UsageGeneratedRuntime,
		SourceType:  SourceGenerated,
		Trigger:     "Execution follow-up or custom continuation",
		Builder:     "execution.buildExecutionPrompt",
		Operations:  []string{"followup", "custom"},
		Purpose:     "Build a follow-up prompt by combining the deliverable with optional operator context.",
		OutputPaths: []string{"plan.md", "conclusion.md"},
	},
	{
		ID:           "execution-review-agent",
		Title:        "Execution Review Agent",
		Group:        GroupExecution,
		UsageType:    UsageDirectRuntime,
		SourceType:   SourceSkill,
		Trigger:      "Post-finalization evidence gathering",
		SkillID:      "swarm-manager-review",
		Operations:   []string{"review"},
		Purpose:      "Gather typed evidence (screenshots, API tests, CLI output) after execution finalization.",
		OutputPaths:  []string{"review/round-NNN.json", "review/captures/*"},
		VariableKeys: []string{"ITEM_FOLDER", "PLAN_CONTENT", "DIFF_SUMMARY", "CHANGED_PATHS", "AFFECTED_SCENARIOS", "ROUND_NUMBER", "USER_REQUEST"},
	},
	{
		ID:           "archive-spec-sync",
		Title:        "Scenario Spec Sync",
		Group:        GroupArchive,
		UsageType:    UsageDirectRuntime,
		SourceType:   SourceSkill,
		Trigger:      "Scenario spec-sync-archive",
		SkillID:      "spec-sync",
		Purpose:      "Synchronize scenario specs before archive and deletion.",
		OutputPaths:  []string{"PRD.md", "requirements/", "README.md", "docs/"},
		VariableKeys: []string{"TARGET"},
	},
	{
		ID:           "support-backlog-tools",
		Title:        "Backlog Tools Reference",
		Group:        GroupSupport,
		UsageType:    UsageSupportReference,
		SourceType:   SourceSkill,
		Trigger:      "Referenced by backlog prompt skills",
		SkillID:      "swarm-manager-backlog-tools",
		Purpose:      "Canonical backlog folder structure, artifact schemas, and CLI interaction rules.",
		VariableKeys: []string{"ITEM_FOLDER"},
	},
	{
		ID:           "support-processing-guidance",
		Title:        "Processing Guidance Reference",
		Group:        GroupSupport,
		UsageType:    UsageSupportReference,
		SourceType:   SourceSkill,
		Trigger:      "Referenced by initialize and processing-oriented prompt skills",
		SkillID:      "swarm-manager-processing-guidance",
		Purpose:      "Shared processing workflow guidance for backlog items.",
		VariableKeys: []string{"ITEM_FOLDER"},
	},
	{
		ID:         "support-implementation-plan-authoring",
		Title:      "Implementation Plan Authoring Reference",
		Group:      GroupSupport,
		UsageType:  UsageSupportReference,
		SourceType: SourceSkill,
		Trigger:    "Referenced by non-research backlog prompt skills",
		SkillID:    "implementation-plan-authoring",
		Purpose:    "Canonical plan structure, convergence rules, and quality gates for plan.md.",
	},
	{
		ID:         "support-research-conclusion-authoring",
		Title:      "Research Conclusion Authoring Reference",
		Group:      GroupSupport,
		UsageType:  UsageSupportReference,
		SourceType: SourceSkill,
		Trigger:    "Referenced by research backlog prompt skills",
		SkillID:    "research-conclusion-authoring",
		Purpose:    "Canonical conclusion structure and quality gates for conclusion.md.",
	},
	{
		ID:         "support-plan-skill-discovery",
		Title:      "Plan Skill Discovery Reference",
		Group:      GroupSupport,
		UsageType:  UsageSupportReference,
		SourceType: SourceSkill,
		Trigger:    "Referenced by workshop and finalize prompt skills",
		SkillID:    "plan-skill-discovery",
		Purpose:    "Guidance for discovering and embedding relevant skills into a plan.",
	},
}

func Entries() []Entry {
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, cloneEntry(entry))
	}
	return result
}

func SkillEntries() []Entry {
	result := make([]Entry, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.SourceType != SourceSkill || strings.TrimSpace(entry.SkillID) == "" {
			continue
		}
		if _, ok := seen[entry.SkillID]; ok {
			continue
		}
		seen[entry.SkillID] = struct{}{}
		result = append(result, cloneEntry(entry))
	}
	return result
}

func Lookup(id string) (Entry, bool) {
	normalized := strings.TrimSpace(id)
	for _, entry := range entries {
		if entry.ID == normalized {
			return cloneEntry(entry), true
		}
	}
	return Entry{}, false
}

func ResolveBacklogSkill(mode, kind string) (Entry, bool) {
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	normalizedKind := strings.ToLower(strings.TrimSpace(kind))
	for _, entry := range entries {
		if entry.Group != GroupBacklog || entry.SourceType != SourceSkill || entry.UsageType != UsageDirectRuntime {
			continue
		}
		if !contains(entry.Modes, normalizedMode) || !contains(entry.BacklogKinds, normalizedKind) {
			continue
		}
		return cloneEntry(entry), true
	}
	return Entry{}, false
}

func ResolveCaptureSkill() (Entry, bool) {
	return Lookup("capture-classify")
}

func ResolveSpecSyncSkill() (Entry, bool) {
	return Lookup("archive-spec-sync")
}

func IsKnownSkillID(skillID string) bool {
	_, ok := skillEntry(skillID)
	return ok
}

func VariableKeysForSkill(skillID string) []string {
	entry, ok := skillEntry(skillID)
	if !ok {
		return nil
	}
	return append([]string{}, entry.VariableKeys...)
}

func SkillUsageType(skillID string) UsageType {
	entry, ok := skillEntry(skillID)
	if !ok {
		return ""
	}
	return entry.UsageType
}

func SkillGroups(skillID string) []string {
	entry, ok := skillEntry(skillID)
	if !ok {
		return nil
	}
	return []string{string(entry.Group)}
}

func SkillUsageCount(skillID string) int {
	entry, ok := skillEntry(skillID)
	if !ok {
		return 0
	}
	switch entry.UsageType {
	case UsageSupportReference:
		count := 0
		for _, candidate := range entries {
			if candidate.UsageType != UsageDirectRuntime {
				continue
			}
			if contains(candidate.ReferenceSkillIDs, entry.SkillID) {
				count++
			}
		}
		return count
	default:
		count := 0
		for _, candidate := range entries {
			if candidate.SourceType != SourceSkill || candidate.UsageType != UsageDirectRuntime {
				continue
			}
			if candidate.SkillID == entry.SkillID {
				count++
			}
		}
		return count
	}
}

func SkillImpactSummary(skillID string) string {
	entry, ok := skillEntry(skillID)
	if !ok {
		return "Not referenced by the prompt catalog."
	}
	count := SkillUsageCount(skillID)
	switch entry.UsageType {
	case UsageSupportReference:
		if count == 1 {
			return "Referenced by 1 runtime prompt path."
		}
		return "Referenced by " + itoa(count) + " runtime prompt paths."
	default:
		if count == 1 {
			return "Used directly by 1 runtime prompt path."
		}
		return "Used directly by " + itoa(count) + " runtime prompt paths."
	}
}

func skillEntry(skillID string) (Entry, bool) {
	normalized := strings.TrimSpace(skillID)
	for _, entry := range entries {
		if entry.SourceType != SourceSkill {
			continue
		}
		if entry.SkillID == normalized {
			return cloneEntry(entry), true
		}
	}
	return Entry{}, false
}

func cloneEntry(entry Entry) Entry {
	entry.BacklogKinds = append([]string{}, entry.BacklogKinds...)
	entry.Modes = append([]string{}, entry.Modes...)
	entry.Operations = append([]string{}, entry.Operations...)
	entry.OutputPaths = append([]string{}, entry.OutputPaths...)
	entry.VariableKeys = append([]string{}, entry.VariableKeys...)
	entry.ReferenceSkillIDs = append([]string{}, entry.ReferenceSkillIDs...)
	return entry
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
