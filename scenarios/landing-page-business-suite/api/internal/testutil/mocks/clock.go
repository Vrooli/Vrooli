// Package mocks contains reusable domain test doubles.
package mocks

import (
	"time"

	"landing-page-business-suite-api/internal/clock"
)

type FakeClock struct {
	Current time.Time
}

func (f FakeClock) Now() time.Time { return f.Current }

var _ clock.Clock = FakeClock{}
