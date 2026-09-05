package heartbeat

import "fmt"

// Prompt sections follow a strict volatility gradient: universal, team, member,
// volatile, then the task. The flat prompt is wrapped in one XML context so
// providers can cache every stable band before live ledger and validation data
// changes. If a run-volatile section moves above a stable section, the first
// differing byte moves up with it and the cacheable prefix collapses. Volatility
// outranks scope when the two conflict; a changing team-shaped value belongs in
// the volatile band rather than weakening the prefix for every member.

const (
	promptSectionKindAgentFile          = "agent-file"
	promptSectionKindSharedDoctrine     = "shared-doctrine"
	promptSectionKindTeamInbox          = "team-inbox"
	promptSectionKindTeamWake           = "team-context-wake"
	promptSectionKindChallengeReview    = "challenge-review"
	promptSectionKindStorageMap         = "team-storage-map"
	promptSectionKindOrgContext         = "team-org-context"
	promptSectionKindOperatingPolicy    = "team-operating-policy"
	promptSectionKindMemberPolicy       = "member-operating-policy"
	promptSectionKindTopicContract      = "topic-contract"
	promptSectionKindInboxFlow          = "inbox-flow"
	promptSectionKindContractFindings   = "contract-findings"
	promptSectionKindResponsibilities   = "team-responsibilities"
	promptSectionKindHeartbeatTask      = "heartbeat-task"
	promptSectionKindContinuityFallback = "continuity-fallback"
)

type promptSectionScope string

const (
	promptScopeUniversal promptSectionScope = "universal"
	promptScopeTeam      promptSectionScope = "team"
	promptScopeMember    promptSectionScope = "member"
	promptScopeVolatile  promptSectionScope = "volatile"
	promptScopeTask      promptSectionScope = "task"
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
	Element string
	Scope   promptSectionScope
}

var promptSectionKinds = map[string]promptSectionKind{
	// The agent-file heading belongs to the merged block that buildSections
	// wraps around every adjacent agent-file section, not to each section.
	promptSectionKindAgentFile: {Label: "Agent File", Heading: "# Agent Files (Markdown)", Element: "agent-files", Scope: promptScopeMember},
	// Emitted first and byte-identical for every member in a given build mode.
	// Anything member-specific belongs in a later section: a single varying byte
	// here destroys the shared prefix this section exists to create.
	promptSectionKindSharedDoctrine:     {Label: "Standing Rules", Heading: "# Standing Rules", Element: "standing-rules", Scope: promptScopeUniversal},
	promptSectionKindTeamInbox:          {Label: "Team Inbox", Heading: "# Team Inbox", Element: "team-inbox", Scope: promptScopeVolatile},
	promptSectionKindTeamWake:           {Label: "Team Context Wake", Heading: "# Team Context Wake", Element: "team-context-wake", Scope: promptScopeVolatile},
	promptSectionKindChallengeReview:    {Label: "Challenge Review", Heading: "# Challenge Review", Element: "challenge-review", Scope: promptScopeVolatile},
	promptSectionKindStorageMap:         {Label: "Storage Map", Heading: "# Storage Map", Element: "storage-map", Scope: promptScopeTeam},
	promptSectionKindOrgContext:         {Label: "Team Org Context", Heading: "# Team Org Context", Element: "org-context", Scope: promptScopeMember},
	promptSectionKindOperatingPolicy:    {Label: "Operating Policy (Team)", Heading: "# Operating Policy (Team)", Element: "operating-policy-team", Scope: promptScopeTeam},
	promptSectionKindMemberPolicy:       {Label: "Operating Policy (Member)", Heading: "# Operating Policy (Member)", Element: "operating-policy-member", Scope: promptScopeMember},
	promptSectionKindTopicContract:      {Label: "Topic Contract", Heading: "# Topic Contract", Element: "topic-contract", Scope: promptScopeMember},
	promptSectionKindInboxFlow:          {Label: "Inbox Flow", Heading: "# Inbox Flow", Element: "inbox-flow", Scope: promptScopeMember},
	promptSectionKindContractFindings:   {Label: "Contract Findings", Heading: "# Contract Findings", Element: "contract-findings", Scope: promptScopeVolatile},
	promptSectionKindResponsibilities:   {Label: "RESPONSIBILITIES.md", Heading: "# Team Responsibilities (RESPONSIBILITIES.md)", Element: "responsibilities", Scope: promptScopeMember},
	promptSectionKindHeartbeatTask:      {Label: "Heartbeat Task", Heading: "# Heartbeat Task (HEARTBEAT.md)", Element: "heartbeat-task", Scope: promptScopeTask},
	promptSectionKindContinuityFallback: {Label: "Continuity Fallback", Heading: "# Continuity Fallback", Element: "continuity-fallback", Scope: promptScopeVolatile},
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

// promptPrecedenceKinds are the reference and task sections ranked by the
// standing rules. They are built from registry entries so a rename cannot leave
// the authority list pointing at a section that no longer exists.
var promptPrecedenceKinds = []string{
	promptSectionKindOperatingPolicy,
	promptSectionKindMemberPolicy,
	promptSectionKindTopicContract,
	promptSectionKindHeartbeatTask,
	promptSectionKindResponsibilities,
	promptSectionKindAgentFile,
}

func promptElement(kind string) string {
	entry, ok := promptSectionKinds[kind]
	if !ok || entry.Element == "" {
		panic(fmt.Sprintf("unregistered prompt section element %q", kind))
	}
	return entry.Element
}

func validatePromptSections(sections []PromptSection) error {
	previousScope := -1
	for _, section := range sections {
		entry, ok := promptSectionKinds[section.Kind]
		if !ok {
			return fmt.Errorf("unregistered prompt section kind %q", section.Kind)
		}
		if entry.Element == "" {
			return fmt.Errorf("prompt section kind %q has no XML element", section.Kind)
		}
		if entry.Scope == "" {
			return fmt.Errorf("prompt section kind %q has no volatility scope", section.Kind)
		}
		scope := promptScopeRank(entry.Scope)
		if scope < previousScope {
			return fmt.Errorf("prompt sections violate volatility order at %q", section.Kind)
		}
		previousScope = scope
	}
	return nil
}

func promptScopeRank(scope promptSectionScope) int {
	switch scope {
	case promptScopeUniversal:
		return 0
	case promptScopeTeam:
		return 1
	case promptScopeMember:
		return 2
	case promptScopeVolatile:
		return 3
	case promptScopeTask:
		return 4
	default:
		return -1
	}
}
