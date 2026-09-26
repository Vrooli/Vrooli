package ledger

import (
	"bufio"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Importer is the read-only boundary over the marketing crew's JSONL store.
// It never retains offsets: each normalized item has a content-addressed key.
type Importer struct {
	db  SQLExecutor
	now func() time.Time
}

func NewImporter(db SQLExecutor) *Importer {
	return &Importer{db: db, now: func() time.Time { return time.Now().UTC() }}
}

func (i *Importer) Import(ctx context.Context, sources []ImportSource) (ImportResult, error) {
	result := ImportResult{RunID: uuid.NewString()}
	started := i.now().UTC()
	if _, err := i.db.ExecContext(ctx, `INSERT INTO ledger_import_runs (id, started_at, status, source_count) VALUES (?, ?, 'running', ?)`, result.RunID, started.Format(time.RFC3339Nano), len(sources)); err != nil {
		return result, fmt.Errorf("start import run: %w", err)
	}
	for _, source := range sources {
		imported, skipped, err := i.importSource(ctx, source)
		result.Imported += imported
		result.Skipped += skipped
		if err != nil {
			result.Failures = append(result.Failures, SourceFailure{Source: source.Name, Err: err})
		}
	}
	result.Complete = len(result.Failures) == 0
	status := "completed"
	if !result.Complete {
		status = "failed"
	}
	_, err := i.db.ExecContext(ctx, `UPDATE ledger_import_runs SET completed_at = ?, status = ?, failure_count = ? WHERE id = ?`, i.now().UTC().Format(time.RFC3339Nano), status, len(result.Failures), result.RunID)
	if err != nil {
		return result, fmt.Errorf("finish import run: %w", err)
	}
	return result, nil
}

func (i *Importer) importSource(ctx context.Context, source ImportSource) (int, int, error) {
	file, err := os.Open(source.Path)
	if err != nil {
		return 0, 0, fmt.Errorf("open %s: %w", source.Path, err)
	}
	defer file.Close()
	var imported, skipped int
	scanner := bufio.NewScanner(file)
	buffer := make([]byte, 0, 64*1024)
	scanner.Buffer(buffer, 4*1024*1024)
	for line := 1; scanner.Scan(); line++ {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		normalized, err := normalizeJSON(raw)
		if err != nil {
			return imported, skipped, fmt.Errorf("normalize %s line %d: %w", source.Name, line, err)
		}
		key := importKey(source, normalized)
		tx, err := i.db.BeginTx(ctx, nil)
		if err != nil {
			return imported, skipped, err
		}
		res, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO ledger_import_keys (import_key, source_name, source_path, imported_at) VALUES (?, ?, ?, ?)`, key, source.Name, source.Path, i.now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			_ = tx.Rollback()
			return imported, skipped, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			_ = tx.Rollback()
			return imported, skipped, err
		}
		if affected == 0 {
			_ = tx.Rollback()
			skipped++
			continue
		}
		if err := i.persistImported(ctx, tx, source.Name, key, normalized); err != nil {
			_ = tx.Rollback()
			return imported, skipped, err
		}
		if err := tx.Commit(); err != nil {
			return imported, skipped, err
		}
		imported++
	}
	if err := scanner.Err(); err != nil {
		return imported, skipped, fmt.Errorf("read %s: %w", source.Name, err)
	}
	return imported, skipped, nil
}

func (i *Importer) persistImported(ctx context.Context, tx *sql.Tx, source, key, normalized string) error {
	var item map[string]any
	if err := json.Unmarshal([]byte(normalized), &item); err != nil {
		return err
	}
	now := i.now().UTC().Format(time.RFC3339Nano)
	switch source {
	case "publish-log":
		_, err := tx.ExecContext(ctx, `INSERT INTO ledger_publish_records (id, import_key, draft_id, channel, audience, published_url, platform_post_id, source_kind, published_at, payload_json) VALUES (?, ?, ?, ?, ?, ?, ?, 'imported', ?, ?)`, uuid.NewString(), key, nullable(stringField(item, "draft_ref")), stringField(item, "channel"), stringField(item, "audience"), stringField(item, "post_url"), stringField(item, "post_id"), timeField(item, "at", now), normalized)
		return err
	case "published-scenario-mentions":
		_, err := tx.ExecContext(ctx, `INSERT INTO ledger_subject_mentions (id, import_key, subject, subject_kind, audience, channel, draft_ref, post_url, post_id, is_first_mention, occurred_at, payload_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), key, stringField(item, "subject"), stringField(item, "subject_kind"), stringField(item, "audience"), stringField(item, "channel"), nullable(stringField(item, "draft_ref")), nullable(stringField(item, "post_url")), nullable(stringField(item, "post_id")), boolInt(item["is_first_mention"]), timeField(item, "at", now), normalized)
		return err
	case "published-improvements-log":
		_, err := tx.ExecContext(ctx, `INSERT INTO ledger_narrated_items (id, import_key, subject, scenario, occurred_at, payload_json) VALUES (?, ?, ?, ?, ?, ?)`, uuid.NewString(), key, stringField(item, "subject"), stringField(item, "scenario"), timeField(item, "at", now), normalized)
		return err
	default:
		return nil // campaign drafts and audience scans are deliberately retained by their source until their owning domains consume them.
	}
}

func normalizeJSON(raw string) (string, error) {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return "", err
	}
	normalized, err := json.Marshal(value)
	return string(normalized), err
}

func importKey(source ImportSource, normalized string) string {
	sum := sha256.Sum256([]byte(source.Name + "\x00" + source.Path + "\x00" + normalized))
	return hex.EncodeToString(sum[:])
}

func stringField(item map[string]any, key string) string {
	value, _ := item[key].(string)
	return value
}

func boolInt(value any) int {
	b, _ := value.(bool)
	if b {
		return 1
	}
	return 0
}

func timeField(item map[string]any, key, fallback string) string {
	value := stringField(item, key)
	if value == "" {
		return fallback
	}
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return fallback
	}
	return value
}
