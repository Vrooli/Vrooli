package mocks

import (
	"extraction-range/internal/httpc"
	"github.com/vrooli/api-core/apihttptest"
)

// FakeDoer re-exports the shared API test fake so local handler/client tests
// use the same ordered, concurrency-safe transport companion as other APIs.
type FakeDoer = apihttptest.FakeDoer

var _ httpc.Doer = (*FakeDoer)(nil)
