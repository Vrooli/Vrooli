package status

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestClientCheckReadsHealthStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"status":"ok","service":"test-genie","version":"1.0.0"}`)
	}))
	defer server.Close()

	api := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{DefaultBase: server.URL}
		},
		func() string { return "" },
	)

	body, resp, err := NewClient(api).Check()
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if string(body) == "" {
		t.Fatal("expected response body to be returned")
	}
	if resp.Status != "ok" || resp.Service != "test-genie" {
		t.Fatalf("expected parsed health response, got %+v", resp)
	}
}

func TestRunUsesFallbackAndPreservesSelectedStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{name: "blank fallback", status: " \t", want: "Service health: unknown"},
		{name: "selected value remains unchanged", status: "  healthy  ", want: "Service health:   healthy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"status":%q}`, tt.status)
			}))
			defer server.Close()
			api := cliutil.NewAPIClient(
				cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
				func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} },
				func() string { return "" },
			)
			output := captureStdout(t, func() {
				if err := Run(NewClient(api)); err != nil {
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
