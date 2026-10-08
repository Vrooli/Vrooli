package commandtree

import (
	"reflect"
	"testing"
)

func TestWalkCommandTreeReturnsSortedLeafPaths(t *testing.T) {
	got := WalkCommandTree([]CommandNode{
		{Name: "runtime", Children: []CommandNode{{Name: "status", Leaf: true}, {Name: "run"}}},
		{Name: "setup", Leaf: true},
		{Name: "runtime", Children: []CommandNode{{Name: "status", Leaf: true}}},
	})
	want := []string{"runtime run", "runtime status", "setup"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("WalkCommandTree() = %#v, want %#v", got, want)
	}
}

func TestCommandTreeFromPathsBuildsNestedLeaves(t *testing.T) {
	paths := CommandTreeFromPaths([]string{"workload list", "host inventory", "setup"})
	got := WalkCommandTree(paths)
	want := []string{"host inventory", "setup", "workload list"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}
