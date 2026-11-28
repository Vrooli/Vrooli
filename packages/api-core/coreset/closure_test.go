package coreset

import (
	"errors"
	"reflect"
	"testing"
)

func id(kind, name string) ComponentID { return ComponentID{Kind: kind, Name: name} }

func TestResolveNestedUnionNormalizesAndDigestsDeterministically(t *testing.T) {
	graph := DependencyGraph{
		Components: []ComponentID{id("resource", "db"), id("scenario", "worker"), id("scenario", "app"), id("tool", "compiler")},
		Edges: []DependencyEdge{
			{From: id("scenario", "app"), To: id("scenario", "worker"), Required: true, Relation: "scenario"},
			{From: id("scenario", "worker"), To: id("resource", "db"), Required: true, Relation: "resource"},
			{From: id("scenario", "app"), To: id("tool", "compiler"), Required: false, Relation: "tool", Policy: "selected"},
			{From: id("scenario", "app"), To: id("scenario", "worker"), Required: true, Relation: "scenario"}, // duplicate
		},
	}
	roots := []ComponentID{id(" SCENARIO ", "APP"), id("scenario", "app")}
	got, err := Resolve(graph, roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Members) != 4 || got.Digest == "" {
		t.Fatalf("closure = %+v", got)
	}
	if got.Members[0].Component != id("resource", "db") || got.Members[1].Component != id("scenario", "app") {
		t.Fatalf("members are not sorted: %+v", got.Members)
	}
	if !got.Members[1].Selected || got.Members[1].Reasons != nil {
		t.Fatalf("root selection/reasons = %+v", got.Members[1])
	}
	if len(got.Members[0].Reasons) != 1 || got.Members[0].Reasons[0].From != id("scenario", "worker") {
		t.Fatalf("nested provenance missing: %+v", got.Members[0])
	}

	reversed := DependencyGraph{
		Components: []ComponentID{id("tool", "COMPILER"), id("scenario", "APP"), id("scenario", "WORKER"), id("resource", "DB")},
		Edges:      []DependencyEdge{graph.Edges[2], graph.Edges[1], graph.Edges[0]},
	}
	again, err := Resolve(reversed, []ComponentID{id("scenario", "app")})
	if err != nil {
		t.Fatal(err)
	}
	if got.Digest != again.Digest || !reflect.DeepEqual(got.Members, again.Members) {
		t.Fatalf("input ordering changed closure:\nfirst=%+v\nsecond=%+v", got, again)
	}
	if err := VerifyDigest(got); err != nil {
		t.Fatalf("valid closure digest rejected: %v", err)
	}
	changed := got
	changed.Members = append([]ClosureMember(nil), got.Members...)
	changed.Members[0].Component.Name = "tampered"
	if err := VerifyDigest(changed); err == nil {
		t.Fatal("tampered closure digest accepted")
	}
}

func TestResolveUnionsSelectedRootsAndRetainsEveryIncomingReason(t *testing.T) {
	shared := id("resource", "shared")
	closure, err := Resolve(DependencyGraph{
		Components: []ComponentID{id("scenario", "alpha"), id("scenario", "beta"), shared},
		Edges: []DependencyEdge{
			{From: id("scenario", "alpha"), To: shared, Required: false, DeclaredBy: "alpha"},
			{From: id("scenario", "beta"), To: shared, Required: true, DeclaredBy: "beta"},
		},
	}, []ComponentID{id("scenario", "beta"), id("scenario", "alpha")})
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range closure.Members {
		if member.Component == shared {
			if member.Selected || len(member.Reasons) != 2 {
				t.Fatalf("shared member did not retain the selected union reasons: %+v", member)
			}
			return
		}
	}
	t.Fatal("shared resource not in union")
}

