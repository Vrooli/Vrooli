//go:build darwin && !cgo

package collectors

import "testing"

func TestDarwinNoCGOExplicitlyUnsupported(t *testing.T) {
	cores, status, err := readDarwinCPUCounters()
	if len(cores) != 0 || status != "unsupported" || err == nil {
		t.Fatalf("no-CGO backend must explicitly refuse: %v %q %v", cores, status, err)
	}
}
