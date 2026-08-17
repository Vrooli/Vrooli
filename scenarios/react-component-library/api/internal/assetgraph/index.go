package assetgraph

import (
	"fmt"
	"sort"

	"react-component-library/internal/assetrung"
	"react-component-library/internal/catalogcoverage"
)

// Node is the immutable catalog projection used by graph consumers.
type Node struct {
	ID          string
	Name        string
	Kind        string
	Rung        assetrung.Rung
	RungName    string
	Domain      string
	DomainOrder int
}

type Index struct {
	nodes   map[string]Node
	forward map[string][]string
	reverse map[string][]string
}

type Band struct {
	Rung   assetrung.Rung
	Name   string
	Assets []Node
	Count  int
}

type UnknownAssetError struct{ ID string }

func (e UnknownAssetError) Error() string { return fmt.Sprintf("unknown catalog asset %q", e.ID) }

type CycleError struct{ Path []string }

func (e CycleError) Error() string {
	return fmt.Sprintf("catalog dependency cycle: %s", joinPath(e.Path))
}

func Build(assets []catalogcoverage.Asset) (*Index, error) {
	i := &Index{nodes: map[string]Node{}, forward: map[string][]string{}, reverse: map[string][]string{}}
	for _, asset := range assets {
		rung, err := assetrung.Of(asset.Kind)
		if err != nil {
			return nil, err
		}
		i.nodes[asset.ID] = Node{ID: asset.ID, Name: asset.Name, Kind: asset.Kind, Rung: rung, RungName: rung.Name(), Domain: asset.Domain, DomainOrder: asset.DomainOrder}
	}
	for _, asset := range assets {
		for _, dep := range asset.Requires {
			if _, ok := i.nodes[dep]; !ok {
				return nil, UnknownAssetError{ID: dep}
			}
			i.forward[asset.ID] = append(i.forward[asset.ID], dep)
			i.reverse[dep] = append(i.reverse[dep], asset.ID)
		}
	}
	for id := range i.nodes {
		sort.Strings(i.forward[id])
		sort.Strings(i.reverse[id])
	}
	return i, nil
}

func (i *Index) Closure(id string) ([]Node, error) {
	if _, ok := i.nodes[id]; !ok {
		return nil, UnknownAssetError{ID: id}
	}
	seen := map[string]bool{}
	var out []Node
	var walk func(string, []string) error
	walk = func(current string, path []string) error {
		if contains(path, current) {
			cycle := append(append([]string{}, path...), current)
			return CycleError{Path: cycle}
		}
		if seen[current] {
			return nil
		}
		seen[current] = true
		out = append(out, i.nodes[current])
		for _, dep := range i.forward[current] {
			if err := walk(dep, append(path, current)); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(id, nil); err != nil {
		return nil, err
	}
	sortNodes(out)
	return out, nil
}

func (i *Index) Dependents(id string) (direct []Node, transitive []Node, err error) {
	if _, ok := i.nodes[id]; !ok {
		return nil, nil, UnknownAssetError{ID: id}
	}
	directIDs := append([]string{}, i.reverse[id]...)
	seen := map[string]bool{id: true}
	queue := append([]string{}, directIDs...)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if seen[current] {
			continue
		}
		seen[current] = true
		queue = append(queue, i.reverse[current]...)
	}
	for _, nodeID := range directIDs {
		direct = append(direct, i.nodes[nodeID])
	}
	for nodeID := range seen {
		if nodeID != id {
			transitive = append(transitive, i.nodes[nodeID])
		}
	}
	sortNodes(direct)
	sortNodes(transitive)
	return direct, transitive, nil
}

func (i *Index) Node(id string) (Node, error) {
	node, ok := i.nodes[id]
	if !ok {
		return Node{}, UnknownAssetError{ID: id}
	}
	return node, nil
}

func (i *Index) Dependencies(id string) ([]Node, error) {
	if _, ok := i.nodes[id]; !ok {
		return nil, UnknownAssetError{ID: id}
	}
	out := make([]Node, 0, len(i.forward[id]))
	for _, dependency := range i.forward[id] {
		out = append(out, i.nodes[dependency])
	}
	sortNodes(out)
	return out, nil
}

func (i *Index) Nodes() []Node {
	out := make([]Node, 0, len(i.nodes))
	for _, node := range i.nodes {
		out = append(out, node)
	}
	sortNodes(out)
	return out
}

func Bands(nodes []Node) []Band {
	grouped := map[assetrung.Rung][]Node{}
	for _, node := range nodes {
		grouped[node.Rung] = append(grouped[node.Rung], node)
	}
	keys := make([]assetrung.Rung, 0, len(grouped))
	for rung := range grouped {
		keys = append(keys, rung)
	}
	sort.Slice(keys, func(a, b int) bool { return keys[a] > keys[b] })
	out := make([]Band, 0, len(keys))
	for _, rung := range keys {
		assets := grouped[rung]
		sortNodes(assets)
		out = append(out, Band{Rung: rung, Name: rung.Name(), Assets: assets, Count: len(assets)})
	}
	return out
}

func sortNodes(nodes []Node) {
	sort.Slice(nodes, func(a, b int) bool {
		if nodes[a].Rung != nodes[b].Rung {
			return nodes[a].Rung > nodes[b].Rung
		}
		return nodes[a].ID < nodes[b].ID
	})
}

func contains(path []string, id string) bool {
	for _, value := range path {
		if value == id {
			return true
		}
	}
	return false
}

func joinPath(path []string) string {
	if len(path) == 0 {
		return ""
	}
	result := path[0]
	for _, value := range path[1:] {
		result += " -> " + value
	}
	return result
}
