//go:build !linux

package nativeisolation

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	"time"
)

type unavailableWitnessReader struct{}

func newWitnessReader() witnessReader           { return unavailableWitnessReader{} }
func (unavailableWitnessReader) now() time.Time { return time.Now().UTC() }
func (unavailableWitnessReader) read(context.Context, ProbeSpec) (witnessSnapshot, error) {
	return witnessSnapshot{}, ErrRefused
}
func (unavailableWitnessReader) qualify(context.Context, *effortauthority.NativeUIDIsolation) error {
	return ErrRefused
}
