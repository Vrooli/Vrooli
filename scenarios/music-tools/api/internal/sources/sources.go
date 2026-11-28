// Package sources owns private reference/audio sources imported into Music Tools.
package sources

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	musicstorage "music-tools/internal/storage"
)

//go:embed schema.sql
var schemaSQL string

const (
	MaxFileBytes int64  = 128 << 20
	OwnerQuota   uint64 = 1 << 30
	ownerPrefix         = "music-tools-source:"
)

var (
	ErrQuotaExceeded = errors.New("private source storage quota exceeded (1 GiB)")
	ErrConflict      = errors.New("source SHA-256 is already registered with conflicting content metadata")
)

// SQLExecutor is the database seam used by the source inventory.
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Record is immutable provenance for one content-addressed private source.
type Record struct {
	SHA256    string    `json:"sha256"`
	BlobKey   string    `json:"source_blob_key"`
	Title     string    `json:"title"`
	Artist    string    `json:"artist,omitempty"`
	Uploader  string    `json:"uploader,omitempty"`
	SourceURL string    `json:"source_url,omitempty"`
	Acquired  string    `json:"acquired_at,omitempty"`
	FileName  string    `json:"file_name"`
	MediaType string    `json:"media_type"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	db    SQLExecutor
	blobs *musicstorage.Store
	mu    sync.Mutex
}

func Schema() string { return schemaSQL }

func New(db SQLExecutor, blobs *musicstorage.Store) (*Store, error) {
	if db == nil || blobs == nil {
		return nil, errors.New("source inventory requires database and managed blob storage")
	}
	return &Store{db: db, blobs: blobs}, nil
}

// Import verifies content-address identity, quota, and non-overwrite semantics
// while serializing concurrent imports through this store instance.
func (s *Store) Import(ctx context.Context, record Record, content io.ReadSeeker) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(record.SHA256) != 64 || strings.ToLower(record.SHA256) != record.SHA256 || record.SizeBytes <= 0 || record.SizeBytes > MaxFileBytes || strings.TrimSpace(record.Title) == "" {
		return Record{}, errors.New("source metadata or upload size is invalid")
	}
	if current, err := s.Get(ctx, record.SHA256); err == nil {
		if current.SizeBytes != record.SizeBytes || current.MediaType != record.MediaType {
			return Record{}, ErrConflict
		}
		if err := verifyBlob(ctx, s.blobs, current.BlobKey, record.SHA256, record.SizeBytes); err != nil {
			return Record{}, fmt.Errorf("verify existing source blob: %w", err)
		}
		return current, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Record{}, err
	}
	key := "sources/" + record.SHA256 + "/original"
	var used uint64
	for _, entry := range s.blobs.Entries() {
		if strings.HasPrefix(entry.Owner, ownerPrefix) && entry.Key != key {
			if entry.Size > OwnerQuota-used {
				return Record{}, ErrQuotaExceeded
			}
			used += entry.Size
		}
	}
	if uint64(record.SizeBytes) > OwnerQuota-used {
		return Record{}, ErrQuotaExceeded
	}

	// A crash after blob commit but before SQL commit leaves an owned blob. Verify
	// it and finish the metadata commit rather than overwriting that key.
	if err := verifyBlob(ctx, s.blobs, key, record.SHA256, record.SizeBytes); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Record{}, fmt.Errorf("source content-address key already exists with conflicting bytes: %w", ErrConflict)
		}
		if _, err := content.Seek(0, io.SeekStart); err != nil {
			return Record{}, err
		}
		if err := s.blobs.PutOwned(ctx, key, ownerPrefix+record.SHA256, content, record.MediaType, false); err != nil {
			return Record{}, fmt.Errorf("store private source blob: %w", err)
		}
	}
	if _, err := content.Seek(0, io.SeekStart); err != nil {
		return Record{}, err
	}
	record.BlobKey = key
	record.CreatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO music_sources (sha256,blob_key,title,artist,uploader,source_url,acquired_at,file_name,media_type,size_bytes,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, record.SHA256, record.BlobKey, record.Title, record.Artist, record.Uploader, record.SourceURL, record.Acquired, record.FileName, record.MediaType, record.SizeBytes, record.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Record{}, fmt.Errorf("register source provenance: %w", err)
	}
	return s.Get(ctx, record.SHA256)
}

func (s *Store) Get(ctx context.Context, digest string) (Record, error) {
	var record Record
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT sha256,blob_key,title,artist,uploader,source_url,acquired_at,file_name,media_type,size_bytes,created_at FROM music_sources WHERE sha256=?`, digest).Scan(&record.SHA256, &record.BlobKey, &record.Title, &record.Artist, &record.Uploader, &record.SourceURL, &record.Acquired, &record.FileName, &record.MediaType, &record.SizeBytes, &created)
	if err != nil {
		return Record{}, err
	}
	record.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *Store) OpenAudio(ctx context.Context, digest string) (Record, io.ReadCloser, error) {
	record, err := s.Get(ctx, digest)
	if err != nil {
		return Record{}, nil, err
	}
	if err := verifyBlob(ctx, s.blobs, record.BlobKey, record.SHA256, record.SizeBytes); err != nil {
		return Record{}, nil, fmt.Errorf("verify private source before playback: %w", err)
	}
	reader, _, err := s.blobs.Get(ctx, record.BlobKey)
	if err != nil {
		return Record{}, nil, err
	}
	return record, reader, nil
}

func verifyBlob(ctx context.Context, blobs *musicstorage.Store, key, expected string, expectedSize int64) error {
	reader, _, err := blobs.Get(ctx, key)
	if err != nil {
		return err
	}
	defer reader.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(reader, MaxFileBytes+1))
	if err != nil {
		return err
	}
	if size != expectedSize || hex.EncodeToString(hash.Sum(nil)) != expected {
		return errors.New("stored source content does not match its SHA-256 and size")
	}
	return nil
}
