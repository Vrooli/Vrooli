package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	apidb "github.com/vrooli/api-core/database"
)

type manifest struct{ CatalogID, Latest string }

func main() {
	dbPath := flag.String("db", "../data/react-component-library.db", "SQLite database path")
	root := flag.String("scenario-root", "..", "React Component Library scenario root")
	flag.Parse()
	db, err := apidb.Open(context.Background(), apidb.Config{
		Driver:       apidb.DriverSQLite,
		DSN:          fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)", *dbPath),
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	paths, err := filepath.Glob(filepath.Join(*root, "library", "*", "*", "component.json"))
	if err != nil {
		log.Fatal(err)
	}
	updated := 0
	latestByID := map[string]string{}
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			log.Fatal(readErr)
		}
		var m manifest
		if json.Unmarshal(data, &m) != nil || m.CatalogID == "" || m.Latest == "" {
			continue
		}
		result, updateErr := db.Primary().ExecContext(ctx, `UPDATE catalog_gate_evidence SET version = ? WHERE asset_id = ? AND version IN ('', 'legacy')`, m.Latest, m.CatalogID)
		if updateErr != nil {
			log.Fatal(updateErr)
		}
		count, _ := result.RowsAffected()
		updated += int(count)
		latestByID[m.CatalogID] = m.Latest
	}
	for oldID, currentID := range map[string]string{"primitives.presence": "motion.presence", "services.scroll-restoration": "navigation.scroll-restoration"} {
		latest := latestByID[currentID]
		if latest == "" {
			latest = "1.0.0"
		}
		_, updateErr := db.Primary().ExecContext(ctx, `
		INSERT OR IGNORE INTO catalog_gate_evidence(asset_id, target, gate, version, result, source_revision, recorded_at)
		SELECT ?, target, gate, ?, result, source_revision, recorded_at FROM catalog_gate_evidence WHERE asset_id = ? AND version IN ('', 'legacy')`, currentID, latest, oldID)
		if updateErr != nil {
			log.Fatal(updateErr)
		}
		result, updateErr := db.Primary().ExecContext(ctx, `DELETE FROM catalog_gate_evidence WHERE asset_id = ? AND version IN ('', 'legacy')`, oldID)
		if updateErr != nil {
			log.Fatal(updateErr)
		}
		count, _ := result.RowsAffected()
		updated += int(count)
	}
	var empty int
	if err := db.Primary().QueryRowContext(ctx, `SELECT COUNT(*) FROM catalog_gate_evidence WHERE version = '' OR version = 'legacy'`).Scan(&empty); err != nil {
		log.Fatal(err)
	}
	if empty != 0 {
		log.Fatal(fmt.Errorf("%d gate evidence rows have no resolved version", empty))
	}
	log.Printf("backfilled versions on %d gate evidence rows", updated)
}
