package commerce

import (
	"strings"
	"testing"
)

func TestFinancialSchemaOwnsCommerceRuntimeTables(t *testing.T) {
	sql := strings.ToLower(FinancialSchema())
	for _, table := range []string{"bundle_products", "checkout_sessions", "subscriptions", "credit_transactions", "payment_anomaly_log"} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Errorf("missing %s", table)
		}
	}
}
