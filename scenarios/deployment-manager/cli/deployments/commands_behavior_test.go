package deployments

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestDeploymentCommandsExerciseSupportedRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/deploy/profile-1":
			_, _ = io.WriteString(w, `{"status":"refused","reason":"dispatcher_not_implemented"}`)
		case "/api/v1/deployments/deploy-1":
			_, _ = io.WriteString(w, `{"status":"not_found"}`)
		case "/api/v1/profiles/profile-1/validate", "/api/v1/profiles/profile-1/cost-estimate":
			_, _ = io.WriteString(w, `{}`)
		case "/api/v1/logs/profile-1":
			_, _ = io.WriteString(w, `[{"timestamp":"now","level":"info","message":"ok"}]`)
		case "/api/v1/build":
			_, _ = io.WriteString(w, `{"status":"success","scenario":"demo","results":[{"service_id":"api","all_succeeded":true,"results":[{"platform":"linux-x64","success":true,"output_path":"bin/api"}]}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	cmd := New(cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: cliutil.APIBaseOptions{DefaultBase: server.URL}}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} },
		func() string { return "" },
	))
	for name, call := range map[string]func() error{
		"deploy":     func() error { return cmd.Deploy([]string{"profile-1", "--dry-run", "--async"}) },
		"status":     func() error { return cmd.Deployment([]string{"status", "deploy-1"}) },
		"validate":   func() error { return cmd.Validate([]string{"profile-1", "--verbose"}) },
		"cost":       func() error { return cmd.EstimateCost([]string{"profile-1"}) },
		"logs json":  func() error { return cmd.Logs([]string{"profile-1", "--level", "info", "--search", "ok"}) },
		"logs table": func() error { return cmd.Logs([]string{"profile-1", "--format", "table"}) },
		"build": func() error {
			return cmd.Build([]string{"--scenario", "demo", "--platforms", "linux-x64", "--services", "api"})
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		})
	}
	if err := cmd.Deploy([]string{}); err == nil || !strings.Contains(err.Error(), "profile id") {
		t.Fatal("missing deploy profile should fail")
	}
	if err := cmd.Deployment([]string{"unknown"}); err == nil {
		t.Fatal("unknown deployment command should fail")
	}
	if err := printLogsTable([]byte("bad")); err == nil {
		t.Fatal("invalid log JSON should fail")
	}
}
