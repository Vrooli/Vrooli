// Command measure-recall records the direct source-ledger recall cost at the
// current corpus size and on a disposable corpus scaled by --scale. The
// source database is copied first; synthetic rows never enter the authority.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"source-ledger/internal/ledgerdb"
	"source-ledger/internal/recall"
	vectorcodec "source-ledger/internal/vector"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

type fixedEmbedder struct{ vector []float64 }

func (e fixedEmbedder) EmbedQuery(context.Context, string) ([]float64, error) { return e.vector, nil }

type measurement struct {
	Scope            string    `json:"scope"`
	Entries          int       `json:"entries"`
	Summaries        int       `json:"summaries"`
	NodeCount        int       `json:"node_count"`
	SamplesSeconds   []float64 `json:"samples_seconds"`
	P50Seconds       float64   `json:"p50_seconds"`
	P95Seconds       float64   `json:"p95_seconds"`
	TopEntryID       string    `json:"top_entry_id"`
	LexicalFallbacks int       `json:"lexical_fallbacks"`
}

func main() {
	dbPath := flag.String("db", "../data/source-ledger.db", "source-ledger SQLite database")
	scope := flag.String("scope", "agent-memory", "scope to measure")
	scale := flag.Int("scale", 3, "synthetic multiple for the disposable measurement")
	flag.Parse()
	if *scale < 1 {
		fatal(fmt.Errorf("scale must be at least 1"))
	}

	root, err := filepath.Abs(*dbPath)
	if err != nil {
		fatal(err)
	}
	current, err := copyDatabase(root)
	if err != nil {
		fatal(err)
	}
	defer storage.RemoveFile(current)
	baseline, err := measure(current, *scope, 1)
	if err != nil {
		fatal(fmt.Errorf("baseline measurement: %w", err))
	}
	// Compare against the same committed snapshot, even if the live corpus is
	// changing while the baseline is measured.
	scaled, err := copyDatabase(current)
	if err != nil {
		fatal(err)
	}
	defer storage.RemoveFile(scaled)
	if err := scaleCorpus(scaled, *scope, *scale); err != nil {
		fatal(fmt.Errorf("scale corpus: %w", err))
	}
	threeX, err := measure(scaled, *scope, *scale)
	if err != nil {
		fatal(fmt.Errorf("scaled measurement: %w", err))
	}
	result := struct {
		Database string      `json:"database"`
		Scale    int         `json:"scale"`
		Baseline measurement `json:"baseline"`
		Scaled   measurement `json:"scaled"`
	}{Database: root, Scale: *scale, Baseline: baseline, Scaled: threeX}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fatal(err)
	}
}

func copyDatabase(path string) (string, error) {
	ctx := context.Background()
	source, err := ledgerdb.OpenExistingReadOnly(ctx, path)
	if err != nil {
		return "", err
	}
	defer source.Close()
	target, err := os.CreateTemp("", "source-ledger-recall-*.db")
	if err != nil {
		return "", err
	}
	targetPath := target.Name()
	if err := target.Close(); err != nil {
		_ = storage.RemoveFile(targetPath)
		return "", err
	}
	// SQLite creates a consistent snapshot including committed WAL pages.
	if _, err := source.ExecContext(ctx, "VACUUM INTO ?", targetPath); err != nil {
		_ = storage.RemoveFile(targetPath)
		return "", err
	}
	return targetPath, nil
}

func open(path string) (*database.RoutedDB, error) {
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		return nil, err
	}
	return database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: dsn, MaxOpenConns: 1, MaxIdleConns: 1})
}

