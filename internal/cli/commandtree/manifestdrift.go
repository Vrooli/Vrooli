package commandtree

import (
	"sort"
	"strings"
)

// CommandNode is a command-tree registration node. A node with children is a
// command group; a node without children is an invocable leaf command.
type CommandNode struct {
	Name     string
	Leaf     bool
	Children []CommandNode
}

// WalkCommandTree returns the normalized leaf paths in a registered command
// tree. It intentionally reports leaves only: a group such as "workload" is
// not itself an invocable command when its registered child is "list".
func WalkCommandTree(roots []CommandNode) []string {
	paths := make([]string, 0, len(roots))
	var walk func([]CommandNode, []string)
	walk = func(nodes []CommandNode, parents []string) {
		for _, node := range nodes {
			name := NormalizeName(node.Name)
			if name == "" {
				continue
			}
			path := append(append([]string(nil), parents...), name)
			if node.Leaf || len(node.Children) == 0 {
				paths = append(paths, strings.Join(path, " "))
			}
			if len(node.Children) > 0 {
				walk(node.Children, path)
			}
		}
	}
	walk(roots, nil)
	sort.Strings(paths)
	return uniqueStrings(paths)
}

// CommandTreeFromPaths turns already-registered command paths into a tree.
// It is useful for registrations whose handlers own their child parser rather
// than exposing a Spec table, while keeping the comparison code independent of
// that implementation detail.
func CommandTreeFromPaths(paths []string) []CommandNode {
	root := make([]CommandNode, 0, len(paths))
	for _, raw := range paths {
		parts := strings.Fields(NormalizeName(raw))
		if len(parts) == 0 {
			continue
		}
		root = insertCommandPath(root, parts)
	}
	return root
}

func insertCommandPath(nodes []CommandNode, parts []string) []CommandNode {
	for i := range nodes {
		if NormalizeName(nodes[i].Name) != parts[0] {
			continue
		}
		if len(parts) == 1 {
			nodes[i].Leaf = true
		}
		if len(parts) > 1 {
			nodes[i].Children = insertCommandPath(nodes[i].Children, parts[1:])
		}
		return nodes
	}
	node := CommandNode{Name: parts[0], Leaf: len(parts) == 1}
	if len(parts) > 1 {
		node.Children = insertCommandPath(nil, parts[1:])
	}
	return append(nodes, node)
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}
