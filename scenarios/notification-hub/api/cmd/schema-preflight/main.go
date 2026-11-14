// Command schema-preflight checks whether existing Ask identities satisfy
// Notification Hub's declared uniqueness contract. It is intended to run as
// an owner lifecycle setup step before replacing a serving process.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"time"

	"notification-hub/internal/conversations"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func main() {
	os.Exit(run(os.Stdout, os.Stderr))
}

func run(stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, err := database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "notification-hub",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		fmt.Fprintf(stderr, "Ask identity preflight unavailable: open configured owner database: %v\n", err)
		return 1
	}
	defer db.Close()
	if err := report(ctx, db.Primary(), stdout, stderr); err != nil {
		return 1
	}
	return 0
}

func report(ctx context.Context, db *sql.DB, stdout, stderr io.Writer) error {
	duplicateGroups, err := conversations.AskIdentityDuplicateGroups(ctx, db)
	if err != nil {
		fmt.Fprintf(stderr, "Ask identity preflight unavailable: %v\n", err)
		return err
	}
	if duplicateGroups > 0 {
		err := fmt.Errorf("duplicate Ask requester identities block schema adoption")
		fmt.Fprintf(stderr, "Ask identity preflight: duplicate_groups=%d; %v\n", duplicateGroups, err)
		return err
	}
	fmt.Fprintln(stdout, "Ask identity preflight: clear (duplicate_groups=0)")
	return nil
}
