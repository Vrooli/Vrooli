package memberflow

import (
	"fmt"
	"sort"
	"strings"
)

const TopicContractHeading = "# Topic Contract"

func RenderTopicContract(teamID, agentID string, memberFlow MemberTopics, catalog ...TopicCatalogEntry) string {
	topics := memberFlow.Topics
	purposes := topicCatalogPurposeByPrefix(catalog)

	var b strings.Builder
	b.WriteString(TopicContractHeading + "\n\n")
	b.WriteString("This section is generated from `topics.json` and team `topicCatalog`. It is the source of truth for topic reads, writes, and Swarm Manager work routing.\n")
	if topics.IsEmpty() {
		b.WriteString("\nNo topic flow is declared for this member.")
		return b.String()
	}

	renderTopicContractIntakes(&b, topics.Intake, purposes)
	renderTopicContractRequiredReads(&b, topics.RequiredRead, purposes)
	renderTopicContractEvidence(&b, topics.EvidenceConsumed, purposes)
	renderTopicContractOutputs(&b, topics.Output, purposes)
	renderTopicContractExternalProducers(&b, topics.ExternalProducers)

	return strings.TrimRight(b.String(), "\n")
}

func renderTopicContractIntakes(b *strings.Builder, entries []IntakeEntry, purposes map[string]string) {
	if len(entries) == 0 {
		return
	}
	entries = append([]IntakeEntry(nil), entries...)
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
		b.WriteString(fmt.Sprintf("- `%s` - %s\n", e.Prefix, topicContractLineDetail(e.Prefix, purposes, strings.Join(parts, ", "))))
	}
}

func renderTopicContractRequiredReads(b *strings.Builder, entries []RequiredReadEntry, purposes map[string]string) {
	if len(entries) == 0 {
		return
	}
	entries = append([]RequiredReadEntry(nil), entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Prefix < entries[j].Prefix })

	b.WriteString("\n## Required Reads\n\n")
	for _, e := range entries {
		var parts []string
		if e.SourceTeam != nil && strings.TrimSpace(*e.SourceTeam) != "" {
			parts = append(parts, "source team `"+*e.SourceTeam+"`")
		}
		if strings.TrimSpace(e.Comment) != "" {
			parts = append(parts, strings.TrimSpace(e.Comment))
		}
		b.WriteString(fmt.Sprintf("- `%s` - %s\n", e.Prefix, topicContractLineDetail(e.Prefix, purposes, strings.Join(parts, "; "))))
	}
}

func renderTopicContractEvidence(b *strings.Builder, entries []EvidenceConsumedEntry, purposes map[string]string) {
	if len(entries) == 0 {
		return
	}
	entries = append([]EvidenceConsumedEntry(nil), entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Prefix < entries[j].Prefix })

	b.WriteString("\n## Evidence Consumed\n\n")
	for _, e := range entries {
		detail := "evidence used when authoring work"
		if e.SourceTeam != nil && strings.TrimSpace(*e.SourceTeam) != "" {
			detail += "; source team `" + *e.SourceTeam + "`"
		}
		b.WriteString(fmt.Sprintf("- `%s` - %s\n", e.Prefix, topicContractLineDetail(e.Prefix, purposes, detail)))
	}
}

func renderTopicContractOutputs(b *strings.Builder, entries []OutputEntry, purposes map[string]string) {
	if len(entries) == 0 {
		return
	}
	entries = append([]OutputEntry(nil), entries...)
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
		b.WriteString(fmt.Sprintf("- `%s` - %s\n", e.Prefix, topicContractLineDetail(e.Prefix, purposes, strings.Join(parts, ", "))))
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

func emptyAs(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func topicContractLineDetail(prefix string, purposes map[string]string, operational string) string {
	purpose := strings.TrimSpace(purposes[prefix])
	operational = strings.TrimSpace(operational)
	switch {
	case purpose != "" && operational != "":
		return purpose + " (" + operational + ")"
	case purpose != "":
		return purpose
	case operational != "":
		return operational
	default:
		return "declared"
	}
}

func topicCatalogPurposeByPrefix(catalog []TopicCatalogEntry) map[string]string {
	out := map[string]string{}
	for _, entry := range catalog {
		prefix := strings.TrimSpace(entry.Prefix)
		purpose := strings.TrimSpace(entry.Purpose)
		if prefix == "" || purpose == "" {
			continue
		}
		out[prefix] = purpose
	}
	return out
}

func sortedUniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
