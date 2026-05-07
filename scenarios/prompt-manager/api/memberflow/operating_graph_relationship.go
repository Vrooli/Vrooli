package memberflow

import (
	"path/filepath"
	"sort"
	"strings"
)

type OperatingRelationshipKind string

const (
	operatingRelTopicRead              OperatingRelationshipKind = "topic_read"
	operatingRelTopicIntake            OperatingRelationshipKind = "topic_intake"
	operatingRelTopicRequiredRead      OperatingRelationshipKind = "topic_required_read"
	operatingRelTopicEvidenceConsumed  OperatingRelationshipKind = "topic_evidence_consumed"
	operatingRelTopicOutput            OperatingRelationshipKind = "topic_output"
	operatingRelPOROutput              OperatingRelationshipKind = "por_output"
	operatingRelDecisionOwned          OperatingRelationshipKind = "decision_owned"
	operatingRelDecisionConsumed       OperatingRelationshipKind = "decision_consumed"
	operatingRelCapabilityGapRaised    OperatingRelationshipKind = "capability_gap_raised"
	operatingRelExternalProducer       OperatingRelationshipKind = "external_producer"
	operatingRelExternalProducerIntake OperatingRelationshipKind = "external_producer_intake"
	operatingRelCrossTeamOutput        OperatingRelationshipKind = "cross_team_output"
)

type OperatingSourceRef struct {
	Path string
	Line int
}

type OperatingRelationship struct {
	Kind       OperatingRelationshipKind
	Team       string
	Member     string
	Topic      string
	Decision   string
	Path       string
	External   string
	TargetTeam string
	Source     OperatingSourceRef
}

type OperatingRelationshipSet struct {
	relationships []OperatingRelationship
	byKey         map[string]OperatingRelationship
}

func NewOperatingRelationshipSet(rels []OperatingRelationship) OperatingRelationshipSet {
	out := OperatingRelationshipSet{byKey: map[string]OperatingRelationship{}}
	for _, rel := range rels {
		out.byKey[operatingRelationshipKey(rel)] = rel
	}
	keys := make([]string, 0, len(out.byKey))
	for key := range out.byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out.relationships = append(out.relationships, out.byKey[key])
	}
	return out
}

func (s OperatingRelationshipSet) All() []OperatingRelationship {
	out := make([]OperatingRelationship, len(s.relationships))
	copy(out, s.relationships)
	return out
}

func (s OperatingRelationshipSet) ByMember(member string) []OperatingRelationship {
	var out []OperatingRelationship
	for _, rel := range s.relationships {
		if rel.Member == member {
			out = append(out, rel)
		}
	}
	return out
}

func (s OperatingRelationshipSet) ByKind(kind OperatingRelationshipKind) []OperatingRelationship {
	var out []OperatingRelationship
	for _, rel := range s.relationships {
		if rel.Kind == kind {
			out = append(out, rel)
		}
	}
	return out
}

type OperatingRelationshipMatcher struct{}

func (OperatingRelationshipMatcher) GraphBackedByRuntime(graphRel OperatingRelationship, runtime OperatingRelationshipSet) bool {
	for _, runtimeRel := range runtime.All() {
		if operatingRelationshipsMatch(graphRel, runtimeRel) {
			return true
		}
	}
	return false
}

func (OperatingRelationshipMatcher) RuntimeShownInGraph(runtimeRel OperatingRelationship, graph OperatingRelationshipSet) bool {
	for _, graphRel := range graph.All() {
		if operatingRelationshipsMatch(graphRel, runtimeRel) {
			return true
		}
	}
	return false
}

