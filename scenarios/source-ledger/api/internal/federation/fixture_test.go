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
    "provider_id": "source-ledger.agent-memory",
    "provider_group": "source-ledger",
    "type": "record",
    "class": "local_live",
    "description": "test memory provider",
    "scope": "SCOPE_PROJECT",
    "endpoint": {"http_json": {"method": "HTTP_METHOD_POST", "path": "/vrooli.source_ledger.v1.recall.RecallService/Recall", "scenario_id": "source-ledger", "body_template": "{\"query\":\"{{query}}\",\"limit\":{{limit}},\"scope\":\"agent-memory\"}"}},
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
