package ruleengine

import (
	"fmt"

	rulespkg "scenario-auditor/rules"
)

// ImplementationStatus captures whether a rule implementation was loaded successfully.
type ImplementationStatus struct {
	Valid   bool   `json:"valid"`
	Error   string `json:"error,omitempty"`
	Details string `json:"details,omitempty"`
}

// Info represents metadata extracted from a rule file together with its executor.
type Info struct {
	rulespkg.Rule
	Reason         string               `json:"reason"`
	FilePath       string               `json:"file_path"`
	Targets        []string             `json:"targets"`
	Implementation ImplementationStatus `json:"implementation"`
	executor       ruleExecutor
}

// WithExecutor returns a copy of the rule info with the executor configured.
func (r Info) WithExecutor(exec ruleExecutor, status ImplementationStatus) Info {
	r.executor = exec
	r.Implementation = status
	return r
}

// Check executes the underlying rule implementation if one is available.
func (r Info) Check(content string, filepath string, scenario string) ([]rulespkg.Violation, error) {
	if r.executor == nil {
		if !r.Implementation.Valid {
			if r.Implementation.Error != "" {
				return nil, fmt.Errorf("rule implementation unavailable: %s", r.Implementation.Error)
			}
			return nil, fmt.Errorf("rule implementation unavailable: executor missing")
		}
		return nil, fmt.Errorf("rule execution not configured for %s", r.Rule.ID)
	}
	return r.executor.Execute(content, filepath, scenario)
}

// ArgumentInfo describes a function argument for documentation purposes.
type ArgumentInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ExecutionCall describes a step in the rule execution flow.
type ExecutionCall struct {
	Source      string `json:"source"`
	Description string `json:"description"`
	Reference   string `json:"reference"`
}

// ExecutionInfo documents how a rule is executed.
type ExecutionInfo struct {
	Signature string          `json:"signature"`
	Arguments []ArgumentInfo  `json:"arguments"`
	CallFlow  []ExecutionCall `json:"call_flow"`
	Notes     []string        `json:"notes"`
}
