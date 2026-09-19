package recipe

import "fmt"

type (
	Method struct {
		ID, Name string
		Steps    []MethodStep
	}
	MethodStep struct {
		ID, Instruction            string
		DependsOn, Inputs, Outputs []string
		AllocationCount            int
	}
)

type GraphError struct{ Path, Reason string }

func (e GraphError) Error() string { return e.Path + ": " + e.Reason }

// ValidateMethod rejects ambiguous execution graphs before they can reach planning.
// Inputs may refer to the canonical ingredient/component namespace supplied by the
// caller; outputs must be unique within a method and dependencies must point back.
func ValidateMethod(m Method, available map[string]bool) error {
	seen := map[string]bool{}
	produced := map[string]bool{}
	for i, step := range m.Steps {
		if step.ID == "" {
			return GraphError{fmt.Sprintf("steps[%d].id", i), "must not be empty"}
		}
		if seen[step.ID] {
			return GraphError{fmt.Sprintf("steps[%d].id", i), "duplicate step id"}
		}
		seen[step.ID] = true
		for _, dep := range step.DependsOn {
			if dep == step.ID {
				return GraphError{fmt.Sprintf("steps[%d].depends_on", i), "self-dependency"}
			}
			found := false
			for j := 0; j < i; j++ {
				if m.Steps[j].ID == dep {
					found = true
				}
			}
			if !found {
				for j := i + 1; j < len(m.Steps); j++ {
					for _, laterDep := range m.Steps[j].DependsOn {
						if laterDep == step.ID && m.Steps[j].ID == dep {
							return GraphError{fmt.Sprintf("steps[%d].depends_on", i), "cycle detected"}
						}
					}
				}
				return GraphError{fmt.Sprintf("steps[%d].depends_on", i), "dependency must reference an earlier step"}
			}
		}
		if step.AllocationCount > 1 {
			return GraphError{fmt.Sprintf("steps[%d].allocation_count", i), "one ingredient use cannot be split across multiple allocations"}
		}
		for _, input := range step.Inputs {
			if !available[input] && !produced[input] {
				return GraphError{fmt.Sprintf("steps[%d].inputs", i), "dangling component reference: " + input}
			}
		}
		for _, output := range step.Outputs {
			if produced[output] {
				return GraphError{fmt.Sprintf("steps[%d].outputs", i), "duplicate output: " + output}
			}
			produced[output] = true
		}
	}
	return nil
}
