package memberflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Generation of a team's `## Operating Graph` block from its declarations.
//
// The block was hand-drawn beside the declarations it restates. Roughly forty
// validation rules existed to compare the two, and 199 `%% @node` annotations
// across six documents had to be updated by hand whenever topics.json changed.
// A generated block cannot disagree with its source, so those rules stop having
// a question to answer.
//
// The relationship registry is the single source for what edges exist —
// DefaultOperatingRelationshipRegistry already maps every family to its runtime
// fields, and this generator reads that rather than restating the mapping.

// GenerateOperatingGraphInput is everything the generator reads. All of it is
// declaration or presentation; nothing is parsed back out of the document being
// generated.
type GenerateOperatingGraphInput struct {
	TeamID       string
	Runtime      OperatingGraphRuntime
	Presentation GraphPresentation
}

// generatedNode is one typed node on its way to Mermaid.
type generatedNode struct {
	Value   string // typed value, e.g. "member:portfolio-manager"
	Kind    OperatingGraphNodeKind
	Raw     string // the kind-stripped value
	ID      string
	Display string
}

// GenerateOperatingGraphBlock renders the fenced Mermaid diagram and its
// `%% @node` annotations from the team's runtime relationships.
func GenerateOperatingGraphBlock(in GenerateOperatingGraphInput) (string, error) {
	rels := BuildRuntimeOperatingRelationships(in.Runtime, in.TeamID)

	nodes := map[string]*generatedNode{}
	addNode := func(kind OperatingGraphNodeKind, raw string) *generatedNode {
		if raw == "" {
			return nil
		}
		value := string(kind) + ":" + raw
		if existing, ok := nodes[value]; ok {
			return existing
		}
		node := &generatedNode{Value: value, Kind: kind, Raw: raw}
		nodes[value] = node
		return node
	}

	type generatedEdge struct{ from, to string }
	edgeSet := map[generatedEdge]bool{}

	registry := DefaultOperatingRelationshipRegistry()
	for _, rel := range rels {
		graphKind := registry.GraphKindForRuntime(rel.Kind)
		spec, ok := registry.Spec(graphKind)
		if !ok {
			continue
		}
		from := generatedEndpointFor(rel, spec.GraphShape.FromKind, in.TeamID)
		to := generatedEndpointFor(rel, spec.GraphShape.ToKind, in.TeamID)
		if from.raw == "" || to.raw == "" {
			continue
		}
		fromNode := addNode(from.kind, from.raw)
		toNode := addNode(to.kind, to.raw)
		if fromNode == nil || toNode == nil {
			continue
		}
		edgeSet[generatedEdge{from: fromNode.Value, to: toNode.Value}] = true
	}

	// Plan-of-record surfaces are declared in the team's manifest.json, not in
	// topics.json, so deriving them from relationships alone misses every
	// document a team registers but no member names as an output destination.
	// The manifest is a declaration like any other; omitting it would have made
	// generation drop real plan-of-record nodes and look like a data loss.
	for _, por := range declaredPlanOfRecordSurfaces(in.Runtime.RepoRoot, in.TeamID) {
		addNode(OperatingGraphNodeKindPOR, por)
	}

	// A team's instrument is declared in its own contract, not in topics.json
	// and not in the plan-of-record manifest, so it needs its own pass. Without
	// it the generator would report a drawn instrument node as undeclared drift
	// even though the team declares the scenario explicitly — which is what
	// happened when infra-health first moved its durable numbers into one.
	if instrument := declaredInstrumentScenario(in.Runtime.RepoRoot, in.TeamID); instrument != "" {
		addNode(OperatingGraphNodeKindInstrument, instrument)
	}

	// Presentation may introduce process/future nodes that no declaration
	// mentions; Validate has already confirmed they carry no contract meaning.
	for _, edge := range in.Presentation.ReadabilityEdges {
		for _, value := range []string{edge.From, edge.To} {
			kind, raw, ok := splitTypedValue(value)
			if !ok {
				return "", fmt.Errorf("readability edge endpoint %q is not a typed value", value)
			}
			addNode(kind, raw)
		}
		edgeSet[generatedEdge{from: edge.From, to: edge.To}] = true
	}

	ordered := orderGeneratedNodes(nodes, in.Presentation)
	assignGeneratedIDs(ordered, in.Presentation)

	var b strings.Builder
	b.WriteString("```mermaid\nflowchart LR\n")

	emitted := map[string]bool{}
	for _, subgraph := range in.Presentation.Subgraphs {
		members := make([]*generatedNode, 0, len(subgraph.Values))
		for _, value := range subgraph.Values {
			if node, ok := nodes[value]; ok && !emitted[value] {
				// Claim the node as this subgraph selects it, not when it is
				// written. Marking on write let a node pass the guard here and
				// be written again by the loop below, emitting two `%% @node`
				// annotations for one topic. The Mermaid parser dedupes by node
				// id, so the second annotation silently disappeared and the
				// topic vanished from the parsed graph while still holding a
				// row in the Topic Catalog table.
				emitted[value] = true
				members = append(members, node)
			}
		}
		if len(members) == 0 {
			continue
		}
		fmt.Fprintf(&b, "  subgraph %s[%q]\n", subgraph.ID, subgraph.Title)
		for _, node := range members {
			writeGeneratedNode(&b, node, "    ")
		}
		b.WriteString("  end\n\n")
	}
	for _, node := range ordered {
		if emitted[node.Value] {
			continue
		}
		emitted[node.Value] = true
		writeGeneratedNode(&b, node, "  ")
	}

	edges := make([]generatedEdge, 0, len(edgeSet))
	for edge := range edgeSet {
		edges = append(edges, edge)
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].from != edges[j].from {
			return edges[i].from < edges[j].from
		}
		return edges[i].to < edges[j].to
	})
	if len(edges) > 0 {
		b.WriteString("\n")
	}
	for _, edge := range edges {
		from, fok := nodes[edge.from]
		to, tok := nodes[edge.to]
		if !fok || !tok {
			continue
		}
		fmt.Fprintf(&b, "  %s --> %s\n", from.ID, to.ID)
	}
	b.WriteString("```\n")
	return b.String(), nil
}

