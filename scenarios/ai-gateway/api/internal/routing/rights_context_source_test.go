package routing

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestJSONRightsContextSourceReadsCurrentVersionAndFailsClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "context.json")
	source := JSONRightsContextSource(path)
	if _, err := source(context.Background()); err == nil {
		t.Fatal("missing trusted context was accepted")
	}
	write := func(value string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"schema_version":"1","context":{"id":"operator-context","version":"v1","revenue_status":"pre_revenue","as_of":"2026-11-10","source":"operator_statement"}}`)
	first, err := source(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	write(`{"schema_version":"1","context":{"id":"operator-context","version":"v2","revenue_status":"at_or_above_threshold","as_of":"2026-11-11","source":"operator_statement"}}`)
	second, err := source(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.Version == second.Version || first.RevenueStatus == second.RevenueStatus {
		t.Fatal("source did not observe changed operator context")
	}
	write(`{"schema_version":"2","context":{}}`)
	if _, err := source(context.Background()); err == nil {
		t.Fatal("unsupported context schema was accepted")
	}
}
