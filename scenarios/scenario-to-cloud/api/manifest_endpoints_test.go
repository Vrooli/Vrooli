package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"scenario-to-cloud/manifest"
)

func validManifestPayload() map[string]interface{} {
	return map[string]interface{}{
		"version": "1.0.0",
		"target": map[string]interface{}{
			"type": "vps",
			"vps": map[string]interface{}{
				"host": "203.0.113.10",
			},
		},
		"scenario": map[string]interface{}{
			"id": "landing-page-business-suite",
		},
		"dependencies": map[string]interface{}{
			"scenarios": []string{"landing-page-business-suite"},
			"resources": []string{},
			"analyzer": map[string]interface{}{
				"tool": "scenario-dependency-analyzer",
			},
		},
		"bundle": map[string]interface{}{
			"include_packages": true,
			"include_autoheal": true,
			"scenarios":        []string{"landing-page-business-suite", "vrooli-autoheal"},
			"resources":        []string{},
		},
		"ports": map[string]interface{}{
			"ui":  float64(3000),
			"api": float64(3001),
			"ws":  float64(3002),
		},
		"edge": map[string]interface{}{
			"domain": "example.com",
			"caddy": map[string]interface{}{
				"enabled": true,
			},
		},
	}
}

func postJSONToServer(t *testing.T, ts *httptest.Server, path string, payload any) (*http.Response, []byte) {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	resp, err := http.Post(ts.URL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, _ := ioReadAll(resp)
	return resp, body
}

func ioReadAll(resp *http.Response) ([]byte, error) {
	var out bytes.Buffer
	_, err := out.ReadFrom(resp.Body)
	return out.Bytes(), err
}

func TestManifestSchemaEndpoint_ReturnsSchema(t *testing.T) {
	t.Setenv("API_PORT", "0")
	srv := newTestServer()
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/manifest/schema")
	if err != nil {
		t.Fatalf("get schema: %v", err)
	}
	defer resp.Body.Close()
	body, _ := ioReadAll(resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("schema status=%d body=%s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Schema map[string]interface{} `json:"schema"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal schema response: %v", err)
	}
	if parsed.Schema["type"] != "object" {
		t.Fatalf("expected object schema, got %v", parsed.Schema["type"])
	}
	if _, ok := parsed.Schema["properties"].(map[string]interface{}); !ok {
		t.Fatalf("expected properties in schema")
	}
}

func TestManifestInitMatchesSchemaAndValidates(t *testing.T) {
	t.Setenv("API_PORT", "0")
	srv := newTestServer()
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	resp, body := postJSONToServer(t, ts, "/api/v1/manifest/init", map[string]interface{}{
		"scenario_id": "landing-page-business-suite",
		"host":        "203.0.113.10",
		"domain":      "example.com",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("init status=%d body=%s", resp.StatusCode, string(body))
	}

	var initResp struct {
		Manifest map[string]interface{} `json:"manifest"`
	}
	if err := json.Unmarshal(body, &initResp); err != nil {
		t.Fatalf("unmarshal init response: %v", err)
	}
	if initResp.Manifest == nil {
		t.Fatalf("expected manifest in init response")
	}

	if issues := manifest.ValidateStructure(initResp.Manifest); len(issues) != 0 {
		t.Fatalf("expected init manifest to match schema, got issues: %+v", issues)
	}

	resp, body = postJSONToServer(t, ts, "/api/v1/manifest/validate", initResp.Manifest)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("validate status=%d body=%s", resp.StatusCode, string(body))
	}

	var validateResp struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(body, &validateResp); err != nil {
		t.Fatalf("unmarshal validate response: %v", err)
	}
	if !validateResp.Valid {
		t.Fatalf("expected init manifest to be valid, got body=%s", string(body))
	}
}

func TestManifestValidateRejectsUnknownFieldFromSchema(t *testing.T) {
	t.Setenv("API_PORT", "0")
	srv := newTestServer()
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	payload := validManifestPayload()
	payload["options"] = map[string]interface{}{"autoheal": true}

	resp, body := postJSONToServer(t, ts, "/api/v1/manifest/validate", payload)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("validate status=%d body=%s", resp.StatusCode, string(body))
	}

	var validateResp struct {
		Valid  bool `json:"valid"`
		Issues []struct {
			Path     string `json:"path"`
			Severity string `json:"severity"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(body, &validateResp); err != nil {
		t.Fatalf("unmarshal validate response: %v", err)
	}
	if validateResp.Valid {
		t.Fatalf("expected invalid manifest when unknown field exists")
	}
	found := false
	for _, issue := range validateResp.Issues {
		if issue.Path == "options" && issue.Severity == "error" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected options structural error, got %+v", validateResp.Issues)
	}
}

func TestManifestFixNormalizesDefaults(t *testing.T) {
	t.Setenv("API_PORT", "0")
	srv := newTestServer()
	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	payload := validManifestPayload()
	payload["bundle"] = map[string]interface{}{
		"include_packages": true,
		"include_autoheal": true,
		"scenarios":        []string{"landing-page-business-suite"},
	}

	resp, body := postJSONToServer(t, ts, "/api/v1/manifest/fix", map[string]interface{}{
		"manifest": payload,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fix status=%d body=%s", resp.StatusCode, string(body))
	}

	var fixResp struct {
		Manifest map[string]interface{} `json:"manifest"`
	}
	if err := json.Unmarshal(body, &fixResp); err != nil {
		t.Fatalf("unmarshal fix response: %v", err)
	}

	target := fixResp.Manifest["target"].(map[string]interface{})
	vps := target["vps"].(map[string]interface{})
	if vps["user"] != "root" {
		t.Fatalf("expected default user root, got %v", vps["user"])
	}
	if vps["port"] != float64(22) {
		t.Fatalf("expected default port 22, got %v", vps["port"])
	}

	bundleMap := fixResp.Manifest["bundle"].(map[string]interface{})
	scenarios := bundleMap["scenarios"].([]interface{})
	hasAutoheal := false
	for _, scenario := range scenarios {
		if scenario == "vrooli-autoheal" {
			hasAutoheal = true
			break
		}
	}
	if !hasAutoheal {
		t.Fatalf("expected vrooli-autoheal to be present in fixed bundle.scenarios")
	}
}

func TestGetDocsDirPrefersContractScenarioDocs(t *testing.T) {
	repoRoot := t.TempDir()
	writeRepoContractFixture(t, repoRoot)
	if err := os.MkdirAll(filepath.Join(repoRoot, "scenarios", "scenario-to-cloud", ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir scenario config dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "scenarios", "scenario-to-cloud", "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(repoRoot, "scenarios", "scenario-to-cloud", ".vrooli", "service.json"),
		[]byte(`{"service":{"name":"scenario-to-cloud"}}`),
		0o644,
	); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(repoRoot, "scenarios", "scenario-to-cloud", "docs", "manifest.json"),
		[]byte(`{"version":"1.0.0","title":"Docs","defaultDocument":"overview.md","sections":[]}`),
		0o644,
	); err != nil {
		t.Fatalf("write docs manifest: %v", err)
	}

	t.Setenv("SCENARIO_TO_CLOUD_REPO_ROOT", repoRoot)
	t.Setenv("SCENARIO_TO_CLOUD_DOCS_DIR", "")

	srv := newTestServer()
	if got, want := srv.getDocsDir(), filepath.Join(repoRoot, "scenarios", "scenario-to-cloud", "docs"); got != want {
		t.Fatalf("getDocsDir = %q, want %q", got, want)
	}
}
