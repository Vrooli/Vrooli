package proposals

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProposal_UnmarshalJSON_FormMutationList(t *testing.T) {
	raw := `{"form":"mutation_list","mutations":[{"id":"m1","op":"add_item","item":{"kind":"execute","name":"foo","title":"Foo"}}]}`
	var p Proposal
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Form != FormMutationList {
		t.Fatalf("expected form=%s, got %s", FormMutationList, p.Form)
	}
	if len(p.Mutations) != 1 || p.Mutations[0].Op != OpAddItem {
		t.Fatalf("expected one add_item mutation, got %+v", p.Mutations)
	}
}

func TestProposal_UnmarshalJSON_FormFullGraph(t *testing.T) {
	raw := `{"form":"full_graph","graph":{"nodes":[{"id":"execute/foo","kind":"execute","name":"foo","title":"Foo"}],"edges":[]}}`
	var p Proposal
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Form != FormFullGraph {
		t.Fatalf("expected form=%s, got %s", FormFullGraph, p.Form)
	}
	if p.Graph == nil || len(p.Graph.Nodes) != 1 {
		t.Fatalf("expected graph with 1 node, got %+v", p.Graph)
	}
}

func TestProposal_UnmarshalJSON_RejectsMissingForm(t *testing.T) {
	raw := `{"mutations":[]}`
	var p Proposal
	err := json.Unmarshal([]byte(raw), &p)
	if err == nil || !strings.Contains(err.Error(), "form is required") {
		t.Fatalf("expected missing-form error, got %v", err)
	}
}

func TestProposal_UnmarshalJSON_RejectsMutationListWithGraph(t *testing.T) {
	raw := `{"form":"mutation_list","graph":{"nodes":[],"edges":[]}}`
	var p Proposal
	err := json.Unmarshal([]byte(raw), &p)
	if err == nil || !strings.Contains(err.Error(), "must not include graph") {
		t.Fatalf("expected mutation_list+graph error, got %v", err)
	}
}

func TestProposal_UnmarshalJSON_RejectsFullGraphWithoutGraph(t *testing.T) {
	raw := `{"form":"full_graph"}`
	var p Proposal
	err := json.Unmarshal([]byte(raw), &p)
	if err == nil || !strings.Contains(err.Error(), "must include graph") {
		t.Fatalf("expected full_graph-without-graph error, got %v", err)
	}
}

func TestProposal_UnmarshalJSON_RejectsFullGraphWithMutations(t *testing.T) {
	raw := `{"form":"full_graph","graph":{"nodes":[],"edges":[]},"mutations":[{"id":"m1","op":"add_item"}]}`
	var p Proposal
	err := json.Unmarshal([]byte(raw), &p)
	if err == nil || !strings.Contains(err.Error(), "must not include mutations") {
		t.Fatalf("expected full_graph+mutations error, got %v", err)
	}
}

func TestProposal_UnmarshalJSON_RejectsUnknownForm(t *testing.T) {
	raw := `{"form":"diff","mutations":[]}`
	var p Proposal
	err := json.Unmarshal([]byte(raw), &p)
	if err == nil || !strings.Contains(err.Error(), "not recognized") {
		t.Fatalf("expected unknown-form error, got %v", err)
	}
}

func TestAllOps_Stable(t *testing.T) {
	ops := AllOps()
	if len(ops) < 8 {
		t.Fatalf("expected AllOps to return the full canonical list, got %d", len(ops))
	}
	seen := make(map[Op]struct{}, len(ops))
	for _, op := range ops {
		if _, dup := seen[op]; dup {
			t.Fatalf("duplicate op in AllOps: %s", op)
		}
		seen[op] = struct{}{}
	}
}

func TestItemSpec_Ref(t *testing.T) {
	s := ItemSpec{Kind: "execute", Name: "foo"}
	if got := s.Ref(); got != "execute/foo" {
		t.Fatalf("expected execute/foo, got %s", got)
	}
}
