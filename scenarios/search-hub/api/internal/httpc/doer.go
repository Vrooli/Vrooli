// Package httpc names search-hub's outbound HTTP seam. The interface and its
// test fake are owned by api-core (apihttp.Doer, apihttptest.FakeDoer); this
// package keeps only the bounded default client.
package httpc

import (
	"net/http"
	"time"

	"github.com/vrooli/api-core/apihttp"
)

// Doer is api-core's outbound HTTP interface.
type Doer = apihttp.Doer

// NewDefault returns the bounded outbound client the provider fan-out uses.
func NewDefault() Doer {
	return &http.Client{Timeout: 30 * time.Second}
}
