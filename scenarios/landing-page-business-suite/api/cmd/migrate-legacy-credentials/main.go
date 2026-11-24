// Command migrate-legacy-credentials plans or applies the retained-data
// compatibility migration against an explicitly named local database copy.
// It never deletes source columns; schema cutover is a separate reviewed step.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/api-core/database"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/credentialmigration"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	copyDSN := flag.String("copy-dsn", "", "explicit PostgreSQL local database-copy DSN; database name must end in _credential_migration_copy")
	apply := flag.Bool("apply", false, "activate prepared authority candidates; omitted means plan-only")
	flag.Parse()
	if !localMigrationCopy(*copyDSN) {
		return errors.New("refusing migration: provide a loopback PostgreSQL DSN whose database name ends in _credential_migration_copy")
	}
	ctx := context.Background()
	routed, err := database.Open(ctx, database.Config{Driver: "postgres", Scenario: "landing-page-business-suite", DSN: *copyDSN})
	if err != nil {
		return errors.New("open local migration copy failed")
	}
	defer routed.Close()
	authority, err := credentialauthority.Default()
	if err != nil {
		return errors.New("credential authority is unavailable")
	}
	plan, err := credentialmigration.Prepare(ctx, routed.Primary(), os.Getenv("STRIPE_MODE"), authority)
	if err != nil {
		return err
	}
	defer plan.Clear()
	for _, item := range plan.Items {
		fmt.Printf("%s %s %s\n", item.Source, item.Field, item.Disposition)
	}
	if !*apply {
		fmt.Println("plan only; legacy source columns remain untouched")
		return nil
	}
	result, err := credentialmigration.Apply(ctx, plan, authority)
	if err != nil {
		return err
	}
	fmt.Printf("activated=%d retained=%d; source columns retained for rollback\n", len(result.Applied), len(result.Retained))
	return nil
}

func localMigrationCopy(dsn string) bool {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.User == nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return false
		}
	}
	name := strings.TrimPrefix(strings.TrimSpace(u.Path), "/")
	return strings.HasSuffix(name, "_credential_migration_copy")
}
