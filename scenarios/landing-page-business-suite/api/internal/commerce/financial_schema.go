package commerce

import (
	_ "embed"
	"strings"
)

//go:embed financial_schema.sql
var financialSchema string

func FinancialSchema() string { return financialSchema }

// FinancialIndexesSchema runs after api-core/database has reconciled additive
// columns declared above. Keeping indexes that reference newly-added columns
// out of the boot-time schema blob prevents an existing SQLite database from
// failing before reconciliation can add those columns.
func FinancialIndexesSchema() string {
	return strings.Join([]string{
		"CREATE INDEX IF NOT EXISTS idx_bundle_prices_external_product ON bundle_prices(external_product_id) WHERE external_product_id IS NOT NULL;",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_source_external ON subscriptions(source, external_subscription_id) WHERE external_subscription_id IS NOT NULL;",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_transactions_source_event ON credit_transactions(source, external_event_id) WHERE external_event_id IS NOT NULL;",
	}, "\n")
}