func BuildRuntimeOperatingRelationships(runtime OperatingGraphRuntime, team string) []OperatingRelationship {
	var rels []OperatingRelationship
	for _, m := range runtime.Members {
		if m.Ref.Team != team {
			continue
		}
		runtimePath := runtimeMemberTopicsPath(runtime, m.Ref.Team, m.Ref.Member)
		source := OperatingSourceRef{Path: runtimePath}
		for _, in := range m.Topics.Intake {
			rels = append(rels, OperatingRelationship{Kind: operatingRelTopicIntake, Team: m.Ref.Team, Member: m.Ref.Member, Topic: in.Prefix, Source: source})
			for _, external := range m.Topics.ExternalProducers {
				rels = append(rels, OperatingRelationship{Kind: operatingRelExternalProducerIntake, Team: m.Ref.Team, Member: m.Ref.Member, Topic: in.Prefix, External: external, Source: source})
			}
		}
		for _, read := range m.Topics.RequiredRead {
			rels = append(rels, OperatingRelationship{Kind: operatingRelTopicRequiredRead, Team: m.Ref.Team, Member: m.Ref.Member, Topic: read.Prefix, Source: source})
		}
		for _, ev := range m.Topics.EvidenceConsumed {
			for _, decision := range ev.ForDecisions {
				rels = append(rels, OperatingRelationship{Kind: operatingRelTopicEvidenceConsumed, Team: m.Ref.Team, Member: m.Ref.Member, Topic: ev.Prefix, Decision: decision, Source: source})
			}
			if len(ev.ForDecisions) == 0 {
				rels = append(rels, OperatingRelationship{Kind: operatingRelTopicEvidenceConsumed, Team: m.Ref.Team, Member: m.Ref.Member, Topic: ev.Prefix, Source: source})
			}
		}
		for _, out := range m.Topics.Output {
			switch out.DestinationKind {
			case DestinationPORFile:
				if out.DestinationPath != nil {
					rels = append(rels, OperatingRelationship{Kind: operatingRelPOROutput, Team: m.Ref.Team, Member: m.Ref.Member, Topic: out.Prefix, Path: *out.DestinationPath, Source: source})
				}
			default:
				rels = append(rels, OperatingRelationship{Kind: operatingRelTopicOutput, Team: m.Ref.Team, Member: m.Ref.Member, Topic: out.Prefix, Source: source})
				if out.DestinationTeam != nil && strings.TrimSpace(*out.DestinationTeam) != "" {
					rels = append(rels, OperatingRelationship{Kind: operatingRelCrossTeamOutput, Team: m.Ref.Team, Member: m.Ref.Member, Topic: out.Prefix, TargetTeam: *out.DestinationTeam, Source: source})
				}
			}
		}
		for _, decision := range m.Topics.DecisionsOwned {
			rels = append(rels, OperatingRelationship{Kind: operatingRelDecisionOwned, Team: m.Ref.Team, Member: m.Ref.Member, Decision: decision, Source: source})
		}
		for _, decision := range m.Topics.DecisionsConsumed {
			rels = append(rels, OperatingRelationship{Kind: operatingRelDecisionConsumed, Team: m.Ref.Team, Member: m.Ref.Member, Decision: decision, Source: source})
		}
		if m.Topics.RaisesCapabilityGaps {
			rels = append(rels, OperatingRelationship{Kind: operatingRelCapabilityGapRaised, Team: m.Ref.Team, Member: m.Ref.Member, Decision: "capability-gap", Source: source})
		}
		for _, external := range m.Topics.ExternalProducers {
			rels = append(rels, OperatingRelationship{Kind: operatingRelExternalProducer, Team: m.Ref.Team, Member: m.Ref.Member, External: external, Source: source})
		}
	}
	return NewOperatingRelationshipSet(rels).All()
}

func BuildGraphOperatingRelationships(block OperatingGraphBlock) []OperatingRelationship {
	index := NewOperatingGraphContractIndex(block, OperatingGraphRuntime{})
	return index.GraphRelationships.All()
}

func operatingRelationshipFromNodes(team string, source OperatingSourceRef, from, to OperatingGraphNode) (OperatingRelationship, bool) {
	base := OperatingRelationship{Team: team, Source: source}
	switch {
	case from.Kind == "topic" && to.Kind == "member":
		base.Kind = operatingRelTopicRead
		base.Topic = from.Value
		base.Member = to.Value
		return base, true
	case from.Kind == "member" && to.Kind == "topic":
		base.Kind = operatingRelTopicOutput
		base.Member = from.Value
		base.Topic = to.Value
		return base, true
	case from.Kind == "member" && to.Kind == "decision":
		base.Member = from.Value
		base.Decision = to.Value
		if to.Value == "capability-gap" {
			base.Kind = operatingRelCapabilityGapRaised
		} else {
			base.Kind = operatingRelDecisionOwned
		}
		return base, true
	case from.Kind == "decision" && to.Kind == "member":
		base.Kind = operatingRelDecisionConsumed
		base.Decision = from.Value
		base.Member = to.Value
		return base, true
	case from.Kind == "member" && to.Kind == "por":
		base.Kind = operatingRelPOROutput
		base.Member = from.Value
		base.Path = to.Value
		return base, true
	case from.Kind == "topic" && to.Kind == "team":
		base.Kind = operatingRelCrossTeamOutput
		base.Topic = from.Value
		base.TargetTeam = to.Value
		return base, true
	case from.Kind == "external" && to.Kind == "member":
		base.Kind = operatingRelExternalProducer
		base.External = from.Value
		base.Member = to.Value
		return base, true
	case from.Kind == "external" && to.Kind == "topic":
		base.Kind = operatingRelExternalProducerIntake
		base.External = from.Value
		base.Topic = to.Value
		return base, true
	default:
		return OperatingRelationship{}, false
	}
}

