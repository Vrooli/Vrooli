package hub

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vrooli/api-core/blobstore"
)

// Attachment blob store: the Hub's own copy of every attached file, addressed
// by SHA-256. Blob bytes live in a blobstore.BlobStore (the scenario data root
// in production); the attachment_blobs table is the inventory the Hub trusts.

var (
	// ErrAttachmentsUnavailable reports a Hub started without an attachment store.
	ErrAttachmentsUnavailable = errors.New("attachment storage is not configured")
	// ErrBlobTooLarge reports a blob over the size the caller may read or store.
	ErrBlobTooLarge = errors.New("attachment content is too large")
	// ErrBlobQuota reports that the Hub already holds as many bytes of uploads
	// that no ask references as it accepts.
	ErrBlobQuota = errors.New("too many uploaded attachment bytes are waiting for an ask")
)

const (
	// unreferencedBlobGrace is how long a blob nothing references survives its
	// last upload or reference, so a sender can upload first and ask later.
	unreferencedBlobGrace = 24 * time.Hour
	sweepBlobBatch        = 200
	maxStatHashes         = 1000
	// maxUnreferencedBlobBytes bounds what uploads that no ask references yet may
	// hold at once. Uploading comes first by design (a sender stores its files,
	// then asks), so unreferenced bytes are normal for a day; an unbounded amount
	// is a disk filled by one sender.
	maxUnreferencedBlobBytes = 1 << 30
	// blobQueryChunk keeps IN lists under SQLite's variable limit.
	blobQueryChunk = 400
)

// BlobStoreResolver returns the store for a request. Test-mode requests resolve
// to a lease-owned store, like the database and every other file root.
type BlobStoreResolver func(context.Context) (blobstore.BlobStore, error)

// StaticBlobStore is a resolver that always returns store.
func StaticBlobStore(store blobstore.BlobStore) BlobStoreResolver {
	return func(context.Context) (blobstore.BlobStore, error) { return store, nil }
}

type attachmentConfig struct {
	storeFor BlobStoreResolver
	// unreferencedLimit is the most bytes of unreferenced uploads PutAttachmentBlob
	// accepts.
	unreferencedLimit int64
}

// SetAttachmentStore wires the attachment blob store. Without one, uploads and
// attachments fail with ErrAttachmentsUnavailable.
func (s *Service) SetAttachmentStore(resolve BlobStoreResolver) {
	if resolve == nil {
		s.attachments.cfg.Store(nil)
		return
	}
	s.attachments.cfg.Store(&attachmentConfig{storeFor: resolve, unreferencedLimit: maxUnreferencedBlobBytes})
}

// blobKey is the content-addressed key of a blob: sha256/<first byte>/<hash>.
func blobKey(sha string) string { return "sha256/" + sha[:2] + "/" + sha }

type blobInfo struct {
	size int64
}

// BlobPutResult is what storing one blob did.
type BlobPutResult struct {
	SHA256    string
	SizeBytes int64
	// Existed reports that the Hub already held these bytes.
	Existed bool
}

// StatAttachmentBlobs returns the hashes from hashes that the Hub does not
// hold, in request order without repeats. A hash that is confirmed present is
// kept alive for another grace period, so the sender can rely on it.
func (s *Service) StatAttachmentBlobs(ctx context.Context, hashes []string) ([]string, error) {
	if len(hashes) > maxStatHashes {
		return nil, fmt.Errorf("%w: at most %d hashes per request", ErrInvalidArgument, maxStatHashes)
	}
	store, err := s.blobStore(ctx)
	if err != nil {
		return nil, err
	}
	ordered := make([]string, 0, len(hashes))
	seen := make(map[string]bool, len(hashes))
	for _, raw := range hashes {
		hash := normalizeBlobHash(raw)
		if hash == "" {
			return nil, fmt.Errorf("%w: %q is not a lowercase hex SHA-256", ErrInvalidArgument, raw)
		}
		if !seen[hash] {
			seen[hash] = true
			ordered = append(ordered, hash)
		}
	}
	held, err := s.touchBlobs(ctx, ordered)
	if err != nil {
		return nil, err
	}
	missing := []string{}
	for _, hash := range ordered {
		// A row without its bytes (a lost file) counts as missing, so the
		// sender uploads it again instead of attaching something unservable.
		if _, ok := held[hash]; !ok || !blobPresent(ctx, store, hash) {
			missing = append(missing, hash)
		}
	}
	return missing, nil
}

