package retrieval

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"math"
)

type (
	queryer interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	}
	SQLiteRepository struct{ db queryer }
)

func NewSQLiteRepository(db queryer) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) AddUnit(ctx context.Context, u Unit) error {
	if _, err := r.db.ExecContext(ctx, `INSERT OR REPLACE INTO retrieval_units (id, collection_id, document_hash, privacy_class, text, anchor_uri) VALUES (?, ?, ?, ?, ?, ?)`, u.ID, u.CollectionID, u.DocumentHash, u.PrivacyClass, u.Text, u.AnchorURI); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `INSERT OR REPLACE INTO retrieval_fts (unit_id, text) VALUES (?, ?)`, u.ID, u.Text)
	return err
}

func (r *SQLiteRepository) AddVector(ctx context.Context, unitID string, vector []float32) error {
	buf := bytes.NewBuffer(make([]byte, 0, len(vector)*4))
	for _, v := range vector {
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	_, err := r.db.ExecContext(ctx, `INSERT OR REPLACE INTO retrieval_vectors (unit_id, dimension, vector) VALUES (?, ?, ?)`, unitID, len(vector), buf.Bytes())
	return err
}

func (r *SQLiteRepository) Candidates(ctx context.Context, q Query) ([]Unit, error) {
	privacy := q.CallerMaxPrivacy
	if privacy == 0 {
		privacy = 1
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, collection_id, document_hash, privacy_class, text, anchor_uri FROM retrieval_units WHERE (? = '' OR collection_id = ?) AND privacy_class <= ? AND (? = 0 OR collection_id IN (SELECT id FROM collections WHERE federated = 1)) ORDER BY id`, q.CollectionID, q.CollectionID, privacy, q.Federated)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Unit
	for rows.Next() {
		var u Unit
		if err := rows.Scan(&u.ID, &u.CollectionID, &u.DocumentHash, &u.PrivacyClass, &u.Text, &u.AnchorURI); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

type SQLiteVectorStore struct{ db queryer }

func NewSQLiteVectorStore(db queryer) *SQLiteVectorStore { return &SQLiteVectorStore{db: db} }
func (s *SQLiteVectorStore) Similar(ctx context.Context, query []float32, candidates []Unit, limit int) map[string]float64 {
	allowed := make(map[string]struct{}, len(candidates))
	for _, c := range candidates {
		allowed[c.ID] = struct{}{}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT unit_id, vector FROM retrieval_vectors`)
	if err != nil {
		return map[string]float64{}
	}
	defer rows.Close()
	type pair struct {
		id    string
		score float64
	}
	var scores []pair
	for rows.Next() {
		var id string
		var raw []byte
		if rows.Scan(&id, &raw) != nil {
			continue
		}
		if _, ok := allowed[id]; !ok || len(raw)%4 != 0 {
			continue
		}
		vector := make([]float32, len(raw)/4)
		for i := range vector {
			_ = binary.Read(bytes.NewReader(raw[i*4:(i+1)*4]), binary.LittleEndian, &vector[i])
		}
		scores = append(scores, pair{id, cosine(query, vector)})
	}
	for i := range scores {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
	if limit <= 0 {
		limit = len(scores)
	}
	if len(scores) > limit {
		scores = scores[:limit]
	}
	out := make(map[string]float64, len(scores))
	for _, p := range scores {
		out[p.id] = p.score
	}
	return out
}

func cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, an, bn float64
	for i := range a {
		dot += float64(a[i] * b[i])
		an += float64(a[i] * a[i])
		bn += float64(b[i] * b[i])
	}
	if an == 0 || bn == 0 {
		return 0
	}
	return dot / (math.Sqrt(an) * math.Sqrt(bn))
}
