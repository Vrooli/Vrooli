// Command schema-preflight checks whether existing Ask identities satisfy
// Notification Hub's declared uniqueness contract. It is intended to run as
// an owner lifecycle setup step before replacing a serving process.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"notification-hub/internal/conversations"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

const scenarioName = "notification-hub"

func main() {
	dir, err := scenarioDir(os.Getenv, os.Getwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ask identity preflight unavailable: %v\n", err)
		os.Exit(1)
	}
	os.Exit(run(os.Stdout, os.Stderr, os.Getenv, dir))
}

// scenarioDir is the scenario's directory: the lifecycle's own value when it
// gave one, else the parent of the api directory this runs from.
func scenarioDir(getenv func(string) string, getwd func() (string, error)) (string, error) {
	if dir := getenv("VROOLI_SCENARIO_DIR"); filepath.IsAbs(dir) {
		return dir, nil
	}
	cwd, err := getwd()
	if err != nil {
		return "", fmt.Errorf("locate the scenario directory: %w", err)
	}
	return filepath.Dir(cwd), nil
}

// servingDatabasePath is the file the serving Hub opens, resolved by the same
// storage.SQLitePath call as the API's database.Open.
//
// That call takes the lifecycle's data directory from the environment the
// lifecycle injects (SCENARIO_NAME, VROOLI_SCENARIO, SCENARIO_DATA_DIR). A
// setup step has them, but this program is also run by hand, by a test harness
// or by an agent, with none of them; resolution then falls to the class root
// (~/.vrooli/data/vrooli/notification-hub), a different and empty database from
// the one the live Hub serves from, and the preflight reported "clear" about
// it. So the environment is completed with what the lifecycle would have
// injected for this scenario's own directory. A shadow variant's namespace
// variables are left alone, and they take precedence inside SQLitePath, so a
// variant still checks its own database.
func servingDatabasePath(getenv func(string) string, scenarioDir string) (string, error) {
	injected := map[string]string{
		"SCENARIO_NAME":     scenarioName,
		"VROOLI_SCENARIO":   scenarioName,
		"SCENARIO_DATA_DIR": storage.LifecycleDataDir(scenarioDir),
	}
	return storage.SQLitePath(storage.SQLiteConfig{
		Scenario: scenarioName,
		EnvGet: func(key string) string {
			if value, ok := injected[key]; ok {
				return value
			}
			return getenv(key)
		},
	})
}

func run(stdout, stderr io.Writer, getenv func(string) string, scenarioDir string) int {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	path, err := servingDatabasePath(getenv, scenarioDir)
	if err != nil {
		fmt.Fprintf(stderr, "Ask identity preflight unavailable: resolve the serving database: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Ask identity preflight: checking %s\n", path)
	// A database that does not exist has no Ask identities to conflict, and
	// opening it would create an empty one.
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintln(stdout, "Ask identity preflight: clear (no database yet)")
		return 0
	} else if err != nil {
		fmt.Fprintf(stderr, "Ask identity preflight unavailable: inspect the serving database: %v\n", err)
		return 1
	}
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		fmt.Fprintf(stderr, "Ask identity preflight unavailable: %v\n", err)
		return 1
	}
	db, err := database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
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
