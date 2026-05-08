package memberflow

import (
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
)

var (
	flowchartRE      = regexp.MustCompile(`^\s*flowchart\s+(LR|TB|RL|BT)\s*$`)
	subgraphRE       = regexp.MustCompile(`^\s*subgraph\s+([A-Za-z][A-Za-z0-9_]*)(?:\s*\[(.*)\])?\s*$`)
	directionRE      = regexp.MustCompile(`^\s*direction\s+(LR|TB|RL|BT)\s*$`)
	edgeRE           = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_]*)\s*-->(?:\|([^|]*)\|)?\s*([A-Za-z][A-Za-z0-9_]*)\s*$`)
	nodeAnnotRE      = regexp.MustCompile(`^\s*%%\s*@node\s+([A-Za-z][A-Za-z0-9_]*)\s+(.+?)\s*$`)
	typedLabelRE     = regexp.MustCompile(`^([a-z][a-z0-9_-]*)(?:\[([a-z][a-z0-9_-]*)\])?:(.+)$`)
	nodeRectangleRE  = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_]*)\s*\[(.*)\]\s*$`)
	nodeCylinderRE   = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_]*)\s*\[\((.*)\)\]\s*$`)
	nodeDiamondRE    = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_]*)\s*\{(.*)\}\s*$`)
	nodeStadiumRE    = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_]*)\s*\(\[(.*)\]\)\s*$`)
	nodeSubroutineRE = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_]*)\s*\[\[(.*)\]\]\s*$`)
	nodeDocumentRE   = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_]*)\s*\[/(.*)/\]\s*$`)
)

type operatingGraphNodeAnnotation struct {
	NodeID     string
	Kind       OperatingGraphNodeKind
	Value      string
	Qualifier  OperatingGraphQualifier
	SourceLine int
}

type operatingGraphNodeLabel struct {
	Raw       string
	Display   string
	Kind      OperatingGraphNodeKind
	Value     string
	Qualifier OperatingGraphQualifier
	Typed     bool
}

type operatingGraphNodeDeclaration struct {
	ID       string
	RawLabel string
	Shape    OperatingGraphNodeShape
}

func ParseOperatingMermaid(id string, lines []string, firstLine int) (OperatingGraph, error) {
	graph := OperatingGraph{ID: id}
	nodes := map[string]OperatingGraphNode{}
	annotations := map[string]operatingGraphNodeAnnotation{}
	var activeGroup *OperatingGraphGroup
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
		if m := subgraphRE.FindStringSubmatch(line); m != nil {
			if activeGroup != nil {
				return graph, fmt.Errorf("nested subgraph %q at line %d is not supported; close %q first", m[1], lineNo, activeGroup.ID)
			}
			display := cleanOperatingGraphLabel(m[2])
			if display == "" {
				display = m[1]
			}
			graph.Groups = append(graph.Groups, OperatingGraphGroup{
				ID:         m[1],
				Display:    display,
				SourceLine: lineNo,
			})
			activeGroup = &graph.Groups[len(graph.Groups)-1]
			continue
		}
		if line == "end" {
			if activeGroup == nil {
				return graph, fmt.Errorf("subgraph end at line %d has no matching subgraph", lineNo)
			}
			activeGroup = nil
			continue
		}
		if activeGroup != nil && directionRE.MatchString(line) {
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
		if decl, ok := parseOperatingGraphNodeDeclaration(line); ok {
			node, err := parseOperatingGraphNode(decl.ID, decl.RawLabel, decl.Shape, lineNo)
			if err != nil {
				return graph, err
			}
			nodes[node.ID] = node
			if activeGroup != nil {
				activeGroup.NodeIDs = append(activeGroup.NodeIDs, node.ID)
			}
			continue
		}
		return graph, fmt.Errorf("unsupported mermaid syntax at line %d: %s", lineNo, line)
	}
	if graph.Direction == "" {
		return graph, fmt.Errorf("missing flowchart direction")
	}
	if activeGroup != nil {
		return graph, fmt.Errorf("subgraph %q opened at line %d is missing end", activeGroup.ID, activeGroup.SourceLine)
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

func parseOperatingGraphNodeDeclaration(line string) (operatingGraphNodeDeclaration, bool) {
	for _, parser := range []struct {
		shape OperatingGraphNodeShape
		re    *regexp.Regexp
	}{
		{OperatingGraphNodeShapeCylinder, nodeCylinderRE},
		{OperatingGraphNodeShapeStadium, nodeStadiumRE},
		{OperatingGraphNodeShapeSubroutine, nodeSubroutineRE},
		{OperatingGraphNodeShapeDocument, nodeDocumentRE},
		{OperatingGraphNodeShapeDiamond, nodeDiamondRE},
		{OperatingGraphNodeShapeRectangle, nodeRectangleRE},
	} {
		m := parser.re.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		return operatingGraphNodeDeclaration{
			ID:       m[1],
			RawLabel: m[2],
			Shape:    parser.shape,
		}, true
	}
	return operatingGraphNodeDeclaration{}, false
}

func parseOperatingGraphNode(id, rawLabel string, shape OperatingGraphNodeShape, line int) (OperatingGraphNode, error) {
	label := parseOperatingGraphNodeLabel(rawLabel)
	node := OperatingGraphNode{
		ID:         id,
		Display:    label.Display,
		RawLabel:   label.Raw,
		Shape:      shape,
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
	label := cleanOperatingGraphLabel(rawLabel)
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

func cleanOperatingGraphLabel(rawLabel string) string {
	label := strings.TrimSpace(rawLabel)
	label = strings.Trim(label, `"`)
	label = html.UnescapeString(label)
	return label
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

func parseOperatingGraphTypedToken(token string) (kind OperatingGraphNodeKind, qualifier OperatingGraphQualifier, value string, ok bool) {
	m := typedLabelRE.FindStringSubmatch(strings.TrimSpace(token))
	if m == nil {
		return "", "", "", false
	}
	return OperatingGraphNodeKind(m[1]), OperatingGraphQualifier(m[2]), strings.TrimSpace(m[3]), true
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
