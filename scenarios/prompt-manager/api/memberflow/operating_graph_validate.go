package memberflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type OperatingGraphRuntime struct {
	RepoRoot  string
	StoreDir  string
	Members   []MemberTopics
	Contracts TeamContractRegistry
}

type OperatingRelationship struct {
	Kind       string
	Team       string
	Member     string
	Topic      string
	Decision   string
	Path       string
	External   string
	TargetTeam string
	SourcePath string
	SourceLine int
}

const (
	operatingRelTopicRead              = "topic_read"
	operatingRelTopicIntake            = "topic_intake"
	operatingRelTopicRequiredRead      = "topic_required_read"
	operatingRelTopicEvidenceConsumed  = "topic_evidence_consumed"
	operatingRelTopicOutput            = "topic_output"
	operatingRelPOROutput              = "por_output"
	operatingRelDecisionOwned          = "decision_owned"
	operatingRelDecisionConsumed       = "decision_consumed"
	operatingRelCapabilityGapRaised    = "capability_gap_raised"
	operatingRelExternalProducer       = "external_producer"
	operatingRelExternalProducerIntake = "external_producer_intake"
	operatingRelCrossTeamOutput        = "cross_team_output"
)

func BuildOperatingGraphRuntime(repoRoot, storeDir string) (OperatingGraphRuntime, error) {
	members, err := LoadAll(storeDir)
	if err != nil {
		return OperatingGraphRuntime{}, err
	}
	contracts, err := LoadAllTeamContracts(storeDir)
	if err != nil {
		return OperatingGraphRuntime{}, err
	}
	return OperatingGraphRuntime{
		RepoRoot:  repoRoot,
		StoreDir:  storeDir,
		Members:   members,
		Contracts: contracts,
	}, nil
}

func ValidateOperatingGraphs(blocks []OperatingGraphBlock, runtime OperatingGraphRuntime, teamFilter, idFilter string) OperatingGraphValidationResult {
	var result OperatingGraphValidationResult
	for _, block := range filterOperatingGraphBlocks(blocks, teamFilter, idFilter) {
		validateOperatingGraphBlock(&result, block, runtime)
	}
	sort.Slice(result.Findings, func(i, j int) bool {
		a, b := result.Findings[i], result.Findings[j]
		if a.Severity != b.Severity {
			return a.Severity < b.Severity
		}
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		return a.Detail < b.Detail
	})
	return result
}

func DiffOperatingGraphs(blocks []OperatingGraphBlock, runtime OperatingGraphRuntime, teamFilter, idFilter string) []OperatingGraphContractDiff {
	diffs := make([]OperatingGraphContractDiff, 0)
	for _, block := range filterOperatingGraphBlocks(blocks, teamFilter, idFilter) {
		if block.Metadata.Mode != "contract" {
			continue
		}
		graphRels := BuildGraphOperatingRelationships(block)
		runtimeRels := BuildRuntimeOperatingRelationships(runtime, block.Metadata.Team)
		for _, rel := range graphRels {
			if !RelationshipBackedByRuntime(rel, runtimeRels) {
				diffs = append(diffs, operatingGraphDiffFromGraphRel(rel, runtime))
			}
		}
		seenRuntimeDiffs := map[string]bool{}
		for _, rel := range runtimeRels {
			if rel.Kind == operatingRelExternalProducerIntake {
				continue
			}
			diffKey := operatingRuntimeDiffKey(rel)
			if seenRuntimeDiffs[diffKey] {
				continue
			}
			seenRuntimeDiffs[diffKey] = true
			if !RuntimeRelationshipShownInGraph(rel, graphRels) {
				diffs = append(diffs, operatingGraphDiffFromRuntimeRel(rel, block))
			}
		}
	}
	sort.Slice(diffs, func(i, j int) bool {
		if diffs[i].Kind != diffs[j].Kind {
			return diffs[i].Kind < diffs[j].Kind
		}
		if diffs[i].Relationship != diffs[j].Relationship {
			return diffs[i].Relationship < diffs[j].Relationship
		}
		return diffs[i].Detail < diffs[j].Detail
	})
	return diffs
}

