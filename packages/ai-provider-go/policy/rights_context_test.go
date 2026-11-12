package policy

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadRightsContextFileRequiresCompleteVersionedOperatorContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "context.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"1","context":{"id":"operator","version":"v1","revenue_status":"pre_revenue","as_of":"2026-11-10","source":"operator_statement"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRightsContextFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "operator" || got.EntityScope != "" || got.RevenueStatus != "pre_revenue" {
		t.Fatalf("context = %+v", got)
	}
	artifact := testConditionalRights()
	decision := EvaluateRights(&artifact, RightsUseLocalInference, got, time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC))
	if decision.Status != RightsUnknown || decision.Reason != "rights_facts_incomplete" {
		t.Fatalf("context without entity scope must remain unknown: %+v", decision)
	}
	for _, document := range []string{
		`{"schema_version":"2","context":{"id":"operator","version":"v1","as_of":"2026-11-10","source":"operator_statement"}}`,
		`{"schema_version":"1","context":{"id":"operator","version":"v1","as_of":"2026-11-10","source":"operator_statement"}}`,
		`{"schema_version":"1","context":{"id":"operator","version":"v1","revenue_status":"pre_revenue","as_of":"2026-11-10"}}`,
	} {
		if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadRightsContextFile(path); err == nil {
			t.Fatalf("accepted incomplete context %s", document)
		}
	}
}
