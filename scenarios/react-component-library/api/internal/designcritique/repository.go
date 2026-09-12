package designcritique

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"
)

//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }

var ErrConflict = errors.New("critique idempotency conflict")

type Record struct {
	ID         string     `json:"id"`
	Hash       string     `json:"hash"`
	Review     Review     `json:"review"`
	Assessment Assessment `json:"assessment"`
	RecordedAt string     `json:"recordedAt"`
}
type Repository interface {
	Create(context.Context, string, Review, Assessment) (Record, error)
	Get(context.Context, string) (Record, error)
	List(context.Context, Target, string) ([]Record, string, error)
}
type SQLiteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }
func identity(review Review) (string, []byte, error) {
	if err := Validate(review); err != nil {
		return "", nil, err
	}
	raw, err := json.Marshal(review)
	if err != nil {
		return "", nil, err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), raw, nil
}
func (r *SQLiteRepository) Create(ctx context.Context, key string, review Review, assessment Assessment) (Record, error) {
	if !bounded(key, 200) {
		return Record{}, fmt.Errorf("critique idempotency key is required")
	}
	if !reflect.DeepEqual(aggregate(review), assessment) {
		return Record{}, fmt.Errorf("critique assessment differs from rubric aggregation")
	}
	hash, raw, err := identity(review)
	if err != nil {
		return Record{}, err
	}
	result, err := json.Marshal(assessment)
	if err != nil {
		return Record{}, err
	}
	digest := sha256.Sum256([]byte(key))
	id := "critique_" + hex.EncodeToString(digest[:])
	_, err = r.db.ExecContext(ctx, `INSERT INTO design_critiques(id,review_hash,review_json,assessment_json,recorded_at) VALUES(?,?,?,?,?) ON CONFLICT(id) DO NOTHING`, id, hash, string(raw), string(result), time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z"))
	if err != nil {
		return Record{}, err
	}
	record, err := r.Get(ctx, id)
	if err != nil {
		return record, err
	}
	if record.Hash != hash {
		return Record{}, ErrConflict
	}
	return record, nil
}
func (r *SQLiteRepository) Get(ctx context.Context, id string) (Record, error) {
	var record Record
	var raw, result string
	err := r.db.QueryRowContext(ctx, `SELECT id,review_hash,review_json,assessment_json,recorded_at FROM design_critiques WHERE id=?`, id).Scan(&record.ID, &record.Hash, &raw, &result, &record.RecordedAt)
	if err != nil {
		return record, err
	}
	return decodeRecord(record, raw, result)
}
func decodeRecord(record Record, raw, result string) (Record, error) {
	var err error
	if err = json.Unmarshal([]byte(raw), &record.Review); err != nil {
		return Record{}, err
	}
	if err = json.Unmarshal([]byte(result), &record.Assessment); err != nil {
		return Record{}, err
	}
	hash, _, err := identity(record.Review)
	if err != nil {
		return Record{}, err
	}
	if record.Hash != hash || !reflect.DeepEqual(aggregate(record.Review), record.Assessment) {
		return Record{}, fmt.Errorf("persisted critique integrity mismatch")
	}
	return record, nil
}

type Service struct {
	Repository Repository
	Verifier   EvidenceVerifier
}

func (s Service) Record(ctx context.Context, key string, review Review) (Record, error) {
	if s.Repository == nil {
		return Record{}, fmt.Errorf("critique repository unavailable")
	}
	if !bounded(key, 200) {
		return Record{}, fmt.Errorf("critique idempotency key is required")
	}
	hash, _, err := identity(review)
	if err != nil {
		return Record{}, err
	}
	digest := sha256.Sum256([]byte(key))
	existing, err := s.Repository.Get(ctx, "critique_"+hex.EncodeToString(digest[:]))
	if err == nil {
		if existing.Hash != hash {
			return Record{}, ErrConflict
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Record{}, err
	}
	verifyCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	assessment, err := Evaluate(verifyCtx, review, s.Verifier)
	if err != nil {
		return Record{}, err
	}
	return s.Repository.Create(ctx, key, review, assessment)
}

// List returns a bounded page for one exact design revision and render.
// Every returned record is integrity-checked before exposing its summary.
func (r *SQLiteRepository) List(ctx context.Context, target Target, before string) ([]Record, string, error) {
	if !bounded(target.Scenario, 128) || !bounded(target.DesignID, 128) || !hashPattern.MatchString(target.Revision) || !hashPattern.MatchString(target.RenderHash) {
		return nil, "", fmt.Errorf("exact critique target is required")
	}
	beforeTime := ""
	if before != "" {
		cursor, err := r.Get(ctx, before)
		if err != nil {
			return nil, "", err
		}
		if cursor.Review.Target != target {
			return nil, "", fmt.Errorf("critique cursor belongs to another target")
		}
		beforeTime = cursor.RecordedAt
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,review_hash,review_json,assessment_json,recorded_at FROM design_critiques
 WHERE json_extract(review_json,'$.target.scenario')=? AND json_extract(review_json,'$.target.designId')=?
 AND json_extract(review_json,'$.target.revision')=? AND json_extract(review_json,'$.target.renderHash')=?
 AND (?='' OR recorded_at<? OR (recorded_at=? AND id<?))
 ORDER BY recorded_at DESC,id DESC LIMIT 21`, target.Scenario, target.DesignID, target.Revision, target.RenderHash, before, beforeTime, beforeTime, before)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	records := []Record{}
	for rows.Next() {
		var record Record
		var raw, result string
		if err := rows.Scan(&record.ID, &record.Hash, &raw, &result, &record.RecordedAt); err != nil {
			return nil, "", err
		}
		verified, err := decodeRecord(record, raw, result)
		if err != nil {
			return nil, "", err
		}
		if verified.Review.Target != target {
			return nil, "", fmt.Errorf("listed critique target differs from requested target")
		}
		records = append(records, verified)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(records) > 20 {
		records = records[:20]
		next = records[len(records)-1].ID
	}
	return records, next, nil
}
