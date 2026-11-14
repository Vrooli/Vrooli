//go:build linux

package platform

import "testing"

func TestParseProcessGroupAcceptsKernelProcessWithZeroGroup(t *testing.T) {
	groupID, state, ok, err := parseProcessGroup([]byte("10 (kernel worker) S 1 0 0"))
	if err != nil {
		t.Fatalf("parseProcessGroup() error = %v", err)
	}
	if !ok || groupID != 0 || state != 'S' {
		t.Fatalf("parseProcessGroup() = (%d, %q, %t), want (0, 'S', true)", groupID, state, ok)
	}
}
