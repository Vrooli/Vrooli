package apihttptest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
)

// FakeDoer is a concurrency-safe, ordered HTTP transport fake for
// scenario API client tests. Queue responses and errors before sharing it
// with requests; inspect Requests and Calls after those requests finish.
type FakeDoer struct {
	mu sync.Mutex

	Responses []*http.Response
	Errs      []error
	Requests  []*http.Request
	Calls     atomic.Int64
}

// Do records req and returns its queued response or error. An unconfigured
// call returns an error so unexpected network traffic fails visibly.
func (f *FakeDoer) Do(req *http.Request) (*http.Response, error) {
	idx := int(f.Calls.Add(1)) - 1

	f.mu.Lock()
	defer f.mu.Unlock()
	f.Requests = append(f.Requests, req)
	if idx < len(f.Errs) && f.Errs[idx] != nil {
		return nil, f.Errs[idx]
	}
	if idx < len(f.Responses) {
		return f.Responses[idx], nil
	}
	return nil, fmt.Errorf("FakeDoer: no canned response for call %d (have %d)", idx+1, len(f.Responses))
}

// AddResponse queues a response with the given status and body.
func (f *FakeDoer) AddResponse(status int, body []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Responses = append(f.Responses, &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	})
	f.Errs = append(f.Errs, nil)
}

// AddError queues a transport failure in response order.
func (f *FakeDoer) AddError(err error) {
	if err == nil {
		err = errors.New("FakeDoer: nil error queued")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Responses = append(f.Responses, nil)
	f.Errs = append(f.Errs, err)
}
