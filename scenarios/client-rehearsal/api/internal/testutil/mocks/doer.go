package mocks

import (
	"client-rehearsal/internal/httpc"

	"github.com/vrooli/api-core/apihttptest"
)

// FakeDoer reuses api-core's shared ordered HTTP fake so scenario tests and
// other APIs exercise the same concurrency-safe request boundary.
type FakeDoer = apihttptest.FakeDoer

var _ httpc.Doer = (*FakeDoer)(nil)
