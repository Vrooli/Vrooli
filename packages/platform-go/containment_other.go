//go:build !linux && !darwin && !windows

package platform

func containedCommand(ContainedSpec) (*Contained, error) { return nil, ErrUnsupported }

func containSelf(string, Containment) (ScopeRef, string, error) {
	return ScopeRef{Kind: ScopeKindNone}, MethodNone, ErrUnsupported
}

func freezeScope(ScopeRef) error { return ErrUnsupported }

func thawScope(ScopeRef) error { return ErrUnsupported }

func scopeFrozen(ScopeRef) (bool, error) { return false, ErrUnsupported }

func scopeProcesses(ScopeRef) ([]int, error) { return nil, ErrUnsupported }

func scopeOccupancy(ScopeRef) (Occupancy, error) { return Occupancy{}, ErrUnsupported }

func scopeChildren(ScopeRef) ([]ScopeRef, error) { return nil, ErrUnsupported }

// adoptIntoScope is unsupported here: neither a process group nor a Job
// Object can take over a process that is already running under another.
// Placement must happen at birth on these platforms.
func adoptIntoScope(AdoptSpec) (ScopeRef, string, error) {
	return ScopeRef{Kind: ScopeKindNone}, MethodNone, ErrUnsupported
}

func sliceCgroupPath(string) (string, error) { return "", ErrUnsupported }
