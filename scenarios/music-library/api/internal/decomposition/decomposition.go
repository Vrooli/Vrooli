// Package decomposition stores Music Tools' per-track analysis and the
// listener's timestamped review. It never performs audio analysis itself.
package decomposition

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
)

//go:embed schema.sql
var schema string

func Schema() string { return schema }

var (
	ErrInvalid  = errors.New("invalid decomposition")
	ErrNotFound = errors.New("decomposition not found")
	ErrConflict = errors.New("analysis job already has a different result")
	hashRE      = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type Result struct {
	TrackSHA256  string          `json:"track_sha256"`
	AnalysisJob  string          `json:"analysis_job_id"`
	SourceSHA256 string          `json:"source_sha256"`
	Source       SourceMetadata  `json:"source,omitempty"`
	ToolResult   json.RawMessage `json:"tool_result"`
	ReceivedAt   time.Time       `json:"received_at"`
	Reviews      []Review        `json:"reviews,omitempty"`
}

// SourceMetadata identifies the private source without copying audio into Library.
type SourceMetadata struct {
	Title     string `json:"title"`
	Artist    string `json:"artist,omitempty"`
	Uploader  string `json:"uploader,omitempty"`
	FileName  string `json:"file_name"`
	MediaType string `json:"media_type"`
	SizeBytes int64  `json:"size_bytes"`
	Acquired  string `json:"acquired_at,omitempty"`
}

// Correction represents a listener's correction to a specific time range.
// End is exclusive, in seconds from the start of the source track.
type Correction struct {
	Start   float64 `json:"start_seconds"`
	End     float64 `json:"end_seconds"`
	Label   string  `json:"label"`
	Comment string  `json:"comment,omitempty"`
}

type Review struct {
	ID          string       `json:"review_id"`
	TrackSHA256 string       `json:"track_sha256"`
	AnalysisJob string       `json:"analysis_job_id"`
	Verdict     string       `json:"verdict"`
	Note        string       `json:"note"`
	Corrections []Correction `json:"corrections"`
	CreatedAt   time.Time    `json:"created_at"`
}

// TasteFeature is a compact, hash-bound acoustic descriptor materialized only
// after an operator confirms a specific analysis result.
type TasteFeature struct {
	Attribute  string
	Value      float64
	Confidence float64
	Source     string
}

type Summary struct {
	TrackSHA256 string    `json:"track_sha256"`
	AnalysisJob string    `json:"analysis_job_id"`
	Title       string    `json:"title,omitempty"`
	Artist      string    `json:"artist,omitempty"`
	Uploader    string    `json:"uploader,omitempty"`
	ReceivedAt  time.Time `json:"received_at"`
	ReviewCount int       `json:"review_count"`
}

type Page struct {
	Items      []Summary `json:"items"`
	NextCursor string    `json:"next_cursor,omitempty"`
}

type pageCursor struct {
	ReceivedAt  string `json:"received_at"`
	TrackSHA    string `json:"track_sha256"`
	AnalysisJob string `json:"analysis_job_id"`
}

type Store struct{ db database.SQLTxExecutor }

func NewStore(db database.SQLTxExecutor) *Store { return &Store{db: db} }

func (s *Store) Put(ctx context.Context, result Result) (Result, error) {
	if !hashRE.MatchString(result.TrackSHA256) || result.TrackSHA256 != result.SourceSHA256 || strings.TrimSpace(result.AnalysisJob) == "" || len(result.AnalysisJob) > 160 {
		return Result{}, fmt.Errorf("%w: track/source SHA-256 and analysis job are required", ErrInvalid)
	}
	if len(result.ToolResult) == 0 || len(result.ToolResult) > 64<<20 || !json.Valid(result.ToolResult) {
		return Result{}, fmt.Errorf("%w: tool_result must be valid JSON up to 64 MiB", ErrInvalid)
	}
	var envelope struct {
		Operation    string `json:"operation"`
		JobID        string `json:"job_id"`
		SourceSHA256 string `json:"source_sha256"`
		Resource     string `json:"resource"`
	}
	if err := json.Unmarshal(result.ToolResult, &envelope); err != nil || envelope.JobID != result.AnalysisJob || envelope.SourceSHA256 != result.SourceSHA256 || envelope.Resource != "music-mir" || (envelope.Operation != "analyze" && envelope.Operation != "separate") {
		return Result{}, fmt.Errorf("%w: result provenance must match job/source and identify music-mir", ErrInvalid)
	}
	if result.Source.Title != "" {
		if err := validateSourceMetadata(result.Source); err != nil {
			return Result{}, err
		}
	}
	now := time.Now().UTC()
	insert, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO music_decompositions (track_sha256,analysis_job_id,source_sha256,result_json,received_at) VALUES (?,?,?,?,?)`, result.TrackSHA256, result.AnalysisJob, result.SourceSHA256, string(result.ToolResult), now.Format(time.RFC3339Nano))
	if err != nil {
		return Result{}, fmt.Errorf("store Music Tools decomposition: %w", err)
	}
	inserted, err := insert.RowsAffected()
	if err != nil {
		return Result{}, err
	}
	if inserted == 0 {
		var existing, received string
		if err := s.db.QueryRowContext(ctx, `SELECT result_json,received_at FROM music_decompositions WHERE track_sha256=? AND analysis_job_id=?`, result.TrackSHA256, result.AnalysisJob).Scan(&existing, &received); err != nil {
			return Result{}, err
		}
		if existing != string(result.ToolResult) {
			return Result{}, ErrConflict
		}
		result.ReceivedAt, err = time.Parse(time.RFC3339Nano, received)
		if err != nil {
			return Result{}, err
		}
	} else {
		result.ReceivedAt = now
	}
	if result.Source.Title != "" {
		_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO music_decomposition_sources (track_sha256,title,artist,uploader,file_name,media_type,size_bytes,acquired_at) VALUES (?,?,?,?,?,?,?,?)`, result.TrackSHA256, result.Source.Title, result.Source.Artist, result.Source.Uploader, result.Source.FileName, result.Source.MediaType, result.Source.SizeBytes, result.Source.Acquired)
		if err != nil {
			return Result{}, fmt.Errorf("store source metadata: %w", err)
		}
	}
	return result, nil
}

