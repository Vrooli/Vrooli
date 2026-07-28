package heartbeat

import (
	"strings"
	"testing"

	"prompt-manager/memberflow"
)

func TestRenderTopicContractSummarizesTopicsJSON(t *testing.T) {
	porPath := "docs/marketing/STRATEGY.md"
	got := RenderTopicContract(&topicContractInputs{
		teamID:  "marketing-crew",
		agentID: "researcher",
		memberFlow: memberflow.MemberTopics{
			Ref: memberflow.MemberRef{Team: "marketing-crew", Member: "researcher"},
			Topics: memberflow.Topics{
				Intake: []memberflow.IntakeEntry{{
					Prefix:          "research-inbox/*",
					Taxonomy:        "marketing-research",
					ClassifierSkill: "signal-classifier",
				}},
				RequiredRead: []memberflow.RequiredReadEntry{{Prefix: "audience-scan/*"}},
				EvidenceConsumed: []memberflow.EvidenceConsumedEntry{{
					Prefix:       "challenge-report/*",
					ForDecisions: []string{"capability-gap", "audience-update"},
				}, {
					Prefix: "marketing-craft-observation/*",
				}},
				Output: []memberflow.OutputEntry{{
					Prefix:          "audience-scan/*",
					DestinationKind: memberflow.DestinationKnowledge,
					Schema:          "audience-scan",
				}, {
					Prefix:          "marketing-canon/*",
					DestinationKind: memberflow.DestinationPORFile,
					DestinationPath: &porPath,
				}},
				DecisionsOwned:       []string{"audience-update"},
				DecisionsConsumed:    []string{"capability-gap"},
				RaisesCapabilityGaps: true,
				ExternalProducers:    []string{"operator"},
			},
		},
	})

	for _, want := range []string{
		"# Topic Contract",
		"- `research-inbox/*` - taxonomy `marketing-research`, classifier `signal-classifier`",
		"- `audience-scan/*`",
		"- `challenge-report/*` - for `audience-update`, `capability-gap`",
		"- `marketing-craft-observation/*` - general evidence",
		"- `audience-scan/*` - knowledge, schema `audience-scan`",
		"- `marketing-canon/*` - por_file, path `docs/marketing/STRATEGY.md`",
		"- own/propose: `audience-update`",
		"- consume: `capability-gap`",
		"- may raise `capability-gap`: yes",
		"- `operator`",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("topic contract missing %q:\n%s", want, got)
		}
	}
}

func TestRenderTopicContractEmptyDeclaration(t *testing.T) {
	got := RenderTopicContract(&topicContractInputs{
		memberFlow: memberflow.MemberTopics{Topics: memberflow.Topics{}},
	})
	if !strings.Contains(got, "No topic flow is declared for this member.") {
		t.Fatalf("empty contract message missing:\n%s", got)
	}
}

func TestRenderTopicContractIncludesTopicCatalogPurpose(t *testing.T) {
	got := RenderTopicContract(&topicContractInputs{
		teamID:  "marketing-crew",
		agentID: "researcher",
		memberFlow: memberflow.MemberTopics{
			Ref: memberflow.MemberRef{Team: "marketing-crew", Member: "researcher"},
			Topics: memberflow.Topics{
				Intake: []memberflow.IntakeEntry{{
					Prefix:   "research-inbox/*",
					Taxonomy: "marketing-research",
				}},
				RequiredRead: []memberflow.RequiredReadEntry{{Prefix: "audience-scan/*"}},
				EvidenceConsumed: []memberflow.EvidenceConsumedEntry{{
					Prefix:       "challenge-report/*",
					ForDecisions: []string{"capability-gap"},
				}},
				Output: []memberflow.OutputEntry{{
					Prefix:          "audience-scan/*",
					DestinationKind: memberflow.DestinationKnowledge,
				}},
			},
		},
		catalog: []memberflow.TopicCatalogEntry{{
			Prefix:  "research-inbox/*",
			Status:  "live",
			Purpose: "Raw research intake.",
		}, {
			Prefix:  "audience-scan/*",
			Status:  "live",
			Purpose: "Audience evidence.",
		}, {
			Prefix:  "challenge-report/*",
			Status:  "live",
			Purpose: "Challenge evidence.",
		}},
	})

	for _, want := range []string{
		"- `research-inbox/*` - Raw research intake. (taxonomy `marketing-research`)",
		"- `audience-scan/*` - Audience evidence.",
		"- `challenge-report/*` - Challenge evidence. (for `capability-gap`)",
		"- `audience-scan/*` - Audience evidence. (knowledge)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("topic contract missing purpose line %q:\n%s", want, got)
		}
	}
}