func operatingRelationshipsMatch(graphRel, runtimeRel OperatingRelationship) bool {
	if graphRel.Team != "" && runtimeRel.Team != "" && graphRel.Team != runtimeRel.Team {
		return false
	}
	switch graphRel.Kind {
	case operatingRelTopicRead:
		return isRuntimeReadRelationship(runtimeRel.Kind) &&
			graphRel.Member == runtimeRel.Member &&
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	case operatingRelTopicOutput:
		return runtimeRel.Kind == operatingRelTopicOutput &&
			graphRel.Member == runtimeRel.Member &&
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	case operatingRelPOROutput:
		return runtimeRel.Kind == operatingRelPOROutput &&
			graphRel.Member == runtimeRel.Member &&
			pathsEqual(graphRel.Path, runtimeRel.Path)
	case operatingRelDecisionOwned:
		return runtimeRel.Kind == operatingRelDecisionOwned &&
			graphRel.Member == runtimeRel.Member &&
			graphRel.Decision == runtimeRel.Decision
	case operatingRelDecisionConsumed:
		return (runtimeRel.Kind == operatingRelDecisionConsumed || runtimeRel.Kind == operatingRelTopicEvidenceConsumed) &&
			graphRel.Member == runtimeRel.Member &&
			graphRel.Decision == runtimeRel.Decision
	case operatingRelCapabilityGapRaised:
		return runtimeRel.Kind == operatingRelCapabilityGapRaised &&
			graphRel.Member == runtimeRel.Member
	case operatingRelExternalProducer:
		return runtimeRel.Kind == operatingRelExternalProducer &&
			graphRel.Member == runtimeRel.Member &&
			graphRel.External == runtimeRel.External
	case operatingRelExternalProducerIntake:
		return runtimeRel.Kind == operatingRelExternalProducerIntake &&
			graphRel.External == runtimeRel.External &&
			(graphRel.Member == "" || runtimeRel.Member == "" || graphRel.Member == runtimeRel.Member) &&
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	case operatingRelCrossTeamOutput:
		return runtimeRel.Kind == operatingRelCrossTeamOutput &&
			graphRel.TargetTeam == runtimeRel.TargetTeam &&
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	default:
		return false
	}
}

func isRuntimeReadRelationship(kind OperatingRelationshipKind) bool {
	switch kind {
	case operatingRelTopicIntake, operatingRelTopicRequiredRead, operatingRelTopicEvidenceConsumed:
		return true
	default:
		return false
	}
}

func topicsOverlap(a, b string) bool {
	return strings.TrimSpace(a) != "" && strings.TrimSpace(b) != "" && Overlap(a, b)
}

func pathsEqual(a, b string) bool {
	return filepath.Clean(strings.TrimSpace(a)) == filepath.Clean(strings.TrimSpace(b))
}

func operatingRelationshipKey(rel OperatingRelationship) string {
	return strings.Join([]string{
		string(rel.Kind),
		rel.Team,
		rel.Member,
		rel.Topic,
		rel.Decision,
		filepath.ToSlash(filepath.Clean(rel.Path)),
		rel.External,
		rel.TargetTeam,
	}, "\x00")
}

func operatingRelationshipDiffDedupeKey(rel OperatingRelationship) string {
	if rel.Kind == operatingRelTopicEvidenceConsumed {
		rel.Decision = ""
	}
	rel.Kind = runtimeRelationshipAsGraphRelationship(rel)
	return operatingRelationshipKey(rel)
}
