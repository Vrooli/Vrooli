// Package mocks re-exports the shared api-core test fake so Treasury tests use
// the same ordered HTTP behavior as other scenario API clients.
package mocks

import (
	"treasury/internal/httpc"

	"github.com/vrooli/api-core/apihttptest"
)

// FakeDoer is the canonical shared fake for the local outbound HTTP seam.
type FakeDoer = apihttptest.FakeDoer

var _ httpc.Doer = (*FakeDoer)(nil)
