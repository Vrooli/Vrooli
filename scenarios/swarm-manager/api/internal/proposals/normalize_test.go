package proposals

import (
	"testing"
)

func TestNormalize_MutationListAssignsMissingIDs(t *testing.T) {
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{Op: OpArchiveItem, Target: "execute/foo"},
			{ID: "keep", Op: OpArchiveItem, Target: "execute/bar"},
			{Op: OpArchiveItem, Target: "execute/baz"},
		},
	}
	out, err := Normalize(p, CurrentState{InitiativeName: "i"})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if out.Mutations[0].ID != "m1" {
		t.Fatalf("expected m1, got %s", out.Mutations[0].ID)
	}
	if out.Mutations[1].ID != "keep" {
		t.Fatalf("expected preserved id, got %s", out.Mutations[1].ID)
	}
	if out.Mutations[2].ID != "m3" {
		t.Fatalf("expected m3, got %s", out.Mutations[2].ID)
	}
}

func TestNormalize_MutationListTrimsWhitespace(t *testing.T) {
	p := Proposal{
		Form: FormMutationList,
		Mutations: []Mutation{
			{Op: OpChangeStatus, Target: "  execute/foo  ", Status: "  READY  "},
			{Op: OpMoveInitiative, Target: " execute/bar ", Initiative: "  dest  "},
		},
	}
	out, _ := Normalize(p, CurrentState{InitiativeName: "i"})
	if out.Mutations[0].Target != "execute/foo" || out.Mutations[0].Status != "ready" {
		t.Fatalf("normalize did not trim/lowercase: %+v", out.Mutations[0])
	}
	if out.Mutations[1].Initiative != "dest" {
		t.Fatalf("normalize did not trim initiative: %+v", out.Mutations[1])
	}
}

func TestNormalize_FullGraphDiff_AddArchiveUpdate(t *testing.T) {
	current := CurrentState{
		InitiativeName: "i",
		Nodes: map[string]GraphNode{
			"execute/keep":   {ID: "execute/keep", Kind: "execute", Name: "keep", Title: "Keep", Priority: 5, Effort: "S"},
			"execute/update": {ID: "execute/update", Kind: "execute", Name: "update", Title: "Old Title", Priority: 5},
			"execute/gone":   {ID: "execute/gone", Kind: "execute", Name: "gone", Title: "Gone"},
		},
		Edges: []GraphEdge{
			{From: "execute/update", To: "execute/keep"},
		},
	}
	target := &Graph{
		Nodes: []GraphNode{
			{ID: "execute/keep", Kind: "execute", Name: "keep", Title: "Keep", Priority: 5, Effort: "S"},
			{ID: "execute/update", Kind: "execute", Name: "update", Title: "New Title", Priority: 7},
			{ID: "execute/new", Kind: "execute", Name: "new", Title: "New Item", Priority: 3},
		},
		Edges: []GraphEdge{
			{From: "execute/update", To: "execute/keep"},
			{From: "execute/new", To: "execute/keep"},
		},
	}
	p := Proposal{Form: FormFullGraph, Graph: target}
	out, err := Normalize(p, current)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if out.Form != FormMutationList {
		t.Fatalf("expected mutation_list, got %s", out.Form)
	}

	ops := make(map[Op]int, len(out.Mutations))
	for _, m := range out.Mutations {
		ops[m.Op]++
	}
	if ops[OpAddItem] != 1 {
		t.Fatalf("expected 1 add_item, got %d", ops[OpAddItem])
	}
	if ops[OpArchiveItem] != 1 {
		t.Fatalf("expected 1 archive_item, got %d", ops[OpArchiveItem])
	}
	if ops[OpUpdateItem] != 1 {
		t.Fatalf("expected 1 update_item, got %d", ops[OpUpdateItem])
	}
	if ops[OpChangePriority] != 1 {
		t.Fatalf("expected 1 change_priority, got %d", ops[OpChangePriority])
	}
	if ops[OpAddEdge] != 1 {
		t.Fatalf("expected 1 add_edge, got %d", ops[OpAddEdge])
	}
}

