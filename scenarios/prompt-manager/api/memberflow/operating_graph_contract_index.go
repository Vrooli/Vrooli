package memberflow

import (
	"strconv"
	"strings"
)

type OperatingGraphContractContext struct {
	Block   OperatingGraphBlock
	Runtime OperatingGraphRuntime
	Index   OperatingGraphContractIndex
	Matcher OperatingRelationshipMatcher
}

type OperatingGraphContractIndex struct {
	NodesByID             map[string]OperatingGraphNode
	NodeIDByKindValue     map[string]string
	GraphRelationships    OperatingRelationshipSet
	RuntimeRelationships  OperatingRelationshipSet
	graphRelationshipsByE map[string]OperatingRelationship
}

func NewOperatingGraphContractContext(block OperatingGraphBlock, runtime OperatingGraphRuntime) OperatingGraphContractContext {
	return OperatingGraphContractContext{
		Block:   block,
		Runtime: runtime,
		Index:   NewOperatingGraphContractIndex(block, runtime),
		Matcher: NewOperatingRelationshipMatcher(),
	}
}

func NewOperatingGraphContractIndex(block OperatingGraphBlock, runtime OperatingGraphRuntime) OperatingGraphContractIndex {
	registry := DefaultOperatingRelationshipRegistry()
	idx := OperatingGraphContractIndex{
		NodesByID:             map[string]OperatingGraphNode{},
		NodeIDByKindValue:     map[string]string{},
		graphRelationshipsByE: map[string]OperatingRelationship{},
	}
	for _, node := range block.Graph.Nodes {
		idx.NodesByID[node.ID] = node
		if node.Kind != "" && node.Value != "" {
			idx.NodeIDByKindValue[string(node.Kind)+"\x00"+node.Value] = node.ID
		}
	}
	var graphRels []OperatingRelationship
	for _, edge := range block.Graph.Edges {
		from, fok := idx.NodesByID[edge.From]
		to, tok := idx.NodesByID[edge.To]
		if !fok || !tok || !operatingGraphEdgeActionable(from, to) {
			continue
		}
		rel, ok := registry.RelationshipFromEdge(block.Metadata.Team, OperatingSourceRef{Path: block.Source.Path, Line: edge.SourceLine}, from, to)
		if !ok {
			continue
		}
		graphRels = append(graphRels, rel)
		idx.graphRelationshipsByE[operatingEdgeKey(edge)] = rel
	}
	idx.GraphRelationships = NewOperatingRelationshipSet(graphRels)
	idx.RuntimeRelationships = NewOperatingRelationshipSet(BuildRuntimeOperatingRelationships(runtime, block.Metadata.Team))
	return idx
}

func (idx OperatingGraphContractIndex) Node(kind, value string) (OperatingGraphNode, bool) {
	nodeID := idx.NodeIDByKindValue[kind+"\x00"+value]
	if nodeID == "" {
		return OperatingGraphNode{}, false
	}
	node, ok := idx.NodesByID[nodeID]
	return node, ok
}

func (idx OperatingGraphContractIndex) GraphHasRelationship(rel OperatingRelationship) bool {
	for _, graphRel := range idx.GraphRelationships.All() {
		if operatingGraphRelationshipsEquivalent(rel, graphRel) {
			return true
		}
	}
	return false
}

func (idx OperatingGraphContractIndex) RuntimeHasRelationship(rel OperatingRelationship, matcher OperatingRelationshipMatcher) bool {
	return matcher.GraphBackedByRuntime(rel, idx.RuntimeRelationships)
}

func (idx OperatingGraphContractIndex) RuntimeRelationshipsByMember(member string) []OperatingRelationship {
	return idx.RuntimeRelationships.ByMember(member)
}

func (idx OperatingGraphContractIndex) RuntimeRelationshipsByKind(kind OperatingRelationshipKind) []OperatingRelationship {
	return idx.RuntimeRelationships.ByKind(kind)
}

func (idx OperatingGraphContractIndex) RelationshipForEdge(edge OperatingGraphEdge) (OperatingRelationship, bool) {
	rel, ok := idx.graphRelationshipsByE[operatingEdgeKey(edge)]
	return rel, ok
}

func operatingGraphNodeNonActionable(node OperatingGraphNode) bool {
	if node.Kind == "" || node.Kind == OperatingGraphNodeKindProcess || node.Kind == OperatingGraphNodeKindFuture {
		return true
	}
	return node.Kind == OperatingGraphNodeKindTopic && (node.Qualifier == OperatingGraphQualifierFuture || node.Qualifier == OperatingGraphQualifierOld || node.Qualifier == OperatingGraphQualifierExternal)
}

func operatingGraphEdgeActionable(from, to OperatingGraphNode) bool {
	return !operatingGraphNodeNonActionable(from) && !operatingGraphNodeNonActionable(to)
}

func operatingGraphRelationshipsEquivalent(a, b OperatingRelationship) bool {
	if a.Kind != b.Kind || a.Team != b.Team || a.Member != b.Member || a.Decision != b.Decision || a.External != b.External || a.ProducerTeam != b.ProducerTeam || a.TargetTeam != b.TargetTeam {
		return false
	}
	if a.Path != "" || b.Path != "" {
		return pathsEqual(a.Path, b.Path)
	}
	if a.Topic != "" || b.Topic != "" {
		return topicsOverlap(a.Topic, b.Topic)
	}
	return true
}

func operatingEdgeKey(edge OperatingGraphEdge) string {
	return strings.Join([]string{edge.From, edge.To, strconv.Itoa(edge.SourceLine)}, "\x00")
}
