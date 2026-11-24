package commerce

import (
	"strings"
	"testing"
)

func TestFinancialSchemaOwnsCommerceRuntimeTables(t *testing.T) {
	sql := strings.ToLower(FinancialSchema())
	for _, table := range []string{"bundle_products", "checkout_sessions", "subscriptions", "credit_transactions", "business_account_credit_wallets", "business_account_credit_transactions", "business_account_credit_lifecycle", "payment_anomaly_log"} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Errorf("missing %s", table)
		}
	}
	for _, migration := range []string{"add column if not exists contract_version", "add column if not exists usd_budget_micros", "add column if not exists cost_multiplier"} {
		if !strings.Contains(sql, migration) {
			t.Errorf("missing lifecycle forward migration %q", migration)
		}
	}
}
