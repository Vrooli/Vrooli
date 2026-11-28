package retrieval

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
)

// SQLiteVectorStore stores generation-scoped dense vectors in the same
// database as Code Facts' source-free search index. Binary float32 encoding
// avoids JSON expansion and foreign keys remove vectors with retired catalog
// generations.
type SQLiteVectorStore struct{ db *sql.DB }

func NewSQLiteVectorStore(db *sql.DB) *SQLiteVectorStore { return &SQLiteVectorStore{db: db} }

func (s *SQLiteVectorStore) Upsert(ctx context.Context, records []VectorRecord) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("semantic vector store requires database")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, record := range records {
		blob := encodeVector(record.Vector)
		if len(record.Vector) == 0 {
			return fmt.Errorf("semantic vector %q is empty", record.ID)
		}
		documentID := record.Payload["document_id"]
		if documentID == "" {
			return fmt.Errorf("semantic vector %q has no source document id", record.ID)
		}
		model := record.Payload["model"]
		if model == "" {
			return fmt.Errorf("semantic vector %q has no resolved embedding model", record.ID)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO code_facts_semantic_vectors(generation_id,id,document_id,model,source_hash,dimensions,vector) VALUES(?,?,?,?,?,?,?) ON CONFLICT(generation_id,id) DO UPDATE SET document_id=excluded.document_id,model=excluded.model,source_hash=excluded.source_hash,dimensions=excluded.dimensions,vector=excluded.vector`, record.Generation, record.ID, documentID, model, record.SourceHash, len(record.Vector), blob); err != nil {
			return fmt.Errorf("upsert semantic vector %q: %w", record.ID, err)
		}
	}
	return tx.Commit()
}

func (s *SQLiteVectorStore) Delete(ctx context.Context, ids []string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("semantic vector store requires database")
	}
	for _, id := range ids {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM code_facts_semantic_vectors WHERE id=?`, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteVectorStore) Query(ctx context.Context, vector []float32, query Query) ([]Candidate, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("semantic vector store requires database")
	}
	if len(vector) == 0 {
		return nil, fmt.Errorf("semantic query vector is empty")
	}
	if query.EmbeddingModel != "" {
		var storedModel string
		err := s.db.QueryRowContext(ctx, `SELECT model FROM code_facts_semantic_vectors WHERE generation_id=? LIMIT 1`, query.Generation).Scan(&storedModel)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if err == nil && storedModel != query.EmbeddingModel {
			return nil, fmt.Errorf("generation %q uses embedding model %q; current role resolves to %q", query.Generation, storedModel, query.EmbeddingModel)
		}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT d.id,d.path,d.title,d.body,d.source_hash,d.generation_id,d.role,d.language,d.scope,d.authority,d.kind,d.start_line,d.end_line,v.vector,v.dimensions FROM code_facts_semantic_vectors v JOIN code_facts_search_documents d ON d.generation_id=v.generation_id AND d.id=v.document_id WHERE v.generation_id=?`, query.Generation)
	if err != nil {
		return nil, fmt.Errorf("query semantic vectors: %w", err)
	}
	defer rows.Close()
	type scored struct {
		candidate Candidate
		score     float64
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	scoredRows := make([]scored, 0, limit*2)
	for rows.Next() {
		var c Candidate
		var blob []byte
		var dims int
		if err := rows.Scan(&c.ID, &c.Path, &c.Title, &c.Text, &c.SourceHash, &c.Generation, &c.Role, &c.Language, &c.Scope, &c.Authority, &c.Kind, &c.StartLine, &c.EndLine, &blob, &dims); err != nil {
			return nil, err
		}
		if dims != len(vector) || len(blob) != dims*4 {
			continue
		}
		if len(query.Roles) > 0 && !contains(query.Roles, c.Role) || len(query.Languages) > 0 && !contains(query.Languages, c.Language) || len(query.Target) > 0 && c.Path != query.Target && !hasPathPrefix(c.Path, query.Target) || len(query.Scope) > 0 && c.Scope != query.Scope {
			continue
		}
		score := cosine(vector, decodeVector(blob))
		if math.IsNaN(score) {
			continue
		}
		c.Score, c.Regime, c.Explanation = score, RegimeNatural, "local dense vector cosine similarity"
		scoredRows = append(scoredRows, scored{candidate: c, score: score})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(scoredRows, func(i, j int) bool {
		if scoredRows[i].score == scoredRows[j].score {
			return scoredRows[i].candidate.ID < scoredRows[j].candidate.ID
		}
		return scoredRows[i].score > scoredRows[j].score
	})
	if len(scoredRows) > limit {
		scoredRows = scoredRows[:limit]
	}
	result := make([]Candidate, len(scoredRows))
	for i, row := range scoredRows {
		result[i] = row.candidate
	}
	return result, nil
}

func encodeVector(vector []float32) []byte {
	out := make([]byte, len(vector)*4)
	for i, v := range vector {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(v))
	}
	return out
}
func decodeVector(blob []byte) []float32 {
	out := make([]float32, len(blob)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[i*4:]))
	}
	return out
}
func cosine(a, b []float32) float64 {
	var dot, aa, bb float64
	for i := range a {
		av, bv := float64(a[i]), float64(b[i])
		dot += av * bv
		aa += av * av
		bb += bv * bv
	}
	if aa == 0 || bb == 0 {
		return 0
	}
	return dot / math.Sqrt(aa*bb)
}
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
func hasPathPrefix(path, prefix string) bool {
	return len(path) > len(prefix) && path[:len(prefix)] == prefix && path[len(prefix)] == '/'
}

var _ VectorStore = (*SQLiteVectorStore)(nil)
