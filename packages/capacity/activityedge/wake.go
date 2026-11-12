package activityedge

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

type wakeRequestKey struct{}

type wakeRoundTripper struct {
	Base        http.RoundTripper
	Wake        func(context.Context) error
	WakeContext func(context.Context) (context.Context, func(), bool)
}

func (t wakeRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	response, err := base.RoundTrip(request)
	if err == nil || t.Wake == nil || request.Context().Value(wakeRequestKey{}) != true || !isRefusedUpstreamConnection(err) {
		return response, err
	}
	wakeParent := request.Context()
	releaseWake := func() {}
	if t.WakeContext != nil {
		var allowed bool
		wakeParent, releaseWake, allowed = t.WakeContext(wakeParent)
		if !allowed {
			// The resource transition has closed admission and is waiting for this
			// request to leave the in-flight set.
			return nil, err
		}
	}
	defer releaseWake()
	ctx, cancel := context.WithTimeout(wakeParent, 2*time.Minute)
	defer cancel()
	if wakeErr := t.Wake(ctx); wakeErr != nil {
		return nil, fmt.Errorf("resource wake failed after upstream connection error (%v): %w", err, wakeErr)
	}
	retry := request
	if request.Body != nil && request.Body != http.NoBody {
		if request.GetBody == nil {
			return nil, fmt.Errorf("resource wake succeeded, but the request body cannot be replayed safely after upstream connection error: %w", err)
		}
		body, bodyErr := request.GetBody()
		if bodyErr != nil {
			return nil, fmt.Errorf("resource wake succeeded, but request body replay failed after upstream connection error: %w", bodyErr)
		}
		retry = request.Clone(request.Context())
		retry.Body = body
	}
	return base.RoundTrip(retry)
}

func isRefusedUpstreamConnection(err error) bool {
	var opErr *net.OpError
	return errors.As(err, &opErr) && errors.Is(err, syscall.ECONNREFUSED)
}
