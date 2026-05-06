package memberflow

import (
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
)

var (
	flowchartRE  = regexp.MustCompile(`^\s*flowchart\s+(LR|TB|RL|BT)\s*$`)
	nodeDeclRE   = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_]*)\s*\[(.*)\]\s*$`)
	edgeRE       = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_]*)\s*-->(?:\|([^|]*)\|)?\s*([A-Za-z][A-Za-z0-9_]*)\s*$`)
	nodeAnnotRE  = regexp.MustCompile(`^\s*%%\s*@node\s+([A-Za-z][A-Za-z0-9_]*)\s+(.+?)\s*$`)
	typedLabelRE = regexp.MustCompile(`^([a-z][a-z0-9_-]*)(?:\[([a-z][a-z0-9_-]*)\])?:(.+)$`)
)

type operatingGraphNodeAnnotation struct {
	NodeID     string
	Kind       string
	Value      string
	Qualifier  string
	SourceLine int
}

type operatingGraphNodeLabel struct {
	Raw       string
	Display   string
	Kind      string
	Value     string
	Qualifier string
	Typed     bool
}

func ParseOperatingMermaid(id string, lines []string, firstLine int) (OperatingGraph, error) {
	graph := OperatingGraph{ID: id}
	nodes := map[string]OperatingGraphNode{}
	annotations := map[string]operatingGraphNodeAnnotation{}
	for i, raw := range lines {
		lineNo := firstLine + i
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if m := nodeAnnotRE.FindStringSubmatch(line); m != nil {
			ann, err := parseOperatingGraphNodeAnnotation(m[1], m[2], lineNo)
			if err != nil {
				return graph, err
			}
			if existing, ok := annotations[ann.NodeID]; ok {
				return graph, fmt.Errorf("duplicate @node annotation for %q at line %d; first declared at line %d", ann.NodeID, lineNo, existing.SourceLine)
			}
			annotations[ann.NodeID] = ann
			continue
		}
		if strings.HasPrefix(line, "%%") {
			continue
		}
		if m := flowchartRE.FindStringSubmatch(line); m != nil {
			if graph.Direction != "" {
				return graph, fmt.Errorf("multiple flowchart declarations")
			}
			graph.Direction = m[1]
			continue
		}
		if m := edgeRE.FindStringSubmatch(line); m != nil {
			graph.Edges = append(graph.Edges, OperatingGraphEdge{
				From:       m[1],
				To:         m[3],
				Label:      strings.TrimSpace(m[2]),
				SourceLine: lineNo,
			})
			if _, ok := nodes[m[1]]; !ok {
				nodes[m[1]] = OperatingGraphNode{ID: m[1], Implicit: true, SourceLine: lineNo}
			}
			if _, ok := nodes[m[3]]; !ok {
				nodes[m[3]] = OperatingGraphNode{ID: m[3], Implicit: true, SourceLine: lineNo}
			}
			continue
		}
		if m := nodeDeclRE.FindStringSubmatch(line); m != nil {
			node, err := parseOperatingGraphNode(m[1], m[2], lineNo)
			if err != nil {
				return graph, err
			}
			nodes[node.ID] = node
			continue
		}
		return graph, fmt.Errorf("unsupported mermaid syntax at line %d: %s", lineNo, line)
	}
	if graph.Direction == "" {
		return graph, fmt.Errorf("missing flowchart direction")
	}
	if err := applyOperatingGraphNodeAnnotations(nodes, annotations); err != nil {
		return graph, err
	}
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		graph.Nodes = append(graph.Nodes, nodes[id])
	}
	return graph, nil
}

func parseOperatingGraphNode(id, rawLabel string, line int) (OperatingGraphNode, error) {
	label := parseOperatingGraphNodeLabel(rawLabel)
	node := OperatingGraphNode{
		ID:         id,
		Display:    label.Display,
		RawLabel:   label.Raw,
		SourceLine: line,
	}
	if label.Typed {
		node.Kind = label.Kind
		node.Qualifier = label.Qualifier
		node.Value = label.Value
	}
	return node, nil
}

func parseOperatingGraphNodeLabel(rawLabel string) operatingGraphNodeLabel {
	label := strings.TrimSpace(rawLabel)
	label = strings.Trim(label, `"`)
	label = html.UnescapeString(label)
	parts := strings.Split(label, "<br/>")
	token := strings.TrimSpace(parts[0])
	display := strings.TrimSpace(strings.Join(parts, " "))
	kind, qualifier, value, ok := parseOperatingGraphTypedToken(token)
	if !ok {
		return operatingGraphNodeLabel{Raw: label, Display: display}
	}
	if len(parts) > 1 {
		display = strings.TrimSpace(strings.Join(parts[1:], " "))
	} else {
		display = ""
	}
	return operatingGraphNodeLabel{
		Raw:       label,
		Display:   display,
		Kind:      kind,
		Qualifier: qualifier,
		Value:     value,
		Typed:     true,
	}
}

func parseOperatingGraphNodeAnnotation(nodeID, token string, line int) (operatingGraphNodeAnnotation, error) {
	kind, qualifier, value, ok := parseOperatingGraphTypedToken(strings.TrimSpace(token))
	if !ok {
		return operatingGraphNodeAnnotation{}, fmt.Errorf("invalid @node annotation at line %d: expected %q to be kind:value", line, token)
	}
	return operatingGraphNodeAnnotation{
		NodeID:     nodeID,
		Kind:       kind,
		Qualifier:  qualifier,
		Value:      value,
		SourceLine: line,
	}, nil
}

func parseOperatingGraphTypedToken(token string) (kind, qualifier, value string, ok bool) {
	m := typedLabelRE.FindStringSubmatch(strings.TrimSpace(token))
	if m == nil {
		return "", "", "", false
	}
	return m[1], m[2], strings.TrimSpace(m[3]), true
}

func applyOperatingGraphNodeAnnotations(nodes map[string]OperatingGraphNode, annotations map[string]operatingGraphNodeAnnotation) error {
	for nodeID, ann := range annotations {
		node, ok := nodes[nodeID]
		if !ok || node.Implicit {
			return fmt.Errorf("@node annotation for %q at line %d does not match a declared node", nodeID, ann.SourceLine)
		}
		if node.Kind != "" || node.Value != "" || node.Qualifier != "" {
			return fmt.Errorf("@node annotation for %q at line %d conflicts with inline typed node label at line %d", nodeID, ann.SourceLine, node.SourceLine)
		}
		node.Kind = ann.Kind
		node.Qualifier = ann.Qualifier
		node.Value = ann.Value
		nodes[nodeID] = node
	}
	return nil
}
