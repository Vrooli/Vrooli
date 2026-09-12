package validation

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateScenarioReportsHTTPSemanticsDefects(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), compliantMain("api-app"))
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "handlers", "users.go"), `package handlers

import (
	"encoding/json"
	"net/http"
)

func Register(r *http.ServeMux) {
	r.HandleFunc("/users", Users)
}

func Users(w http.ResponseWriter, r *http.Request) {
	if err := validate(r); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func validate(*http.Request) error { return nil }
`)
	svc := New(Deps{RepoRoot: root})
	report, err := svc.ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.False(t, report.Passed)
	requireFinding(t, report, CodeUnversionedEndpoint)
	requireFinding(t, report, CodeImplicitErrorSuccess)
	requireFinding(t, report, CodeRawStatusCode)
	requireFinding(t, report, CodeContentTypeMissing)
	require.NotEmpty(t, report.Target.HTTP.InspectedFiles)
	require.NotEmpty(t, report.Target.HTTP.Routes)
	require.NotEmpty(t, report.Target.HTTP.ResponsePatterns)
}

func TestValidateScenarioAllowsSupportedErrorFallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), compliantMain("api-app"))
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "handlers", "preview.go"), `package handlers

import (
	"errors"
	"net/http"
)

var errPreviewUnavailable = errors.New("preview unavailable")

func Preview(w http.ResponseWriter, err error) {
	if err != nil {
		if errors.Is(err, errPreviewUnavailable) {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
`)
	svc := New(Deps{RepoRoot: root})
	report, err := svc.ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	for _, finding := range report.Findings {
		if finding.Code == CodeImplicitErrorSuccess {
			t.Fatalf("supported fallback must not be reported as implicit success: %+v", finding)
		}
	}
}

func TestValidateScenarioAllowsVersionedAndExemptRoutes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "endpoints.json"), `{
	"endpoints":[
		{"path":"/health","method":"GET","category":"system","rest_exception":{"reason":"ops_probe"}},
		{"path":"/callback","method":"POST","category":"integration","rest_exception":{"reason":"external_webhook"}}
	]
}`)
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), compliantMain("api-app"))
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "handlers", "routes.go"), `package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func Register(r *mux.Router) {
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/users", Users).Methods(http.MethodGet)
	r.HandleFunc("/health", Health).Methods(http.MethodGet)
	r.HandleFunc("/callback", Callback).Methods(http.MethodPost)
}

func Users(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func Callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}
`)
	svc := New(Deps{RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.True(t, report.Passed, "findings: %#v", report.Findings)
	require.Empty(t, report.Findings)
	require.Len(t, report.Target.HTTP.Routes, 4)
}

func TestValidateScenarioSkipsTestsAndFrameworkOwnedResponses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scenarios", "api-app", ".vrooli", "service.json"), validServiceJSON())
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "main.go"), compliantMain("api-app"))
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "handlers", "framework.go"), `package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GinHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
`)
	writeFile(t, filepath.Join(root, "scenarios", "api-app", "api", "handlers", "framework_test.go"), `package handlers

import (
	"encoding/json"
	"net/http"
)

func fixture(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"test": "only"})
}
`)
	svc := New(Deps{RepoRoot: root})

	report, err := svc.ValidateScenario(context.Background(), "api-app", "", false)
	require.NoError(t, err)
	require.True(t, report.Passed)
	require.Empty(t, report.Findings)
}
