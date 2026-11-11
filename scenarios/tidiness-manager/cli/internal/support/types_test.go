package support

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestScanDurationAcceptsAPIAndStringForms(t *testing.T) {
	var result SmartScanResult
	if err := json.Unmarshal([]byte(`{"duration":2000000000,"batch_results":[{"duration":"150ms"}]}`), &result); err != nil {
		t.Fatalf("decode mixed duration forms: %v", err)
	}
	if got, want := result.Duration.String(), "2s"; got != want {
		t.Fatalf("top-level duration = %q, want %q", got, want)
	}
	if got, want := result.BatchResults[0].Duration.String(), "150ms"; got != want {
		t.Fatalf("batch duration = %q, want %q", got, want)
	}
}

func TestScanDurationRejectsAmbiguousNumericForms(t *testing.T) {
	var result SmartScanResult
	if err := json.Unmarshal([]byte(`{"duration":1.5}`), &result); err == nil {
		t.Fatal("expected fractional duration to be rejected")
	}
}

func TestLightScanResultRetainsSharedLanguageDuplicateEvidence(t *testing.T) {
	var result LightScanResult
	if err := json.Unmarshal([]byte(`{"scenario":"demo","language_metrics":{"javascript":{"language":"javascript","file_count":3,"total_lines":44,"code_metrics":{"todo_count":1,"fixme_count":2,"hack_count":3,"avg_imports_per_file":2.5,"avg_functions_per_file":1.5,"max_imports_in_file":8,"max_functions_in_file":4,"comment_to_code_ratio":0.25,"future_metric":9},"complexity":{"average_complexity":3.5,"max_complexity":12,"high_complexity_count":2,"threshold":10,"total_functions":20,"tool":"code-facts"},"duplicates":{"total_duplicates":2,"duplicate_blocks":[{"files":[{"path":"a.go","start_line":1,"end_line":6},{"path":"b.go","start_line":3,"end_line":8}],"lines":6}],"total_lines":17,"tool":"code-facts"}}}}`), &result); err != nil {
		t.Fatalf("decode light scan language metrics: %v", err)
	}
	javascript := result.LanguageMetrics["javascript"]
	if javascript.Language != "javascript" || javascript.FileCount != 3 || javascript.TotalLines != 44 {
		t.Fatalf("language summary was lost by CLI projection: %+v", javascript)
	}
	var duplicates struct {
		TotalDuplicates int `json:"total_duplicates"`
		TotalLines      int `json:"total_lines"`
		DuplicateBlocks []struct {
			Files []json.RawMessage `json:"files"`
		} `json:"duplicate_blocks"`
	}
	if err := json.Unmarshal(javascript.Duplicates, &duplicates); err != nil || duplicates.TotalDuplicates != 2 || duplicates.TotalLines != 17 || len(duplicates.DuplicateBlocks) != 1 || len(duplicates.DuplicateBlocks[0].Files) != 2 {
		t.Fatalf("duplicate graph evidence was lost by CLI projection: %s, err=%v", javascript.Duplicates, err)
	}
	encoded, err := json.Marshal(javascript)
	if err != nil || !strings.Contains(string(encoded), `"comment_to_code_ratio":0.25`) || !strings.Contains(string(encoded), `"future_metric":9`) {
		t.Fatalf("unknown detailed scan fields were not preserved: %s, err=%v", encoded, err)
	}
}
