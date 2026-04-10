package depgraph

import (
	"reflect"
	"testing"
)

func TestEmptyGraph(t *testing.T) {
	g := New()

	cycle, found := g.DetectCycle()
	if found {
		t.Errorf("empty graph should have no cycle, got %v", cycle)
	}

	order, err := g.TopologicalSort()
	if err != nil {
		t.Errorf("empty graph topological sort should succeed: %v", err)
	}
	if len(order) != 0 {
		t.Errorf("expected empty sort result, got %v", order)
	}

	unblocked := g.UnblockedItems(nil)
	if len(unblocked) != 0 {
		t.Errorf("expected no unblocked items, got %v", unblocked)
	}

	blocked := g.BlockedItems(nil)
	if len(blocked) != 0 {
		t.Errorf("expected no blocked items, got %v", blocked)
	}

	edges := g.Edges()
	if len(edges) != 0 {
		t.Errorf("expected no edges, got %v", edges)
	}
}

func TestSingleNodeNoDeps(t *testing.T) {
	g := New()
	g.AddNode("A", nil)

	cycle, found := g.DetectCycle()
	if found {
		t.Errorf("single node should have no cycle, got %v", cycle)
	}

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("topological sort failed: %v", err)
	}
	if !reflect.DeepEqual(order, []string{"A"}) {
		t.Errorf("expected [A], got %v", order)
	}

	unblocked := g.UnblockedItems(map[string]bool{})
	if !reflect.DeepEqual(unblocked, []string{"A"}) {
		t.Errorf("expected [A] unblocked, got %v", unblocked)
	}

	dependents := g.Dependents("A")
	if len(dependents) != 0 {
		t.Errorf("expected no dependents for A, got %v", dependents)
	}
}

func TestLinearChain(t *testing.T) {
	// A depends on B, B depends on C (C must complete first)
	g := New()
	g.AddNode("A", []string{"B"})
	g.AddNode("B", []string{"C"})
	g.AddNode("C", nil)

	cycle, found := g.DetectCycle()
	if found {
		t.Errorf("linear chain should have no cycle, got %v", cycle)
	}

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("topological sort failed: %v", err)
	}
	// C has no deps -> processed first, then B, then A
	if !reflect.DeepEqual(order, []string{"C", "B", "A"}) {
		t.Errorf("expected [C B A], got %v", order)
	}

	// Nothing completed: only C is unblocked
	unblocked := g.UnblockedItems(map[string]bool{})
	if !reflect.DeepEqual(unblocked, []string{"C"}) {
		t.Errorf("expected [C] unblocked, got %v", unblocked)
	}

	// After C completes: B is unblocked
	unblocked = g.UnblockedItems(map[string]bool{"C": true})
	if !reflect.DeepEqual(unblocked, []string{"B"}) {
		t.Errorf("expected [B] unblocked, got %v", unblocked)
	}

	// After B and C complete: A is unblocked
	unblocked = g.UnblockedItems(map[string]bool{"B": true, "C": true})
	if !reflect.DeepEqual(unblocked, []string{"A"}) {
		t.Errorf("expected [A] unblocked, got %v", unblocked)
	}

	// Dependents
	depOfC := g.Dependents("C")
	if !reflect.DeepEqual(depOfC, []string{"B"}) {
		t.Errorf("expected [B] as dependents of C, got %v", depOfC)
	}

	depOfB := g.Dependents("B")
	if !reflect.DeepEqual(depOfB, []string{"A"}) {
		t.Errorf("expected [A] as dependents of B, got %v", depOfB)
	}
}

func TestDiamondDeps(t *testing.T) {
	// A depends on B and C; B depends on D; C depends on D
	g := New()
	g.AddNode("A", []string{"B", "C"})
	g.AddNode("B", []string{"D"})
	g.AddNode("C", []string{"D"})
	g.AddNode("D", nil)

	cycle, found := g.DetectCycle()
	if found {
		t.Errorf("diamond should have no cycle, got %v", cycle)
	}

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("topological sort failed: %v", err)
	}
	// D first, then B and C (alphabetical), then A
	if !reflect.DeepEqual(order, []string{"D", "B", "C", "A"}) {
		t.Errorf("expected [D B C A], got %v", order)
	}

	// Only D is unblocked initially
	unblocked := g.UnblockedItems(map[string]bool{})
	if !reflect.DeepEqual(unblocked, []string{"D"}) {
		t.Errorf("expected [D] unblocked, got %v", unblocked)
	}

	// After D: B and C unblocked
	unblocked = g.UnblockedItems(map[string]bool{"D": true})
	if !reflect.DeepEqual(unblocked, []string{"B", "C"}) {
		t.Errorf("expected [B C] unblocked, got %v", unblocked)
	}

	// After B, C, D: A unblocked
	unblocked = g.UnblockedItems(map[string]bool{"B": true, "C": true, "D": true})
	if !reflect.DeepEqual(unblocked, []string{"A"}) {
		t.Errorf("expected [A] unblocked, got %v", unblocked)
	}

	// Edges
	edges := g.Edges()
	expected := [][2]string{{"A", "B"}, {"A", "C"}, {"B", "D"}, {"C", "D"}}
	if !reflect.DeepEqual(edges, expected) {
		t.Errorf("expected edges %v, got %v", expected, edges)
	}
}

