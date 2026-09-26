// Command migrate-memory-corpus moves the engine-owned tables from a
// vrooli-memory SQLite database into the source-ledger database.
//
// The command is intentionally one-shot and explicit. It does not move
// harness_projections, harness_import_runs, or journal_retry_queue: those
// tables describe the coding-agent integration and remain owned by
// vrooli-memory. The migration runs in one transaction and reports the
// deterministic entries content hash used by the phase acceptance gate.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

type tableSpec struct {
	name    string
	columns string
}

var engineTables = []tableSpec{
	{name: "scopes", columns: "id,label,frontier_target,wake_budget,max_entry_lines,created_at,updated_at"},
	{name: "facet_definitions", columns: "id,scope,label,classification_guidance,created_at"},
	{name: "facet_policies", columns: "facet_id,scope,retention_policy,compaction_eligible,resident_budget"},
	{name: "entries", columns: "id,scope,body,facet_id,kind,actor_id,actor_kind,source_runtime,verification_status,harness_session_id,harness_kind,run_id,workflow_execution_id,import_key,source_harness,source_path,imported_at,created_at"},
	{name: "facet_assignments", columns: "id,entry_id,facet_id,assigned_at,actor_id"},
	{name: "facet_examples", columns: "id,scope,facet_id,entry_id,body,created_at"},
	{name: "facet_texts", columns: "id,entry_id,kind,text,embedding_ref"},
	{name: "embeddings", columns: "id,facet_text_id,vector_json,vector_blob,created_at"},
	{name: "summaries", columns: "id,scope,body,facet_id,vector_json,vector_blob,depth,generation,created_at"},
	{name: "tree_edges", columns: "parent_id,child_id,child_kind"},
	{name: "pins", columns: "entry_id,review_at,pinned_at,actor_id"},
	{name: "pin_reviews", columns: "id,entry_id,due_at,resolved_at"},
	{name: "merge_proposals", columns: "id,scope,rationale,entry_ids_json,resolved_at"},
	{name: "marks", columns: "id,entry_id,kind,replacement_entry_id,created_at"},
	{name: "facet_recall_stats", columns: "entry_id,recall_count,last_recalled_at"},
	{name: "classification_rules", columns: "id,scope,priority,facet_id,source_runtime,kind,source_path_glob,body_pattern,enabled,created_at,updated_at"},
	{name: "classification_rule_dry_runs", columns: "id,rule_id,scope,corpus_fingerprint,match_count,samples_json,created_at"},
}

var deleteOrder = []string{
	"classification_rule_dry_runs",
	"classification_rules",
	"tree_edges",
	"embeddings",
	"facet_texts",
	"facet_examples",
	"facet_assignments",
	"facet_recall_stats",
	"pin_reviews",
	"pins",
	"marks",
	"merge_proposals",
	"summaries",
	"entries",
	"facet_policies",
	"facet_definitions",
	"scopes",
}

