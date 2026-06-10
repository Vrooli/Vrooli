package fetch_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"

	"web-search/internal/research/fetch"

	"github.com/stretchr/testify/require"
)

// fakeCaptureClient records the last Capture request and returns a canned
// response or error.
type fakeCaptureClient struct {
	lastReq *capturev1.CaptureRequest
	resp    *capturev1.CaptureResponse
	err     error
	calls   int
}

func (f *fakeCaptureClient) Capture(
	_ context.Context,
	req *connect.Request[capturev1.CaptureRequest],
) (*connect.Response[capturev1.CaptureResponse], error) {
	f.calls++
	f.lastReq = req.Msg
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.resp), nil
}

func basWithClient(client fetch.CaptureClient, resolveErr error) *fetch.BASFetcher {
	return &fetch.BASFetcher{
		Resolve: func(_ context.Context) (fetch.CaptureClient, error) {
			if resolveErr != nil {
				return nil, resolveErr
			}
			return client, nil
		},
	}
}

func TestBASFetcherExtractsReadableTextFromInlineDom(t *testing.T) {
	client := &fakeCaptureClient{resp: &capturev1.CaptureResponse{
		ExecutionId: "exec-1",
		DomHtml:     `<html><body><nav>menu</nav><p>Rendered article text.</p></body></html>`,
	}}
	f := basWithClient(client, nil)

	text, err := f.Fetch(context.Background(), "https://example.com/spa")
	require.NoError(t, err)
	require.Contains(t, text, "Rendered article text.")
	require.NotContains(t, text, "menu")

	// The request must use inline DOM (no artifact-path coupling), capture
	// type DOM, and wait for networkidle so JS shells hydrate.
	require.True(t, client.lastReq.GetInlineDom())
	require.Equal(t, []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_DOM}, client.lastReq.GetCaptures())
	require.True(t, client.lastReq.GetWaitFor().GetNetworkidle())
}

func TestBASFetcherErrorsOnEmptyDom(t *testing.T) {
	client := &fakeCaptureClient{resp: &capturev1.CaptureResponse{ExecutionId: "exec-1"}}
	f := basWithClient(client, nil)

	_, err := f.Fetch(context.Background(), "https://example.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no DOM")
}

func TestBASFetcherErrorsWhenBASUnavailable(t *testing.T) {
	f := basWithClient(nil, errors.New("scenario not running"))
	_, err := f.Fetch(context.Background(), "https://example.com")
	require.Error(t, err)
}

func TestBASFetcherRetriesResolutionAfterCaptureError(t *testing.T) {
	// First capture fails (e.g. BAS restarted on a new port): the cached
	// client must be dropped so the next fetch re-resolves.
	failing := &fakeCaptureClient{err: errors.New("connection refused")}
	working := &fakeCaptureClient{resp: &capturev1.CaptureResponse{DomHtml: "<p>recovered</p>"}}
	clients := []fetch.CaptureClient{failing, working}
	resolves := 0
	f := &fetch.BASFetcher{
		Resolve: func(_ context.Context) (fetch.CaptureClient, error) {
			client := clients[resolves]
			resolves++
			return client, nil
		},
	}

	_, err := f.Fetch(context.Background(), "https://example.com")
	require.Error(t, err)

	text, err := f.Fetch(context.Background(), "https://example.com")
	require.NoError(t, err)
	require.Contains(t, text, "recovered")
	require.Equal(t, 2, resolves, "capture failure must invalidate the cached client")
}

func TestBASFetcherRejectsEmptyURL(t *testing.T) {
	f := basWithClient(&fakeCaptureClient{}, nil)
	_, err := f.Fetch(context.Background(), " ")
	require.Error(t, err)
}