func validateSourceMetadata(source SourceMetadata) error {
	if strings.TrimSpace(source.Title) == "" || len(source.Title) > 512 || len(source.Artist) > 256 || len(source.Uploader) > 256 || len(source.FileName) > 160 || strings.ContainsAny(source.FileName, "/\\\r\n") || !allowedSourceMediaType(source.MediaType) || source.SizeBytes <= 0 || source.SizeBytes > 128<<20 {
		return fmt.Errorf("%w: source metadata is invalid", ErrInvalid)
	}
	return nil
}

func allowedSourceMediaType(value string) bool {
	switch value {
	case "audio/mp4", "audio/aac", "audio/mpeg", "audio/wav", "audio/flac", "audio/ogg", "audio/wave", "audio/x-wav":
		return true
	default:
		return false
	}
}

func (s *Store) Get(ctx context.Context, trackSHA, jobID string) (Result, error) {
	var out Result
	var raw, received string
	query := `SELECT track_sha256,analysis_job_id,source_sha256,result_json,received_at FROM music_decompositions WHERE track_sha256=?`
	args := []any{trackSHA}
	if jobID != "" {
		query += ` AND analysis_job_id=?`
		args = append(args, jobID)
	} else {
		query += ` ORDER BY received_at DESC LIMIT 1`
	}
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&out.TrackSHA256, &out.AnalysisJob, &out.SourceSHA256, &raw, &received)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, ErrNotFound
	}
	if err != nil {
		return Result{}, err
	}
	if err := json.Unmarshal([]byte(raw), &out.ToolResult); err != nil {
		return Result{}, err
	}
	if out.ReceivedAt, err = time.Parse(time.RFC3339Nano, received); err != nil {
		return Result{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT title,artist,uploader,file_name,media_type,size_bytes,acquired_at FROM music_decomposition_sources WHERE track_sha256=?`, out.TrackSHA256).Scan(&out.Source.Title, &out.Source.Artist, &out.Source.Uploader, &out.Source.FileName, &out.Source.MediaType, &out.Source.SizeBytes, &out.Source.Acquired); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Result{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT review_id,verdict,note,corrections_json,created_at FROM music_decomposition_reviews WHERE track_sha256=? AND analysis_job_id=? ORDER BY created_at,review_id`, out.TrackSHA256, out.AnalysisJob)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var review Review
		var corrections, created string
		review.TrackSHA256, review.AnalysisJob = out.TrackSHA256, out.AnalysisJob
		if err := rows.Scan(&review.ID, &review.Verdict, &review.Note, &corrections, &created); err != nil {
			return Result{}, err
		}
		if err := json.Unmarshal([]byte(corrections), &review.Corrections); err != nil {
			return Result{}, err
		}
		if review.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			return Result{}, err
		}
		out.Reviews = append(out.Reviews, review)
	}
	return out, rows.Err()
}

