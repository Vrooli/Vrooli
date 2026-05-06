package memberflow

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type OperatingGraphMetadata struct {
	ID     string            `json:"id"`
	Scope  string            `json:"scope"`
	Team   string            `json:"team"`
	Mode   string            `json:"mode"`
	Status string            `json:"status,omitempty"`
	Allow  []string          `json:"allow,omitempty"`
	Extra  map[string]string `json:"extra,omitempty"`
}

type OperatingGraphSource struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	FenceLine int    `json:"fence_line"`
}

type OperatingGraphBlock struct {
	Metadata OperatingGraphMetadata `json:"metadata"`
	Graph    OperatingGraph         `json:"graph"`
	Source   OperatingGraphSource   `json:"source"`
}

type OperatingGraph struct {
	ID        string               `json:"id"`
	Direction string               `json:"direction"`
	Nodes     []OperatingGraphNode `json:"nodes"`
	Edges     []OperatingGraphEdge `json:"edges"`
}

type OperatingGraphNode struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Value      string `json:"value"`
	Qualifier  string `json:"qualifier,omitempty"`
	Display    string `json:"display,omitempty"`
	RawLabel   string `json:"raw_label"`
	SourceLine int    `json:"source_line"`
	Implicit   bool   `json:"implicit,omitempty"`
}

type OperatingGraphEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Label      string `json:"label,omitempty"`
	SourceLine int    `json:"source_line"`
}

type OperatingGraphFinding struct {
	Rule       string `json:"rule"`
	Severity   string `json:"severity"`
	GraphID    string `json:"graph_id,omitempty"`
	Team       string `json:"team,omitempty"`
	NodeID     string `json:"node_id,omitempty"`
	Edge       string `json:"edge,omitempty"`
	Member     string `json:"member,omitempty"`
	Topic      string `json:"topic,omitempty"`
	Decision   string `json:"decision,omitempty"`
	Path       string `json:"path,omitempty"`
	SourcePath string `json:"source_path,omitempty"`
	Line       int    `json:"line,omitempty"`
	Detail     string `json:"detail"`
}

type OperatingGraphValidationResult struct {
	Findings []OperatingGraphFinding `json:"findings"`
	Errors   int                     `json:"errors"`
	Warnings int                     `json:"warnings"`
}

type OperatingGraphListResponse struct {
	Graphs []OperatingGraphBlock `json:"graphs"`
}

type OperatingGraphValidationResponse struct {
	Graphs     []OperatingGraphBlock          `json:"graphs"`
	Validation OperatingGraphValidationResult `json:"validation"`
}

type OperatingGraphDiffResponse struct {
	Graphs []OperatingGraphBlock        `json:"graphs"`
	Diff   []OperatingGraphContractDiff `json:"diff"`
}

type OperatingGraphContractDiff struct {
	Kind             string   `json:"kind"`
	Relationship     string   `json:"relationship"`
	Team             string   `json:"team"`
	Member           string   `json:"member,omitempty"`
	Topic            string   `json:"topic,omitempty"`
	Decision         string   `json:"decision,omitempty"`
	Path             string   `json:"path,omitempty"`
	External         string   `json:"external,omitempty"`
	TargetTeam       string   `json:"target_team,omitempty"`
	SourcePath       string   `json:"source_path,omitempty"`
	Line             int      `json:"line,omitempty"`
	RuntimePath      string   `json:"runtime_path,omitempty"`
	AcceptableFields []string `json:"acceptable_fields,omitempty"`
	Suggestions      []string `json:"suggestions,omitempty"`
	Detail           string   `json:"detail"`
}

var (
	metadataStartRE = regexp.MustCompile(`^\s*<!--\s*prompt-manager-graph:\s*$`)
	metadataEndRE   = regexp.MustCompile(`^\s*-->\s*$`)
	flowchartRE     = regexp.MustCompile(`^\s*flowchart\s+(LR|TB|RL|BT)\s*$`)
	nodeDeclRE      = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_]*)\s*\[(.*)\]\s*$`)
	edgeRE          = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_]*)\s*-->(?:\|([^|]*)\|)?\s*([A-Za-z][A-Za-z0-9_]*)\s*$`)
	nodeAnnotRE     = regexp.MustCompile(`^\s*%%\s*@node\s+([A-Za-z][A-Za-z0-9_]*)\s+(.+?)\s*$`)
	typedLabelRE    = regexp.MustCompile(`^([a-z][a-z0-9_-]*)(?:\[([a-z][a-z0-9_-]*)\])?:(.+)$`)
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

