package browsercapture

import (
	"context"

	capturepb "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

// FakeCaptureClient is the test double for CaptureClient. It records every
// request and returns a canned response (or per-call responses keyed by URL).
//
// seam: FakeCaptureClient is the test wiring for the CaptureClient seam
// (capture.go).
type FakeCaptureClient struct {
	// Response is returned for every Capture call when Responses has no match.
	Response *capturepb.CaptureResponse
	// Responses, when set, maps a CaptureRequest.Url to a specific response.
	Responses map[string]*capturepb.CaptureResponse
	// Err, when set, fails every Capture call.
	Err error

	// Requests records every request passed to Capture, in order.
	Requests []*capturepb.CaptureRequest
}

// Capture records the request and returns the configured response/error.
func (f *FakeCaptureClient) Capture(ctx context.Context, req *capturepb.CaptureRequest) (*capturepb.CaptureResponse, error) {
	f.Requests = append(f.Requests, req)
	if f.Err != nil {
		return nil, f.Err
	}
	if resp, ok := f.Responses[req.GetUrl()]; ok {
		return resp, nil
	}
	if f.Response != nil {
		return f.Response, nil
	}
	return &capturepb.CaptureResponse{}, nil
}

// CallCount reports how many Capture calls were recorded.
func (f *FakeCaptureClient) CallCount() int { return len(f.Requests) }
