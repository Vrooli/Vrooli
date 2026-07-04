package phases

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestRunListJSONUsesAPIPayload(t *testing.T) {
	api := testAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/phases" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"items":[{"name":"unit","provider":"unit-health","source":"validation-provider"}],"count":1}`)
	})

	var out bytes.Buffer
	if err := run(api, []string{"list", "--json"}, &out); err != nil {
		t.Fatalf("run list: %v", err)
	}
	if !strings.Contains(out.String(), `"provider":"unit-health"`) {
		t.Fatalf("expected raw JSON payload, got %s", out.String())
	}
}

func TestRunApplicabilityJSONPassesTargetAndPhase(t *testing.T) {
	api := testAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/phases/applicability" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("target"); got != "demo" {
			t.Fatalf("target query = %q, want demo", got)
		}
		if got := r.URL.Query().Get("phase"); got != "search" {
			t.Fatalf("phase query = %q, want search", got)
		}
		fmt.Fprint(w, `{"scenarioName":"demo","phase":{"name":"search","applicabilityStatus":"not_applicable"}}`)
	})

	var out bytes.Buffer
	if err := run(api, []string{"applicability", "demo", "--phase", "search", "--json"}, &out); err != nil {
		t.Fatalf("run applicability: %v", err)
	}
	if !strings.Contains(out.String(), `"applicabilityStatus":"not_applicable"`) {
		t.Fatalf("expected applicability JSON, got %s", out.String())
	}
}

func testAPIClient(t *testing.T, handler http.HandlerFunc) *cliutil.APIClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
}
