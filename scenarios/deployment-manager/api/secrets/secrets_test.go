package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"deployment-manager/shared"

	"github.com/gorilla/mux"
)

// --- Mock ProfileLookup ---

type mockProfileLookup struct {
	scenario  string
	tierCount int
	err       error
}

func (m *mockProfileLookup) GetScenarioAndTier(_ context.Context, _ string) (string, int, error) {
	return m.scenario, m.tierCount, m.err
}

// --- Mock TimeProvider ---

type fixedTime struct {
	t time.Time
}

func (f fixedTime) Now() time.Time { return f.t }

// --- Helpers ---

func newTestHandler(profiles ProfileLookup) *Handler {
	return NewHandler(profiles, func(string, map[string]interface{}) {})
}

func makeRequest(method, url string) *http.Request {
	return httptest.NewRequest(method, url, nil)
}

func setRouteVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	return result
}

// --- Pure function tests: GenerateTemplate ---

func TestGenerateTemplate(t *testing.T) {
	params := TemplateParams{
		ProfileID: "prof-123",
		Scenario:  "my-scenario",
		TierName:  "Tier 2 - Desktop",
	}

	tests := []struct {
		name      string
		format    TemplateFormat
		wantErr   bool
		errSubstr string
		checks    func(t *testing.T, out string)
	}{
		{
			name:   "env format contains expected placeholders",
			format: TemplateFormatEnv,
			checks: func(t *testing.T, out string) {
				for _, want := range []string{
					"DATABASE_URL=",
					"API_KEY=",
					"DEBUG_MODE=false",
					"prof-123",
					"my-scenario",
					"Tier 2 - Desktop",
				} {
					if !strings.Contains(out, want) {
						t.Errorf("env template missing %q", want)
					}
				}
			},
		},
		{
			name:   "vault format is valid JSON with paths",
			format: TemplateFormatVault,
			checks: func(t *testing.T, out string) {
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(out), &parsed); err != nil {
					t.Fatalf("vault template is not valid JSON: %v", err)
				}
				secrets, ok := parsed["secrets"].([]interface{})
				if !ok || len(secrets) == 0 {
					t.Fatal("vault template missing secrets array")
				}
				first := secrets[0].(map[string]interface{})
				path, _ := first["path"].(string)
				if !strings.Contains(path, "secret/data/deployment-manager/prof-123") {
					t.Errorf("vault path missing profile ID, got: %s", path)
				}
			},
		},
		{
			name:   "aws format is valid JSON with ARNs",
			format: TemplateFormatAWS,
			checks: func(t *testing.T, out string) {
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(out), &parsed); err != nil {
					t.Fatalf("aws template is not valid JSON: %v", err)
				}
				secrets, ok := parsed["secrets"].([]interface{})
				if !ok || len(secrets) == 0 {
					t.Fatal("aws template missing secrets array")
				}
				first := secrets[0].(map[string]interface{})
				arn, _ := first["arn"].(string)
				if !strings.HasPrefix(arn, "arn:aws:secretsmanager:") {
					t.Errorf("expected ARN prefix, got: %s", arn)
				}
				if !strings.Contains(arn, "prof-123") {
					t.Errorf("ARN missing profile ID, got: %s", arn)
				}
			},
		},
		{
			name:      "unknown format returns error",
			format:    TemplateFormat("yaml"),
			wantErr:   true,
			errSubstr: "unsupported format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := GenerateTemplate(tt.format, params)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.checks != nil {
				tt.checks(t, out)
			}
		})
	}
}

// --- Pure function tests: IsEnvFormat ---

func TestIsEnvFormat(t *testing.T) {
	tests := []struct {
		format TemplateFormat
		want   bool
	}{
		{TemplateFormatEnv, true},
		{TemplateFormatVault, false},
		{TemplateFormatAWS, false},
		{TemplateFormat("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			if got := IsEnvFormat(tt.format); got != tt.want {
				t.Errorf("IsEnvFormat(%q) = %v, want %v", tt.format, got, tt.want)
			}
		})
	}
}

// --- Handler tests: IdentifySecrets ---

