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
		idx := indexOperatingGraph(block.Graph)
		for _, m := range runtime.Members {
			if m.Ref.Team != block.Metadata.Team {
				continue
			}
			memberID := nodeIDFor(idx, "member", m.Ref.Member)
			for _, in := range m.Topics.Intake {
				if memberID == "" || !idx.hasEdgeToMemberWithTopic(memberID, in.Prefix) {
					diffs = append(diffs, OperatingGraphContractDiff{Kind: "declared_intake_missing", Team: m.Ref.Team, Member: m.Ref.Member, Topic: in.Prefix, Detail: fmt.Sprintf("%s/%s intake %q is absent from contract graph", m.Ref.Team, m.Ref.Member, in.Prefix)})
				}
			}
			for _, out := range m.Topics.Output {
				if memberID == "" || !idx.hasMemberOutput(memberID, out) {
					diffs = append(diffs, OperatingGraphContractDiff{Kind: "declared_output_missing", Team: m.Ref.Team, Member: m.Ref.Member, Topic: out.Prefix, Detail: fmt.Sprintf("%s/%s output %q is absent from contract graph", m.Ref.Team, m.Ref.Member, out.Prefix)})
				}
			}
		}
	}
	sort.Slice(diffs, func(i, j int) bool {
		if diffs[i].Kind != diffs[j].Kind {
			return diffs[i].Kind < diffs[j].Kind
		}
		return diffs[i].Detail < diffs[j].Detail
	})
	return diffs
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
	switch {
	case from.Kind == "external" && to.Kind == "topic":
		for _, m := range runtime.Members {
			if m.Ref.Team != team || !stringInSlice(from.Value, m.Topics.ExternalProducers) {
				continue
			}
			for _, in := range m.Topics.Intake {
				if Overlap(in.Prefix, to.Value) {
					return true
				}
			}
		}
	case from.Kind == "topic" && to.Kind == "member":
		if mt, ok := runtime.member(team, to.Value); ok {
			return memberReadsTopic(mt.Topics, from.Value)
		}
	case from.Kind == "member" && to.Kind == "topic":
		if mt, ok := runtime.member(team, from.Value); ok {
			for _, out := range mt.Topics.Output {
				if out.DestinationKind == DestinationKnowledge && Overlap(out.Prefix, to.Value) {
					return true
				}
			}
		}
	case from.Kind == "member" && to.Kind == "decision":
		if mt, ok := runtime.member(team, from.Value); ok {
			return stringInSlice(to.Value, mt.Topics.DecisionsOwned) || (to.Value == "capability-gap" && mt.Topics.RaisesCapabilityGaps)
		}
	case from.Kind == "decision" && to.Kind == "member":
		if mt, ok := runtime.member(team, to.Value); ok {
			return stringInSlice(from.Value, mt.Topics.DecisionsConsumed) || evidenceForDecision(mt.Topics, from.Value)
		}
	case from.Kind == "member" && to.Kind == "por":
		if mt, ok := runtime.member(team, from.Value); ok {
			for _, out := range mt.Topics.Output {
				if out.DestinationKind == DestinationPORFile && out.DestinationPath != nil && *out.DestinationPath == to.Value {
					return true
				}
			}
		}
	case from.Kind == "topic" && to.Kind == "team":
		for _, m := range runtime.Members {
			if m.Ref.Team != team {
				continue
			}
			for _, out := range m.Topics.Output {
				if out.DestinationTeam != nil && *out.DestinationTeam == to.Value && Overlap(out.Prefix, from.Value) {
					return true
				}
			}
		}
	case from.Kind == "external" && to.Kind == "member":
		if mt, ok := runtime.member(team, to.Value); ok {
			return stringInSlice(from.Value, mt.Topics.ExternalProducers)
		}
	}
	return false
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
