//go:build audio_lab

package corpus

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface the sqliteRepository depends
// on. Both *sql.DB (repository unit tests) and *database.RoutedDB
// (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type sqlWriter interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production corpus Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

// clipTimeFormat matches RFC3339Nano so created_at sorts lexicographically
// in time order for a fixed zone.
const clipTimeFormat = time.RFC3339Nano

const (
	insertClipSQL = `
	INSERT INTO corpus_clips (id, reference_text, tags, duration_ms, sample_rate_hz, format, blob_key, source, corpus_revision, license_reference, speaker_pseudonym, split, noise_condition, snr_db, content_sha256, source_id, accent_group, manifest_transcript_sha256, lineage_manifest_sha256, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	selectClipColumns       = `id, reference_text, tags, duration_ms, sample_rate_hz, format, blob_key, source, corpus_revision, license_reference, speaker_pseudonym, split, noise_condition, snr_db, content_sha256, source_id, accent_group, manifest_transcript_sha256, lineage_manifest_sha256, created_at`
	selectClipByIDSQL       = `SELECT ` + selectClipColumns + ` FROM corpus_clips WHERE id = ?`
	deleteClipSQL           = `DELETE FROM corpus_clips WHERE id = ?`
	selectTimedReferenceSQL = `SELECT annotation_json FROM corpus_timed_references WHERE clip_id = ?`
	countCreatedInWindowSQL = `SELECT COUNT(*) FROM corpus_clips WHERE created_at >= ? AND created_at < ?`
)

func (s *sqliteRepository) Create(ctx context.Context, c Clip) (created Clip, retErr error) {
	if err := c.TimedReference.Validate(c.DurationMs); err != nil {
		return Clip{}, fmt.Errorf("corpus: clip %q: %w", c.ID, err)
	}
	if err := c.TimedReference.ValidateTranscript(c.ReferenceText); err != nil {
		return Clip{}, fmt.Errorf("corpus: clip %q: timed reference transcript mismatch: %w", c.ID, err)
	}
	if err := c.ReferenceReview.Validate(c.ReferenceText); err != nil {
		return Clip{}, fmt.Errorf("corpus: clip %q: %w", c.ID, err)
	}
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = s.clock.Now().UTC()
	}
	c.Source = c.Source.Normalize()
	if c.BlobKey == "" {
		return Clip{}, fmt.Errorf("corpus: clip %q requires a blob_key", c.ID)
	}
	tags, err := marshalTags(c.Tags)
	if err != nil {
		return Clip{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Clip{}, fmt.Errorf("begin corpus clip transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			retErr = errors.Join(retErr, fmt.Errorf("rollback corpus clip transaction: %w", rollbackErr))
		}
	}()
	if _, err := tx.ExecContext(ctx, insertClipSQL,
		c.ID, c.ReferenceText, tags, c.DurationMs, c.SampleRateHz, c.Format, c.BlobKey, string(c.Source), c.CorpusRevision, c.LicenseReference, c.SpeakerPseudonym, string(c.Split), string(c.NoiseCondition), c.SNRDB, c.ContentSHA256, c.SourceID, c.AccentGroup, c.ManifestTranscriptSHA256, c.LineageManifestSHA256,
		c.CreatedAt.Format(clipTimeFormat),
	); err != nil {
		return Clip{}, fmt.Errorf("insert clip %q: %w", c.ID, err)
	}
	if c.TimedReference != nil {
		if err := s.storeTimedReference(ctx, tx, c.ID, c.TimedReference); err != nil {
			return Clip{}, err
		}
	}
	if c.ReferenceReview != nil {
		if err := s.storeReferenceReview(ctx, tx, c.ID, c.ReferenceReview); err != nil {
			return Clip{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Clip{}, fmt.Errorf("commit corpus clip %q: %w", c.ID, err)
	}
	return c, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Clip, error) {
	row := s.db.QueryRowContext(ctx, selectClipByIDSQL, id)
	c, err := scanClip(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Clip{}, ErrClipNotFound{ID: id}
	}
	if err != nil {
		return Clip{}, fmt.Errorf("get clip %q: %w", id, err)
	}
	if err := s.loadTimedReference(ctx, &c); err != nil {
		return Clip{}, err
	}
	if err := s.loadReferenceReview(ctx, &c); err != nil {
		return Clip{}, err
	}
	return c, nil
}

func (s *sqliteRepository) ReviewReference(ctx context.Context, id, expectedText, referenceText string, review ReferenceReview) (Clip, error) {
	if err := review.Validate(referenceText); err != nil {
		return Clip{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Clip{}, fmt.Errorf("begin corpus reference review: %w", err)
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE corpus_clips SET reference_text = ? WHERE id = ? AND reference_text = ?`, referenceText, id, expectedText)
	if err != nil {
		return Clip{}, fmt.Errorf("update reference transcript for clip %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Clip{}, err
	}
	if n == 0 {
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM corpus_clips WHERE id = ?`, id).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
			return Clip{}, ErrClipNotFound{ID: id}
		} else if err != nil {
			return Clip{}, err
		}
		return Clip{}, ErrStaleReference{ID: id}
	}
	if referenceText != expectedText {
		if _, err := tx.ExecContext(ctx, `DELETE FROM corpus_timed_references WHERE clip_id = ?`, id); err != nil {
			return Clip{}, fmt.Errorf("clear stale word timings for clip %q: %w", id, err)
		}
	}
	encoded, err := json.Marshal(review)
	if err != nil {
		return Clip{}, fmt.Errorf("encode reference review for clip %q: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO corpus_reference_reviews (clip_id, review_json, updated_at) VALUES (?, ?, ?) ON CONFLICT(clip_id) DO UPDATE SET review_json = excluded.review_json, updated_at = excluded.updated_at`, id, string(encoded), s.clock.Now().UTC().Format(clipTimeFormat)); err != nil {
		return Clip{}, fmt.Errorf("store reference review for clip %q: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return Clip{}, fmt.Errorf("commit corpus reference review for clip %q: %w", id, err)
	}
	return s.Get(ctx, id)
}

func (s *sqliteRepository) ReviewReferenceGroup(ctx context.Context, members []ReviewMember, sourceID, revision, speaker, accent, manifestTranscriptSHA256, lineageManifestSHA256, referenceText string, review ReferenceReview) ([]Clip, error) {
	if err := review.Validate(referenceText); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin grouped corpus reference review: %w", err)
	}
	defer tx.Rollback()
	var groupCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM corpus_clips WHERE source_id = ?`, sourceID).Scan(&groupCount); err != nil {
		return nil, err
	}
	if groupCount != len(members) {
		return nil, fmt.Errorf("corpus: source variants changed during grouped review; reload the group")
	}
	encoded, err := json.Marshal(review)
	if err != nil {
		return nil, fmt.Errorf("encode grouped reference review: %w", err)
	}
	for _, member := range members {
		var savedText, gotSourceID, gotRevision, gotSpeaker, gotAccent, gotManifestDigest, gotLineageDigest string
		if err := tx.QueryRowContext(ctx, `SELECT reference_text, source_id, corpus_revision, speaker_pseudonym, accent_group, manifest_transcript_sha256, lineage_manifest_sha256 FROM corpus_clips WHERE id = ?`, member.ID).Scan(&savedText, &gotSourceID, &gotRevision, &gotSpeaker, &gotAccent, &gotManifestDigest, &gotLineageDigest); errors.Is(err, sql.ErrNoRows) {
			return nil, ErrClipNotFound{ID: member.ID}
		} else if err != nil {
			return nil, err
		}
		if gotSourceID != sourceID || gotRevision != revision || gotSpeaker != speaker || gotAccent != accent || gotManifestDigest != manifestTranscriptSHA256 || gotLineageDigest != lineageManifestSHA256 || ReferenceDigest(savedText) != gotManifestDigest {
			return nil, fmt.Errorf("corpus: source variant metadata changed during grouped review")
		}
		if ReferenceDigest(savedText) != member.ExpectedReferenceSHA256 {
			return nil, ErrStaleReference{ID: member.ID}
		}
		res, err := tx.ExecContext(ctx, `UPDATE corpus_clips SET reference_text = ? WHERE id = ? AND reference_text = ?`, referenceText, member.ID, savedText)
		if err != nil {
			return nil, fmt.Errorf("update grouped transcript for %q: %w", member.ID, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if n != 1 {
			return nil, ErrStaleReference{ID: member.ID}
		}
		if savedText != referenceText {
			if _, err := tx.ExecContext(ctx, `DELETE FROM corpus_timed_references WHERE clip_id = ?`, member.ID); err != nil {
				return nil, fmt.Errorf("clear stale timings for %q: %w", member.ID, err)
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO corpus_reference_reviews (clip_id, review_json, updated_at) VALUES (?, ?, ?) ON CONFLICT(clip_id) DO UPDATE SET review_json = excluded.review_json, updated_at = excluded.updated_at`, member.ID, string(encoded), s.clock.Now().UTC().Format(clipTimeFormat)); err != nil {
			return nil, fmt.Errorf("store grouped review for %q: %w", member.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit grouped corpus reference review: %w", err)
	}
	clips := make([]Clip, 0, len(members))
	for _, member := range members {
		clip, err := s.Get(ctx, member.ID)
		if err != nil {
			return nil, err
		}
		clips = append(clips, clip)
	}
	return clips, nil
}

