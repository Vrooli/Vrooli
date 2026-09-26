package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func TestMaintenanceCLIUsesOwnerProtocol(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer owner-fixture" {
			t.Error("owner credential was lost")
		}
		if strings.HasSuffix(r.URL.Path, "resume") {
			var input map[string]any
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input["revision"] != float64(7) {
				t.Error("resume revision was lost", input, err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"closed":true,"revision":7,"admitting":0,"remaining":0,"drained":true}`))
	}))
	defer server.Close()
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} }, func() string { return "owner-fixture" })
	app := &App{services: NewServices(api)}
	for _, args := range [][]string{{"status", "--json"}, {"begin", "--reason", "planned rollout", "--json"}, {"drain", "--timeout", "2s", "--json"}, {"resume", "--revision", "7", "--json"}} {
		if err := app.cmdMaintenance(args); err != nil {
			t.Fatal(args, err)
		}
	}
	want := []string{"GET /api/v1/maintenance/admission", "POST /api/v1/maintenance/admission/enter", "POST /api/v1/maintenance/admission/wait", "POST /api/v1/maintenance/admission/resume"}
	if strings.Join(paths, "\n") != strings.Join(want, "\n") {
		t.Fatal(paths)
	}
}

func TestMaintenanceCLIPreservesTimeoutStandingAndDoesNotRetry(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		select {
		case <-time.After(40 * time.Millisecond):
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusRequestTimeout)
		_, _ = w.Write([]byte(`{"state":{"closed":true,"revision":8,"remaining":1,"drained":false},"error":"context deadline exceeded"}`))
	}))
	defer server.Close()
	base := cliutil.NewHTTPClient(cliutil.HTTPClientOptions{Timeout: time.Millisecond})
	api := cliutil.NewAPIClient(base, func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} }, nil)
	app := &App{services: NewServices(api)}
	err := app.cmdMaintenance([]string{"drain", "--timeout", "1s", "--json"})
	if err == nil || calls != 1 || base.Timeout() != time.Millisecond {
		t.Fatalf("timeout lost owner standing or retried: %v calls=%d", err, calls)
	}
}

func TestMaintenanceCLILocalOwnerCannotElevateAgentAndValidatesBeforeExchange(t *testing.T) {
	services, recorder := newContractServices(t)
	app := &App{services: services}
	t.Setenv(cliutil.EnvIdentityToken, "identified-agent-fixture")
	if err := app.cmdMaintenance([]string{"begin", "--reason", "rollout", "--local-owner"}); err == nil || !strings.Contains(err.Error(), "identified agent") {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"resume"}, {"begin"}, {"drain", "--timeout", "0s"}, {"status", "unexpected"}} {
		if err := app.cmdMaintenance(args); err == nil {
			t.Fatal("invalid maintenance arguments accepted", args)
		}
	}
	if len(recorder.Requests()) != 0 {
		t.Fatal("invalid request reached owner")
	}
}

func TestMaintenanceCLIStatusRetains503OwnerEvidence(t *testing.T) {
	const body = `{"state":{"closed":true,"revision":7,"admitting":0,"remaining":null,"drained":false,"inventory":{"remaining":null,"work":[],"executors":[{"pid":315576,"alive":true}],"unknown":["scope exclusion unavailable"]}},"error":"physical executor inventory is incomplete"}`
	for _, jsonOut := range []bool{false, true} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(body))
		}))
		api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} }, nil)
		app := &App{services: NewServices(api)}
		args := []string{"status"}
		if jsonOut {
			args = append(args, "--json")
		}
		var requestErr error
		output := captureStdout(t, func() error { requestErr = app.cmdMaintenance(args); return nil })
		server.Close()
		if requestErr == nil || !strings.Contains(requestErr.Error(), "physical executor inventory is incomplete") || calls != 1 {
			t.Fatalf("unknown inventory became success or retried: %v calls=%d", requestErr, calls)
		}
		if jsonOut {
			var envelope struct {
				State maintenanceStanding `json:"state"`
				Error string              `json:"error"`
			}
			if err := json.Unmarshal([]byte(output), &envelope); err != nil || !envelope.State.Closed || envelope.State.Revision != 7 || envelope.State.Remaining != nil || envelope.Error == "" {
				t.Fatalf("JSON owner evidence lost: %q err=%v", output, err)
			}
		} else if !strings.Contains(output, "closed=true revision=7") || !strings.Contains(output, "remaining=unobserved") {
			t.Fatalf("human owner evidence lost: %q", output)
		}
		if !strings.Contains(output, "315576") || !strings.Contains(output, "scope exclusion unavailable") {
			t.Fatalf("physical evidence lost: %q", output)
		}
	}
}

func TestMaintenanceCLIUnobservedErrorDoesNotInventOpenFence(t *testing.T) {
	for _, body := range []string{`{"error":"owner unavailable"}`, `{"state":null,"error":"lifecycle lock held"}`, `{"state":{},"error":"owner unavailable"}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(body))
		}))
		api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} }, nil)
		app := &App{services: NewServices(api)}
		var requestErr error
		output := captureStdout(t, func() error { requestErr = app.cmdMaintenance([]string{"status"}); return nil })
		server.Close()
		if requestErr == nil || strings.Contains(output, "closed=false") || strings.Contains(output, "revision=0") {
			t.Fatalf("unobserved fence fabricated: %q %v", output, requestErr)
		}
	}
}

func TestMaintenanceCLIEvidenceDecodeIsBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"state":{"closed":true,"revision":7},"error":"` + strings.Repeat("x", 256*1024) + `"}`))
	}))
	defer server.Close()
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} }, nil)
	app := &App{services: NewServices(api)}
	var requestErr error
	output := captureStdout(t, func() error { requestErr = app.cmdMaintenance([]string{"status", "--json"}); return nil })
	if requestErr == nil || !strings.Contains(requestErr.Error(), "evidence limit") || len(requestErr.Error()) > 512 || len(output) > 512 {
		t.Fatalf("unbounded evidence emitted: bytes=%d err-bytes=%d", len(output), len(fmt.Sprint(requestErr)))
	}
}

func TestMaintenanceCLIHelpShowsOwnerRequirements(t *testing.T) {
	for _, tc := range []struct {
		operation string
		want      []string
	}{
		{"status", []string{"agent-manager maintenance status [--json]"}},
		{"begin", []string{"agent-manager maintenance begin --reason", "1-512 bytes", "--local-owner"}},
		{"drain", []string{"agent-manager maintenance drain [--timeout", "1s to 120s", "--local-owner"}},
		{"resume", []string{"agent-manager maintenance resume --revision", "--local-owner"}},
	} {
		t.Run(tc.operation, func(t *testing.T) {
			output := captureStdout(t, func() error {
				app, err := NewApp()
				if err != nil {
					return err
				}
				return app.Run([]string{"maintenance", tc.operation, "--help"})
			})
			for _, want := range tc.want {
				if !strings.Contains(output, want) {
					t.Fatalf("help is missing %q: %s", want, output)
				}
			}
		})
	}
}
