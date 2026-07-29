//go:build ignore

// Seed prepares the routed SQLite test database used by the mutating manual
// workflow BAS playbook. It creates no platform identity and has no external
// side effect; the browser journey creates its synthetic identity itself.
package main

import (
	"database/sql"
	"log"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	dsn := strings.TrimSpace(os.Getenv("SQLITE_PATH"))
	if dsn == "" { dsn = strings.TrimSpace(os.Getenv("SQLITE_DB")) }
	if dsn == "" { log.Fatal("seed: SQLITE_PATH or SQLITE_DB is required") }
	db, err := sql.Open("sqlite", dsn)
	if err != nil { log.Fatal(err) }
	defer db.Close()
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS channel_manager_state (id INTEGER PRIMARY KEY CHECK(id=1), state_json TEXT NOT NULL)`); err != nil { log.Fatal(err) }
	// The playbook uses a stable synthetic ID so its screenshots and evidence
	// remain readable. Clear only the scenario-owned snapshot before each
	// routed run to make that fixture repeatable without touching live data.
	if _, err = db.Exec(`DELETE FROM channel_manager_state`); err != nil { log.Fatal(err) }
}
