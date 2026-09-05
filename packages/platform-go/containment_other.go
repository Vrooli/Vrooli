//go:build !linux && !darwin && !windows

package platform

func containedCommand(ContainedSpec) (*Contained, error) { return nil, ErrUnsupported }

func containSelf(string, Containment) (ScopeRef, string, error) {
	return ScopeRef{Kind: ScopeKindNone}, MethodNone, ErrUnsupported
}

func freezeScope(ScopeRef) error { return ErrUnsupported }

func thawScope(ScopeRef) error { return ErrUnsupported }

func scopeFrozen(ScopeRef) (bool, error) { return false, ErrUnsupported }