func (s *sqliteRepository) BackfillSourceLineage(ctx context.Context, entries []LineageEntry, dryRun bool) (int, error) {
	if len(entries) == 0 {
		return 0, fmt.Errorf("corpus: lineage backfill requires at least one manifest entry")
	}
	manifestDigest := entries[0].ManifestSHA256
	if !validSHA256(manifestDigest) {
		return 0, fmt.Errorf("corpus: manifest SHA-256 is required")
	}
	seen := make(map[string]struct{}, len(entries))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin source lineage backfill: %w", err)
	}
	defer tx.Rollback()
	wouldUpdate := 0
	updates := make([]LineageEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.ClipID == "" || entry.SourceID == "" || entry.CorpusRevision == "" || entry.SpeakerPseudonym == "" || entry.AccentGroup == "" || entry.LicenseReference == "" || !validSHA256(entry.ContentSHA256) || !validSHA256(entry.ManifestTranscriptSHA256) || entry.ManifestSHA256 != manifestDigest {
			return 0, fmt.Errorf("corpus: lineage backfill entry %q is incomplete or belongs to another manifest", entry.ClipID)
		}
		if _, ok := seen[entry.ClipID]; ok {
			return 0, fmt.Errorf("corpus: duplicate lineage backfill clip %q", entry.ClipID)
		}
		seen[entry.ClipID] = struct{}{}
		var referenceText, revision, contentHash, sourceID, speaker, accent, transcriptHash, priorManifest string
		var split, license string
		if err := tx.QueryRowContext(ctx, `SELECT reference_text, corpus_revision, content_sha256, source_id, speaker_pseudonym, accent_group, manifest_transcript_sha256, lineage_manifest_sha256, split, license_reference FROM corpus_clips WHERE id = ?`, entry.ClipID).Scan(&referenceText, &revision, &contentHash, &sourceID, &speaker, &accent, &transcriptHash, &priorManifest, &split, &license); errors.Is(err, sql.ErrNoRows) {
			return 0, ErrClipNotFound{ID: entry.ClipID}
		} else if err != nil {
			return 0, err
		}
		if revision != entry.CorpusRevision || contentHash != entry.ContentSHA256 || ReferenceDigest(referenceText) != entry.ManifestTranscriptSHA256 {
			return 0, fmt.Errorf("corpus: lineage backfill exact revision/content/transcript check failed for clip %q", entry.ClipID)
		}
		conflict := (sourceID != "" && sourceID != entry.SourceID) || (speaker != "" && speaker != entry.SpeakerPseudonym) || (accent != "" && accent != entry.AccentGroup) || (transcriptHash != "" && transcriptHash != entry.ManifestTranscriptSHA256) || (priorManifest != "" && priorManifest != entry.ManifestSHA256) || (split != "" && split != entry.Split) || (license != "" && license != entry.LicenseReference)
		if conflict {
			return 0, fmt.Errorf("corpus: lineage backfill conflicts with existing metadata for clip %q", entry.ClipID)
		}
		if sourceID == "" || speaker == "" || accent == "" || transcriptHash == "" || priorManifest == "" || split == "" || license == "" {
			wouldUpdate++
			updates = append(updates, entry)
		}
	}
	if dryRun {
		return wouldUpdate, nil
	}
	for _, entry := range updates {
		if _, err := tx.ExecContext(ctx, `UPDATE corpus_clips SET source_id = ?, speaker_pseudonym = ?, accent_group = ?, manifest_transcript_sha256 = ?, lineage_manifest_sha256 = ?, split = ?, license_reference = ? WHERE id = ?`, entry.SourceID, entry.SpeakerPseudonym, entry.AccentGroup, entry.ManifestTranscriptSHA256, entry.ManifestSHA256, entry.Split, entry.LicenseReference, entry.ClipID); err != nil {
			return 0, fmt.Errorf("backfill lineage for clip %q: %w", entry.ClipID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit source lineage backfill: %w", err)
	}
	return wouldUpdate, nil
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func (s *sqliteRepository) List(ctx context.Context, filter ListFilter) ([]Clip, error) {
	query := `SELECT ` + selectClipColumns + ` FROM corpus_clips`
	var args []any
	if filter.TagContains != "" {
		query += ` WHERE tags LIKE ?`
		args = append(args, "%"+filter.TagContains+"%")
	}
	query += ` ORDER BY created_at DESC, id DESC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list clips: %w", err)
	}

	var clips []Clip
	for rows.Next() {
		c, err := scanClip(rows)
		if err != nil {
			return nil, fmt.Errorf("scan clip: %w", err)
		}
		clips = append(clips, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate clips: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close corpus clip rows: %w", err)
	}
	for i := range clips {
		if err := s.loadTimedReference(ctx, &clips[i]); err != nil {
			return nil, err
		}
		if err := s.loadReferenceReview(ctx, &clips[i]); err != nil {
			return nil, err
		}
	}
	return clips, nil
}

func (s *sqliteRepository) ListBySourceID(ctx context.Context, sourceID string) ([]Clip, error) {
	if sourceID == "" {
		return nil, fmt.Errorf("corpus: source id is required")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+selectClipColumns+` FROM corpus_clips WHERE source_id = ? ORDER BY noise_condition, snr_db, id`, sourceID)
	if err != nil {
		return nil, fmt.Errorf("list source variants: %w", err)
	}
	var clips []Clip
	for rows.Next() {
		clip, err := scanClip(rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		clips = append(clips, clip)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range clips {
		if err := s.loadTimedReference(ctx, &clips[i]); err != nil {
			return nil, err
		}
		if err := s.loadReferenceReview(ctx, &clips[i]); err != nil {
			return nil, err
		}
	}
	return clips, nil
}

// CountCreatedInWindow returns the number of clips created in [from, to),
// with bounds interpreted as UTC instants.
func (s *sqliteRepository) CountCreatedInWindow(ctx context.Context, from, to time.Time) (int64, error) {
	if !from.Before(to) {
		return 0, fmt.Errorf("corpus: count window start must be before end")
	}
	var count int64
	if err := s.db.QueryRowContext(ctx, countCreatedInWindowSQL, from.UTC().Format(clipTimeFormat), to.UTC().Format(clipTimeFormat)).Scan(&count); err != nil {
		return 0, fmt.Errorf("count corpus clips created in window: %w", err)
	}
	return count, nil
}

func (s *sqliteRepository) Delete(ctx context.Context, id string) (retErr error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin corpus clip deletion transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			retErr = errors.Join(retErr, fmt.Errorf("rollback corpus clip deletion transaction: %w", rollbackErr))
		}
	}()
	if _, err := tx.ExecContext(ctx, `DELETE FROM corpus_timed_references WHERE clip_id = ?`, id); err != nil {
		return fmt.Errorf("delete timed reference for clip %q: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM corpus_reference_reviews WHERE clip_id = ?`, id); err != nil {
		return fmt.Errorf("delete reference review for clip %q: %w", id, err)
	}
	res, err := tx.ExecContext(ctx, deleteClipSQL, id)
	if err != nil {
		return fmt.Errorf("delete clip %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete clip %q rows: %w", id, err)
	}
	if n == 0 {
		return ErrClipNotFound{ID: id}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit corpus clip deletion %q: %w", id, err)
	}
	return nil
}

func (s *sqliteRepository) storeReferenceReview(ctx context.Context, writer sqlWriter, clipID string, review *ReferenceReview) error {
	encoded, err := json.Marshal(review)
	if err != nil {
		return fmt.Errorf("marshal reference review for clip %q: %w", clipID, err)
	}
	if _, err := writer.ExecContext(ctx, `INSERT INTO corpus_reference_reviews (clip_id, review_json, updated_at) VALUES (?, ?, ?)`, clipID, string(encoded), s.clock.Now().UTC().Format(clipTimeFormat)); err != nil {
		return fmt.Errorf("store reference review for clip %q: %w", clipID, err)
	}
	return nil
}

func (s *sqliteRepository) loadReferenceReview(ctx context.Context, clip *Clip) error {
	var encoded string
	err := s.db.QueryRowContext(ctx, `SELECT review_json FROM corpus_reference_reviews WHERE clip_id = ?`, clip.ID).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load reference review for clip %q: %w", clip.ID, err)
	}
	var review ReferenceReview
	if err := json.Unmarshal([]byte(encoded), &review); err != nil {
		return fmt.Errorf("decode reference review for clip %q: %w", clip.ID, err)
	}
	if err := review.Validate(clip.ReferenceText); err != nil {
		return fmt.Errorf("invalid stored reference review for clip %q: %w", clip.ID, err)
	}
	clip.ReferenceReview = &review
	return nil
}

