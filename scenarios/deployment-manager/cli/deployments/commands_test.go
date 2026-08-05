package deployments

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestDeployValidateOnlyShortCircuits(t *testing.T) {
	var validateCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/validate") {
			validateCalled = true
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))
	if err := cmd.Deploy([]string{"demo", "--validate-only"}); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}
	if !validateCalled {
		t.Fatalf("expected validate endpoint to be called")
	}
}

func TestLogsAcceptsFilters(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))
	if err := cmd.Logs([]string{"demo", "--level", "error", "--search", "fail"}); err != nil {
		t.Fatalf("logs failed: %v", err)
	}
	if !strings.Contains(query, "level=error") || !strings.Contains(query, "search=fail") {
		t.Fatalf("expected query params, got %s", query)
	}
}

func testAPIClient(base string) *cliutil.APIClient {
	return cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: cliutil.APIBaseOptions{DefaultBase: base}}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: base} },
		func() string { return "" },
	)
}