func LoadOperatingGraphBlocks(repoRoot string) ([]OperatingGraphBlock, error) {
	docsDir := filepath.Join(repoRoot, "docs")
	var blocks []OperatingGraphBlock
	err := filepath.WalkDir(docsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}
		rel, _ := filepath.Rel(repoRoot, path)
		found, err := ExtractOperatingGraphBlocks(path, rel)
		if err != nil {
			return err
		}
		blocks = append(blocks, found...)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].Metadata.ID != blocks[j].Metadata.ID {
			return blocks[i].Metadata.ID < blocks[j].Metadata.ID
		}
		return blocks[i].Source.Path < blocks[j].Source.Path
	})
	return blocks, nil
}

func ExtractOperatingGraphBlocks(path, sourcePath string) ([]OperatingGraphBlock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var blocks []OperatingGraphBlock
	inFence := false
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if !metadataStartRE.MatchString(lines[i]) {
			continue
		}
		metaStart := i + 1
		var metaLines []string
		i++
		for i < len(lines) && !metadataEndRE.MatchString(lines[i]) {
			metaLines = append(metaLines, lines[i])
			i++
		}
		if i >= len(lines) {
			return nil, fmt.Errorf("%s:%d: unterminated prompt-manager-graph metadata", sourcePath, metaStart)
		}
		meta, err := parseOperatingGraphMetadata(metaLines)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", sourcePath, metaStart, err)
		}
		i++
		for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
			i++
		}
		if i >= len(lines) || strings.TrimSpace(lines[i]) != "```mermaid" {
			return nil, fmt.Errorf("%s:%d: prompt-manager-graph metadata must be followed by a mermaid fence", sourcePath, metaStart)
		}
		fenceLine := i + 1
		i++
		var mermaid []string
		for i < len(lines) && strings.TrimSpace(lines[i]) != "```" {
			mermaid = append(mermaid, lines[i])
			i++
		}
		if i >= len(lines) {
			return nil, fmt.Errorf("%s:%d: unterminated mermaid fence", sourcePath, fenceLine)
		}
		graph, err := ParseOperatingMermaid(meta.ID, mermaid, fenceLine+1)
		if err != nil && meta.Mode == "contract" {
			return nil, fmt.Errorf("%s:%d: %w", sourcePath, fenceLine, err)
		}
		if err == nil {
			blocks = append(blocks, OperatingGraphBlock{
				Metadata: meta,
				Graph:    graph,
				Source: OperatingGraphSource{
					Path:      sourcePath,
					Line:      metaStart,
					FenceLine: fenceLine,
				},
			})
		}
	}
	return blocks, nil
}

func parseOperatingGraphMetadata(lines []string) (OperatingGraphMetadata, error) {
	meta := OperatingGraphMetadata{Extra: map[string]string{}}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return meta, fmt.Errorf("metadata line %q must be key: value", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "id":
			meta.ID = value
		case "scope":
			meta.Scope = value
		case "team":
			meta.Team = value
		case "mode":
			meta.Mode = value
		case "status":
			meta.Status = value
		case "allow":
			meta.Allow = splitCSV(value)
		default:
			meta.Extra[key] = value
		}
	}
	if meta.ID == "" || meta.Scope == "" || meta.Team == "" || meta.Mode == "" {
		return meta, fmt.Errorf("metadata requires id, scope, team, and mode")
	}
	switch meta.Mode {
	case "explanatory", "checkable", "contract":
	default:
		return meta, fmt.Errorf("unsupported mode %q", meta.Mode)
	}
	return meta, nil
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

func splitCSV(value string) []string {
	value = strings.Trim(value, "[]")
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.Trim(strings.TrimSpace(part), `"`)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
