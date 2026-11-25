package hub

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"time"
)

// Embedding is one embedder's answer for a batch of texts. Model and
// Dimensions are the embedder's, not a caller's assumption: a changed model
// must never be compared with vectors from the old one.
type Embedding struct {
	Model      string
	Dimensions int
	Vectors    [][]float32
}

// Embedder turns text into vectors. It is optional: a nil embedder leaves the
// deterministic lexical read in place. Implementations must be safe for
// concurrent use.
type Embedder interface {
	Embed(ctx context.Context, texts []string) (Embedding, error)
}

// decisionEmbedText is the document side of the index. It carries the question,
// the recommended answer and the first part of the situation: enough for a
// paraphrase to land near its kin without dragging the whole packet in.
func decisionEmbedText(question, recommended, bigPicture string) string {
	text := strings.TrimSpace(question)
	recommended = strings.TrimSpace(recommended)
	if recommended != "" {
		text += "\nRecommended: " + recommended
	}
	if bigPicture = strings.TrimSpace(bigPicture); bigPicture != "" {
		if runes := []rune(bigPicture); len(runes) > 400 {
			bigPicture = string(runes[:400])
		}
		text += "\n" + bigPicture
	}
	return "search_document: " + text
}

func queryEmbedText(question string) string {
	return "search_query: " + strings.TrimSpace(question)
}

func textDigest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// encodeVector stores little-endian float32, the shape code-facts uses for its
// SQLite vectors.
func encodeVector(vector []float32) []byte {
	out := make([]byte, 4*len(vector))
	for i, value := range vector {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(value))
	}
	return out
}

func decodeVector(raw []byte, dimensions int) ([]float32, bool) {
	if dimensions <= 0 || len(raw) != 4*dimensions {
		return nil, false
	}
	out := make([]float32, dimensions)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
	}
	return out, true
}

// cosine is the similarity the embedder's space is built for; vectors are L2
// normalized by the gateway, so a dot product would do, but the norm guard
// keeps a hand-written vector from producing a bogus score.
func cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		av, bv := float64(a[i]), float64(b[i])
		dot += av * bv
		na += av * av
		nb += bv * bv
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// semanticFloor is the cosine below which a suggestion is noise. It is a band,
// not a probability; the UI labels it instead of showing a bare number.
const semanticFloor = 0.68

type storedEmbedding struct {
	model      string
	dimensions int
	vector     []float32
}

// relatedVectors loads the stored vectors of candidate asks. Missing rows are
// simply absent: the lexical read covers them.
func (s *Service) relatedVectors(ctx context.Context, ids []string) map[string]storedEmbedding {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx, `SELECT ask_id, model, dimensions, vector FROM ask_embeddings WHERE ask_id IN (`+placeholders+`)`, args...)
	if err != nil {
		s.log.Printf("related: read vectors: %v", err)
		return nil
	}
	defer rows.Close()
	out := map[string]storedEmbedding{}
	for rows.Next() {
		var id, model string
		var dimensions int
		var raw []byte
		if err := rows.Scan(&id, &model, &dimensions, &raw); err != nil {
			s.log.Printf("related: scan vector: %v", err)
			return out
		}
		vector, ok := decodeVector(raw, dimensions)
		if !ok {
			continue
		}
		out[id] = storedEmbedding{model: model, dimensions: dimensions, vector: vector}
	}
	return out
}

// semanticScores embeds the question once and scores every candidate that has a
// vector from the same model and dimension. A failure leaves the lexical scores
// exactly as they were.
func (s *Service) semanticScores(ctx context.Context, question string, candidates map[string]storedEmbedding) map[string]float64 {
	embedder := s.currentEmbedder()
	if embedder == nil || len(candidates) == 0 || strings.TrimSpace(question) == "" {
		return nil
	}
	// A detail read must not wait on a cold model. The lexical read is the
	// floor; this leg gets a short budget and is skipped when it misses.
	embedCtx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
	defer cancel()
	result, err := embedder.Embed(embedCtx, []string{queryEmbedText(question)})
	if err != nil || len(result.Vectors) != 1 {
		if err != nil {
			s.logSemantic(err)
		}
		return nil
	}
	query := result.Vectors[0]
	scores := map[string]float64{}
	for id, candidate := range candidates {
		if candidate.model != result.Model || candidate.dimensions != result.Dimensions || len(candidate.vector) != len(query) {
			continue
		}
		score := cosine(query, candidate.vector)
		if score >= semanticFloor {
			scores[id] = math.Round(score*100) / 100
		}
	}
	return scores
}

