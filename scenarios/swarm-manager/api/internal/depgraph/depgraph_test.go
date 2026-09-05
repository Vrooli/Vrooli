package depgraph

import "testing"

type testNode struct {
	key    string
	deps   []string
	status string
}

func (n testNode) Key() string    { return n.key }
func (n testNode) Deps() []string { return n.deps }
func (n testNode) Status() string { return n.status }

func TestComputeBlocking_NoDeps(t *testing.T) {
	nodes := []Node{testNode{key: "a", status: "ready"}}
	got := ComputeBlocking(nodes, map[string]bool{"backlog": true}, true)
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestComputeBlocking_BlockedByBacklogDep(t *testing.T) {
	nodes := []Node{
		testNode{key: "a", status: "backlog"},
		testNode{key: "b", deps: []string{"a"}, status: "ready"},
	}
	got := ComputeBlocking(nodes, map[string]bool{"backlog": true}, true)
	info, ok := got["b"]
	if !ok {
		t.Fatal("expected b to be blocked")
	}
	if !info.Blocked || !info.AllForceable {
		t.Errorf("unexpected info: %+v", info)
	}
	if len(info.BlockingKeys) != 1 || info.BlockingKeys[0] != "a" {
		t.Errorf("expected blocking keys [a], got %v", info.BlockingKeys)
	}
}

func TestComputeBlocking_MissingDepIsFailOpen(t *testing.T) {
	nodes := []Node{
		testNode{key: "b", deps: []string{"ghost"}, status: "ready"},
	}
	got := ComputeBlocking(nodes, map[string]bool{"backlog": true}, true)
	if len(got) != 0 {
		t.Errorf("missing dep should be treated as completed, got %v", got)
	}
}

func TestComputeBlocking_ForceableFlag(t *testing.T) {
	nodes := []Node{
		testNode{key: "a", status: "backlog"},
		testNode{key: "b", deps: []string{"a"}, status: "ready"},
	}
	got := ComputeBlocking(nodes, map[string]bool{"backlog": true}, false)
	if info := got["b"]; info.AllForceable {
		t.Error("expected AllForceable=false")
	}
}

func TestDetectCycleFrom_None(t *testing.T) {
	g := map[string][]string{"a": {"b"}, "b": {"c"}, "c": nil}
	if p := DetectCycleFrom(g, "a"); p != "" {
		t.Errorf("expected no cycle, got %q", p)
	}
}

func TestDetectCycleFrom_Direct(t *testing.T) {
	g := map[string][]string{"a": {"b"}, "b": {"a"}}
	p := DetectCycleFrom(g, "a")
	if p == "" {
		t.Fatal("expected cycle")
	}
}

func TestDetectCycleFrom_Transitive(t *testing.T) {
	g := map[string][]string{"a": {"b"}, "b": {"c"}, "c": {"a"}}
	if p := DetectCycleFrom(g, "a"); p == "" {
		t.Fatal("expected cycle")
	}
}

func TestBuildGraph(t *testing.T) {
	nodes := []Node{
		testNode{key: "a", deps: []string{"b"}},
		testNode{key: "b"},
	}
	g := BuildGraph(nodes)
	if len(g["a"]) != 1 || g["a"][0] != "b" {
		t.Errorf("unexpected graph: %v", g)
	}
	// Mutating the returned slice must not affect the source node.
	g["a"][0] = "c"
	if nodes[0].Deps()[0] != "b" {
		t.Error("BuildGraph must deep-copy deps")
	}
}
