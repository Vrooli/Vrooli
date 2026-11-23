package coreset

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ComponentID identifies a member of a dependency graph. Kind is extensible
// so callers can represent scenarios, resources, tools, safeguards, packages
// and other governed components without teaching the shared resolver their
// domain-specific policy.
type ComponentID struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// DependencyEdge records one declared dependency. Relation and DeclaredBy
// retain the owning vocabulary for projection, while Required and Policy
// preserve the decision made by the manifest or operator.
type DependencyEdge struct {
	From       ComponentID `json:"from"`
	To         ComponentID `json:"to"`
	Relation   string      `json:"relation,omitempty"`
	DeclaredBy string      `json:"declared_by,omitempty"`
	Detail     string      `json:"detail,omitempty"`
	Required   bool        `json:"required"`
	Policy     string      `json:"policy,omitempty"`
}

// DependencyGraph is a catalog snapshot. Components can be supplied in any
// order; Resolve canonicalizes them before traversal and digesting.
type DependencyGraph struct {
	Components []ComponentID
	Edges      []DependencyEdge
}

// ClosureReason explains the edge by which a selected root reached a member.
type ClosureReason struct {
	From       ComponentID `json:"from"`
	To         ComponentID `json:"to"`
	Relation   string      `json:"relation,omitempty"`
	DeclaredBy string      `json:"declared_by,omitempty"`
	Detail     string      `json:"detail,omitempty"`
	Required   bool        `json:"required"`
	Policy     string      `json:"policy,omitempty"`
}

// ClosureMember is one deduplicated member of the selected union.
type ClosureMember struct {
	Component ComponentID     `json:"component"`
	Selected  bool            `json:"selected"`
	Root      ComponentID     `json:"root"`
	Path      []ClosureReason `json:"path,omitempty"`
	Reasons   []ClosureReason `json:"reasons,omitempty"`
}

// DependencyClosure is a deterministic union of selected roots and their
// reachable dependencies. Digest is the SHA-256 of the canonical versioned
// closure content, excluding the digest field itself.
type DependencyClosure struct {
	SchemaVersion int             `json:"schema_version"`
	Members       []ClosureMember `json:"members"`
	Digest        string          `json:"digest"`
}

// ClosureSchemaVersion is the only canonical dependency-closure wire format
// currently implemented by Resolve and VerifyDigest.
const ClosureSchemaVersion = 1

// ResolveOptions selects compatibility behavior needed by existing graph
// owners. When enabled, cycles containing an optional edge remain represented
// in the closure; cycles made entirely of required edges still fail.
type ResolveOptions struct {
	AllowCyclesWithOptionalEdges bool
	// AllowCycles preserves cycles whose owner treats graph reachability as a
	// union rather than requiring a runnable topological order.
	AllowCycles bool
}

// DependencySource loads the outgoing declarations for one component. The
// shared package owns reachability traversal; adapters own only how a node's
// declarations are read and which domain policies remove an edge.
type DependencySource interface {
	Dependencies(ComponentID) ([]DependencyEdge, error)
}

// DependencySourceFunc adapts a function to DependencySource.
type DependencySourceFunc func(ComponentID) ([]DependencyEdge, error)

func (fn DependencySourceFunc) Dependencies(id ComponentID) ([]DependencyEdge, error) {
	return fn(id)
}

// ResolveFromSource loads only the reachable graph, then applies the same
// deterministic closure and cycle rules as ResolveWithOptions. Every edge
// target is retained even when its source has no outgoing dependencies.
func ResolveFromSource(source DependencySource, roots []ComponentID, options ResolveOptions) (DependencyClosure, error) {
	if source == nil {
		return DependencyClosure{}, fmt.Errorf("%w: dependency source is required", ErrInvalidGraph)
	}
	graph := DependencyGraph{}
	state := map[ComponentID]uint8{} // 0 unseen, 1 loading, 2 loaded
	var load func(ComponentID) error
	load = func(raw ComponentID) error {
		id, err := normalizeID(raw)
		if err != nil {
			return err
		}
		if state[id] != 0 {
			return nil
		}
		state[id] = 1
		graph.Components = append(graph.Components, id)
		edges, err := source.Dependencies(id)
		if err != nil {
			return err
		}
		for _, edge := range edges {
			edge.From = id
			target, err := normalizeID(edge.To)
			if err != nil {
				return err
			}
			edge.To = target
			graph.Components = append(graph.Components, target)
			graph.Edges = append(graph.Edges, edge)
			if state[target] == 0 {
				if err := load(target); err != nil {
					return err
				}
			}
		}
		state[id] = 2
		return nil
	}
	for _, root := range roots {
		if err := load(root); err != nil {
			return DependencyClosure{}, err
		}
	}
	return ResolveWithOptions(graph, roots, options)
}

var (
	ErrInvalidGraph = errors.New("invalid dependency graph")
	ErrMissingNode  = errors.New("dependency graph references a missing component")
	ErrCycle        = errors.New("dependency graph contains a cycle")
)

