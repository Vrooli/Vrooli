package heartbeat

import (
	"strings"
	"testing"

	"prompt-manager/memberflow"
)

func TestRenderTopicContractSummarizesTopicsJSON(t *testing.T) {
	got := RenderTopicContract(&topicContractInputs{
		teamID:  "marketing-crew",
		agentID: "researcher",
		memberFlow: memberflow.MemberTopics{
			Ref: memberflow.MemberRef{Team: "marketing-crew", Member: "researcher"},
			Topics: memberflow.Topics{
				Intake: []memberflow.IntakeEntry{{
					Prefix:          "research-inbox/*",
					Taxonomy:        "marketing-research",
					ClassifierSkill: "marketing-signal-classifier",
				}},
				RequiredRead: []memberflow.RequiredReadEntry{{Prefix: "audience-scan/*"}},
				EvidenceConsumed: []memberflow.EvidenceConsumedEntry{{
					Prefix:       "challenge-report/*",
					ForDecisions: []string{"capability-gap", "audience-update"},
				}},
				Output: []memberflow.OutputEntry{{
					Prefix:          "audience-scan/*",
					DestinationKind: memberflow.DestinationKnowledge,
					Schema:          "audience-scan",
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
		"- `research-inbox/*` - taxonomy `marketing-research`, classifier `marketing-signal-classifier`",
		"- `audience-scan/*`",
		"- `challenge-report/*` - for `audience-update`, `capability-gap`",
		"- `audience-scan/*` - knowledge, schema `audience-scan`",
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