func main() {
	source := flag.String("source", "", "vrooli-memory SQLite database to read")
	target := flag.String("target", "", "source-ledger SQLite database to replace engine data in")
	flag.Parse()
	if strings.TrimSpace(*source) == "" || strings.TrimSpace(*target) == "" {
		flag.Usage()
		os.Exit(2)
	}
	if filepath.Clean(*source) == filepath.Clean(*target) {
		log.Fatal("source and target must be different files")
	}

	ctx := context.Background()
	sourceDB, err := openSQLite(ctx, *source)
	if err != nil {
		log.Fatalf("open source database: %v", err)
	}
	defer sourceDB.Close()
	targetDB, err := openSQLite(ctx, *target)
	if err != nil {
		log.Fatalf("open target database: %v", err)
	}
	defer targetDB.Close()

	sourceHash, sourceCount, err := entriesHash(ctx, sourceDB, "main")
	if err != nil {
		log.Fatalf("hash source entries: %v", err)
	}
	if _, err := targetDB.ExecContext(ctx, `PRAGMA foreign_keys=OFF`); err != nil {
		log.Fatalf("disable foreign-key checks for migration transaction: %v", err)
	}
	attachPath := quoteSQLiteString(*source)
	if _, err := targetDB.ExecContext(ctx, "ATTACH DATABASE '"+attachPath+"' AS source_memory"); err != nil {
		log.Fatalf("attach source database: %v", err)
	}
	defer targetDB.ExecContext(ctx, "DETACH DATABASE source_memory") //nolint:errcheck

	tx, err := targetDB.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("begin migration transaction: %v", err)
	}
	rollback := func(err error) {
		_ = tx.Rollback()
		log.Fatal(err)
	}

	for _, trigger := range []string{"entries_append_only_update", "entries_append_only_delete"} {
		if _, err := tx.ExecContext(ctx, "DROP TRIGGER IF EXISTS "+trigger); err != nil {
			rollback(fmt.Errorf("drop temporary migration trigger %s: %w", trigger, err))
		}
	}
	for _, table := range deleteOrder {
		if _, err := tx.ExecContext(ctx, "DELETE FROM main."+table); err != nil {
			rollback(fmt.Errorf("clear target table %s: %w", table, err))
		}
	}
	for _, table := range engineTables {
		query := fmt.Sprintf("INSERT INTO main.%s (%s) SELECT %s FROM source_memory.%s", table.name, table.columns, table.columns, table.name)
		if _, err := tx.ExecContext(ctx, query); err != nil {
			rollback(fmt.Errorf("copy table %s: %w", table.name, err))
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM main.journal_high_water_mark`); err != nil {
		rollback(fmt.Errorf("reset journal high-water mark: %w", err))
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO main.journal_high_water_mark(id,max_rowid,recorded_at) SELECT 1,COALESCE(MAX(rowid),0),CURRENT_TIMESTAMP FROM main.entries`); err != nil {
		rollback(fmt.Errorf("write journal high-water mark: %w", err))
	}
	for _, trigger := range []string{
		`CREATE TRIGGER entries_append_only_update BEFORE UPDATE ON entries BEGIN SELECT RAISE(ABORT, 'journal entries are append-only'); END`,
		`CREATE TRIGGER entries_append_only_delete BEFORE DELETE ON entries BEGIN SELECT RAISE(ABORT, 'journal entries are append-only'); END`,
	} {
		if _, err := tx.ExecContext(ctx, trigger); err != nil {
			rollback(fmt.Errorf("restore append-only guard: %w", err))
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatalf("commit migration: %v", err)
	}

	targetHash, targetCount, err := entriesHash(ctx, targetDB, "main")
	if err != nil {
		log.Fatalf("hash migrated entries: %v", err)
	}
	if sourceCount != targetCount || sourceHash != targetHash {
		log.Fatalf("migration verification failed: source entries=%d hash=%s, target entries=%d hash=%s", sourceCount, sourceHash, targetCount, targetHash)
	}

	fmt.Printf("migration complete: entries=%d content_hash=%s\n", targetCount, targetHash)
	for _, table := range engineTables {
		var sourceCount, targetCount int64
		if err := sourceDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM main."+table.name).Scan(&sourceCount); err != nil {
			log.Fatalf("count source table %s: %v", table.name, err)
		}
		if err := targetDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM main."+table.name).Scan(&targetCount); err != nil {
			log.Fatalf("count target table %s: %v", table.name, err)
		}
		if sourceCount != targetCount {
			log.Fatalf("migration verification failed: table %s source=%d target=%d", table.name, sourceCount, targetCount)
		}
		fmt.Printf("table %-30s %d\n", table.name, targetCount)
	}
	fmt.Println("harness tables intentionally not migrated: harness_projections harness_import_runs journal_retry_queue")
}

func openSQLite(ctx context.Context, path string) (*database.RoutedDB, error) {
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		return nil, err
	}
	return database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func entriesHash(ctx context.Context, db queryer, schema string) (string, int64, error) {
	rows, err := db.QueryContext(ctx, "SELECT id,body FROM "+schema+`.entries ORDER BY id`)
	if err != nil {
		return "", 0, err
	}
	defer rows.Close()
	h := sha256.New()
	var count int64
	for rows.Next() {
		var id, body string
		if err := rows.Scan(&id, &body); err != nil {
			return "", 0, err
		}
		_, _ = h.Write([]byte(id))
		_, _ = h.Write([]byte(body))
		count++
	}
	if err := rows.Err(); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), count, nil
}

func quoteSQLiteString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