func TestIdentifySecrets_ProfileFound(t *testing.T) {
	fixedNow := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	shared.SetTimeProvider(fixedTime{fixedNow})
	defer shared.SetTimeProvider(shared.RealTimeProvider{})

	h := newTestHandler(&mockProfileLookup{scenario: "web-app", tierCount: 3})
	req := setRouteVars(makeRequest(http.MethodGet, "/api/v1/secrets/prof-1/identify"), map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.IdentifySecrets(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := decodeJSON(t, rec)

	if body["profile_id"] != "prof-1" {
		t.Errorf("profile_id = %v, want prof-1", body["profile_id"])
	}
	if body["scenario"] != "web-app" {
		t.Errorf("scenario = %v, want web-app", body["scenario"])
	}
	count, ok := body["count"].(float64)
	if !ok || count != 3 {
		t.Errorf("count = %v, want 3", body["count"])
	}
	secrets, ok := body["secrets"].([]interface{})
	if !ok || len(secrets) != 3 {
		t.Fatalf("expected 3 secrets, got %v", body["secrets"])
	}
	if body["timestamp"] != "2025-06-15T12:00:00Z" {
		t.Errorf("timestamp = %v, want 2025-06-15T12:00:00Z", body["timestamp"])
	}

	// Verify the secrets contain expected names
	names := make(map[string]bool)
	for _, s := range secrets {
		entry := s.(map[string]interface{})
		names[entry["name"].(string)] = true
	}
	for _, expected := range []string{"DATABASE_URL", "API_KEY", "DEBUG_MODE"} {
		if !names[expected] {
			t.Errorf("missing secret %q in response", expected)
		}
	}
}

func TestIdentifySecrets_ProfileNotFound(t *testing.T) {
	h := newTestHandler(&mockProfileLookup{err: ErrProfileNotFound})
	req := setRouteVars(makeRequest(http.MethodGet, "/api/v1/secrets/missing/identify"), map[string]string{"id": "missing"})
	rec := httptest.NewRecorder()

	h.IdentifySecrets(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "profile not found") {
		t.Errorf("expected 'profile not found' error, got: %s", rec.Body.String())
	}
}

func TestIdentifySecrets_DatabaseError(t *testing.T) {
	h := newTestHandler(&mockProfileLookup{err: errors.New("connection refused")})
	req := setRouteVars(makeRequest(http.MethodGet, "/api/v1/secrets/prof-1/identify"), map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.IdentifySecrets(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "database error") {
		t.Errorf("expected 'database error' message, got: %s", rec.Body.String())
	}
}

// --- Handler tests: GenerateSecretTemplate ---

func TestGenerateSecretTemplate_EnvFormat(t *testing.T) {
	fixedNow := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	shared.SetTimeProvider(fixedTime{fixedNow})
	defer shared.SetTimeProvider(shared.RealTimeProvider{})

	h := newTestHandler(&mockProfileLookup{scenario: "my-app", tierCount: 2})
	req := setRouteVars(makeRequest(http.MethodGet, "/api/v1/secrets/prof-1/template?format=env"), map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.GenerateSecretTemplate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := decodeJSON(t, rec)

	if body["format"] != "env" {
		t.Errorf("format = %v, want env", body["format"])
	}
	if body["profile_id"] != "prof-1" {
		t.Errorf("profile_id = %v, want prof-1", body["profile_id"])
	}
	template, ok := body["template"].(string)
	if !ok {
		t.Fatal("template field missing or not a string")
	}
	if !strings.Contains(template, "DATABASE_URL=") {
		t.Errorf("env template missing DATABASE_URL placeholder")
	}
	if body["timestamp"] != "2025-06-15T12:00:00Z" {
		t.Errorf("timestamp = %v, want 2025-06-15T12:00:00Z", body["timestamp"])
	}
}

func TestGenerateSecretTemplate_VaultFormat(t *testing.T) {
	h := newTestHandler(&mockProfileLookup{scenario: "my-app", tierCount: 2})
	req := setRouteVars(makeRequest(http.MethodGet, "/api/v1/secrets/prof-1/template?format=vault"), map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.GenerateSecretTemplate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Vault format is written directly as JSON (not wrapped)
	var parsed map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&parsed); err != nil {
		t.Fatalf("vault response is not valid JSON: %v", err)
	}
	secrets, ok := parsed["secrets"].([]interface{})
	if !ok || len(secrets) == 0 {
		t.Fatal("vault response missing secrets array")
	}
	first := secrets[0].(map[string]interface{})
	if _, ok := first["path"]; !ok {
		t.Error("vault secret missing path field")
	}
}

func TestGenerateSecretTemplate_AWSFormat(t *testing.T) {
	h := newTestHandler(&mockProfileLookup{scenario: "my-app", tierCount: 2})
	req := setRouteVars(makeRequest(http.MethodGet, "/api/v1/secrets/prof-1/template?format=aws"), map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.GenerateSecretTemplate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// AWS format is written directly as JSON (not wrapped)
	var parsed map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&parsed); err != nil {
		t.Fatalf("aws response is not valid JSON: %v", err)
	}
	secrets, ok := parsed["secrets"].([]interface{})
	if !ok || len(secrets) == 0 {
		t.Fatal("aws response missing secrets array")
	}
	first := secrets[0].(map[string]interface{})
	arn, _ := first["arn"].(string)
	if !strings.HasPrefix(arn, "arn:aws:secretsmanager:") {
		t.Errorf("expected ARN prefix, got: %s", arn)
	}
}

func TestGenerateSecretTemplate_DefaultsToEnv(t *testing.T) {
	h := newTestHandler(&mockProfileLookup{scenario: "my-app", tierCount: 2})
	// No format query param
	req := setRouteVars(makeRequest(http.MethodGet, "/api/v1/secrets/prof-1/template"), map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.GenerateSecretTemplate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := decodeJSON(t, rec)
	if body["format"] != "env" {
		t.Errorf("expected default format env, got %v", body["format"])
	}
	template, ok := body["template"].(string)
	if !ok {
		t.Fatal("template field missing or not a string")
	}
	if !strings.Contains(template, "DATABASE_URL=") {
		t.Error("default env template missing expected placeholders")
	}
}

func TestGenerateSecretTemplate_UnsupportedFormat(t *testing.T) {
	h := newTestHandler(&mockProfileLookup{scenario: "my-app", tierCount: 2})
	req := setRouteVars(makeRequest(http.MethodGet, "/api/v1/secrets/prof-1/template?format=yaml"), map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.GenerateSecretTemplate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "unsupported format") {
		t.Errorf("expected 'unsupported format' error, got: %s", rec.Body.String())
	}
}

func TestGenerateSecretTemplate_ProfileNotFound(t *testing.T) {
	h := newTestHandler(&mockProfileLookup{err: ErrProfileNotFound})
	req := setRouteVars(makeRequest(http.MethodGet, "/api/v1/secrets/missing/template?format=env"), map[string]string{"id": "missing"})
	rec := httptest.NewRecorder()

	h.GenerateSecretTemplate(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "profile not found") {
		t.Errorf("expected 'profile not found' error, got: %s", rec.Body.String())
	}
}

func TestGenerateSecretTemplate_DatabaseError(t *testing.T) {
	h := newTestHandler(&mockProfileLookup{err: errors.New("timeout")})
	req := setRouteVars(makeRequest(http.MethodGet, "/api/v1/secrets/prof-1/template?format=env"), map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.GenerateSecretTemplate(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "database error") {
		t.Errorf("expected 'database error' message, got: %s", rec.Body.String())
	}
}

// --- Handler tests: ValidateSecrets ---

func TestValidateSecrets(t *testing.T) {
	fixedNow := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	shared.SetTimeProvider(fixedTime{fixedNow})
	defer shared.SetTimeProvider(shared.RealTimeProvider{})

	h := newTestHandler(&mockProfileLookup{scenario: "web-app", tierCount: 2})
	req := setRouteVars(makeRequest(http.MethodPost, "/api/v1/secrets/prof-1/validate"), map[string]string{"id": "prof-1"})
	rec := httptest.NewRecorder()

	h.ValidateSecrets(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := decodeJSON(t, rec)

	if body["profile_id"] != "prof-1" {
		t.Errorf("profile_id = %v, want prof-1", body["profile_id"])
	}
	if body["status"] != "pass" {
		t.Errorf("status = %v, want pass", body["status"])
	}
	tests, ok := body["tests"].([]interface{})
	if !ok || len(tests) == 0 {
		t.Fatal("expected non-empty tests array")
	}
	// Verify first test entry
	first := tests[0].(map[string]interface{})
	if first["secret"] != "DATABASE_URL" {
		t.Errorf("first test secret = %v, want DATABASE_URL", first["secret"])
	}
	if first["status"] != "pass" {
		t.Errorf("first test status = %v, want pass", first["status"])
	}
	if body["timestamp"] != "2025-06-15T12:00:00Z" {
		t.Errorf("timestamp = %v, want 2025-06-15T12:00:00Z", body["timestamp"])
	}
}

// --- Handler tests: ValidateSecret ---

func TestValidateSecret(t *testing.T) {
	fixedNow := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	shared.SetTimeProvider(fixedTime{fixedNow})
	defer shared.SetTimeProvider(shared.RealTimeProvider{})

	h := newTestHandler(&mockProfileLookup{scenario: "web-app", tierCount: 2})
	req := makeRequest(http.MethodPost, "/api/v1/secrets/validate")
	rec := httptest.NewRecorder()

	h.ValidateSecret(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := decodeJSON(t, rec)

	if body["status"] != "pass" {
		t.Errorf("status = %v, want pass", body["status"])
	}
	if body["message"] != "Secret validation not yet implemented" {
		t.Errorf("unexpected message: %v", body["message"])
	}
	if body["timestamp"] != "2025-06-15T12:00:00Z" {
		t.Errorf("timestamp = %v, want 2025-06-15T12:00:00Z", body["timestamp"])
	}
}

// --- Handler tests: TestSecret ---

func TestTestSecret(t *testing.T) {
	fixedNow := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	shared.SetTimeProvider(fixedTime{fixedNow})
	defer shared.SetTimeProvider(shared.RealTimeProvider{})

	h := newTestHandler(&mockProfileLookup{scenario: "web-app", tierCount: 2})
	req := makeRequest(http.MethodPost, "/api/v1/secrets/test")
	rec := httptest.NewRecorder()

	h.TestSecret(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := decodeJSON(t, rec)

	if body["status"] != "available" {
		t.Errorf("status = %v, want available", body["status"])
	}
	if body["message"] != "Secret testing endpoint ready" {
		t.Errorf("unexpected message: %v", body["message"])
	}
	if body["timestamp"] != "2025-06-15T12:00:00Z" {
		t.Errorf("timestamp = %v, want 2025-06-15T12:00:00Z", body["timestamp"])
	}
}

// --- Handler tests: Content-Type header ---

func TestHandlersSetJSONContentType(t *testing.T) {
	h := newTestHandler(&mockProfileLookup{scenario: "app", tierCount: 1})

	tests := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		setup   func(r *http.Request) *http.Request
	}{
		{
			name:    "IdentifySecrets",
			handler: h.IdentifySecrets,
			setup:   func(r *http.Request) *http.Request { return setRouteVars(r, map[string]string{"id": "p1"}) },
		},
		{
			name:    "GenerateSecretTemplate env",
			handler: h.GenerateSecretTemplate,
			setup: func(r *http.Request) *http.Request {
				r = setRouteVars(r, map[string]string{"id": "p1"})
				return r
			},
		},
		{
			name:    "ValidateSecrets",
			handler: h.ValidateSecrets,
			setup:   func(r *http.Request) *http.Request { return setRouteVars(r, map[string]string{"id": "p1"}) },
		},
		{
			name:    "ValidateSecret",
			handler: h.ValidateSecret,
			setup:   func(r *http.Request) *http.Request { return r },
		},
		{
			name:    "TestSecret",
			handler: h.TestSecret,
			setup:   func(r *http.Request) *http.Request { return r },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setup(makeRequest(http.MethodGet, "/test"))
			rec := httptest.NewRecorder()
			tt.handler(rec, req)

			ct := rec.Header().Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}

// --- NewHandler ---

func TestNewHandler(t *testing.T) {
	mock := &mockProfileLookup{scenario: "test", tierCount: 1}
	logged := false
	logFn := func(msg string, fields map[string]interface{}) { logged = true }

	h := NewHandler(mock, logFn)

	if h.profiles != mock {
		t.Error("profiles field not set correctly")
	}
	if h.log == nil {
		t.Error("log field is nil")
	}
	h.log("test", nil)
	if !logged {
		t.Error("log function not called")
	}
}