func TestResolveEmptySelectionHasStableEmptyDigest(t *testing.T) {
	closure, err := Resolve(DependencyGraph{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(closure.Members) != 0 || closure.Digest == "" {
		t.Fatalf("empty closure = %+v", closure)
	}
}

func TestVerifyDigestRejectsUnknownSchemaVersionWithValidDigest(t *testing.T) {
	closure, err := Resolve(DependencyGraph{Components: []ComponentID{id("scenario", "demo")}}, []ComponentID{id("scenario", "demo")})
	if err != nil {
		t.Fatal(err)
	}
	closure.SchemaVersion = ClosureSchemaVersion + 1
	closure.Digest, err = closureDigest(closure.SchemaVersion, closure.Members)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyDigest(closure); !errors.Is(err, ErrInvalidGraph) {
		t.Fatalf("future schema error = %v, want ErrInvalidGraph", err)
	}
}

func TestResolveFailsClosedOnMissingNodesAndCycles(t *testing.T) {
	_, err := Resolve(DependencyGraph{Components: []ComponentID{id("scenario", "app")}, Edges: []DependencyEdge{{From: id("scenario", "app"), To: id("resource", "missing")}}}, []ComponentID{id("scenario", "app")})
	if !errors.Is(err, ErrMissingNode) {
		t.Fatalf("missing node error = %v", err)
	}
	_, err = Resolve(DependencyGraph{
		Components: []ComponentID{id("scenario", "a"), id("scenario", "b")},
		Edges:      []DependencyEdge{{From: id("scenario", "a"), To: id("scenario", "b")}, {From: id("scenario", "b"), To: id("scenario", "a")}},
	}, []ComponentID{id("scenario", "a")})
	if !errors.Is(err, ErrCycle) {
		t.Fatalf("cycle error = %v", err)
	}
}

func TestResolveOptionAllowsOnlyCyclesWithAnOptionalEdge(t *testing.T) {
	graph := DependencyGraph{
		Components: []ComponentID{id("scenario", "a"), id("scenario", "b")},
		Edges: []DependencyEdge{
			{From: id("scenario", "a"), To: id("scenario", "b"), Required: false},
			{From: id("scenario", "b"), To: id("scenario", "a"), Required: true},
		},
	}
	if _, err := Resolve(graph, []ComponentID{id("scenario", "a")}); !errors.Is(err, ErrCycle) {
		t.Fatalf("default cycle policy error = %v", err)
	}
	closure, err := ResolveWithOptions(graph, []ComponentID{id("scenario", "a")}, ResolveOptions{AllowCyclesWithOptionalEdges: true})
	if err != nil {
		t.Fatalf("optional cycle rejected: %v", err)
	}
	if len(closure.Members) != 2 || closure.Digest == "" {
		t.Fatalf("optional cycle closure = %+v", closure)
	}
}

func TestResolveOptionCanPreserveOwnerApprovedRequiredCycles(t *testing.T) {
	graph := DependencyGraph{
		Components: []ComponentID{id("scenario", "a"), id("scenario", "b")},
		Edges: []DependencyEdge{
			{From: id("scenario", "a"), To: id("scenario", "b"), Required: true},
			{From: id("scenario", "b"), To: id("scenario", "a"), Required: true},
		},
	}
	if _, err := Resolve(graph, []ComponentID{id("scenario", "a")}); !errors.Is(err, ErrCycle) {
		t.Fatalf("default cycle policy error = %v", err)
	}
	closure, err := ResolveWithOptions(graph, []ComponentID{id("scenario", "a")}, ResolveOptions{AllowCycles: true})
	if err != nil || len(closure.Members) != 2 {
		t.Fatalf("owner-approved cycle closure = %+v, err=%v", closure, err)
	}
}

func TestResolveFromSourceOwnsReachabilityAndPreservesCycleEdges(t *testing.T) {
	ids := map[string]ComponentID{
		"a": {Kind: "scenario", Name: "a"},
		"b": {Kind: "scenario", Name: "b"},
		"c": {Kind: "resource", Name: "c"},
	}
	calls := map[ComponentID]int{}
	source := DependencySourceFunc(func(id ComponentID) ([]DependencyEdge, error) {
		calls[id]++
		switch id {
		case ids["a"]:
			return []DependencyEdge{{To: ids["b"], Required: true}}, nil
		case ids["b"]:
			return []DependencyEdge{{To: ids["a"], Required: false}, {To: ids["c"], Required: true}}, nil
		default:
			return nil, nil
		}
	})
	closure, err := ResolveFromSource(source, []ComponentID{ids["a"]}, ResolveOptions{AllowCyclesWithOptionalEdges: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(closure.Members) != 3 {
		t.Fatalf("members=%d, want 3", len(closure.Members))
	}
	for id, count := range calls {
		if count != 1 {
			t.Errorf("source called %v times for %v, want once", count, id)
		}
	}
	if calls[ids["c"]] != 1 {
		t.Errorf("leaf source calls=%d, want 1", calls[ids["c"]])
	}
}

func TestResolveFromSourcePropagatesOwnerLoadFailure(t *testing.T) {
	want := errors.New("manifest unavailable")
	_, err := ResolveFromSource(DependencySourceFunc(func(id ComponentID) ([]DependencyEdge, error) {
		return nil, want
	}), []ComponentID{{Kind: "scenario", Name: "root"}}, ResolveOptions{})
	if !errors.Is(err, want) {
		t.Fatalf("err=%v, want wrapped owner failure", err)
	}
}