func BuildRuntimeOperatingRelationships(runtime OperatingGraphRuntime, team string) []OperatingRelationship {
	var rels []OperatingRelationship
	for _, m := range runtime.Members {
		if m.Ref.Team != team {
			continue
		}
		runtimePath := runtimeMemberTopicsPath(runtime, m.Ref.Team, m.Ref.Member)
		for _, in := range m.Topics.Intake {
			rels = append(rels, OperatingRelationship{Kind: operatingRelTopicIntake, Team: m.Ref.Team, Member: m.Ref.Member, Topic: in.Prefix, SourcePath: runtimePath})
			for _, external := range m.Topics.ExternalProducers {
				rels = append(rels, OperatingRelationship{Kind: operatingRelExternalProducerIntake, Team: m.Ref.Team, Member: m.Ref.Member, Topic: in.Prefix, External: external, SourcePath: runtimePath})
			}
		}
		for _, read := range m.Topics.RequiredRead {
			rels = append(rels, OperatingRelationship{Kind: operatingRelTopicRequiredRead, Team: m.Ref.Team, Member: m.Ref.Member, Topic: read.Prefix, SourcePath: runtimePath})
		}
		for _, ev := range m.Topics.EvidenceConsumed {
			for _, decision := range ev.ForDecisions {
				rels = append(rels, OperatingRelationship{Kind: operatingRelTopicEvidenceConsumed, Team: m.Ref.Team, Member: m.Ref.Member, Topic: ev.Prefix, Decision: decision, SourcePath: runtimePath})
			}
			if len(ev.ForDecisions) == 0 {
				rels = append(rels, OperatingRelationship{Kind: operatingRelTopicEvidenceConsumed, Team: m.Ref.Team, Member: m.Ref.Member, Topic: ev.Prefix, SourcePath: runtimePath})
			}
		}
		for _, out := range m.Topics.Output {
			switch out.DestinationKind {
			case DestinationPORFile:
				if out.DestinationPath != nil {
					rels = append(rels, OperatingRelationship{Kind: operatingRelPOROutput, Team: m.Ref.Team, Member: m.Ref.Member, Topic: out.Prefix, Path: *out.DestinationPath, SourcePath: runtimePath})
				}
			default:
				rels = append(rels, OperatingRelationship{Kind: operatingRelTopicOutput, Team: m.Ref.Team, Member: m.Ref.Member, Topic: out.Prefix, SourcePath: runtimePath})
				if out.DestinationTeam != nil && strings.TrimSpace(*out.DestinationTeam) != "" {
					rels = append(rels, OperatingRelationship{Kind: operatingRelCrossTeamOutput, Team: m.Ref.Team, Member: m.Ref.Member, Topic: out.Prefix, TargetTeam: *out.DestinationTeam, SourcePath: runtimePath})
				}
			}
		}
		for _, decision := range m.Topics.DecisionsOwned {
			rels = append(rels, OperatingRelationship{Kind: operatingRelDecisionOwned, Team: m.Ref.Team, Member: m.Ref.Member, Decision: decision, SourcePath: runtimePath})
		}
		for _, decision := range m.Topics.DecisionsConsumed {
			rels = append(rels, OperatingRelationship{Kind: operatingRelDecisionConsumed, Team: m.Ref.Team, Member: m.Ref.Member, Decision: decision, SourcePath: runtimePath})
		}
		if m.Topics.RaisesCapabilityGaps {
			rels = append(rels, OperatingRelationship{Kind: operatingRelCapabilityGapRaised, Team: m.Ref.Team, Member: m.Ref.Member, Decision: "capability-gap", SourcePath: runtimePath})
		}
		for _, external := range m.Topics.ExternalProducers {
			rels = append(rels, OperatingRelationship{Kind: operatingRelExternalProducer, Team: m.Ref.Team, Member: m.Ref.Member, External: external, SourcePath: runtimePath})
		}
	}
	return dedupeOperatingRelationships(rels)
}

func BuildGraphOperatingRelationships(block OperatingGraphBlock) []OperatingRelationship {
	idx := indexOperatingGraph(block.Graph)
	var rels []OperatingRelationship
	for _, edge := range block.Graph.Edges {
		from, fok := idx.nodes[edge.From]
		to, tok := idx.nodes[edge.To]
		if !fok || !tok || operatingGraphNodeNonActionable(from) || operatingGraphNodeNonActionable(to) {
			continue
		}
		rel, ok := operatingRelationshipFromNodes(block.Metadata.Team, block.Source.Path, edge.SourceLine, from, to)
		if ok {
			rels = append(rels, rel)
		}
	}
	return dedupeOperatingRelationships(rels)
}

func RelationshipBackedByRuntime(graphRel OperatingRelationship, runtimeRels []OperatingRelationship) bool {
	for _, runtimeRel := range runtimeRels {
		if operatingRelationshipsMatch(graphRel, runtimeRel) {
			return true
		}
	}
	return false
}

