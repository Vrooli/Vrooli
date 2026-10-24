package administration

import (
	"net/http"
)

// HTTPDoer is the outbound-HTTP seam for remote-profile administration.
// Production wires an http.Client once; tests provide a deterministic client.
// seam: HTTPDoer keeps remote-profile transport independently substitutable.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}
