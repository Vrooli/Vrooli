//go:build linux

package network

import "testing"

func TestSSOutputHasListenerForPort(t *testing.T) {
	output := []byte(`
LISTEN 0      4096                                             *:18060            *:*
LISTEN 0      128                                      127.0.0.1:18800      0.0.0.0:*
`)

	if !ssOutputHasListenerForPort(output, 18060) {
		t.Fatal("expected ss listener output to match wildcard listener")
	}
	if !ssOutputHasListenerForPort(output, 18800) {
		t.Fatal("expected ss listener output to match localhost listener")
	}
	if ssOutputHasListenerForPort(output, 1806) {
		t.Fatal("did not expect partial port match")
	}
}