func TestSelfReferentialCycle(t *testing.T) {
	g := New()
	g.AddNode("A", []string{"A"})

	cycle, found := g.DetectCycle()
	if !found {
		t.Fatal("expected self-referential cycle")
	}
	if len(cycle) == 0 {
		t.Error("expected non-empty cycle path")
	}

	_, err := g.TopologicalSort()
	if err == nil {
		t.Error("expected topological sort to fail with cycle")
	}
}

func TestMultiNodeCycle(t *testing.T) {
	// A -> B -> C -> A
	g := New()
	g.AddNode("A", []string{"B"})
	g.AddNode("B", []string{"C"})
	g.AddNode("C", []string{"A"})

	cycle, found := g.DetectCycle()
	if !found {
		t.Fatal("expected multi-node cycle")
	}
	if len(cycle) < 3 {
		t.Errorf("expected cycle of at least 3 nodes, got %v", cycle)
	}

	_, err := g.TopologicalSort()
	if err == nil {
		t.Error("expected topological sort to fail with cycle")
	}
}

func TestDisconnectedSubgraphs(t *testing.T) {
	g := New()
	g.AddNode("A", []string{"B"})
	g.AddNode("B", nil)
	g.AddNode("X", []string{"Y"})
	g.AddNode("Y", nil)

	cycle, found := g.DetectCycle()
	if found {
		t.Errorf("disconnected subgraphs should have no cycle, got %v", cycle)
	}

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("topological sort failed: %v", err)
	}
	// All 4 nodes should appear. B before A, Y before X.
	if len(order) != 4 {
		t.Fatalf("expected 4 nodes, got %v", order)
	}
	// Verify ordering constraints: B before A, Y before X.
	posOf := map[string]int{}
	for i, v := range order {
		posOf[v] = i
	}
	if posOf["B"] > posOf["A"] {
		t.Errorf("B should come before A in topological order, got %v", order)
	}
	if posOf["Y"] > posOf["X"] {
		t.Errorf("Y should come before X in topological order, got %v", order)
	}

	unblocked := g.UnblockedItems(map[string]bool{})
	if !reflect.DeepEqual(unblocked, []string{"B", "Y"}) {
		t.Errorf("expected [B Y] unblocked, got %v", unblocked)
	}
}

func TestBlockedItems(t *testing.T) {
	g := New()
	g.AddNode("A", []string{"B", "C"})
	g.AddNode("B", nil)
	g.AddNode("C", nil)

	blocked := g.BlockedItems(map[string]bool{})
	if !reflect.DeepEqual(blocked, []string{"A"}) {
		t.Errorf("expected [A] blocked, got %v", blocked)
	}

	// After B completes, A is still blocked (needs C)
	blocked = g.BlockedItems(map[string]bool{"B": true})
	if !reflect.DeepEqual(blocked, []string{"A"}) {
		t.Errorf("expected [A] still blocked, got %v", blocked)
	}

	// After both complete, nothing blocked
	blocked = g.BlockedItems(map[string]bool{"B": true, "C": true})
	if len(blocked) != 0 {
		t.Errorf("expected no blocked items, got %v", blocked)
	}
}

func TestExternalDependencies(t *testing.T) {
	// A depends on "external" which is not in the graph.
	g := New()
	g.AddNode("A", []string{"external"})

	cycle, found := g.DetectCycle()
	if found {
		t.Errorf("external deps should not cause cycles, got %v", cycle)
	}

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("topological sort failed: %v", err)
	}
	if !reflect.DeepEqual(order, []string{"A"}) {
		t.Errorf("expected [A], got %v", order)
	}

	// A is unblocked because external deps are not tracked.
	unblocked := g.UnblockedItems(map[string]bool{})
	if !reflect.DeepEqual(unblocked, []string{"A"}) {
		t.Errorf("expected [A] unblocked, got %v", unblocked)
	}
}

func TestUnblockedWithPartialCompletion(t *testing.T) {
	// A->B, A->C, D->C, E (independent)
	g := New()
	g.AddNode("A", []string{"B", "C"})
	g.AddNode("B", nil)
	g.AddNode("C", nil)
	g.AddNode("D", []string{"C"})
	g.AddNode("E", nil)

	// Initially: B, C, E are unblocked
	unblocked := g.UnblockedItems(map[string]bool{})
	if !reflect.DeepEqual(unblocked, []string{"B", "C", "E"}) {
		t.Errorf("expected [B C E] unblocked, got %v", unblocked)
	}

	// After C: B, D, E unblocked (A still blocked on B)
	unblocked = g.UnblockedItems(map[string]bool{"C": true})
	if !reflect.DeepEqual(unblocked, []string{"B", "D", "E"}) {
		t.Errorf("expected [B D E] unblocked, got %v", unblocked)
	}

	// After B and C: A, D, E unblocked
	unblocked = g.UnblockedItems(map[string]bool{"B": true, "C": true})
	if !reflect.DeepEqual(unblocked, []string{"A", "D", "E"}) {
		t.Errorf("expected [A D E] unblocked, got %v", unblocked)
	}
}

func TestAddNodeReplace(t *testing.T) {
	g := New()
	g.AddNode("A", []string{"B"})
	g.AddNode("A", []string{"C"}) // replace

	_ = g.Edges() // first call before C is added as a node
	// Should only have A->C, not A->B
	g.AddNode("C", nil) // add C so edge is visible
	edges := g.Edges()
	expected := [][2]string{{"A", "C"}}
	if !reflect.DeepEqual(edges, expected) {
		t.Errorf("expected edges %v after replace, got %v", expected, edges)
	}
}
