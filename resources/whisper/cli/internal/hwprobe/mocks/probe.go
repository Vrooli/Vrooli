// Package mocks hosts the FakeProbe used by recommender and CLI tests.
package mocks

import (
	"context"

	"resource-whisper/cli/internal/hwprobe"
)

// FakeProbe returns a programmable HostCapabilities. Tests set Caps
// and (optionally) Err to drive specific recommender rows without
// touching the real OS.
type FakeProbe struct {
	Caps hwprobe.HostCapabilities
	Err  error
}

// Detect satisfies hwprobe.Probe.
func (f *FakeProbe) Detect(context.Context) (hwprobe.HostCapabilities, error) {
	return f.Caps, f.Err
}

var _ hwprobe.Probe = (*FakeProbe)(nil)
