package runlocal

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestParseArgsParsesRepeatableSelectors(t *testing.T) {
	parsed, err := ParseArgs([]string{
		"demo",
		"--type", "phased",
		"--path", "api/foo.go",
		"--path", "ui/src/App.tsx",
		"--playbook", "bas/one.json",
		"--playbook", "bas/two.json",
		"--filter", "UserFlow",
		"--json",
	})
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}

	if parsed.Scenario != "demo" || parsed.Type != "phased" || parsed.Filter != "UserFlow" {
		t.Fatalf("unexpected parsed args: %+v", parsed)
	}
	if !reflect.DeepEqual(parsed.Paths, []string{"api/foo.go", "ui/src/App.tsx"}) {
		t.Fatalf("expected parsed paths, got %v", parsed.Paths)
	}
	if !reflect.DeepEqual(parsed.Playbooks, []string{"bas/one.json", "bas/two.json"}) {
		t.Fatalf("expected parsed playbooks, got %v", parsed.Playbooks)
	}
	if !parsed.JSON {
		t.Fatal("expected json output to be enabled")
	}
}

func TestParseArgsRejectsMissingScenario(t *testing.T) {
	_, err := ParseArgs(nil)
	if err == nil {
		t.Fatal("expected missing scenario to fail")
	}
	if !strings.Contains(err.Error(), "usage: run-tests") {
		t.Fatalf("expected usage error, got %v", err)
	}
}

func TestMultiFlagSetIgnoresBlankValues(t *testing.T) {
	var values multiFlag
	if err := values.Set(" first "); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if err := values.Set("   "); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if err := values.Set("second"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if !reflect.DeepEqual([]string(values), []string{"first", "second"}) {
		t.Fatalf("expected trimmed values, got %v", []string(values))
	}
}

func TestRunUsesFallbackAndPreservesSelectedResponseType(t *testing.T) {
	tests := []struct {
		name         string
		responseType string
		want         string
	}{
		{name: "blank fallback", responseType: " \t", want: "Scenario tests completed via unknown"},
		{name: "selected value remains unchanged", responseType: "  phased  ", want: "Scenario tests completed via   phased  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/scenarios/demo/run-tests" {
					t.Fatalf("request path = %q", r.URL.Path)
				}
				fmt.Fprintf(w, `{"type":%q}`, tt.responseType)
			}))
			defer server.Close()
			api := cliutil.NewAPIClient(
				cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
				func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} },
				func() string { return "" },
			)
			client := NewClient(api)
			output := captureStdout(t, func() {
				if err := Run(client, []string{"demo"}); err != nil {
					t.Errorf("Run() error = %v", err)
				}
			})
			if !strings.Contains(output, tt.want) {
				t.Fatalf("Run() output = %q, want it to contain %q", output, tt.want)
			}
		})
	}
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	original := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = original }()
	run()
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close stdout reader: %v", err)
	}
	return string(output)
}
