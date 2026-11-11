package apihttptest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/api-core/apihttp"
)

func fetchBody(t *testing.T, doer apihttp.Doer, url string) string {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := doer.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestDoerContractAcceptsClientAndFake(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "from http client")
	}))
	t.Cleanup(server.Close)
	if got := fetchBody(t, server.Client(), server.URL); got != "from http client" {
		t.Fatalf("client response = %q", got)
	}

	fake := &FakeDoer{}
	fake.AddResponse(http.StatusOK, []byte("from fake"))
	if got := fetchBody(t, fake, "https://example.invalid/path"); got != "from fake" {
		t.Fatalf("fake response = %q", got)
	}
	if got := fake.Calls.Load(); got != 1 {
		t.Fatalf("fake calls = %d, want 1", got)
	}
	if len(fake.Requests) != 1 || fake.Requests[0].Method != http.MethodGet || !strings.HasSuffix(fake.Requests[0].URL.Path, "/path") {
		t.Fatalf("fake requests = %#v, want one GET ending in /path", fake.Requests)
	}
}
