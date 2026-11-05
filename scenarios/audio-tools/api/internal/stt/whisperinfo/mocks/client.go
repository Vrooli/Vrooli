// Package mocks hosts the FakeClient used in LocalProvider tests.
package mocks

import "audio-tools/internal/stt/whisperinfo"

// FakeClient returns a fixed Info. Tests override Info to assert that
// LocalProvider surfaces it verbatim.
type FakeClient struct {
	Info whisperinfo.Info
}

// CurrentModel satisfies whisperinfo.Client.
func (f *FakeClient) CurrentModel() whisperinfo.Info { return f.Info }

var _ whisperinfo.Client = (*FakeClient)(nil)
