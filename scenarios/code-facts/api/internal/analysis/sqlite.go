package analysis

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type SQLiteProjectionStore struct{ db *sql.DB }

func NewSQLiteProjectionStore(db *sql.DB) *SQLiteProjectionStore {
	return &SQLiteProjectionStore{db: db}
}

func (store *SQLiteProjectionStore) Replace(ctx context.Context, generation, sourceFileID string, facts []Fact) error {
	if store == nil || store.db == nil {
		return fmt.Errorf("graph projection store requires database")
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var state string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM code_facts_generations WHERE id=?`, generation).Scan(&state); err != nil {
		return fmt.Errorf("read graph generation: %w", err)
	}
	if state != "shadow" {
		return fmt.Errorf("graph generation %q is %s, want shadow", generation, state)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM code_facts_graph_facts WHERE generation_id=? AND source_file_id=?`, generation, sourceFileID); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO code_facts_graph_facts(
 generation_id,id,source_file_id,subject_id,predicate,object_id,proof_status,source_hash,metadata_json
) VALUES(?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, fact := range facts {
		if strings.TrimSpace(fact.ID) == "" || strings.TrimSpace(fact.Subject) == "" || strings.TrimSpace(fact.Predicate) == "" || strings.TrimSpace(fact.Object) == "" || strings.TrimSpace(fact.SourceHash) == "" || strings.TrimSpace(fact.Proof) == "" || strings.TrimSpace(fact.Analyzer) == "" {
			return fmt.Errorf("graph fact requires id, subject, predicate, object, source hash, proof, and analyzer")
		}
		metadata := map[string]any{"family": fact.Family, "kind": fact.Kind, "path": fact.Path, "analyzer": fact.Analyzer, "analyzer_version": fact.Version, "attributes": fact.Attributes}
		payload, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		if _, err := stmt.ExecContext(ctx, generation, fact.ID, sourceFileID, fact.Subject, fact.Predicate, fact.Object, fact.Proof, fact.SourceHash, string(payload)); err != nil {
			return fmt.Errorf("insert graph fact %q: %w", fact.ID, err)
		}
	}
	return tx.Commit()
}

func (store *SQLiteProjectionStore) Expand(ctx context.Context, generation, subject string, families []string, limit int) ([]Fact, error) {
	if store == nil || store.db == nil {
		return nil, fmt.Errorf("graph projection store requires database")
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	familyCSV := strings.Join(families, ",")
	rows, err := store.db.QueryContext(ctx, `
SELECT gf.id,gf.subject_id,gf.predicate,gf.object_id,gf.proof_status,gf.source_hash,gf.metadata_json
FROM code_facts_graph_facts gf
JOIN code_facts_source_files sf ON sf.generation_id=gf.generation_id AND sf.id=gf.source_file_id AND sf.content_hash=gf.source_hash
JOIN code_facts_generations g ON g.id=gf.generation_id
WHERE gf.generation_id=? AND gf.subject_id=? AND g.state IN ('active','shadow')
  AND (?='' OR instr(','||?||',',','||json_extract(gf.metadata_json,'$.family')||',')>0)
ORDER BY gf.predicate,gf.object_id,gf.id LIMIT ?`, generation, subject, familyCSV, familyCSV, limit)
	if err != nil {
		return nil, fmt.Errorf("expand graph projection: %w", err)
	}
	defer rows.Close()
	var facts []Fact
	for rows.Next() {
		var fact Fact
		var metadataJSON string
		if err := rows.Scan(&fact.ID, &fact.Subject, &fact.Predicate, &fact.Object, &fact.Proof, &fact.SourceHash, &metadataJSON); err != nil {
			return nil, err
		}
		var metadata struct {
			Family     string            `json:"family"`
			Kind       string            `json:"kind"`
			Path       string            `json:"path"`
			Analyzer   string            `json:"analyzer"`
			Version    string            `json:"analyzer_version"`
			Attributes map[string]string `json:"attributes"`
		}
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			return nil, err
		}
		fact.Family, fact.Kind, fact.Path, fact.Analyzer, fact.Version, fact.Attributes = metadata.Family, metadata.Kind, metadata.Path, metadata.Analyzer, metadata.Version, metadata.Attributes
		facts = append(facts, fact)
	}
	return facts, rows.Err()
}

var _ ProjectionStore = (*SQLiteProjectionStore)(nil)
