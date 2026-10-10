//go:build !linux

package main

import (
	"context"
	"errors"
	"github.com/vrooli/vrooli/internal/nativeisolation/probeprotocol"
	"time"
)

type fixedObservationEffects struct{}

func (fixedObservationEffects) markers(probeprotocol.Observation) error {
	return errors.New("fixed observation unavailable")
}
func (fixedObservationEffects) child(context.Context, probeprotocol.Observation) (int, func() error, error) {
	return 0, nil, errors.New("fixed observation unavailable")
}
func (fixedObservationEffects) report([]byte) error {
	return errors.New("fixed observation unavailable")
}
func (fixedObservationEffects) parentPID() int                  { return 0 }
func (fixedObservationEffects) wait(context.Context, time.Time) {}

func (fixedObservationEffects) now() time.Time { return time.Now().UTC() }
