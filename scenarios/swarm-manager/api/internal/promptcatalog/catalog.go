// Package promptcatalog defines the canonical prompt inventory used by
// swarm-manager runtime flows and the Prompt Center.
//
// DOC: docs/concepts/ARCHITECTURE.md#physical-structure
// DOC: docs/internal/SEAMS.md#prompt-catalog-boundary
package promptcatalog

import (
	"strings"
)

type Group string

const (
	GroupCapture   Group = "capture"
	GroupBacklog   Group = "backlog"
	GroupExecution Group = "execution"
	GroupArchive   Group = "archive"
	GroupSupport   Group = "support"
	GroupGoal      Group = "goal"
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
	ExperimentID      string     `json:"experiment_id,omitempty"`
}

var staticEntries = []Entry{
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
		ID:          "execution-process",
		Title:       "Execution Process Prompt",
		Group:       GroupExecution,
		UsageType:   UsageGeneratedRuntime,
		SourceType:  SourceGenerated,
		Trigger:     "Execution start / retry",
		Builder:     "execution.buildExecutionPrompt",
		Operations:  []string{"generator", "improver"},
		Purpose:     "Build the runtime execution prompt from the rendered plan_ref or research conclusion.",
		OutputPaths: []string{"conclusion.md"},
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
		OutputPaths: []string{"conclusion.md"},
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
		OutputPaths: []string{"conclusion.md"},
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
		VariableKeys: []string{"ITEM_FOLDER", "PLAN_CONTENT", "DIFF_SUMMARY", "CHANGED_PATHS", "AFFECTED_SCENARIOS", "ROUND_NUMBER", "USER_REQUEST", "GCT_REVIEW_RESULTS"},
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
		Purpose:    "Canonical plan-manager authoring, convergence rules, and quality gates.",
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
		ID:         "session-proposals",
		Title:      "Session-backed Proposal Agent",
		Group:      GroupGoal,
		UsageType:  UsageDirectRuntime,
		SourceType: SourceSkill,
		Trigger:    "A session is launched against a goal or an item in its derived scope",
		SkillID:    "swarm-manager-proposals",
		Purpose:    "Produce an operator-reviewed mutation-list proposal from hydrated goal context, including item renewal and scoped dependency changes when stale work needs controlled renewal.",
		ReferenceSkillIDs: []string{
			"swarm-manager-backlog-tools",
			"swarm-manager-goal-context",
			"implementation-plan-authoring",
		},
	},
	{
		ID:         "support-goal-context",
		Title:      "Goal Context Reference",
		Group:      GroupSupport,
		UsageType:  UsageSupportReference,
		SourceType: SourceSkill,
		Trigger:    "Referenced by goal-scoped prompt skills and milestone review",
		SkillID:    "swarm-manager-goal-context",
		Purpose:    "Goal-derived scope, milestone partitioning, and read-only CLI context loading.",
	},
}

func Entries() []Entry {
	entries := catalogEntries()
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, cloneEntry(entry))
	}
	return result
}

func SkillEntries() []Entry {
	entries := catalogEntries()
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
	for _, entry := range catalogEntries() {
		if entry.ID == normalized {
			return cloneEntry(entry), true
		}
	}
	return Entry{}, false
}

func ResolveCaptureSkill() (Entry, bool) {
	return Lookup("capture-classify")
}

func ResolveSpecSyncSkill() (Entry, bool) {
	return Lookup("archive-spec-sync")
}

func catalogEntries() []Entry {
	entries := make([]Entry, 0, len(staticEntries))
	for _, entry := range staticEntries {
		entries = append(entries, entry)
	}
	return entries
}
