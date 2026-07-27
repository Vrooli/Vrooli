package financial

import (
	"strings"
	"testing"
)

func TestSchemaOwnsFinancialRuntimeTables(t *testing.T) {
	sql := strings.ToLower(Schema())
	for _, table := range []string{"bundle_products", "checkout_sessions", "subscriptions", "credit_transactions", "payment_anomaly_log"} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Errorf("missing %s", table)
		}
	}
}