func RuntimeRelationshipShownInGraph(runtimeRel OperatingRelationship, graphRels []OperatingRelationship) bool {
	for _, graphRel := range graphRels {
		if runtimeRel.Kind == operatingRelExternalProducer &&
			graphRel.Kind == operatingRelExternalProducerIntake &&
			graphRel.Team == runtimeRel.Team &&
			graphRel.External == runtimeRel.External {
			return true
		}
		if operatingRelationshipsMatch(graphRel, runtimeRel) {
			return true
		}
	}
	return false
}

func operatingRelationshipFromNodes(team, sourcePath string, sourceLine int, from, to OperatingGraphNode) (OperatingRelationship, bool) {
	base := OperatingRelationship{
		Team:       team,
		SourcePath: sourcePath,
		SourceLine: sourceLine,
	}
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

func operatingGraphNodeNonActionable(node OperatingGraphNode) bool {
	if node.Kind == "" || node.Kind == "process" || node.Kind == "future" {
		return true
	}
	return node.Kind == "topic" && (node.Qualifier == "future" || node.Qualifier == "old" || node.Qualifier == "external")
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
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	case operatingRelCrossTeamOutput:
		return runtimeRel.Kind == operatingRelCrossTeamOutput &&
			graphRel.TargetTeam == runtimeRel.TargetTeam &&
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	default:
		return false
	}
}

