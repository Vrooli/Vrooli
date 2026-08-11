package mocks

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"backdrop-studio/internal/httpc"
)

// FakeDoer satisfies httpc.Doer for tests that don't want real network
// IO. Arrange via AddResponse (or by populating Responses / Errs
// directly) before the code under test runs; assert via Requests and
// Calls afterwards.
//
// The fake serves responses in order: the Nth Do call returns the Nth
// pre-loaded response (or error). Requests beyond the configured
// length surface a clear error rather than panicking — so an
// off-by-one in the test arrangement is obvious.
type FakeDoer struct {
	mu sync.Mutex

	Responses []*http.Response
	Errs      []error

	Requests []*http.Request
	Calls    atomic.Int64
}

// Do records the request, then returns the next pre-loaded response
// or error. Requests is appended in arrival order so tests can assert
// on method / URL / body after the fact.
func (f *FakeDoer) Do(req *http.Request) (*http.Response, error) {
	idx := int(f.Calls.Add(1)) - 1

	f.mu.Lock()
	f.Requests = append(f.Requests, req)
	var (
		resp *http.Response
		err  error
	)
	if idx < len(f.Errs) && f.Errs[idx] != nil {
		err = f.Errs[idx]
	} else if idx < len(f.Responses) {
		resp = f.Responses[idx]
	} else {
		err = fmt.Errorf("FakeDoer: no canned response for call %d (have %d)", idx+1, len(f.Responses))
	}
	f.mu.Unlock()

	return resp, err
}

// AddResponse is a convenience helper for the common case of "queue a
// status + body". The body becomes the response's Body via a bytes.Reader
// wrapped in io.NopCloser. Header is empty; tests that need a specific
// header populate Responses directly.
func (f *FakeDoer) AddResponse(status int, body []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Responses = append(f.Responses, &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	})
}

// AddError queues an error response (used when the test wants the
// transport itself to fail before a body would be read).
func (f *FakeDoer) AddError(err error) {
	if err == nil {
		err = errors.New("FakeDoer: nil error queued")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	// Pad Responses with a placeholder so the index alignment between
	// Errs and Responses stays correct.
	f.Responses = append(f.Responses, nil)
	f.Errs = append(f.Errs, err)
}

// Compile-time guarantee that *FakeDoer satisfies httpc.Doer.
var _ httpc.Doer = (*FakeDoer)(nil)
