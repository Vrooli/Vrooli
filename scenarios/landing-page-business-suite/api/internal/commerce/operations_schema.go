package commerce

import (
	_ "embed"
	"strings"
)

// OperationsSchema is declarative runtime DDL. api-core/database reconciles
// additive columns as part of EnsureSchemas, so a newly declared column is
// added to an existing table at boot without a hand-run ALTER or destructive
// migration. This is the greenfield-with-data tier: retain one-shot operator
// scripts for data moves, and introduce versioned migrations only once a
// production schema evolution requires them.
//
//go:embed operations_schema.sql
var operationsSchema string

//go:embed stripe_webhook_receipts.sql
var stripeWebhookReceiptsSchema string

//go:embed stripe_test_economic_receipts.sql
var stripeTestEconomicReceiptsSchema string

//go:embed stripe_test_lifecycle_receipts.sql
var stripeTestLifecycleReceiptsSchema string

func OperationsSchema() string {
	return strings.Join([]string{operationsSchema, stripeWebhookReceiptsSchema, stripeTestEconomicReceiptsSchema, stripeTestLifecycleReceiptsSchema}, "\n")
}