// VerifyDigest confirms that a transported closure still matches the canonical
// BR1 representation. Consumers must call this before using closure members as
// an authority-bearing shipment input.
func VerifyDigest(closure DependencyClosure) error {
	if closure.SchemaVersion != ClosureSchemaVersion {
		return fmt.Errorf("%w: unsupported closure schema version %d", ErrInvalidGraph, closure.SchemaVersion)
	}
	if len(closure.Members) == 0 {
		return fmt.Errorf("%w: closure members are required", ErrInvalidGraph)
	}
	want, err := closureDigest(closure.SchemaVersion, closure.Members)
	if err != nil {
		return err
	}
	if closure.Digest != want {
		return fmt.Errorf("%w: closure digest mismatch", ErrInvalidGraph)
	}
	return nil
}

// Resolve computes the sorted transitive union for roots. Duplicate roots,
// components and edges normalize away; missing roots/targets and cycles fail
// closed with stable error paths.
func Resolve(graph DependencyGraph, roots []ComponentID) (DependencyClosure, error) {
	return ResolveWithOptions(graph, roots, ResolveOptions{})
}

// ResolveWithOptions computes a dependency closure with the requested cycle
// policy. Most callers should use Resolve, which rejects every cycle.
func ResolveWithOptions(graph DependencyGraph, roots []ComponentID, options ResolveOptions) (DependencyClosure, error) {
	nodes := make(map[ComponentID]struct{}, len(graph.Components))
	for _, raw := range graph.Components {
		id, err := normalizeID(raw)
		if err != nil {
			return DependencyClosure{}, err
		}
		if _, exists := nodes[id]; exists {
			continue
		}
		nodes[id] = struct{}{}
	}
	adjacency := make(map[ComponentID][]DependencyEdge, len(nodes))
	seenEdges := make(map[string]struct{}, len(graph.Edges))
	for _, raw := range graph.Edges {
		from, err := normalizeID(raw.From)
		if err != nil {
			return DependencyClosure{}, err
		}
		to, err := normalizeID(raw.To)
		if err != nil {
			return DependencyClosure{}, err
		}
		if _, ok := nodes[from]; !ok {
			return DependencyClosure{}, fmt.Errorf("%w: edge source %s", ErrMissingNode, formatID(from))
		}
		if _, ok := nodes[to]; !ok {
			return DependencyClosure{}, fmt.Errorf("%w: edge %s -> %s", ErrMissingNode, formatID(from), formatID(to))
		}
		edge := raw
		edge.From, edge.To = from, to
		edge.Relation = strings.TrimSpace(edge.Relation)
		edge.DeclaredBy = strings.TrimSpace(edge.DeclaredBy)
		edge.Detail = strings.TrimSpace(edge.Detail)
		edge.Policy = strings.TrimSpace(edge.Policy)
		key, _ := json.Marshal(edge)
		if _, exists := seenEdges[string(key)]; exists {
			continue
		}
		seenEdges[string(key)] = struct{}{}
		adjacency[from] = append(adjacency[from], edge)
	}
	for from := range adjacency {
		sort.Slice(adjacency[from], func(i, j int) bool { return edgeKey(adjacency[from][i]) < edgeKey(adjacency[from][j]) })
	}

	normalizedRoots := make(map[ComponentID]struct{}, len(roots))
	for _, raw := range roots {
		id, err := normalizeID(raw)
		if err != nil {
			return DependencyClosure{}, err
		}
		if _, ok := nodes[id]; !ok {
			return DependencyClosure{}, fmt.Errorf("%w: selected root %s", ErrMissingNode, formatID(id))
		}
		normalizedRoots[id] = struct{}{}
	}
	rootList := make([]ComponentID, 0, len(normalizedRoots))
	for id := range normalizedRoots {
		rootList = append(rootList, id)
	}
	sort.Slice(rootList, func(i, j int) bool { return componentKey(rootList[i]) < componentKey(rootList[j]) })

	selected := make(map[ComponentID]bool, len(rootList))
	visited := make(map[ComponentID]bool, len(nodes))
	onStack := make(map[ComponentID]bool, len(nodes))
	stack := make([]ComponentID, 0, len(nodes))
	stackRequired := make([]bool, 0, len(nodes))
	reasons := make(map[ComponentID]map[string]ClosureReason)
	var visit func(ComponentID, bool) error
	visit = func(id ComponentID, viaRequired bool) error {
		if onStack[id] {
			start := 0
			for i, item := range stack {
				if item == id {
					start = i
					break
				}
			}
			cycle := append(append([]ComponentID(nil), stack[start:]...), id)
			parts := make([]string, len(cycle))
			for i, item := range cycle {
				parts[i] = formatID(item)
			}
			return fmt.Errorf("%w: %s", ErrCycle, strings.Join(parts, " -> "))
		}
		if visited[id] {
			return nil
		}
		visited[id], onStack[id] = true, true
		stack = append(stack, id)
		stackRequired = append(stackRequired, viaRequired)
		for _, edge := range adjacency[id] {
			reason := ClosureReason{From: edge.From, To: edge.To, Relation: edge.Relation, DeclaredBy: edge.DeclaredBy, Detail: edge.Detail, Required: edge.Required, Policy: edge.Policy}
			if reasons[edge.To] == nil {
				reasons[edge.To] = map[string]ClosureReason{}
			}
			serialized, _ := json.Marshal(reason)
			reasons[edge.To][string(serialized)] = reason
			if onStack[edge.To] {
				if !options.AllowCycles && (!options.AllowCyclesWithOptionalEdges || closesRequiredCycle(stack, stackRequired, edge.To, edge.Required)) {
					start := 0
					for i, item := range stack {
						if item == edge.To {
							start = i
							break
						}
					}
					cycle := append(append([]ComponentID(nil), stack[start:]...), edge.To)
					parts := make([]string, len(cycle))
					for i, item := range cycle {
						parts[i] = formatID(item)
					}
					return fmt.Errorf("%w: %s", ErrCycle, strings.Join(parts, " -> "))
				}
				continue
			}
			if err := visit(edge.To, edge.Required); err != nil {
				return err
			}
		}
		stack = stack[:len(stack)-1]
		stackRequired = stackRequired[:len(stackRequired)-1]
		onStack[id] = false
		return nil
	}
	for _, root := range rootList {
		selected[root] = true
		if err := visit(root, false); err != nil {
			return DependencyClosure{}, err
		}
	}

	// Choose one canonical shortest provenance path per member. All incoming
	// reasons remain on the member; this path is the stable attribution chain
	// for consumers whose presentation needs one explanation.
	type pathState struct {
		id   ComponentID
		root ComponentID
		path []ClosureReason
	}
	canonicalPath := make(map[ComponentID]pathState, len(visited))
	queue := make([]pathState, 0, len(rootList))
	for _, root := range rootList {
		state := pathState{id: root, root: root}
		canonicalPath[root] = state
		queue = append(queue, state)
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, edge := range adjacency[current.id] {
			if !visited[edge.To] {
				continue
			}
			if _, exists := canonicalPath[edge.To]; exists {
				continue
			}
			path := append(append([]ClosureReason(nil), current.path...), ClosureReason{
				From: edge.From, To: edge.To, Relation: edge.Relation, DeclaredBy: edge.DeclaredBy, Detail: edge.Detail,
				Required: edge.Required, Policy: edge.Policy,
			})
			state := pathState{id: edge.To, root: current.root, path: path}
			canonicalPath[edge.To] = state
			queue = append(queue, state)
		}
	}

	closure := DependencyClosure{SchemaVersion: ClosureSchemaVersion, Members: make([]ClosureMember, 0, len(visited))}
	for id := range visited {
		path := canonicalPath[id]
		member := ClosureMember{Component: id, Selected: selected[id], Root: path.root, Path: path.path}
		for _, reason := range reasons[id] {
			member.Reasons = append(member.Reasons, reason)
		}
		sort.Slice(member.Reasons, func(i, j int) bool { return reasonKey(member.Reasons[i]) < reasonKey(member.Reasons[j]) })
		closure.Members = append(closure.Members, member)
	}
	sort.Slice(closure.Members, func(i, j int) bool {
		return componentKey(closure.Members[i].Component) < componentKey(closure.Members[j].Component)
	})
	digest, err := closureDigest(closure.SchemaVersion, closure.Members)
	if err != nil {
		return DependencyClosure{}, err
	}
	closure.Digest = digest
	return closure, nil
}