func writeGeneratedNode(b *strings.Builder, node *generatedNode, indent string) {
	fmt.Fprintf(b, "%s%%%% @node %s %s\n", indent, node.ID, node.Value)
	fmt.Fprintf(b, "%s%s%s\n", indent, node.ID, generatedShape(node.Kind, node.Display))
}

// generatedShape applies the shape convention operatingGraphNodeShapeMatchesKind
// enforces, so a generated node can never trip the shape-drift rule.
func generatedShape(kind OperatingGraphNodeKind, display string) string {
	switch kind {
	case OperatingGraphNodeKindTopic:
		return "[(" + display + ")]"
	case OperatingGraphNodeKindExternal, OperatingGraphNodeKindProcess, OperatingGraphNodeKindFuture:
		return "([" + display + "])"
	case OperatingGraphNodeKindTeam:
		return "[[" + display + "]]"
	case OperatingGraphNodeKindPOR:
		return "[/" + display + "/]"
	default:
		return "[" + display + "]"
	}
}

type generatedEndpoint struct {
	kind OperatingGraphNodeKind
	raw  string
}

// generatedEndpointFor maps one side of a relationship's graph shape onto the
// relationship's own fields.
func generatedEndpointFor(rel OperatingRelationship, kind OperatingGraphNodeKind, teamID string) generatedEndpoint {
	switch kind {
	case OperatingGraphNodeKindMember:
		return generatedEndpoint{kind: kind, raw: rel.Member}
	case OperatingGraphNodeKindTopic:
		return generatedEndpoint{kind: kind, raw: rel.Topic}
	case OperatingGraphNodeKindPOR:
		return generatedEndpoint{kind: kind, raw: rel.Path}
	case OperatingGraphNodeKindExternal:
		return generatedEndpoint{kind: kind, raw: rel.External}
	case OperatingGraphNodeKindTeam:
		target := rel.TargetTeam
		if target == "" {
			target = rel.ProducerTeam
		}
		if target == teamID {
			target = ""
		}
		return generatedEndpoint{kind: kind, raw: target}
	default:
		return generatedEndpoint{kind: kind}
	}
}

