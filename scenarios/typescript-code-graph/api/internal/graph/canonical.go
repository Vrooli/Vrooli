package graph

import (
	"bytes"
	"encoding/json"
	"sort"
)

// CanonicalJSON marshals a domain Graph to the canonical wire-shape JSON used
// by the committed expected-graph.json fixtures (captured from the sidecar's
// `jq -S .graph` output). Output is pretty-printed with a two-space indent and
// a trailing newline so it is byte-identical to the fixtures.
//
// This is the production counterpart of the integration test's
// canonicalGraphJSON helper; ValidateFixture uses it so the UI can run a
// determinism check without re-deriving the comparison shape. Keep the two in
// lockstep — if one changes, regenerate fixtures and update the other.
func CanonicalJSON(g Graph) ([]byte, error) {
	wg := wireGraph{
		Nodes: make([]wireNode, 0, len(g.Nodes)),
		Edges: make([]wireEdge, 0, len(g.Edges)),
	}
	for _, n := range g.Nodes {
		lc := n.LeadingComments
		if lc == nil {
			lc = []string{}
		}
		wg.Nodes = append(wg.Nodes, wireNode{
			ID:              n.ID,
			Kind:            nodeKindToWire(n.Kind),
			Name:            n.Name,
			Path:            n.Path,
			Attributes:      canonicalAttrs(n.Attributes),
			LeadingComments: lc,
		})
	}
	for _, e := range g.Edges {
		wg.Edges = append(wg.Edges, wireEdge{
			ID:         e.ID,
			Kind:       edgeKindToWire(e.Kind),
			FromNodeID: e.From,
			ToNodeID:   e.To,
			Attributes: canonicalAttrs(e.Attributes),
		})
	}
	// Mirror what the sidecar emits (already sorted by Normalize, but be
	// explicit so the comparison shape is self-contained).
	sort.Slice(wg.Nodes, func(i, j int) bool { return wg.Nodes[i].ID < wg.Nodes[j].ID })
	sort.Slice(wg.Edges, func(i, j int) bool {
		if wg.Edges[i].FromNodeID != wg.Edges[j].FromNodeID {
			return wg.Edges[i].FromNodeID < wg.Edges[j].FromNodeID
		}
		return wg.Edges[i].ToNodeID < wg.Edges[j].ToNodeID
	})

	// encoding/json sorts map keys alphabetically (matching `jq -S`). Disable
	// HTMLEscape so `>` survives as `>` rather than `>` — jq emits the
	// raw character.
	var raw bytes.Buffer
	enc := json.NewEncoder(&raw)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(wg); err != nil {
		return nil, err
	}
	// Drop the trailing newline json.Encoder appends so json.Indent produces a
	// clean buffer, then re-indent and re-append a single newline.
	raw0 := bytes.TrimRight(raw.Bytes(), "\n")
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw0, "", "  "); err != nil {
		return nil, err
	}
	pretty.WriteByte('\n')
	return pretty.Bytes(), nil
}

// wireNode/wireEdge/wireGraph mirror the canonical shape the sidecar emits and
// the fixtures capture. Field order follows `jq -S` (alphabetic).
type wireNode struct {
	Attributes      map[string]string `json:"attributes,omitempty"`
	ID              string            `json:"id"`
	Kind            int32             `json:"kind"`
	LeadingComments []string          `json:"leading_comments"`
	Name            string            `json:"name"`
	Path            string            `json:"path"`
}

type wireEdge struct {
	Attributes map[string]string `json:"attributes,omitempty"`
	FromNodeID string            `json:"from_node_id"`
	ID         string            `json:"id"`
	Kind       int32             `json:"kind"`
	ToNodeID   string            `json:"to_node_id"`
}

type wireGraph struct {
	Edges []wireEdge `json:"edges"`
	Nodes []wireNode `json:"nodes"`
}

// canonicalAttrs returns a copy of in, or nil if empty (encoding/json sorts
// map keys when marshaling, so we only need to nil-out empties for omitempty).
func canonicalAttrs(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// nodeKindToWire reverses graph.NodeKind to the int32 wire enum values, mirroring
// normalize.decodeNodeKind / the common.v1 + TsNodeKind extension ranges.
func nodeKindToWire(k NodeKind) int32 {
	switch k {
	case NodeKindFile:
		return 1
	case NodeKindModule:
		return 200
	case NodeKindComponent:
		return 201
	case NodeKindHook:
		return 202
	case NodeKindClass:
		return 203
	case NodeKindInterface:
		return 204
	case NodeKindType:
		return 205
	case NodeKindFunction:
		return 206
	case NodeKindVar:
		return 207
	case NodeKindConst:
		return 208
	case NodeKindReExport:
		return 209
	default:
		return 0
	}
}

func edgeKindToWire(k EdgeKind) int32 {
	switch k {
	case EdgeKindReExport:
		return 3
	case EdgeKindImport:
		return 1
	default:
		return 0
	}
}
