//go:build !linux

package lightingbinding

import "testing"

func TestNativeLightingRequiresSupportedKernelPeerIdentity(t *testing.T) {
	if listener, err := Listen("unused", Identity{}); err == nil {
		listener.Close()
		t.Fatal("unsupported native peer identity admitted a listener")
	}
}
