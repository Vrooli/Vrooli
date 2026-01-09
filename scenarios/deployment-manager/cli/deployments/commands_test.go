package deployments

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
		io.WriteString(w, `{}`)
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

func TestPackageRequiresPackager(t *testing.T) {
	cmd := New(nil)
	if err := cmd.PackageProfile([]string{"demo"}); err == nil || !strings.Contains(err.Error(), "--packager is required") {
		t.Fatalf("expected packager requirement, got %v", err)
	}
}

func TestLogsAcceptsFilters(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		io.WriteString(w, `{}`)
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

func TestPackagersStubOutputsMessage(t *testing.T) {
	cmd := New(nil)
	output := captureOutput(t, func() {
		if err := cmd.Packagers([]string{"discover"}); err != nil {
			t.Fatalf("packagers stub failed: %v", err)
		}
	})
	if !strings.Contains(strings.ToLower(output), "packager") {
		t.Fatalf("expected packager message, got %s", output)
	}
}

func TestPackageStubbedResponse(t *testing.T) {
	cmd := New(nil)
	output := captureOutput(t, func() {
		if err := cmd.PackageProfile([]string{"demo", "--packager", "scenario-to-desktop", "--dry-run"}); err != nil {
			t.Fatalf("package stub failed: %v", err)
		}
	})
	if !strings.Contains(strings.ToLower(output), "deploy-desktop") && !strings.Contains(strings.ToLower(output), "stub") {
		t.Fatalf("expected stub messaging, got %s", output)
	}
}

func testAPIClient(base string) *cliutil.APIClient {
	return cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: cliutil.APIBaseOptions{DefaultBase: base}}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: base} },
		func() string { return "" },
	)
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	_ = r.Close()
	return buf.String()
}