func (s *sqliteRepository) storeTimedReference(ctx context.Context, writer sqlWriter, clipID string, annotation *TimedReference) error {
	encoded, err := json.Marshal(annotation)
	if err != nil {
		return fmt.Errorf("marshal timed reference for clip %q: %w", clipID, err)
	}
	if len(encoded) > MaxTimedReferenceBytes {
		return fmt.Errorf("timed reference for clip %q exceeds %d bytes", clipID, MaxTimedReferenceBytes)
	}
	_, err = writer.ExecContext(ctx, `INSERT INTO corpus_timed_references (clip_id, annotation_json, updated_at) VALUES (?, ?, ?)`, clipID, string(encoded), s.clock.Now().UTC().Format(clipTimeFormat))
	if err != nil {
		return fmt.Errorf("store timed reference for clip %q: %w", clipID, err)
	}
	return nil
}

func (s *sqliteRepository) loadTimedReference(ctx context.Context, clip *Clip) error {
	var encoded string
	err := s.db.QueryRowContext(ctx, selectTimedReferenceSQL, clip.ID).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load timed reference for clip %q: %w", clip.ID, err)
	}
	var annotation TimedReference
	if err := json.Unmarshal([]byte(encoded), &annotation); err != nil {
		return fmt.Errorf("decode timed reference for clip %q: %w", clip.ID, err)
	}
	clip.TimedReference = &annotation
	return nil
}