func isRuntimeReadRelationship(kind string) bool {
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

func operatingGraphDiffFromGraphRel(rel OperatingRelationship, runtime OperatingGraphRuntime) OperatingGraphContractDiff {
	diff := OperatingGraphContractDiff{
		Kind:             "graph_relationship_missing_in_runtime",
		Relationship:     rel.Kind,
		Team:             rel.Team,
		Member:           rel.Member,
		Topic:            rel.Topic,
		Decision:         rel.Decision,
		Path:             rel.Path,
		External:         rel.External,
		TargetTeam:       rel.TargetTeam,
		SourcePath:       rel.SourcePath,
		Line:             rel.SourceLine,
		RuntimePath:      runtimePathForGraphRelationship(rel, runtime),
		AcceptableFields: acceptableRuntimeFields(rel),
	}
	diff.Suggestions = suggestionsForGraphRelationship(diff)
	diff.Detail = detailForGraphRelationshipMissingRuntime(diff)
	return diff
}

func operatingGraphDiffFromRuntimeRel(rel OperatingRelationship, block OperatingGraphBlock) OperatingGraphContractDiff {
	diff := OperatingGraphContractDiff{
		Kind:         "runtime_relationship_missing_in_graph",
		Relationship: runtimeRelationshipAsGraphRelationship(rel),
		Team:         rel.Team,
		Member:       rel.Member,
		Topic:        rel.Topic,
		Decision:     rel.Decision,
		Path:         rel.Path,
		External:     rel.External,
		TargetTeam:   rel.TargetTeam,
		SourcePath:   block.Source.Path,
		Line:         block.Source.FenceLine,
		RuntimePath:  rel.SourcePath,
	}
	diff.Suggestions = suggestionsForRuntimeRelationship(diff)
	diff.Detail = detailForRuntimeRelationshipMissingGraph(diff)
	return diff
}

func runtimeRelationshipAsGraphRelationship(rel OperatingRelationship) string {
	if isRuntimeReadRelationship(rel.Kind) {
		return operatingRelTopicRead
	}
	return rel.Kind
}

func runtimePathForGraphRelationship(rel OperatingRelationship, runtime OperatingGraphRuntime) string {
	if rel.Member != "" {
		return runtimeMemberTopicsPath(runtime, rel.Team, rel.Member)
	}
	if rel.Kind == operatingRelCrossTeamOutput || rel.Kind == operatingRelExternalProducerIntake {
		if match := matchingRuntimeRelationshipPath(rel, runtime); match != "" {
			return match
		}
	}
	return runtimeTeamPath(runtime, rel.Team)
}

func matchingRuntimeRelationshipPath(rel OperatingRelationship, runtime OperatingGraphRuntime) string {
	runtimeRels := BuildRuntimeOperatingRelationships(runtime, rel.Team)
	for _, candidate := range runtimeRels {
		if operatingRelationshipsMatch(rel, candidate) || graphlessRelationshipLooksRelated(rel, candidate) {
			return candidate.SourcePath
		}
	}
	return ""
}

func graphlessRelationshipLooksRelated(graphRel, runtimeRel OperatingRelationship) bool {
	if graphRel.Team != runtimeRel.Team {
		return false
	}
	switch graphRel.Kind {
	case operatingRelCrossTeamOutput:
		return runtimeRel.Kind == operatingRelCrossTeamOutput &&
			graphRel.TargetTeam == runtimeRel.TargetTeam &&
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	case operatingRelExternalProducerIntake:
		return runtimeRel.Kind == operatingRelExternalProducerIntake &&
			graphRel.External == runtimeRel.External &&
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	default:
		return false
	}
}

func acceptableRuntimeFields(rel OperatingRelationship) []string {
	switch rel.Kind {
	case operatingRelTopicRead:
		return []string{"intake", "required_read", "evidence_consumed"}
	case operatingRelTopicOutput:
		return []string{"output"}
	case operatingRelPOROutput:
		return []string{"output.destination_kind=por_file"}
	case operatingRelDecisionOwned:
		return []string{"decisions_owned"}
	case operatingRelDecisionConsumed:
		return []string{"decisions_consumed", "evidence_consumed.for_decisions"}
	case operatingRelCapabilityGapRaised:
		return []string{"raises_capability_gaps"}
	case operatingRelExternalProducer, operatingRelExternalProducerIntake:
		return []string{"external_producers", "intake"}
	case operatingRelCrossTeamOutput:
		return []string{"output.destination_team"}
	default:
		return nil
	}
}

func suggestionsForGraphRelationship(diff OperatingGraphContractDiff) []string {
	switch diff.Relationship {
	case operatingRelTopicRead:
		return []string{
			fmt.Sprintf("add required_read %q to %s/topics.json", diff.Topic, diff.Member),
			"or remove the topic -> member edge from the operating graph",
		}
	case operatingRelTopicOutput:
		return []string{
			fmt.Sprintf("add output %q to %s/topics.json", diff.Topic, diff.Member),
			"or remove the member -> topic edge from the operating graph",
		}
	case operatingRelPOROutput:
		return []string{
			fmt.Sprintf("add a por_file output to %q in %s/topics.json", diff.Path, diff.Member),
			"or remove the member -> PoR edge from the operating graph",
		}
	case operatingRelDecisionOwned:
		return []string{
			fmt.Sprintf("add decisions_owned %q to %s/topics.json", diff.Decision, diff.Member),
			"or remove the member -> decision edge from the operating graph",
		}
	case operatingRelDecisionConsumed:
		return []string{
			fmt.Sprintf("add decisions_consumed %q to %s/topics.json", diff.Decision, diff.Member),
			"or remove the decision -> member edge from the operating graph",
		}
	case operatingRelCapabilityGapRaised:
		return []string{
			fmt.Sprintf("set raises_capability_gaps to true in %s/topics.json", diff.Member),
			"or remove the member -> capability-gap edge from the operating graph",
		}
	case operatingRelExternalProducer:
		return []string{
			fmt.Sprintf("add external_producers %q to %s/topics.json", diff.External, diff.Member),
			"or remove the external -> member edge from the operating graph",
		}
	case operatingRelExternalProducerIntake:
		return []string{
			fmt.Sprintf("declare an intake for %q and external_producers %q on the receiving member topics.json", diff.Topic, diff.External),
			"or remove the external -> topic edge from the operating graph",
		}
	case operatingRelCrossTeamOutput:
		return []string{
			fmt.Sprintf("add destination_team %q to an output for %q", diff.TargetTeam, diff.Topic),
			"or remove the topic -> team edge from the operating graph",
		}
	default:
		return nil
	}
}

func suggestionsForRuntimeRelationship(diff OperatingGraphContractDiff) []string {
	switch diff.Relationship {
	case operatingRelTopicRead:
		return []string{
			fmt.Sprintf("add topic:%s -> member:%s to the operating graph", diff.Topic, diff.Member),
			"or remove the runtime read declaration if it is no longer part of the operating contract",
		}
	case operatingRelTopicOutput:
		return []string{
			fmt.Sprintf("add member:%s -> topic:%s to the operating graph", diff.Member, diff.Topic),
			"or remove the runtime output declaration if it is obsolete",
		}
	case operatingRelPOROutput:
		return []string{
			fmt.Sprintf("add member:%s -> por:%s to the operating graph", diff.Member, diff.Path),
			"or remove the runtime por_file output if it is obsolete",
		}
	case operatingRelDecisionOwned:
		return []string{
			fmt.Sprintf("add member:%s -> decision:%s to the operating graph", diff.Member, diff.Decision),
			"or remove the runtime decision ownership if it is obsolete",
		}
	case operatingRelDecisionConsumed:
		return []string{
			fmt.Sprintf("add decision:%s -> member:%s to the operating graph", diff.Decision, diff.Member),
			"or remove the runtime decision consumption if it is obsolete",
		}
	case operatingRelCapabilityGapRaised:
		return []string{
			fmt.Sprintf("add member:%s -> decision:capability-gap to the operating graph", diff.Member),
			"or unset raises_capability_gaps if this member should not raise gaps",
		}
	case operatingRelExternalProducer:
		return []string{
			fmt.Sprintf("add external:%s -> member:%s to the operating graph", diff.External, diff.Member),
			"or remove the runtime external producer declaration if it is obsolete",
		}
	case operatingRelExternalProducerIntake:
		return []string{
			fmt.Sprintf("add external:%s -> topic:%s to the operating graph", diff.External, diff.Topic),
			"or remove the runtime external producer/intake relationship if it is obsolete",
		}
	case operatingRelCrossTeamOutput:
		return []string{
			fmt.Sprintf("add topic:%s -> team:%s to the operating graph", diff.Topic, diff.TargetTeam),
			"or remove destination_team if this is not a cross-team output",
		}
	default:
		return nil
	}
}

func detailForGraphRelationshipMissingRuntime(diff OperatingGraphContractDiff) string {
	return fmt.Sprintf("%s says %s. Runtime has no matching declaration.", relationshipLocation(diff), relationshipStatement(diff))
}

func detailForRuntimeRelationshipMissingGraph(diff OperatingGraphContractDiff) string {
	return fmt.Sprintf("%s declares %s. The contract graph does not show a matching relationship.", diff.RuntimePath, relationshipStatement(diff))
}

func relationshipLocation(diff OperatingGraphContractDiff) string {
	if diff.SourcePath == "" {
		return "contract graph"
	}
	if diff.Line > 0 {
		return fmt.Sprintf("%s:%d", diff.SourcePath, diff.Line)
	}
	return diff.SourcePath
}

func relationshipStatement(diff OperatingGraphContractDiff) string {
	switch diff.Relationship {
	case operatingRelTopicRead:
		return fmt.Sprintf("topic:%s -> member:%s", diff.Topic, diff.Member)
	case operatingRelTopicOutput:
		return fmt.Sprintf("member:%s -> topic:%s", diff.Member, diff.Topic)
	case operatingRelPOROutput:
		return fmt.Sprintf("member:%s -> por:%s", diff.Member, diff.Path)
	case operatingRelDecisionOwned:
		return fmt.Sprintf("member:%s -> decision:%s", diff.Member, diff.Decision)
	case operatingRelDecisionConsumed:
		return fmt.Sprintf("decision:%s -> member:%s", diff.Decision, diff.Member)
	case operatingRelCapabilityGapRaised:
		return fmt.Sprintf("member:%s -> decision:capability-gap", diff.Member)
	case operatingRelExternalProducer:
		return fmt.Sprintf("external:%s -> member:%s", diff.External, diff.Member)
	case operatingRelExternalProducerIntake:
		return fmt.Sprintf("external:%s -> topic:%s", diff.External, diff.Topic)
	case operatingRelCrossTeamOutput:
		return fmt.Sprintf("topic:%s -> team:%s", diff.Topic, diff.TargetTeam)
	default:
		return diff.Relationship
	}
}

func runtimeMemberTopicsPath(runtime OperatingGraphRuntime, team, member string) string {
	return relativeRuntimePath(runtime, filepath.Join(runtime.StoreDir, "teams", team, "members", member, "topics.json"))
}

func runtimeTeamPath(runtime OperatingGraphRuntime, team string) string {
	return relativeRuntimePath(runtime, filepath.Join(runtime.StoreDir, "teams", team, "team.json"))
}

func relativeRuntimePath(runtime OperatingGraphRuntime, path string) string {
	if runtime.RepoRoot != "" {
		if rel, err := filepath.Rel(runtime.RepoRoot, path); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(path)
}

func dedupeOperatingRelationships(rels []OperatingRelationship) []OperatingRelationship {
	seen := map[string]bool{}
	var out []OperatingRelationship
	for _, rel := range rels {
		key := operatingRelationshipKey(rel)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, rel)
	}
	sort.Slice(out, func(i, j int) bool {
		return operatingRelationshipKey(out[i]) < operatingRelationshipKey(out[j])
	})
	return out
}

func operatingRelationshipKey(rel OperatingRelationship) string {
	return strings.Join([]string{
		rel.Kind,
		rel.Team,
		rel.Member,
		rel.Topic,
		rel.Decision,
		filepath.ToSlash(filepath.Clean(rel.Path)),
		rel.External,
		rel.TargetTeam,
	}, "\x00")
}

func operatingRuntimeDiffKey(rel OperatingRelationship) string {
	if rel.Kind == operatingRelTopicEvidenceConsumed {
		rel.Decision = ""
	}
	rel.Kind = runtimeRelationshipAsGraphRelationship(rel)
	return operatingRelationshipKey(rel)
}

func filterOperatingGraphBlocks(blocks []OperatingGraphBlock, teamFilter, idFilter string) []OperatingGraphBlock {
	var out []OperatingGraphBlock
	for _, block := range blocks {
		if teamFilter != "" && block.Metadata.Team != teamFilter {
			continue
		}
		if idFilter != "" && block.Metadata.ID != idFilter {
			continue
		}
		out = append(out, block)
	}
	return out
}

type operatingGraphIndex struct {
	nodes       map[string]OperatingGraphNode
	byKindValue map[string]string
	edges       []OperatingGraphEdge
}

func indexOperatingGraph(graph OperatingGraph) operatingGraphIndex {
	idx := operatingGraphIndex{
		nodes:       map[string]OperatingGraphNode{},
		byKindValue: map[string]string{},
		edges:       graph.Edges,
	}
	for _, n := range graph.Nodes {
		idx.nodes[n.ID] = n
		if n.Kind != "" && n.Value != "" {
			idx.byKindValue[n.Kind+"\x00"+n.Value] = n.ID
		}
	}
	return idx
}

func validateOperatingGraphBlock(result *OperatingGraphValidationResult, block OperatingGraphBlock, runtime OperatingGraphRuntime) {
	if block.Metadata.Mode == "explanatory" {
		return
	}
	idx := indexOperatingGraph(block.Graph)
	contract := runtime.Contracts[block.Metadata.Team]

	for _, node := range block.Graph.Nodes {
		validateOperatingGraphNode(result, block, node, runtime, contract)
	}
	for _, edge := range block.Graph.Edges {
		validateOperatingGraphEdge(result, block, idx, edge, runtime)
	}
	if block.Metadata.Mode == "contract" {
		validateOperatingGraphCompleteness(result, block, idx, runtime)
	}
}

func validateOperatingGraphNode(result *OperatingGraphValidationResult, block OperatingGraphBlock, node OperatingGraphNode, runtime OperatingGraphRuntime, contract *LoadedTeamContract) {
	if node.Kind == "" {
		addOperatingFinding(result, OperatingGraphFinding{Rule: "graph_untyped_node", Severity: string(SeverityError), GraphID: block.Metadata.ID, Team: block.Metadata.Team, NodeID: node.ID, SourcePath: block.Source.Path, Line: node.SourceLine, Detail: fmt.Sprintf("node %q lacks a typed machine label", node.ID)})
		return
	}
	switch node.Kind {
	case "member":
		if contract == nil || contract.Contract == nil {
			addOperatingFinding(result, OperatingGraphFinding{Rule: "graph_unknown_member", Severity: string(SeverityError), GraphID: block.Metadata.ID, Team: block.Metadata.Team, NodeID: node.ID, Member: node.Value, SourcePath: block.Source.Path, Line: node.SourceLine, Detail: fmt.Sprintf("member %q cannot be resolved because team contract is unavailable", node.Value)})
			return
		}
		if _, ok := contract.Contract.Members[node.Value]; !ok {
			addOperatingFinding(result, OperatingGraphFinding{Rule: "graph_unknown_member", Severity: string(SeverityError), GraphID: block.Metadata.ID, Team: block.Metadata.Team, NodeID: node.ID, Member: node.Value, SourcePath: block.Source.Path, Line: node.SourceLine, Detail: fmt.Sprintf("member %q is not declared in %s/team.json", node.Value, block.Metadata.Team)})
		}
	case "decision":
		if !runtime.Contracts.HasDecisionContext(node.Value) {
			addOperatingFinding(result, OperatingGraphFinding{Rule: "graph_unknown_decision", Severity: string(SeverityError), GraphID: block.Metadata.ID, Team: block.Metadata.Team, NodeID: node.ID, Decision: node.Value, SourcePath: block.Source.Path, Line: node.SourceLine, Detail: fmt.Sprintf("decision context %q is not declared in any team contract", node.Value)})
		}
	case "team":
		if _, ok := runtime.Contracts[node.Value]; !ok {
			addOperatingFinding(result, OperatingGraphFinding{Rule: "graph_unknown_team", Severity: string(SeverityWarning), GraphID: block.Metadata.ID, Team: block.Metadata.Team, NodeID: node.ID, SourcePath: block.Source.Path, Line: node.SourceLine, Detail: fmt.Sprintf("team %q is not declared in the team registry", node.Value)})
		}
	case "por":
		if node.Value == "" || !operatingGraphFileExists(filepath.Join(runtime.RepoRoot, node.Value)) {
			addOperatingFinding(result, OperatingGraphFinding{Rule: "graph_unknown_por", Severity: string(SeverityError), GraphID: block.Metadata.ID, Team: block.Metadata.Team, NodeID: node.ID, Path: node.Value, SourcePath: block.Source.Path, Line: node.SourceLine, Detail: fmt.Sprintf("plan-of-record path %q does not exist", node.Value)})
		}
	case "topic":
		if node.Qualifier == "future" || node.Qualifier == "old" || node.Qualifier == "external" {
			return
		}
		if !runtime.topicDeclared(block.Metadata.Team, node.Value) {
			addOperatingFinding(result, OperatingGraphFinding{Rule: "graph_topic_unresolved", Severity: string(SeverityError), GraphID: block.Metadata.ID, Team: block.Metadata.Team, NodeID: node.ID, Topic: node.Value, SourcePath: block.Source.Path, Line: node.SourceLine, Detail: fmt.Sprintf("live topic %q is not declared by any %s member topics.json", node.Value, block.Metadata.Team)})
		}
	case "external", "process", "future":
	default:
		addOperatingFinding(result, OperatingGraphFinding{Rule: "graph_unknown_node_kind", Severity: string(SeverityError), GraphID: block.Metadata.ID, Team: block.Metadata.Team, NodeID: node.ID, SourcePath: block.Source.Path, Line: node.SourceLine, Detail: fmt.Sprintf("node kind %q is not supported", node.Kind)})
	}
}

func validateOperatingGraphEdge(result *OperatingGraphValidationResult, block OperatingGraphBlock, idx operatingGraphIndex, edge OperatingGraphEdge, runtime OperatingGraphRuntime) {
	from, fok := idx.nodes[edge.From]
	to, tok := idx.nodes[edge.To]
	if !fok || !tok || from.Kind == "" || to.Kind == "" {
		return
	}
	if from.Kind == "process" || to.Kind == "process" || from.Kind == "future" || to.Kind == "future" {
		return
	}
	if from.Kind == "topic" && from.Qualifier == "future" {
		addOperatingFinding(result, OperatingGraphFinding{Rule: "graph_future_topic_live_edge", Severity: string(SeverityWarning), GraphID: block.Metadata.ID, Team: block.Metadata.Team, Edge: edge.From + "->" + edge.To, Topic: from.Value, SourcePath: block.Source.Path, Line: edge.SourceLine, Detail: fmt.Sprintf("future topic %q is used as an active edge source", from.Value)})
		return
	}
	if to.Kind == "topic" && to.Qualifier == "future" {
		addOperatingFinding(result, OperatingGraphFinding{Rule: "graph_future_topic_live_edge", Severity: string(SeverityWarning), GraphID: block.Metadata.ID, Team: block.Metadata.Team, Edge: edge.From + "->" + edge.To, Topic: to.Value, SourcePath: block.Source.Path, Line: edge.SourceLine, Detail: fmt.Sprintf("future topic %q is used as an active edge target", to.Value)})
		return
	}
	if operatingEdgeBacked(block.Metadata.Team, from, to, runtime) {
		return
	}
	addOperatingFinding(result, OperatingGraphFinding{Rule: "graph_edge_unbacked", Severity: string(SeverityError), GraphID: block.Metadata.ID, Team: block.Metadata.Team, Edge: edge.From + "->" + edge.To, SourcePath: block.Source.Path, Line: edge.SourceLine, Detail: fmt.Sprintf("edge %s:%s -> %s:%s is not backed by runtime declarations", from.Kind, from.Value, to.Kind, to.Value)})
}

func operatingEdgeBacked(team string, from, to OperatingGraphNode, runtime OperatingGraphRuntime) bool {
	rel, ok := operatingRelationshipFromNodes(team, "", 0, from, to)
	if !ok {
		return false
	}
	return RelationshipBackedByRuntime(rel, BuildRuntimeOperatingRelationships(runtime, team))
}

func validateOperatingGraphCompleteness(result *OperatingGraphValidationResult, block OperatingGraphBlock, idx operatingGraphIndex, runtime OperatingGraphRuntime) {
	for _, m := range runtime.Members {
		if m.Ref.Team != block.Metadata.Team {
			continue
		}
		memberID := nodeIDFor(idx, "member", m.Ref.Member)
		if memberID == "" {
			continue
		}
		for _, in := range m.Topics.Intake {
			if !idx.hasEdgeToMemberWithTopic(memberID, in.Prefix) {
				addOperatingFinding(result, OperatingGraphFinding{Rule: "graph_declared_intake_missing", Severity: string(SeverityError), GraphID: block.Metadata.ID, Team: block.Metadata.Team, Member: m.Ref.Member, Topic: in.Prefix, SourcePath: block.Source.Path, Detail: fmt.Sprintf("declared intake %s/%s %q is missing from the contract graph", m.Ref.Team, m.Ref.Member, in.Prefix)})
			}
		}
		for _, out := range m.Topics.Output {
			if !idx.hasMemberOutput(memberID, out) {
				addOperatingFinding(result, OperatingGraphFinding{Rule: "graph_declared_output_missing", Severity: string(SeverityWarning), GraphID: block.Metadata.ID, Team: block.Metadata.Team, Member: m.Ref.Member, Topic: out.Prefix, SourcePath: block.Source.Path, Detail: fmt.Sprintf("declared output %s/%s %q is missing from the contract graph", m.Ref.Team, m.Ref.Member, out.Prefix)})
			}
		}
	}
}

func (r OperatingGraphRuntime) member(team, member string) (MemberTopics, bool) {
	for _, m := range r.Members {
		if m.Ref.Team == team && m.Ref.Member == member {
			return m, true
		}
	}
	return MemberTopics{}, false
}

func (r OperatingGraphRuntime) topicDeclared(team, topic string) bool {
	for _, m := range r.Members {
		if m.Ref.Team != team {
			continue
		}
		if memberReadsTopic(m.Topics, topic) {
			return true
		}
		for _, out := range m.Topics.Output {
			if Overlap(out.Prefix, topic) {
				return true
			}
		}
	}
	return false
}

func memberReadsTopic(topics Topics, topic string) bool {
	for _, in := range topics.Intake {
		if Overlap(in.Prefix, topic) {
			return true
		}
	}
	for _, read := range topics.RequiredRead {
		if Overlap(read.Prefix, topic) {
			return true
		}
	}
	for _, ev := range topics.EvidenceConsumed {
		if Overlap(ev.Prefix, topic) {
			return true
		}
	}
	return false
}

func evidenceForDecision(topics Topics, decision string) bool {
	for _, ev := range topics.EvidenceConsumed {
		if stringInSlice(decision, ev.ForDecisions) {
			return true
		}
	}
	return false
}

func nodeIDFor(idx operatingGraphIndex, kind, value string) string {
	return idx.byKindValue[kind+"\x00"+value]
}

func (idx operatingGraphIndex) hasEdgeToMemberWithTopic(memberID, topic string) bool {
	for _, edge := range idx.edges {
		if edge.To != memberID {
			continue
		}
		from := idx.nodes[edge.From]
		if from.Kind == "topic" && Overlap(from.Value, topic) {
			return true
		}
	}
	return false
}

func (idx operatingGraphIndex) hasMemberOutput(memberID string, out OutputEntry) bool {
	for _, edge := range idx.edges {
		if edge.From != memberID {
			continue
		}
		to := idx.nodes[edge.To]
		switch out.DestinationKind {
		case DestinationPORFile:
			if to.Kind == "por" && out.DestinationPath != nil && to.Value == *out.DestinationPath {
				return true
			}
		default:
			if to.Kind == "topic" && Overlap(out.Prefix, to.Value) {
				return true
			}
		}
	}
	return false
}

func addOperatingFinding(result *OperatingGraphValidationResult, f OperatingGraphFinding) {
	result.Findings = append(result.Findings, f)
	switch f.Severity {
	case string(SeverityError):
		result.Errors++
	case string(SeverityWarning):
		result.Warnings++
	}
}

func stringInSlice(value string, values []string) bool {
	value = strings.TrimSpace(value)
	for _, candidate := range values {
		if strings.TrimSpace(candidate) == value {
			return true
		}
	}
	return false
}

func operatingGraphFileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
