package recipe

import "testing"

func TestMethodGraphRejectsNegativeFixtures(t *testing.T) {
	cases := []struct {
		name   string
		method Method
		reason string
	}{
		{"self-dependency", Method{Steps: []MethodStep{{ID: "a", DependsOn: []string{"a"}}}}, "self-dependency"},
		{"cycle", Method{Steps: []MethodStep{{ID: "a", DependsOn: []string{"b"}}, {ID: "b", DependsOn: []string{"a"}}}}, "cycle"},
		{"forward-dependency", Method{Steps: []MethodStep{{ID: "a", DependsOn: []string{"b"}}, {ID: "b"}}}, "earlier step"},
		{"missing-output", Method{Steps: []MethodStep{{ID: "a", Inputs: []string{"missing"}}}}, "dangling"},
		{"over-split", Method{Steps: []MethodStep{{ID: "a", AllocationCount: 2}}}, "split"},
		{"duplicate-output", Method{Steps: []MethodStep{{ID: "a", Outputs: []string{"x"}}, {ID: "b", Outputs: []string{"x"}}}}, "duplicate output"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateMethod(tc.method, map[string]bool{"ingredient": true}); err == nil || !contains(err.Error(), tc.reason) {
				t.Fatalf("error=%v, want %q", err, tc.reason)
			}
		})
	}
}

func contains(value, want string) bool {
	for i := 0; i+len(want) <= len(value); i++ {
		if value[i:i+len(want)] == want {
			return true
		}
	}
	return false
}
