//go:build !linux

package platform

import (
	"errors"
	"testing"
)

// No non-Linux body is host-verified, so StopScope must say so instead of
// pretending a tree it never touched is gone.
func TestStopScopeIsUnsupportedOffLinux(t *testing.T) {
	err := StopScope(ScopeRef{Kind: ScopeKindProcessGroup, PID: 4242})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("StopScope = %v, want ErrUnsupported", err)
	}
}
