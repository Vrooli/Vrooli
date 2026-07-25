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
	operatingRelUniversalSourceWrite   OperatingRelationshipKind = "universal_source_write"
)

type OperatingSourceRef struct {
	Path string
	Line int
}

type OperatingRelationship struct {
	Kind         OperatingRelationshipKind
	Team         string
	Member       string
	Topic        string
	Decision     string
	Path         string
	External     string
	ProducerTeam string
	TargetTeam   string
	Source       OperatingSourceRef
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

type OperatingRelationshipMatcher struct {
	registry OperatingRelationshipRegistry
}

func NewOperatingRelationshipMatcher() OperatingRelationshipMatcher {
	return OperatingRelationshipMatcher{registry: DefaultOperatingRelationshipRegistry()}
}

func (m OperatingRelationshipMatcher) GraphBackedByRuntime(graphRel OperatingRelationship, runtime OperatingRelationshipSet) bool {
	if len(m.registry.specs) == 0 {
		m = NewOperatingRelationshipMatcher()
	}
	for _, runtimeRel := range runtime.All() {
		if m.registry.Match(graphRel, runtimeRel) {
			return true
		}
	}
	return false
}

func (m OperatingRelationshipMatcher) RuntimeShownInGraph(runtimeRel OperatingRelationship, graph OperatingRelationshipSet) bool {
	if len(m.registry.specs) == 0 {
		m = NewOperatingRelationshipMatcher()
	}
	for _, graphRel := range graph.All() {
		if m.registry.Match(graphRel, runtimeRel) {
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
	// A writer-skill is portable by design, so its writes_to[] declaration is
	// not owned by one member. For a universal-source intake, materialize the
	// declared peer-team producers explicitly. This lets the contract graph
	// distinguish real intra-swarm flow from an outside-system producer.
	writerDeclarations, err := LoadWriterSkillProducerDeclarations(runtime.RepoRoot)
	if err == nil {
		allTeams := runtimeTeamIDs(runtime)
		for _, member := range runtime.Members {
			if member.Ref.Team != team {
				continue
			}
			source := OperatingSourceRef{Path: runtimeMemberTopicsPath(runtime, member.Ref.Team, member.Ref.Member)}
			for _, intake := range member.Topics.Intake {
				if intake.SourceTeam == nil || *intake.SourceTeam != SourceTeamWildcard {
					continue
				}
				for _, declaration := range writerDeclarations {
					if !topicsOverlap(declaration.Prefix, intake.Prefix) || !containsString(member.Topics.ExternalProducers, declaration.SkillID) {
						continue
					}
					for _, producerTeam := range allTeams {
						if producerTeam == team {
							continue
						}
						rels = append(rels, OperatingRelationship{Kind: operatingRelUniversalSourceWrite, Team: team, ProducerTeam: producerTeam, Topic: intake.Prefix, External: declaration.SkillID, Source: source})
					}
				}
			}
		}
	}
	return NewOperatingRelationshipSet(rels).All()
}

func runtimeTeamIDs(runtime OperatingGraphRuntime) []string {
	seen := map[string]bool{}
	for _, member := range runtime.Members {
		if member.Ref.Team != "" {
			seen[member.Ref.Team] = true
		}
	}
	teams := make([]string, 0, len(seen))
	for team := range seen {
		teams = append(teams, team)
	}
	sort.Strings(teams)
	return teams
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func BuildGraphOperatingRelationships(block OperatingGraphBlock) []OperatingRelationship {
	index := NewOperatingGraphContractIndex(block, OperatingGraphRuntime{})
	return index.GraphRelationships.All()
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
		rel.ProducerTeam,
		rel.TargetTeam,
	}, "\x00")
}

func operatingRelationshipDiffDedupeKey(rel OperatingRelationship) string {
	if rel.Kind == operatingRelTopicEvidenceConsumed {
		rel.Decision = ""
	}
	rel.Kind = DefaultOperatingRelationshipRegistry().GraphKindForRuntime(rel.Kind)
	return operatingRelationshipKey(rel)
}
