package validation

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateScenarioReportsRuntimeHygieneDefects(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), compliantMain("api-app"))
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "handlers", "runtime.go"), `package handlers

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://example.test", nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("request failed: %v", err)
		return
	}
	_, _ = io.ReadAll(resp.Body)
	_ = context.Background()
	go func() {
		for {
			time.Sleep(time.Second)
		}
	}()
}
`)
	svc := New(Deps{RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeHTTPClientUnbounded)
	requireFinding(t, report, CodeResponseBodyUnclosed)
	requireFinding(t, report, CodeRequestContextDrop)
	requireFinding(t, report, CodeGoroutineUncancelled)
	requireFinding(t, report, CodeUnstructuredLogging)
	require.NotEmpty(t, report.Target.Runtime.InspectedFiles)
	require.NotEmpty(t, report.Target.Runtime.Signals)
}

func TestValidateScenarioAllowsBoundedRuntimePatterns(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), compliantMain("api-app"))
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "handlers", "runtime.go"), `package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://example.test", nil)
	resp, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	slog.InfoContext(ctx, "outbound request complete", "path", r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}(ctx)
}
`)
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "handlers", "runtime_test.go"), `package handlers

import (
	"net/http"
)

func testOnly() {
	_, _ = http.Get("https://example.test")
}
`)
	svc := New(Deps{RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.True(t, report.Passed, "findings: %#v", report.Findings)
	require.Empty(t, report.Findings)
	require.NotEmpty(t, report.Target.Runtime.InspectedFiles)
	require.NotEmpty(t, report.Target.Runtime.Signals)
}
