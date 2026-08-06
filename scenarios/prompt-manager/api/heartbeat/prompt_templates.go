package heartbeat

import "fmt"

const (
	promptSectionKindAgentFile        = "agent-file"
	promptSectionKindActiveTaskBrief  = "active-task-brief"
	promptSectionKindTeamInbox        = "team-inbox"
	promptSectionKindLastHandoff      = "last-handoff"
	promptSectionKindChallengeReview  = "challenge-review"
	promptSectionKindStorageMap       = "team-storage-map"
	promptSectionKindOrgContext       = "team-org-context"
	promptSectionKindOperatingPolicy  = "team-operating-policy"
	promptSectionKindTopicContract    = "topic-contract"
	promptSectionKindInboxFlow        = "inbox-flow"
	promptSectionKindContractFindings = "contract-findings"
	promptSectionKindResponsibilities = "team-responsibilities"
	promptSectionKindHeartbeatTask    = "heartbeat-task"
	promptSectionKindTaskReminder     = "task-reminder"
)

// promptSectionKind describes a stable section identity emitted by the
// heartbeat prompt. The registry prevents a renderer from introducing an
// untracked section kind that clients cannot reliably interpret.
//
// Heading is the exact level-one heading the section emits. It lives here so
// that the assembled prompt can be checked against the registry: a level-one
// heading the registry does not name is an injected document title that has
// escaped its section, which is the defect that left `# Operating Policy` and
// `# Heartbeat Task (HEARTBEAT.md)` holding no content.
//
// Label and Heading are read from here at every call site rather than copied
// into constants beside them. Heading text that lived at call sites is how the
// precedence list came to name headings the builder never emitted.
type promptSectionKind struct {
	Label   string
	Heading string
}

var promptSectionKinds = map[string]promptSectionKind{
	// The agent-file heading belongs to the merged block that buildSections
	// wraps around every adjacent agent-file section, not to each section.
	promptSectionKindAgentFile:        {Label: "Agent File", Heading: "# Agent Files (Markdown)"},
	promptSectionKindActiveTaskBrief:  {Label: "Active Task Brief", Heading: "# Active Task Brief"},
	promptSectionKindTeamInbox:        {Label: "Team Inbox", Heading: "# Team Inbox"},
	promptSectionKindLastHandoff:      {Label: "Previous Handoff", Heading: "# Previous Heartbeat Handoff"},
	promptSectionKindChallengeReview:  {Label: "Challenge Review", Heading: "# Challenge Review"},
	promptSectionKindStorageMap:       {Label: "Storage Map", Heading: "# Storage Map"},
	promptSectionKindOrgContext:       {Label: "Team Org Context", Heading: "# Team Org Context"},
	promptSectionKindOperatingPolicy:  {Label: "Operating Policy", Heading: "# Operating Policy"},
	promptSectionKindTopicContract:    {Label: "Topic Contract", Heading: "# Topic Contract"},
	promptSectionKindInboxFlow:        {Label: "Inbox Flow", Heading: "# Inbox Flow"},
	promptSectionKindContractFindings: {Label: "Contract Findings", Heading: "# Contract Findings"},
	promptSectionKindResponsibilities: {Label: "RESPONSIBILITIES.md", Heading: "# Team Responsibilities (RESPONSIBILITIES.md)"},
	promptSectionKindHeartbeatTask:    {Label: "Heartbeat Task", Heading: "# Heartbeat Task (HEARTBEAT.md)"},
	promptSectionKindTaskReminder:     {Label: "Task Reminder", Heading: "# Task Reminder"},
}

// promptHeading returns the registered level-one heading for a section kind.
// An unregistered kind panics rather than returning empty: a section with no
// heading is the failure this registry exists to prevent, and it must not be
// discoverable only by reading an assembled prompt.
func promptHeading(kind string) string {
	entry, ok := promptSectionKinds[kind]
	if !ok {
		panic(fmt.Sprintf("unregistered prompt section kind %q", kind))
	}
	return entry.Heading
}

// newPromptSection builds a section carrying its registered identity. Label and
// heading come from the registry, so adding a section means adding a registry
// entry and one call here.
func newPromptSection(kind, sourcePath, content string) PromptSection {
	entry, ok := promptSectionKinds[kind]
	if !ok {
		panic(fmt.Sprintf("unregistered prompt section kind %q", kind))
	}
	return PromptSection{
		Kind:       kind,
		Label:      entry.Label,
		SourcePath: sourcePath,
		Content:    content,
	}
}

// promptSectionHeadings returns every level-one heading the builder may emit.
func promptSectionHeadings() map[string]string {
	headings := make(map[string]string, len(promptSectionKinds))
	for kind, entry := range promptSectionKinds {
		headings[entry.Heading] = kind
	}
	return headings
}

// promptPrecedenceKinds are the section kinds the `# Active Task Brief`
// precedence list ranks, in rank order. The list is built from these rather
// than from hand-typed heading strings, so a heading rename cannot leave the
// precedence list pointing at a section that no longer exists.
var promptPrecedenceKinds = []string{
	promptSectionKindActiveTaskBrief,
	promptSectionKindOperatingPolicy,
	promptSectionKindTopicContract,
	promptSectionKindHeartbeatTask,
	promptSectionKindResponsibilities,
	promptSectionKindAgentFile,
}

func validatePromptSections(sections []PromptSection) error {
	for _, section := range sections {
		if _, ok := promptSectionKinds[section.Kind]; !ok {
			return fmt.Errorf("unregistered prompt section kind %q", section.Kind)
		}
	}
	return nil
}
