package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestRunContinueHumanOutputDistinguishesInteractiveTransport(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	t.Setenv("TMPDIR", t.TempDir()) // isolate the process-scoped identity broker socket
	const runID = "8f7813d8-66f4-4c8a-9a53-63a28f7d3c44"
	for _, tc := range []struct {
		name, mode, want, unwanted string
	}{
		{"interactive transport", "EXECUTION_MODE_INTERACTIVE", "Transport accepted into running", "Continuation started for run"},
		{"codec pipe continuation", "EXECUTION_MODE_CODEC_PIPE", "Continuation started for run", "Transport accepted into running interactive session"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/runs/"+runID+"/continue" {
					t.Errorf("unexpected continuation request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":true,"run":{"id":"` + runID + `","status":"RUN_STATUS_RUNNING","execution_mode":"` + tc.mode + `"},"transport_disposition":"` + map[string]string{"EXECUTION_MODE_INTERACTIVE":"accepted","EXECUTION_MODE_CODEC_PIPE":"continuation_started"}[tc.mode] + `","consumption_disposition":"unknown"}`))
			}))
			defer server.Close()

			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
				return cliutil.APIBaseOptions{DefaultBase: server.URL}
			}, nil)
			app.services = NewServices(api)

			output := captureStdout(t, func() error {
				return app.runContinue([]string{runID, "--message", "resume"})
			})
			if !strings.Contains(output, tc.want) || strings.Contains(output, tc.unwanted) {
				t.Fatalf("output = %q; want %q and not %q", output, tc.want, tc.unwanted)
			}
			if !strings.Contains(output, "provider consumption unknown") {
				t.Fatalf("output must preserve consumption uncertainty: %q", output)
			}
		})
	}
}