func blobPresent(ctx context.Context, store blobstore.BlobStore, hash string) bool {
	reader, _, err := store.Get(ctx, blobKey(hash))
	if err != nil {
		return false
	}
	_ = reader.Close()
	return true
}

// PutAttachmentBlob stores content under its SHA-256. The Hub recomputes the
// hash and refuses a mismatch, so a blob's key always names its bytes. mediaType
// is the sender's declaration, kept with the blob for the record; the Hub never
// serves it (see classifyBlob). Uploads that no ask references yet are capped
// (ErrBlobQuota). The caller authenticates the sender.
func (s *Service) PutAttachmentBlob(ctx context.Context, sha, mediaType string, content []byte) (BlobPutResult, error) {
	hash := normalizeBlobHash(sha)
	if hash == "" {
		return BlobPutResult{}, fmt.Errorf("%w: sha256 must be 64 lowercase hex characters", ErrInvalidArgument)
	}
	if len(content) > MaxAttachmentBlobBytes {
		return BlobPutResult{}, fmt.Errorf("%w: a file is at most %d MiB", ErrInvalidArgument, MaxAttachmentBlobBytes>>20)
	}
	sum := sha256.Sum256(content)
	if hex.EncodeToString(sum[:]) != hash {
		return BlobPutResult{}, fmt.Errorf("%w: sha256 does not match the content", ErrInvalidArgument)
	}
	mediaType = strings.TrimSpace(mediaType)
	if !utf8.ValidString(mediaType) || len(mediaType) > maxMediaTypeBytes {
		return BlobPutResult{}, fmt.Errorf("%w: media_type must be valid UTF-8 of at most %d bytes", ErrInvalidArgument, maxMediaTypeBytes)
	}
	cfg := s.attachments.cfg.Load()
	if cfg == nil {
		return BlobPutResult{}, ErrAttachmentsUnavailable
	}
	store, err := cfg.storeFor(ctx)
	if err != nil {
		return BlobPutResult{}, err
	}
	s.attachments.writeMu.Lock()
	defer s.attachments.writeMu.Unlock()
	now := storedTime(s.clock.Now())
	var storedAt string
	err = s.db.QueryRowContext(ctx, `SELECT stored_at FROM attachment_blobs WHERE sha256 = ?`, hash).Scan(&storedAt)
	row := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return BlobPutResult{}, err
	}
	existed := row && blobPresent(ctx, store, hash)
	if !existed {
		if err := s.checkUnreferencedQuota(ctx, cfg, int64(len(content))); err != nil {
			return BlobPutResult{}, err
		}
		if err := store.Put(ctx, blobKey(hash), bytes.NewReader(content), mediaType); err != nil {
			return BlobPutResult{}, fmt.Errorf("store attachment: %w", err)
		}
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO attachment_blobs (sha256, size_bytes, media_type, stored_at, last_touched_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(sha256) DO UPDATE SET last_touched_at = excluded.last_touched_at`, hash, len(content), mediaType, now, now); err != nil {
		return BlobPutResult{}, err
	}
	return BlobPutResult{SHA256: hash, SizeBytes: int64(len(content)), Existed: existed}, nil
}

// checkUnreferencedQuota refuses an upload that would take the bytes waiting
// for an ask past the limit. Bytes an ask references are bounded by the asks
// themselves and are not counted. The caller holds writeMu.
func (s *Service) checkUnreferencedQuota(ctx context.Context, cfg *attachmentConfig, incoming int64) error {
	var held int64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(size_bytes), 0) FROM attachment_blobs b
		WHERE NOT EXISTS (SELECT 1 FROM ask_attachments a WHERE a.sha256 = b.sha256)
		AND NOT EXISTS (SELECT 1 FROM ask_attachment_entries e WHERE e.sha256 = b.sha256)`).Scan(&held); err != nil {
		return err
	}
	if held+incoming > cfg.unreferencedLimit {
		return fmt.Errorf("%w: %d MiB are waiting and the limit is %d MiB; ask with them, or wait for unused uploads to expire after %s",
			ErrBlobQuota, held>>20, cfg.unreferencedLimit>>20, unreferencedBlobGrace)
	}
	return nil
}