// marshalTags serializes the JSON tags column; a nil slice encodes as [].
func marshalTags(tags []string) (string, error) {
	if tags == nil {
		tags = []string{}
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return "", fmt.Errorf("marshal tags: %w", err)
	}
	return string(b), nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanClip(sc rowScanner) (Clip, error) {
	var (
		c                                                                      Clip
		tagsRaw                                                                string
		sourceRaw                                                              string
		corpusRevision, licenseReference, speakerPseudonym                     string
		sourceID, accentGroup, manifestTranscriptSHA256, lineageManifestSHA256 string
		splitRaw, noiseRaw, contentHash                                        string
		snrDB                                                                  float64
		createdRaw                                                             string
	)
	if err := sc.Scan(&c.ID, &c.ReferenceText, &tagsRaw, &c.DurationMs, &c.SampleRateHz,
		&c.Format, &c.BlobKey, &sourceRaw, &corpusRevision, &licenseReference, &speakerPseudonym, &splitRaw, &noiseRaw, &snrDB, &contentHash, &sourceID, &accentGroup, &manifestTranscriptSHA256, &lineageManifestSHA256, &createdRaw,
	); err != nil {
		return Clip{}, err
	}
	if tagsRaw != "" && tagsRaw != "null" {
		if err := json.Unmarshal([]byte(tagsRaw), &c.Tags); err != nil {
			return Clip{}, fmt.Errorf("parse tags: %w", err)
		}
	}
	c.Source = Source(sourceRaw).Normalize()
	c.CorpusRevision = corpusRevision
	c.LicenseReference = licenseReference
	c.SpeakerPseudonym = speakerPseudonym
	c.Split = Split(splitRaw)
	c.NoiseCondition = NoiseCondition(noiseRaw)
	c.SNRDB = snrDB
	c.ContentSHA256 = contentHash
	c.SourceID = sourceID
	c.AccentGroup = accentGroup
	c.ManifestTranscriptSHA256 = manifestTranscriptSHA256
	c.LineageManifestSHA256 = lineageManifestSHA256
	created, err := time.Parse(clipTimeFormat, createdRaw)
	if err != nil {
		return Clip{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	c.CreatedAt = created
	return c, nil
}
