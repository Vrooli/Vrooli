package recipe

import "testing"

func TestFIX05ParallelPreparationBranchesJoin(t *testing.T) {
	method := Method{ID: "parallel-prep", Steps: []MethodStep{
		{ID: "sauce", Inputs: []string{"tomato"}, Outputs: []string{"sauce"}},
		{ID: "vegetables", Inputs: []string{"beans"}, Outputs: []string{"vegetables"}},
		{ID: "join", DependsOn: []string{"sauce", "vegetables"}, Inputs: []string{"sauce", "vegetables"}, Outputs: []string{"bowl"}},
	}}
	if err := ValidateMethod(method, map[string]bool{"tomato": true, "beans": true}); err != nil {
		t.Fatal(err)
	}
}
