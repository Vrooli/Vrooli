// Command migrate-legacy-credentials performs the one-time write-through
// migration for pre-authority LPBS payment and delivery settings.
//
// Run it against a copy of an existing database before deploying the schema
// without the retired cleartext columns. It intentionally does not provision
// values and refuses to continue when the credential authority is unavailable.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/vrooli/api-core/database"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/delivery"
)

func main() {
	ctx := context.Background()
	authority, err := credentialauthority.Default()
	if err != nil {
		log.Fatalf("initialize credential authority: %v", err)
	}

	routed, err := database.Open(ctx, database.Config{Driver: "postgres"})
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer routed.Close()

	read := func(ctx context.Context, field string) (string, error) {
		return authority.Require(credentialauthority.Identity("vrooli/landing-page-business-suite"), field)
	}
	write := func(_ context.Context, field, value string) error {
		return authority.Put(credentialauthority.Identity("vrooli/landing-page-business-suite"), field, value)
	}

	paymentCount, err := commerce.MigrateLegacyPaymentCredentials(ctx, routed.Primary(), read, write)
	if err != nil {
		log.Fatalf("migrate payment credentials: %v", err)
	}
	deliveryCount, err := delivery.MigrateLegacyStorageCredentials(ctx, routed.Primary(), read, write)
	if err != nil {
		log.Fatalf("migrate delivery credentials: %v", err)
	}
	fmt.Printf("legacy credential migration complete: payment=%d delivery=%d total=%d\n", paymentCount, deliveryCount, paymentCount+deliveryCount)
}