// ReadAttachmentBlob returns a stored blob's bytes, at most max of them. It is
// for the Hub's own readers (the packet check reads an explainer this way); the
// operator reads attachments through OpenAskAttachment.
func (s *Service) ReadAttachmentBlob(ctx context.Context, sha string, max int64) ([]byte, error) {
	hash := normalizeBlobHash(sha)
	if hash == "" {
		return nil, fmt.Errorf("%w: sha256 must be 64 lowercase hex characters", ErrInvalidArgument)
	}
	store, err := s.blobStore(ctx)
	if err != nil {
		return nil, err
	}
	reader, _, err := store.Get(ctx, blobKey(hash))
	if err != nil {
		return nil, fmt.Errorf("%w: attachment content is not stored", ErrNotFound)
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, ErrBlobTooLarge
	}
	return data, nil
}

// sniffBlob returns the first bytes of a blob for content classification.
func (s *Service) sniffBlob(ctx context.Context, sha string) ([]byte, error) {
	store, err := s.blobStore(ctx)
	if err != nil {
		return nil, err
	}
	reader, _, err := store.Get(ctx, blobKey(sha))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, sniffBlobBytes))
}

// touchBlobs looks up hashes in the inventory and marks the ones found as
// referenced now, so the sweep keeps them for another grace period.
func (s *Service) touchBlobs(ctx context.Context, hashes []string) (map[string]blobInfo, error) {
	found := make(map[string]blobInfo, len(hashes))
	unique := make([]string, 0, len(hashes))
	seen := make(map[string]bool, len(hashes))
	for _, hash := range hashes {
		if !seen[hash] {
			seen[hash] = true
			unique = append(unique, hash)
		}
	}
	if len(unique) == 0 {
		return found, nil
	}
	s.attachments.writeMu.Lock()
	defer s.attachments.writeMu.Unlock()
	now := storedTime(s.clock.Now())
	for start := 0; start < len(unique); start += blobQueryChunk {
		chunk := unique[start:min(start+blobQueryChunk, len(unique))]
		marks := strings.TrimSuffix(strings.Repeat("?,", len(chunk)), ",")
		args := make([]any, len(chunk))
		for i, hash := range chunk {
			args[i] = hash
		}
		rows, err := s.db.QueryContext(ctx, `SELECT sha256, size_bytes FROM attachment_blobs WHERE sha256 IN (`+marks+`)`, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var hash string
			var info blobInfo
			if err := rows.Scan(&hash, &info.size); err != nil {
				_ = rows.Close()
				return nil, err
			}
			found[hash] = info
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE attachment_blobs SET last_touched_at = ? WHERE sha256 IN (`+marks+`)`, append([]any{now}, args...)...); err != nil {
			return nil, err
		}
	}
	return found, nil
}

// SweepAttachmentBlobs deletes blobs that no ask references and that were last
// uploaded or referenced more than a day ago. It reports how many it deleted.
// The maintenance loop calls it every minute.
func (s *Service) SweepAttachmentBlobs(ctx context.Context) (int, error) {
	cfg := s.attachments.cfg.Load()
	if cfg == nil {
		return 0, nil
	}
	store, err := cfg.storeFor(ctx)
	if err != nil {
		return 0, err
	}
	cutoff := storedTime(s.clock.Now().UTC().Add(-unreferencedBlobGrace))

	s.attachments.writeMu.Lock()
	defer s.attachments.writeMu.Unlock()
	rows, err := s.db.QueryContext(ctx, `SELECT sha256 FROM attachment_blobs b WHERE b.last_touched_at < ?
		AND NOT EXISTS (SELECT 1 FROM ask_attachments a WHERE a.sha256 = b.sha256)
		AND NOT EXISTS (SELECT 1 FROM ask_attachment_entries e WHERE e.sha256 = b.sha256)
		ORDER BY b.last_touched_at LIMIT ?`, cutoff, sweepBlobBatch)
	if err != nil {
		return 0, err
	}
	var doomed []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			_ = rows.Close()
			return 0, err
		}
		doomed = append(doomed, hash)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	deleted := 0
	for _, hash := range doomed {
		// Bytes first: a failure here leaves the row, and the next pass retries.
		if err := store.Delete(ctx, blobKey(hash)); err != nil {
			s.log.Printf("attachment sweep: delete %s: %v", hash, err)
			continue
		}
		if _, err := s.db.ExecContext(ctx, `DELETE FROM attachment_blobs WHERE sha256 = ?`, hash); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}
