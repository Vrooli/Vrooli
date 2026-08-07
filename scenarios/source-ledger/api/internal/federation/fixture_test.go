package federation

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSearchFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "search.json")
	const fixture = `{
  "version": "1.0.0",
  "providers": [{
    "provider_id": "vrooli-memory.memories",
    "provider_group": "vrooli-memory",
    "type": "record",
    "class": "local_live",
    "description": "test memory provider",
    "scope": "SCOPE_PROJECT",
    "endpoint": {"http_json": {"method": "HTTP_METHOD_POST", "path": "/Recall", "scenario_id": "vrooli-memory", "body_template": "{\"query\":\"{{query}}\",\"limit\":{{limit}}}"}},
    "result_mapping": {"results_path": "hits", "id_field": "entryId", "score_field": "score", "snippet_field": "text", "title_field": "facetId"},
    "tuning": {"engine": "dense"},
    "tests": {"cases": [{"id": "memory-work-record", "query": "durable memory"}]}
  }]
}`
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