func TestNormalize_FullGraphIsDeterministic(t *testing.T) {
	current := CurrentState{
		InitiativeName: "i",
		Nodes: map[string]GraphNode{
			"execute/a": {ID: "execute/a", Kind: "execute", Name: "a", Title: "A"},
		},
	}
	target := &Graph{
		Nodes: []GraphNode{
			{ID: "execute/x", Kind: "execute", Name: "x", Title: "X"},
			{ID: "execute/y", Kind: "execute", Name: "y", Title: "Y"},
			{ID: "execute/a", Kind: "execute", Name: "a", Title: "A"},
		},
	}
	p := Proposal{Form: FormFullGraph, Graph: target}
	out1, err := Normalize(p, current)
	if err != nil {
		t.Fatal(err)
	}
	out2, err := Normalize(p, current)
	if err != nil {
		t.Fatal(err)
	}
	if len(out1.Mutations) != len(out2.Mutations) {
		t.Fatalf("mutation counts differ")
	}
	for i := range out1.Mutations {
		if out1.Mutations[i].ID != out2.Mutations[i].ID || out1.Mutations[i].Op != out2.Mutations[i].Op {
			t.Fatalf("mutation[%d] differs: %+v vs %+v", i, out1.Mutations[i], out2.Mutations[i])
		}
	}
	// First add should be the lexicographically earliest ref (execute/x < execute/y).
	if out1.Mutations[0].Op != OpAddItem || out1.Mutations[0].Item.Ref() != "execute/x" {
		t.Fatalf("expected first mutation to add execute/x, got %+v", out1.Mutations[0])
	}
}

func TestNormalize_FullGraphNoDiff_EmitsNoMutations(t *testing.T) {
	current := CurrentState{
		InitiativeName: "i",
		Nodes: map[string]GraphNode{
			"execute/a": {ID: "execute/a", Kind: "execute", Name: "a", Title: "A", Priority: 5},
		},
		Edges: []GraphEdge{{From: "execute/a", To: "execute/a"}},
	}
	target := &Graph{
		Nodes: []GraphNode{
			{ID: "execute/a", Kind: "execute", Name: "a", Title: "A", Priority: 5},
		},
		Edges: []GraphEdge{{From: "execute/a", To: "execute/a"}},
	}
	p := Proposal{Form: FormFullGraph, Graph: target}
	out, err := Normalize(p, current)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Mutations) != 0 {
		t.Fatalf("expected no mutations, got %d: %+v", len(out.Mutations), out.Mutations)
	}
}

func TestNormalize_FullGraphBackfillsKindNameFromID(t *testing.T) {
	target := &Graph{
		Nodes: []GraphNode{{ID: "execute/alpha", Title: "Alpha"}},
	}
	p := Proposal{Form: FormFullGraph, Graph: target}
	out, err := Normalize(p, CurrentState{InitiativeName: "i"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Mutations) != 1 {
		t.Fatalf("expected 1 mutation, got %d", len(out.Mutations))
	}
	if out.Mutations[0].Item.Kind != "execute" || out.Mutations[0].Item.Name != "alpha" {
		t.Fatalf("kind/name not backfilled from ID: %+v", out.Mutations[0].Item)
	}
}

func TestNormalize_FullGraphDedupesEdges(t *testing.T) {
	current := CurrentState{
		InitiativeName: "i",
		Nodes: map[string]GraphNode{
			"execute/a": {ID: "execute/a", Kind: "execute", Name: "a", Title: "A"},
			"execute/b": {ID: "execute/b", Kind: "execute", Name: "b", Title: "B"},
		},
	}
	target := &Graph{
		Nodes: []GraphNode{
			{ID: "execute/a", Kind: "execute", Name: "a", Title: "A"},
			{ID: "execute/b", Kind: "execute", Name: "b", Title: "B"},
		},
		Edges: []GraphEdge{
			{From: "execute/a", To: "execute/b"},
			{From: "execute/a", To: "execute/b"},
		},
	}
	p := Proposal{Form: FormFullGraph, Graph: target}
	out, err := Normalize(p, current)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, m := range out.Mutations {
		if m.Op == OpAddEdge {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 add_edge after dedupe, got %d", count)
	}
}
