package swaps

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestRunRequiresSubcommand(t *testing.T) {
	cmd := New(nil)
	if err := cmd.Run([]string{}); err == nil || !strings.Contains(err.Error(), "swaps subcommand is required") {
		t.Fatalf("expected subcommand error, got %v", err)
	}
}

func TestApplyCallsAPI(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/profiles/demo/swaps" && r.Method == http.MethodPost {
			called = true
		}
		io.WriteString(w, `{}`)
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))
	if err := cmd.Run([]string{"apply", "demo", "from", "to"}); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if !called {
		t.Fatalf("expected swap apply request to be sent")
	}
}

func testAPIClient(base string) *cliutil.APIClient {
	return cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: cliutil.APIBaseOptions{DefaultBase: base}}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: base} },
		func() string { return "" },
	)
}
