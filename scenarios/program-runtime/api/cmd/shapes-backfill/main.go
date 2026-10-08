// Command shapes-backfill derives the shape ledger from successful programs.
// It is safe to run more than once because the ledger records each program id.
package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	_ "modernc.org/sqlite"
	"program-runtime/internal/contracts"
	"program-runtime/internal/shapes"
)

func backfill(ctx context.Context, db shapes.SQLExecutor) (int, error) {
	rows, err := db.QueryContext(ctx, `SELECT id,session_id,provenance,created_at FROM programs WHERE status='succeeded' ORDER BY created_at,id`)
	if err != nil {
		return 0, err
	}
	type program struct{ id, sessionID, provenance, created string }
	programs := make([]program, 0)
	for rows.Next() {
		var id, sessionID, provenance, created string
		if err := rows.Scan(&id, &sessionID, &provenance, &created); err != nil {
			rows.Close()
			return 0, err
		}
		programs = append(programs, program{id, sessionID, provenance, created})
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	repo := shapes.NewRepository(db)
	count := 0
	for _, item := range programs {
		id, sessionID, provenance, created := item.id, item.sessionID, item.provenance, item.created
		value, parseErr := strconv.Atoi(provenance)
		if parseErr != nil {
			value = 0
		}
		when, parseErr := time.Parse(time.RFC3339Nano, created)
		if parseErr != nil {
			when = time.Now().UTC()
		}
		if _, err := repo.Observe(ctx, id, sessionID, programsv1.Provenance(value), when); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func main() {
	ctx := context.Background()
	dsn, err := storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "program-runtime"})
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: dsn, MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	count, err := backfill(ctx, db.Primary())
	if err != nil {
		log.Fatal(err)
	}
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		log.Fatal(err)
	}
	index := contracts.NewIndex()
	if err := index.Load(root); err != nil {
		log.Fatal(err)
	}
	repo := shapes.NewRepository(db.Primary(), index)
	if _, err := repo.ResolveCoverage(ctx, index); err != nil {
		log.Fatal(err)
	}
	if _, err := repo.ApplyGate(ctx, time.Now().UTC()); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("processed successful programs: %d\n", count)
}
