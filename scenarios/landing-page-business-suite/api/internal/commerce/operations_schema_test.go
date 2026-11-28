package commerce

import (
	"strings"
	"testing"
)

func TestOperationsSchemaOwnsCommerceRuntimeTables(t *testing.T) {
	sql := strings.ToLower(OperationsSchema())
	if strings.Contains(sql, "alter table") || strings.Contains(sql, "drop table") {
		t.Fatal("commerce runtime schema must be declarative; additive columns belong in CREATE definitions")
	}
	for _, table := range []string{"usage_records", "usage_events", "credit_reservations", "api_keys", "users", "auth_tokens", "user_sessions"} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Errorf("missing %s", table)
		}
	}
	for _, column := range []string{"cost_micros", "prompt_tokens", "completion_tokens", "provider", "browser_session_minutes", "browser_actions", "browser_bytes", "browser_run_id", "browser_session_id", "browser_evidence_fingerprint"} {
		if !strings.Contains(sql, column) {
			t.Errorf("usage_events schema missing provider cost column %s", column)
		}
	}
}

func TestServiceOrderReceiptColumnsArePartOfDeclarativeOwnerTable(t *testing.T) {
	sql := strings.ToLower(OperationsSchema())
	start := strings.Index(sql, "create table if not exists service_orders")
	if start < 0 {
		t.Fatal("service_orders declaration missing")
	}
	end := strings.Index(sql[start:], ");")
	if end < 0 {
		t.Fatal("service_orders declaration is not closed")
	}
	declaration := sql[start : start+end]
	for _, column := range []string{"ledger_receipt_id", "treasury_external_id"} {
		if !strings.Contains(declaration, column) {
			t.Errorf("service_orders declaration missing %s", column)
		}
	}
}
