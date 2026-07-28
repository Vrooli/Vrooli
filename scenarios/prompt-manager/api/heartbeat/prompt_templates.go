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

	promptSectionLabelActiveTaskBrief  = "Active Task Brief"
	promptSectionLabelChallengeReview  = "Challenge Review"
	promptSectionLabelOperatingPolicy  = "Operating Policy"
	promptSectionLabelTopicContract    = "Topic Contract"
	promptSectionLabelInboxFlow        = "Inbox Flow"
	promptSectionLabelContractFindings = "Contract Findings"
	promptSectionLabelTaskReminder     = "Task Reminder"

	promptHeadingActiveTaskBrief  = "# Active Task Brief"
	promptHeadingChallengeReview  = "# Challenge Review"
	promptHeadingOperatingPolicy  = "# Operating Policy"
	promptHeadingTopicContract    = "# Topic Contract"
	promptHeadingInboxFlow        = "# Inbox Flow"
	promptHeadingContractFindings = "# Contract Findings"
	promptHeadingTaskReminder     = "# Task Reminder"
	promptHeadingHeartbeatTask    = "# Heartbeat Task (HEARTBEAT.md)"
)

// promptSectionKind describes a stable section identity emitted by the
// heartbeat prompt. The registry prevents a renderer from introducing an
// untracked section kind that clients cannot reliably interpret.
type promptSectionKind struct {
	Label string
}

var promptSectionKinds = map[string]promptSectionKind{
	promptSectionKindAgentFile:        {Label: "Agent File"},
	promptSectionKindActiveTaskBrief:  {Label: promptSectionLabelActiveTaskBrief},
	promptSectionKindTeamInbox:        {Label: "Team Inbox"},
	promptSectionKindLastHandoff:      {Label: "Previous Handoff"},
	promptSectionKindChallengeReview:  {Label: promptSectionLabelChallengeReview},
	promptSectionKindStorageMap:       {Label: "Storage Map"},
	promptSectionKindOrgContext:       {Label: "Team Org Context"},
	promptSectionKindOperatingPolicy:  {Label: promptSectionLabelOperatingPolicy},
	promptSectionKindTopicContract:    {Label: promptSectionLabelTopicContract},
	promptSectionKindInboxFlow:        {Label: promptSectionLabelInboxFlow},
	promptSectionKindContractFindings: {Label: promptSectionLabelContractFindings},
	promptSectionKindResponsibilities: {Label: "RESPONSIBILITIES.md"},
	promptSectionKindHeartbeatTask:    {Label: "Heartbeat Task"},
	promptSectionKindTaskReminder:     {Label: promptSectionLabelTaskReminder},
}

func validatePromptSections(sections []PromptSection) error {
	for _, section := range sections {
		if _, ok := promptSectionKinds[section.Kind]; !ok {
			return fmt.Errorf("unregistered prompt section kind %q", section.Kind)
		}
	}
	return nil
}
