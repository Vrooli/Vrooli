//go:build audio_lab

package soak

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type leaseBodyTracker struct {
	io.Reader
	closed bool
}

func (b *leaseBodyTracker) Close() error { b.closed = true; return nil }

type leaseRoundTripper func(*http.Request) (*http.Response, error)

func (f leaseRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestVerifyAudioToolsDemandLeaseClosesTransferredResponseBody(t *testing.T) {
	body := &leaseBodyTracker{Reader: strings.NewReader(`{"verified":true,"receipt_id":"tglease_opaque","owner":"test-genie","target":"audio-tools"}`)}
	client := &http.Client{Transport: leaseRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: body, Request: req}, nil
	})}

	require.NoError(t, VerifyAudioToolsDemandLease(context.Background(), "https://test-genie.example", "tglease_opaque", client))
	require.True(t, body.closed, "the caller must close the transferred HTTP response body")
}

func TestVerifyAudioToolsDemandLeaseDoesNotFollowRedirect(t *testing.T) {
	var redirectedRequests int
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirectedRequests++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"verified":true,"receipt_id":"tglease_opaque","owner":"test-genie","target":"audio-tools"}`))
	}))
	defer target.Close()

	verifier := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, &http.Request{}, target.URL, http.StatusTemporaryRedirect)
	}))
	defer verifier.Close()

	err := VerifyAudioToolsDemandLease(context.Background(), verifier.URL, "tglease_opaque", verifier.Client())
	require.Error(t, err)
	require.Zero(t, redirectedRequests, "receipt token must not be sent to a redirect target")
}

func TestVerifyAudioToolsDemandLeaseRejectsUntrustedEndpointsAndReceipts(t *testing.T) {
	for _, endpoint := range []string{
		"file:///tmp/verifier", "http://user:pass@example.test", "http://example.test?target=other",
		"http://example.test#fragment", "//example.test",
	} {
		t.Run(endpoint, func(t *testing.T) {
			require.Error(t, VerifyAudioToolsDemandLease(context.Background(), endpoint, "tglease_opaque", nil))
		})
	}

	for _, body := range []string{
		`not-json`,
		`{"verified":false,"receipt_id":"tglease_opaque","owner":"test-genie","target":"audio-tools"}`,
		`{"verified":true,"receipt_id":"other","owner":"test-genie","target":"audio-tools"}`,
		`{"verified":true,"receipt_id":"tglease_opaque","owner":"other","target":"audio-tools"}`,
		`{"verified":true,"receipt_id":"tglease_opaque","owner":"test-genie","target":"other"}`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}))
		require.Error(t, VerifyAudioToolsDemandLease(context.Background(), server.URL, "tglease_opaque", server.Client()))
		server.Close()
	}
}