func orderGeneratedNodes(nodes map[string]*generatedNode, presentation GraphPresentation) []*generatedNode {
	ordered := make([]*generatedNode, 0, len(nodes))
	seen := map[string]bool{}
	for _, value := range presentation.NodeOrder {
		if node, ok := nodes[value]; ok && !seen[value] {
			ordered = append(ordered, node)
			seen[value] = true
		}
	}
	rest := make([]*generatedNode, 0, len(nodes))
	for value, node := range nodes {
		if !seen[value] {
			rest = append(rest, node)
		}
	}
	// Deterministic: kind first, then value, so a new declaration slots in
	// beside its family instead of reshuffling the diagram.
	sort.Slice(rest, func(i, j int) bool {
		if rest[i].Kind != rest[j].Kind {
			return rest[i].Kind < rest[j].Kind
		}
		return rest[i].Value < rest[j].Value
	})
	return append(ordered, rest...)
}

func assignGeneratedIDs(ordered []*generatedNode, presentation GraphPresentation) {
	used := map[string]bool{}
	for _, node := range ordered {
		if short, ok := presentation.ShortNames[node.Value]; ok && short != "" && !used[short] {
			node.ID = short
			used[short] = true
		}
		if display, ok := presentation.Displays[node.Value]; ok && display != "" {
			node.Display = display
		} else {
			node.Display = node.Raw
		}
	}
	for _, node := range ordered {
		if node.ID != "" {
			continue
		}
		node.ID = uniqueGeneratedID(node.Raw, used)
		used[node.ID] = true
	}
}

// uniqueGeneratedID derives a stable mermaid identifier from a node's value.
func uniqueGeneratedID(raw string, used map[string]bool) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(raw) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	base := b.String()
	if base == "" {
		base = "N"
	}
	if len(base) > 12 {
		base = base[:12]
	}
	candidate := base
	for i := 2; used[candidate]; i++ {
		candidate = fmt.Sprintf("%s%d", base, i)
	}
	return candidate
}

func splitTypedValue(value string) (OperatingGraphNodeKind, string, bool) {
	kind, raw, ok := strings.Cut(value, ":")
	if !ok || kind == "" || raw == "" {
		return "", "", false
	}
	return OperatingGraphNodeKind(kind), raw, true
}

// declaredPlanOfRecordSurfaces returns every document a team's plan-of-record
// manifest registers, as repo-relative paths.
// declaredInstrumentScenario returns the scenario a team names as its
// instrument, or "" when the team declares none. A team without an instrument
// is the normal case; only a team that has moved computable numbers out of its
// plan of record has one.
func declaredInstrumentScenario(repoRoot, teamID string) string {
	if repoRoot == "" || teamID == "" {
		return ""
	}
	path := filepath.Join(repoRoot, "scenarios", "prompt-manager", "store", "teams", teamID, "team.json")
	payload, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var contract struct {
		Instrument struct {
			Scenario string `json:"scenario"`
		} `json:"instrument"`
	}
	if err := json.Unmarshal(payload, &contract); err != nil {
		return ""
	}
	return strings.TrimSpace(contract.Instrument.Scenario)
}

func declaredPlanOfRecordSurfaces(repoRoot, teamID string) []string {
	if repoRoot == "" || teamID == "" {
		return nil
	}
	manifestPath := filepath.Join(repoRoot, "docs", teamID, "manifest.json")
	manifest, err := LoadResolvedPlanOfRecordManifest(repoRoot, manifestPath)
	if err != nil {
		return nil
	}
	var out []string
	for _, section := range manifest.Sections {
		for _, doc := range section.Documents {
			rel := cleanJoin(section.Path, doc.Path)
			if rel == "" {
				continue
			}
			out = append(out, filepath.ToSlash(filepath.Join("docs", teamID, rel)))
		}
	}
	sort.Strings(out)
	return out
}