func scaleCorpus(path, scope string, scale int) error {
	db, err := open(path)
	if err != nil {
		return err
	}
	defer db.Close()
	ctx := context.Background()
	var count int
	if err := db.Primary().QueryRowContext(ctx, `SELECT COUNT(*) FROM entries WHERE scope=?`, scope).Scan(&count); err != nil {
		return err
	}
	if scale == 1 {
		return nil
	}
	target := count * scale
	var vectorBlob []byte
	if err := db.Primary().QueryRowContext(ctx, `SELECT em.vector_blob FROM entries e JOIN facet_texts ft ON ft.entry_id=e.id JOIN embeddings em ON em.facet_text_id=ft.id WHERE e.scope=? AND length(em.vector_blob)>0 ORDER BY e.id LIMIT 1`, scope).Scan(&vectorBlob); err != nil {
		return err
	}
	fillerVector, err := vectorcodec.Decode(vectorBlob)
	if err != nil {
		return err
	}
	// Exercise the same payload size without creating thousands of exact
	// matches for the fixed query embedding.
	for i, value := range fillerVector {
		fillerVector[i] = -value
	}
	vectorBlob = vectorcodec.Encode(fillerVector)
	var facet string
	if err := db.Primary().QueryRowContext(ctx, `SELECT facet_id FROM entries WHERE scope=? ORDER BY id LIMIT 1`, scope).Scan(&facet); err != nil {
		return err
	}
	tx, err := db.Primary().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for i := count; i < target; i++ {
		id := fmt.Sprintf("perf-scale:%08d", i)
		if _, err := tx.ExecContext(ctx, `INSERT INTO entries(id,scope,body,facet_id,kind,created_at) VALUES(?,?,?,?,?,?)`, id, scope, fmt.Sprintf("synthetic filler entry %d", i), facet, "performance-fixture", now); err != nil {
			return err
		}
		for _, kind := range []string{"topic", "rule", "entities"} {
			textID := id + ":" + kind
			if _, err := tx.ExecContext(ctx, `INSERT INTO facet_texts(id,entry_id,kind,text) VALUES(?,?,?,?)`, textID, id, kind, id); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO embeddings(id,facet_text_id,vector_json,vector_blob,created_at) VALUES(?,?,?,?,?)`, textID, textID, "", vectorBlob, now); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func measure(path, scope string, _ int) (measurement, error) {
	db, err := open(path)
	if err != nil {
		return measurement{}, err
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.EnsureSchemas(ctx, db.Primary(), database.SchemaProviderFunc(vectorcodec.Schema)); err != nil {
		return measurement{}, err
	}
	var entries, summaries int
	if err := db.Primary().QueryRowContext(ctx, `SELECT COUNT(*) FROM entries WHERE scope=?`, scope).Scan(&entries); err != nil {
		return measurement{}, err
	}
	if err := db.Primary().QueryRowContext(ctx, `SELECT COUNT(*) FROM summaries WHERE scope=?`, scope).Scan(&summaries); err != nil {
		return measurement{}, err
	}
	var vectorJSON string
	var vectorBlob []byte
	if err := db.Primary().QueryRowContext(ctx, `SELECT em.vector_json,em.vector_blob FROM entries e JOIN facet_texts ft ON ft.entry_id=e.id JOIN embeddings em ON em.facet_text_id=ft.id WHERE e.scope=? ORDER BY e.id LIMIT 1`, scope).Scan(&vectorJSON, &vectorBlob); err != nil {
		return measurement{}, err
	}
	var vector []float64
	if vectorJSON != "" && vectorJSON != "[]" {
		if err := json.Unmarshal([]byte(vectorJSON), &vector); err != nil {
			return measurement{}, err
		}
	} else if len(vectorBlob) > 0 {
		vector, err = vectorcodec.Decode(vectorBlob)
		if err != nil {
			return measurement{}, err
		}
	}
	readDSN, err := storage.SQLiteReadPoolDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		return measurement{}, err
	}
	reader, err := database.Connect(ctx, database.Config{
		Driver: database.DriverSQLite, DSN: readDSN,
		MaxOpenConns: database.DefaultReadMaxOpenConns,
		MaxIdleConns: database.DefaultReadMaxOpenConns,
	})
	if err != nil {
		return measurement{}, err
	}
	defer reader.Close()
	generations, err := vectorcodec.NewSQLiteGenerationServing(reader)
	if err != nil {
		return measurement{}, err
	}
	if _, active, err := generations.ActiveRecipeID(ctx); err != nil || active {
		return measurement{}, fmt.Errorf("fixed-vector benchmark requires column-based serving, active=%t, lookup error=%v", active, err)
	}
	service := recall.NewService(recall.NewSQLiteSource(reader, generations), fixedEmbedder{vector: vector}, recall.Config{})
	result := measurement{Scope: scope, Entries: entries, Summaries: summaries}
	for i := 0; i < 5; i++ {
		start := time.Now()
		recalled, err := service.Recall(ctx, "fixed recall performance fixture", 5)
		if err != nil {
			return measurement{}, err
		}
		result.SamplesSeconds = append(result.SamplesSeconds, time.Since(start).Seconds())
		if recalled.Lexical {
			result.LexicalFallbacks++
		}
		if i == 0 && len(recalled.Hits) > 0 {
			result.TopEntryID = recalled.Hits[0].Node.EntryID
		}
	}
	result.NodeCount, err = countNodes(ctx, db, scope)
	if err != nil {
		return measurement{}, err
	}
	sorted := append([]float64(nil), result.SamplesSeconds...)
	sort.Float64s(sorted)
	result.P50Seconds = sorted[len(sorted)/2]
	result.P95Seconds = sorted[len(sorted)-1]
	return result, nil
}

func countNodes(ctx context.Context, db *database.RoutedDB, scope string) (int, error) {
	var entries, summaries int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entries WHERE scope=?`, scope).Scan(&entries); err != nil {
		return 0, err
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM summaries WHERE scope=?`, scope).Scan(&summaries); err != nil {
		return 0, err
	}
	return entries + summaries, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