// ListSources loads only lightweight source metadata for the requested exact
// hashes. Recommendation reads must not deserialize large analysis envelopes
// or open audio simply to obtain artist attribution.
func (s *Store) ListSources(ctx context.Context, trackHashes []string) (map[string]SourceMetadata, error) {
	out := make(map[string]SourceMetadata, len(trackHashes))
	for start := 0; start < len(trackHashes); start += 500 {
		end := min(start+500, len(trackHashes))
		batch := trackHashes[start:end]
		if len(batch) == 0 {
			continue
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")
		args := make([]any, len(batch))
		for i, hash := range batch {
			args[i] = hash
		}
		query := `SELECT track_sha256,title,artist,uploader,file_name,media_type,size_bytes,acquired_at FROM music_decomposition_sources WHERE track_sha256 IN (` + placeholders + `)`
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var hash string
			var source SourceMetadata
			if err := rows.Scan(&hash, &source.Title, &source.Artist, &source.Uploader, &source.FileName, &source.MediaType, &source.SizeBytes, &source.Acquired); err != nil {
				rows.Close()
				return nil, err
			}
			out[hash] = source
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// List returns light summaries in a stable newest-first order. The opaque
// cursor makes discovery independent of known reference hashes and avoids
// loading large analysis documents for the review queue.
func (s *Store) List(ctx context.Context, limit int, cursor string) (Page, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	query := `SELECT d.track_sha256,d.analysis_job_id,d.received_at,COUNT(r.review_id),COALESCE(s.title,''),COALESCE(s.artist,''),COALESCE(s.uploader,'') FROM music_decompositions d LEFT JOIN music_decomposition_reviews r ON r.track_sha256=d.track_sha256 AND r.analysis_job_id=d.analysis_job_id LEFT JOIN music_decomposition_sources s ON s.track_sha256=d.track_sha256`
	args := []any{}
	if cursor != "" {
		if len(cursor) > 1024 {
			return Page{}, fmt.Errorf("%w: pagination cursor is too long", ErrInvalid)
		}
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return Page{}, fmt.Errorf("%w: invalid pagination cursor", ErrInvalid)
		}
		var c pageCursor
		if err := json.Unmarshal(raw, &c); err != nil || c.ReceivedAt == "" || !hashRE.MatchString(c.TrackSHA) || strings.TrimSpace(c.AnalysisJob) == "" {
			return Page{}, fmt.Errorf("%w: invalid pagination cursor", ErrInvalid)
		}
		query += ` WHERE (d.received_at < ? OR (d.received_at = ? AND d.track_sha256 < ?) OR (d.received_at = ? AND d.track_sha256 = ? AND d.analysis_job_id < ?))`
		args = append(args, c.ReceivedAt, c.ReceivedAt, c.TrackSHA, c.ReceivedAt, c.TrackSHA, c.AnalysisJob)
	}
	query += ` GROUP BY d.track_sha256,d.analysis_job_id,d.received_at,s.title,s.artist,s.uploader ORDER BY d.received_at DESC,d.track_sha256 DESC,d.analysis_job_id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()
	items := make([]Summary, 0, limit+1)
	for rows.Next() {
		var item Summary
		var received string
		if err := rows.Scan(&item.TrackSHA256, &item.AnalysisJob, &received, &item.ReviewCount, &item.Title, &item.Artist, &item.Uploader); err != nil {
			return Page{}, err
		}
		if item.ReceivedAt, err = time.Parse(time.RFC3339Nano, received); err != nil {
			return Page{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	page := Page{Items: items}
	if len(items) > limit {
		last := items[limit-1]
		page.Items = items[:limit]
		c, err := json.Marshal(pageCursor{ReceivedAt: last.ReceivedAt.Format(time.RFC3339Nano), TrackSHA: last.TrackSHA256, AnalysisJob: last.AnalysisJob})
		if err != nil {
			return Page{}, err
		}
		page.NextCursor = base64.RawURLEncoding.EncodeToString(c)
	}
	return page, nil
}

func (s *Store) Review(ctx context.Context, review Review) (Review, error) {
	if review.ID == "" || len(review.ID) > 160 || !hashRE.MatchString(review.TrackSHA256) || strings.TrimSpace(review.AnalysisJob) == "" || len(review.AnalysisJob) > 160 || (review.Verdict != "confirmed" && review.Verdict != "corrected" && review.Verdict != "needs_work") || len(review.Note) > 4000 || len(review.Corrections) > 500 {
		return Review{}, fmt.Errorf("%w: invalid review fields", ErrInvalid)
	}
	for _, c := range review.Corrections {
		if c.Start < 0 || c.End <= c.Start || c.End-c.Start > 24*60*60 || strings.TrimSpace(c.Label) == "" || len(c.Label) > 240 || len(c.Comment) > 2000 {
			return Review{}, fmt.Errorf("%w: timestamp corrections need a positive bounded range and label", ErrInvalid)
		}
	}
	features := []TasteFeature(nil)
	if review.Verdict == "confirmed" {
		result, err := s.Get(ctx, review.TrackSHA256, review.AnalysisJob)
		if err != nil {
			return Review{}, err
		}
		features, err = extractTasteFeatures(result)
		if err != nil {
			return Review{}, err
		}
	}
	b, err := json.Marshal(review.Corrections)
	if err != nil {
		return Review{}, err
	}
	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Review{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO music_decomposition_reviews (review_id,track_sha256,analysis_job_id,verdict,note,corrections_json,created_at) VALUES (?,?,?,?,?,?,?)`, review.ID, review.TrackSHA256, review.AnalysisJob, review.Verdict, review.Note, string(b), now.Format(time.RFC3339Nano))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "foreign key") {
			return Review{}, ErrNotFound
		}
		return Review{}, fmt.Errorf("save decomposition review: %w", err)
	}
	for _, feature := range features {
		_, err = tx.ExecContext(ctx, `INSERT INTO music_decomposition_taste_features (track_sha256,analysis_job_id,review_id,attribute,value,confidence,evidence_source,created_at) VALUES (?,?,?,?,?,?,?,?)`, review.TrackSHA256, review.AnalysisJob, review.ID, feature.Attribute, feature.Value, feature.Confidence, feature.Source, now.Format(time.RFC3339Nano))
		if err != nil {
			return Review{}, fmt.Errorf("materialize confirmed taste feature: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Review{}, err
	}
	review.CreatedAt = now
	return review, nil
}

func extractTasteFeatures(result Result) ([]TasteFeature, error) {
	var envelope struct {
		JobID        string `json:"job_id"`
		SourceSHA256 string `json:"source_sha256"`
		Resource     string `json:"resource"`
		Operation    string `json:"operation"`
		Analysis     struct {
			SourceSHA256 string `json:"source_sha256"`
			Media        struct {
				DurationSeconds float64 `json:"duration_seconds"`
			} `json:"media"`
			Beat struct {
				TempoBPMEstimate float64 `json:"tempo_bpm_estimate"`
			} `json:"beat"`
			Key struct {
				PrimaryEstimate string `json:"primary_estimate"`
			} `json:"key"`
		} `json:"analysis"`
	}
	if err := json.Unmarshal(result.ToolResult, &envelope); err != nil {
		return nil, err
	}
	if envelope.JobID != result.AnalysisJob || envelope.SourceSHA256 != result.TrackSHA256 || envelope.Analysis.SourceSHA256 != result.TrackSHA256 || envelope.Resource != "music-mir" || envelope.Operation != "analyze" {
		return nil, ErrInvalid
	}
	source := "decomposition:" + result.TrackSHA256 + ":" + result.AnalysisJob
	features := make([]TasteFeature, 0, 3)
	duration := envelope.Analysis.Media.DurationSeconds
	if finite(duration) && duration >= 1 && duration <= 600 {
		value := math.Log2(duration/180) / math.Log2(600.0/180.0)
		features = append(features, TasteFeature{Attribute: "acoustic:duration_log2_180s", Value: clamp(value, -1, 1), Confidence: 1, Source: source})
	}
	tempo := envelope.Analysis.Beat.TempoBPMEstimate
	if finite(tempo) && tempo >= 30 && tempo <= 300 {
		value := math.Log2(tempo/120) / math.Log2(300.0/120.0)
		features = append(features, TasteFeature{Attribute: "acoustic:tempo_log2_120bpm", Value: clamp(value, -1, 1), Confidence: 1, Source: source})
	}
	if key := strings.ToLower(strings.TrimSpace(envelope.Analysis.Key.PrimaryEstimate)); validKey(key) {
		key = strings.ReplaceAll(key, " ", "_")
		features = append(features, TasteFeature{Attribute: "key:" + key, Value: 1, Confidence: 1, Source: source})
	}
	return features, nil
}

func validKey(value string) bool {
	parts := strings.Fields(value)
	if len(parts) != 2 || (parts[1] != "major" && parts[1] != "minor") {
		return false
	}
	switch parts[0] {
	case "c", "c♯", "d", "d♯", "e", "f", "f♯", "g", "g♯", "a", "a♯", "b":
		return true
	default:
		return false
	}
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
func clamp(value, low, high float64) float64 {
	return math.Max(low, math.Min(high, value))
}

// ListTasteFeatures returns only the latest operator-confirmed feature
// projection for each requested hash. It never reads full result blobs or audio.
func (s *Store) ListTasteFeatures(ctx context.Context, trackHashes []string) (map[string][]TasteFeature, error) {
	out := make(map[string][]TasteFeature)
	for start := 0; start < len(trackHashes); start += 500 {
		end := min(start+500, len(trackHashes))
		batch := trackHashes[start:end]
		if len(batch) == 0 {
			continue
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")
		args := make([]any, len(batch))
		for i, hash := range batch {
			args[i] = hash
		}
		query := `SELECT f.track_sha256,f.attribute,f.value,f.confidence,f.evidence_source FROM music_decomposition_taste_features f JOIN music_decomposition_reviews r ON r.review_id=f.review_id JOIN music_decompositions d ON d.track_sha256=f.track_sha256 AND d.analysis_job_id=f.analysis_job_id WHERE f.track_sha256 IN (` + placeholders + `) AND r.verdict='confirmed' AND NOT EXISTS (SELECT 1 FROM music_decomposition_reviews newer WHERE newer.track_sha256=r.track_sha256 AND newer.analysis_job_id=r.analysis_job_id AND newer.rowid>r.rowid) AND NOT EXISTS (SELECT 1 FROM music_decompositions newer_job WHERE newer_job.track_sha256=d.track_sha256 AND newer_job.rowid>d.rowid) ORDER BY f.track_sha256,f.attribute`
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var hash string
			var feature TasteFeature
			if err := rows.Scan(&hash, &feature.Attribute, &feature.Value, &feature.Confidence, &feature.Source); err != nil {
				rows.Close()
				return nil, err
			}
			out[hash] = append(out[hash], feature)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	return out, nil
}
