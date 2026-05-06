package heartbeat

import (
	"fmt"
	"sort"
	"strings"

	"prompt-manager/memberflow"
)

type topicContractInputs struct {
	teamID     string
	agentID    string
	memberFlow memberflow.MemberTopics
}

func LoadTopicContractInputs(storeDir, teamID, agentID string) (*topicContractInputs, error) {
	mt, err := memberflow.LoadMember(storeDir, teamID, agentID)
	if err != nil {
		return nil, fmt.Errorf("load topics.json: %w", err)
	}
	return &topicContractInputs{
		teamID:     teamID,
		agentID:    agentID,
		memberFlow: mt,
	}, nil
}

func RenderTopicContract(in *topicContractInputs) string {
	if in == nil {
		return ""
	}
	topics := in.memberFlow.Topics

	var b strings.Builder
	b.WriteString(promptHeadingTopicContract + "\n\n")
	b.WriteString("This section is generated from `topics.json`. It is the source of truth for topic reads, writes, decisions, and capability-gap routing.\n")
	if topics.IsEmpty() {
		b.WriteString("\nNo topic flow is declared for this member.")
		return b.String()
	}

	renderTopicContractIntakes(&b, topics.Intake)
	renderTopicContractRequiredReads(&b, topics.RequiredRead)
	renderTopicContractEvidence(&b, topics.EvidenceConsumed)
	renderTopicContractOutputs(&b, topics.Output)
	renderTopicContractDecisions(&b, topics)
	renderTopicContractExternalProducers(&b, topics.ExternalProducers)

	return strings.TrimRight(b.String(), "\n")
}

func renderTopicContractIntakes(b *strings.Builder, entries []memberflow.IntakeEntry) {
	if len(entries) == 0 {
		return
	}
	entries = append([]memberflow.IntakeEntry(nil), entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Prefix < entries[j].Prefix })

	b.WriteString("\n## Inboxes Drained\n\n")
	for _, e := range entries {
		parts := []string{"taxonomy `" + emptyAs(e.Taxonomy, "none") + "`"}
		if strings.TrimSpace(e.ClassifierSkill) != "" {
			parts = append(parts, "classifier `"+e.ClassifierSkill+"`")
		}
		if e.SourceTeam != nil && strings.TrimSpace(*e.SourceTeam) != "" {
			parts = append(parts, "source team `"+*e.SourceTeam+"`")
		}
		b.WriteString(fmt.Sprintf("- `%s` - %s\n", e.Prefix, strings.Join(parts, ", ")))
	}
}

func renderTopicContractRequiredReads(b *strings.Builder, entries []memberflow.RequiredReadEntry) {
	if len(entries) == 0 {
		return
	}
	entries = append([]memberflow.RequiredReadEntry(nil), entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Prefix < entries[j].Prefix })

	b.WriteString("\n## Required Reads\n\n")
	for _, e := range entries {
		line := "- `" + e.Prefix + "`"
		if e.SourceTeam != nil && strings.TrimSpace(*e.SourceTeam) != "" {
			line += " - source team `" + *e.SourceTeam + "`"
		}
		if strings.TrimSpace(e.Comment) != "" {
			line += " - " + strings.TrimSpace(e.Comment)
		}
		b.WriteString(line + "\n")
	}
}

func renderTopicContractEvidence(b *strings.Builder, entries []memberflow.EvidenceConsumedEntry) {
	if len(entries) == 0 {
		return
	}
	entries = append([]memberflow.EvidenceConsumedEntry(nil), entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Prefix < entries[j].Prefix })

	b.WriteString("\n## Evidence Consumed\n\n")
	for _, e := range entries {
		decisions := append([]string(nil), e.ForDecisions...)
		sort.Strings(decisions)
		line := fmt.Sprintf("- `%s` - general evidence", e.Prefix)
		if len(decisions) > 0 {
			line = fmt.Sprintf("- `%s` - for `%s`", e.Prefix, strings.Join(decisions, "`, `"))
		}
		if e.SourceTeam != nil && strings.TrimSpace(*e.SourceTeam) != "" {
			line += "; source team `" + *e.SourceTeam + "`"
		}
		b.WriteString(line + "\n")
	}
}

func renderTopicContractOutputs(b *strings.Builder, entries []memberflow.OutputEntry) {
	if len(entries) == 0 {
		return
	}
	entries = append([]memberflow.OutputEntry(nil), entries...)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Prefix != entries[j].Prefix {
			return entries[i].Prefix < entries[j].Prefix
		}
		return string(entries[i].DestinationKind) < string(entries[j].DestinationKind)
	})

	b.WriteString("\n## Outputs\n\n")
	for _, e := range entries {
		parts := []string{string(e.DestinationKind)}
		if strings.TrimSpace(e.Schema) != "" {
			parts = append(parts, "schema `"+e.Schema+"`")
		}
		if e.DestinationTeam != nil && strings.TrimSpace(*e.DestinationTeam) != "" {
			parts = append(parts, "team `"+*e.DestinationTeam+"`")
		}
		if e.DestinationPath != nil && strings.TrimSpace(*e.DestinationPath) != "" {
			parts = append(parts, "path `"+*e.DestinationPath+"`")
		}
		b.WriteString(fmt.Sprintf("- `%s` - %s\n", e.Prefix, strings.Join(parts, ", ")))
	}
}

func renderTopicContractDecisions(b *strings.Builder, topics memberflow.Topics) {
	if len(topics.DecisionsOwned) == 0 && len(topics.DecisionsConsumed) == 0 && !topics.RaisesCapabilityGaps {
		return
	}
	b.WriteString("\n## Decisions\n\n")
	if len(topics.DecisionsOwned) > 0 {
		owned := append([]string(nil), topics.DecisionsOwned...)
		sort.Strings(owned)
		b.WriteString("- own/propose: `" + strings.Join(owned, "`, `") + "`\n")
	}
	if len(topics.DecisionsConsumed) > 0 {
		consumed := append([]string(nil), topics.DecisionsConsumed...)
		sort.Strings(consumed)
		b.WriteString("- consume: `" + strings.Join(consumed, "`, `") + "`\n")
	}
	if topics.RaisesCapabilityGaps {
		b.WriteString("- may raise `capability-gap`: yes\n")
	}
}

func renderTopicContractExternalProducers(b *strings.Builder, producers []string) {
	producers = sortedUniqueStrings(producers)
	if len(producers) == 0 {
		return
	}
	b.WriteString("\n## External Producers\n\n")
	for _, producer := range producers {
		b.WriteString("- `" + producer + "`\n")
	}
}
