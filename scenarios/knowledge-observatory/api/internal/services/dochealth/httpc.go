package dochealth

import (
	"net/http"
	"time"
)

// Doer is the seam used by the external-link probe so tests can inject a
// fake transport without booting a real HTTP server. The standard library
// *http.Client satisfies this interface.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultDoer constructs an HTTP client with the configured per-request
// timeout. It is used when callers do not inject a Doer of their own.
func defaultDoer(timeout time.Duration) Doer {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &http.Client{Timeout: timeout}
}
