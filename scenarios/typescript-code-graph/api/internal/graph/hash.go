package graph

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// GraphHash returns the hex sha256 of a canonical JSON serialization
// of g. "Canonical" here means: map keys are sorted, every collection
// is already sorted (the caller is expected to have run Normalize),
// and the encoder uses no HTML escaping. Identical Graph values produce
// identical hashes byte-for-byte.
//
// Powers REQ-P1-005 (caller cache-hit detection) and the determinism
// test suite.
func GraphHash(g Graph) string {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "")
	_ = enc.Encode(canonicalGraph(g))

	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:])
}

// canonicalNode rebuilds Node with attribute maps materialized as
// sorted key/value pairs. JSON serialization of Go maps already sorts
// keys lexicographically, but we explicitly project to an ordered
// representation so the canonical form is obvious from reading the
// code. LeadingComments is carried verbatim — REQ-P0-003 demands that
// the bytes survive into the hash too, so a comment edit changes the
// graph hash.
type canonicalNode struct {
	ID              string     `json:"id"`
	Kind            NodeKind   `json:"kind"`
	Name            string     `json:"name"`
	Path            string     `json:"path"`
	Attributes      [][]string `json:"attributes,omitempty"`
	LeadingComments []string   `json:"leading_comments,omitempty"`
}

type canonicalEdge struct {
	ID         string     `json:"id"`
	Kind       EdgeKind   `json:"kind"`
	From       string     `json:"from"`
	To         string     `json:"to"`
	Attributes [][]string `json:"attributes,omitempty"`
}

type canonicalGraphShape struct {
	Nodes []canonicalNode `json:"nodes"`
	Edges []canonicalEdge `json:"edges"`
}

func canonicalGraph(g Graph) canonicalGraphShape {
	out := canonicalGraphShape{
		Nodes: make([]canonicalNode, 0, len(g.Nodes)),
		Edges: make([]canonicalEdge, 0, len(g.Edges)),
	}
	for _, n := range g.Nodes {
		out.Nodes = append(out.Nodes, canonicalNode{
			ID:              n.ID,
			Kind:            n.Kind,
			Name:            n.Name,
			Path:            n.Path,
			Attributes:      sortedPairs(n.Attributes),
			LeadingComments: n.LeadingComments,
		})
	}
	for _, e := range g.Edges {
		out.Edges = append(out.Edges, canonicalEdge{
			ID:         e.ID,
			Kind:       e.Kind,
			From:       e.From,
			To:         e.To,
			Attributes: sortedPairs(e.Attributes),
		})
	}
	return out
}

func sortedPairs(m map[string]string) [][]string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([][]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, []string{k, m[k]})
	}
	return out
}
