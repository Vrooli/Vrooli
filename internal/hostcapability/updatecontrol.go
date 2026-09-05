// Package hostcapability owns typed control semantics for host capabilities
// whose updates must be observed, guarded as a coupled set, or explicitly
// reported as not implemented. It does not perform host mutation itself.
package hostcapability

import "fmt"

type UpdateControl string

const (
	UpdateObserve UpdateControl = "observe"
	UpdateGuard   UpdateControl = "guard"
	UpdateOwn     UpdateControl = "own"
)

type CoupledSet struct {
	Name    string
	Members []string
}

type UpdateControlResult struct {
	Mode          UpdateControl `json:"mode"`
	FrozenMembers []string      `json:"frozen_members,omitempty"`
	Policy        string        `json:"policy,omitempty"`
	Reason        string        `json:"reason,omitempty"`
	Implemented   bool          `json:"implemented"`
}

// ReconcileCoupledSet is deliberately pure. The control plane may use its
// result to render or apply a policy, while observe never mutates host state.
func ReconcileCoupledSet(mode UpdateControl, set CoupledSet) UpdateControlResult {
	result := UpdateControlResult{Mode: mode, Implemented: true}
	if len(set.Members) == 0 {
		result.Reason = "declared coupled set has no members"
		return result
	}
	switch mode {
	case UpdateObserve:
		result.Reason = "observe records drift without writing host policy"
	case UpdateGuard:
		result.FrozenMembers = append([]string(nil), set.Members...)
		result.Policy = RenderGuardPolicy(set)
		result.Reason = "guard freezes every member of the declared coupled set"
	case UpdateOwn:
		result.Implemented = false
		result.Reason = "own update control is not implemented"
	default:
		result.Implemented = false
		result.Reason = fmt.Sprintf("unknown update-control mode %q", mode)
	}
	return result
}

func RenderGuardPolicy(set CoupledSet) string {
	policy := "# Managed by Vrooli safeguard: nvidia-driver\n# Coupled set: " + set.Name + "\n"
	for _, member := range set.Members {
		policy += "# Freeze member: " + member + "\n"
	}
	return policy
}
