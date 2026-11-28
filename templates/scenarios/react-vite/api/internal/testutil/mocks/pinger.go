package mocks

import (
	"github.com/vrooli/api-core/databasetest"
	"{{SCENARIO_ID}}/internal/database"
)

// FakePinger re-exports the shared API test fake so generated scenarios do
// not maintain a second implementation of the same database seam.
type FakePinger = databasetest.FakePinger

// Compile-time guarantee that *FakePinger satisfies database.Pinger.
var _ database.Pinger = (*FakePinger)(nil)