func closureDigest(schemaVersion int, members []ClosureMember) (string, error) {
	canonical, err := json.Marshal(struct {
		SchemaVersion int             `json:"schema_version"`
		Members       []ClosureMember `json:"members"`
	}{SchemaVersion: schemaVersion, Members: members})
	if err != nil {
		return "", fmt.Errorf("marshal canonical closure: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func closesRequiredCycle(stack []ComponentID, stackRequired []bool, target ComponentID, closingRequired bool) bool {
	if !closingRequired {
		return false
	}
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] == target {
			for j := i + 1; j < len(stackRequired); j++ {
				if !stackRequired[j] {
					return false
				}
			}
			return true
		}
	}
	return false
}

func normalizeID(id ComponentID) (ComponentID, error) {
	id.Kind = strings.ToLower(strings.TrimSpace(id.Kind))
	id.Name = strings.ToLower(strings.TrimSpace(id.Name))
	if id.Kind == "" || id.Name == "" {
		return ComponentID{}, fmt.Errorf("%w: component kind and name are required", ErrInvalidGraph)
	}
	return id, nil
}

func componentKey(id ComponentID) string { return id.Kind + ":" + id.Name }

func formatID(id ComponentID) string { return componentKey(id) }

func edgeKey(edge DependencyEdge) string {
	encoded, _ := json.Marshal(edge)
	return string(encoded)
}

func reasonKey(reason ClosureReason) string {
	encoded, _ := json.Marshal(reason)
	return string(encoded)
}
