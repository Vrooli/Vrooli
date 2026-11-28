package search

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func changeIntentFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	base := filepath.Join(root, "scenarios", "business-health")
	files := map[string]string{
		"requirements/index.json":          `{"imports":["01-test/module.json"]}`,
		"requirements/01-test/module.json": `{"requirements":[{"id":"BH-TEST-001","title":"Bounded test requirement","description":"Keep the source scan bounded.","status":"planned"}]}`,
		"api/contract.go":                  "package api\n// [REQ:BH-TEST-001]\n",
	}
	for name, content := range files {
		path := filepath.Join(base, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestChangeIntentHTTPPositiveAndNegative(t *testing.T) {
	handler := assessIntent(changeIntentFixture(t))
	t.Run("positive preserves scoped provenance and limitation", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"scenario": "business-health", "intent": "Change BH-TEST-001", "limit": 10})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/change-intent/assess", bytes.NewReader(body))
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
		}
		var decoded map[string]any
		if err := json.Unmarshal(res.Body.Bytes(), &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded["scenario"] != "business-health" {
			t.Fatalf("response scope=%v", decoded["scenario"])
		}
		results := decoded["results"].([]any)
		if len(results) != 1 || results[0].(map[string]any)["relation"] != "support" {
			t.Fatalf("response results=%v", results)
		}
		result := results[0].(map[string]any)
		if result["scenario"] != "business-health" {
			t.Fatalf("relation missing explicit scenario identity: %v", result)
		}
		source, ok := result["requirement_source"].(map[string]any)
		if !ok || source["path"] == "" || source["line"].(float64) < 1 || len(source["sha256"].(string)) != 64 {
			t.Fatalf("relation missing exact declaration provenance: %v", result)
		}
		if len(decoded["limitations"].([]any)) == 0 {
			t.Fatal("missing non-verification limits")
		}
	})
	t.Run("rejects unsupported scenario and unknown fields", func(t *testing.T) {
		for _, payload := range []string{`{"scenario":"other","intent":"x"}`, `{"scenario":"business-health","intent":"x","scope":"fleet"}`} {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/change-intent/assess", bytes.NewBufferString(payload))
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != http.StatusBadRequest {
				t.Fatalf("payload %s: status=%d body=%s", payload, res.Code, res.Body.String())
			}
		}
	})
}
