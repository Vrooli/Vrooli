package mocks

import (
	"extraction-range/internal/database"
	"github.com/vrooli/api-core/databasetest"
)

// FakePinger re-exports the shared database ping fake used by API test suites.
type FakePinger = databasetest.FakePinger

var _ database.Pinger = (*FakePinger)(nil)
