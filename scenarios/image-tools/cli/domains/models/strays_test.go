package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStrayReportShowsEveryFindingWithItsActionAndWhereItLooked(t *testing.T) {
	var report strayReport
	if err := json.Unmarshal([]byte(`{
	  "managed_root": "/m",
	  "scanned": [{"path": "/home/x/.insightface", "exists": false, "source": "insightface"}, {"path": "/home/x/.cache/huggingface", "exists": true, "source": "huggingface_hub"}],
	  "findings": [
	    {"id": "a1", "path": "/home/x/.cache/huggingface/hub/models--o--n", "owner": "image_tools_catalog", "catalog_ids": ["model:sd-1.5"], "bytes": 2097152, "action": "adopt", "destination": "/m/cache/huggingface/hub/models--o--n", "reason": "catalog refers"},
	    {"id": "b2", "path": "/home/x/.ollama/models/blobs", "owner": "other_application", "bytes": 5, "action": "report_only", "reason": "owned elsewhere"}
	  ],
	  "totals": {"stray_bytes": 2097157, "adoptable_bytes": 2097152, "report_only_bytes": 5}
	}`), &report); err != nil {
		t.Fatal(err)
	}
	list := renderStrayReport(report)
	joined := strings.Join(list.Results, "\n")
	for _, want := range []string{"a1", "2 MiB", "[ADOPT]", "model:sd-1.5", "-> /m/cache/huggingface/hub/models--o--n", "[REPORT_ONLY]", "owned elsewhere"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in\n%s", want, joined)
		}
	}
	summary := strings.Join(list.Summary, "\n")
	if !strings.Contains(summary, ".insightface (insightface, absent)") || !strings.Contains(summary, "adoptable 2 MiB") {
		t.Fatalf("summary must say what was scanned and totals: %s", summary)
	}
}
