package apihttp_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
)

type recordingRoundTripper struct {
	request *http.Request
}

func (r *recordingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	r.request = request
	return &http.Response{
		StatusCode: http.StatusNoContent,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
		Request:    request,
	}, nil
}

func TestTestModeTransportPropagatesMarkerWithoutMutatingCaller(t *testing.T) {
	recorder := &recordingRoundTripper{}
	client := apihttp.NewTestModeClient(&http.Client{Transport: recorder})
	request, err := http.NewRequestWithContext(database.WithTestMode(t.Context()), http.MethodGet, "http://scenario.test/check", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.Do(request); err != nil {
		t.Fatal(err)
	}
	if got := recorder.request.Header.Get(apihttp.TestModeHeader); got != apihttp.TestModeValue {
		t.Fatalf("outbound test-mode header = %q, want %q", got, apihttp.TestModeValue)
	}
	if got := request.Header.Get(apihttp.TestModeHeader); got != "" {
		t.Fatalf("caller request was mutated with test-mode header %q", got)
	}
}

func TestTestModeTransportPreservesOrdinaryRequest(t *testing.T) {
	recorder := &recordingRoundTripper{}
	client := apihttp.NewTestModeClient(&http.Client{Transport: recorder})
	request, err := http.NewRequest(http.MethodGet, "http://scenario.test/check", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Trace", "unchanged")

	if _, err := client.Do(request); err != nil {
		t.Fatal(err)
	}
	if got := recorder.request.Header.Get(apihttp.TestModeHeader); got != "" {
		t.Fatalf("ordinary request gained test-mode header %q", got)
	}
	if got := recorder.request.Header.Get("X-Trace"); got != "unchanged" {
		t.Fatalf("ordinary request trace header = %q, want unchanged", got)
	}
}
