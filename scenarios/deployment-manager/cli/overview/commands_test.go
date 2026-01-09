package overview

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestAnalyzeRequiresScenario(t *testing.T) {
	cmd := New(dummyAPIClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	if err := cmd.Analyze([]string{}); err == nil || !strings.Contains(err.Error(), "scenario is required") {
		t.Fatalf("expected scenario required error, got %v", err)
	}
}

func TestFitnessPostsPayload(t *testing.T) {
	var gotPath string
	var method string
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		method = r.Method
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cmd := New(testAPIClient(server.URL))

	if err := cmd.Fitness([]string{"demo", "--tier", "3"}); err != nil {
		t.Fatalf("fitness failed: %v", err)
	}
	if gotPath != "/api/v1/fitness/score" || method != http.MethodPost {
		t.Fatalf("unexpected call: %s %s", method, gotPath)
	}
	if !strings.Contains(body, `"tiers":[3]`) {
		t.Fatalf("expected tier payload, got %s", body)
	}
}

func dummyAPIClient(t *testing.T, handler http.Handler) *cliutil.APIClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return testAPIClient(server.URL)
}

func testAPIClient(base string) *cliutil.APIClient {
	return cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{
			BaseOptions: cliutil.APIBaseOptions{DefaultBase: base},
		}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: base} },
		func() string { return "" },
	)
}