// embedAsk stores one ask's vector. It is called after a write when a semantic
// embedder is configured; a failure is logged and never fails the send.
func (s *Service) embedAsk(ctx context.Context, askID string) error {
	embedder := s.currentEmbedder()
	if embedder == nil {
		return nil
	}
	record, err := scanAsk(s.db.QueryRowContext(ctx, askSelect+` WHERE a.id = ?`, askID))
	if err != nil {
		return err
	}
	recommended := record.Recommended
	for _, option := range record.Options {
		if option.Key == recommended {
			recommended = option.Label
			break
		}
	}
	text := decisionEmbedText(record.Question, recommended, record.BigPicture)
	if strings.TrimSpace(record.Question) == "" {
		return nil
	}
	// A boot backfill must not re-embed an unchanged question.
	digest := textDigest(text)
	var existing string
	if err := s.db.QueryRowContext(ctx, `SELECT text_hash FROM ask_embeddings WHERE ask_id = ?`, askID).Scan(&existing); err == nil && existing == digest {
		return nil
	}
	result, err := embedder.Embed(ctx, []string{text})
	if err != nil {
		return err
	}
	if len(result.Vectors) != 1 || len(result.Vectors[0]) != result.Dimensions {
		return errors.New("embedder returned a vector that does not match its dimensions")
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO ask_embeddings (ask_id, model, dimensions, text_hash, vector, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(ask_id) DO UPDATE SET model=excluded.model, dimensions=excluded.dimensions,
		text_hash=excluded.text_hash, vector=excluded.vector, created_at=excluded.created_at`,
		askID, result.Model, result.Dimensions, digest, encodeVector(result.Vectors[0]),
		s.clock.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// BackfillEmbeddings builds the optional semantic cache for open asks. It is
// bounded and rebuildable; a failure on one batch is logged and the rest go on.
func (s *Service) BackfillEmbeddings(ctx context.Context, limit int) (int, error) {
	if s.currentEmbedder() == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 300
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM asks
		WHERE state IN ('pending','escalated') AND kind <> 'status' AND question <> ''
		ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()

	// A vector is only useful for an open ask; answered ones are not related to
	// anything. Drop them so the cache follows the open queue instead of every
	// ask the Hub ever held.
	if _, err := s.db.ExecContext(ctx, `DELETE FROM ask_embeddings WHERE ask_id IN
		(SELECT id FROM asks WHERE state NOT IN ('pending','escalated'))`); err != nil {
		s.log.Printf("semantic: prune closed asks: %v", err)
	}

	stored := 0
	for start := 0; start < len(ids); start += 32 {
		if ctx.Err() != nil {
			return stored, ctx.Err()
		}
		end := start + 32
		if end > len(ids) {
			end = len(ids)
		}
		for _, id := range ids[start:end] {
			if err := s.embedAsk(ctx, id); err != nil {
				s.log.Printf("semantic: backfill %s: %v", id, err)
				continue
			}
			stored++
		}
	}
	return stored, nil
}

// askVectorExists is a small read the tests and diagnostics use; it says whether
// the cache holds a vector for the ask.
func (s *Service) askVectorExists(ctx context.Context, askID string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM ask_embeddings WHERE ask_id = ?`, askID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// logSemantic records a failing semantic leg at most once a minute: a detail
// page must not turn a down gateway into a log flood.
func (s *Service) logSemantic(err error) {
	s.semanticLogMu.Lock()
	defer s.semanticLogMu.Unlock()
	if time.Since(s.lastSemanticAt) < time.Minute {
		return
	}
	s.lastSemanticAt = time.Now()
	s.log.Printf("related: semantic leg unavailable: %v", err)
}
