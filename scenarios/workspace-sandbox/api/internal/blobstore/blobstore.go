// Package blobstore is the on-disk store for diff archive content blobs.
//
// Blobs are content-addressed by the SHA-256 of their **uncompressed**
// content and persisted as gzip-compressed files. They live under the
// scenario's ClassData root, scoped per-sandbox so retention can drop a
// sandbox's archive in a single directory removal:
//
//	<ClassData>/<app>/workspace-sandbox/archives/<sandbox_id>/<sha256>.gz
//
// The store is the disk-side of the hybrid DB+filesystem pattern in
// docs/internal/ARCHIVE_DESIGN.md. SQLite holds the metadata
// (sandbox_diff_archives); this package holds the bytes. The two are
// kept consistent by writing blobs first, then committing the SQL
// transaction that records them — see Service.snapshotDiff.
package blobstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/vrooli/api-core/storage"
)

// ScenarioID is the storage scope for workspace-sandbox blobs.
const ScenarioID = "workspace-sandbox"

// archivesSubdir is the subdirectory under ClassData that holds all
// per-sandbox blob trees.
const archivesSubdir = "archives"

// blobSuffix is appended to every on-disk blob filename. Content is
// gzipped so the suffix advertises the wire format to humans browsing
// the storage tree; the loader does not depend on it.
const blobSuffix = ".gz"

// ErrNotFound is returned by Get/Stat when no blob exists for the given
// sandbox ID + hash. Distinguished from generic IO errors so callers
// (HTTP handlers, retention) can map it to 404 instead of 500.
var ErrNotFound = errors.New("blobstore: blob not found")

// ErrInvalidSandboxID is returned when a caller passes a sandbox ID
// that is not a syntactically-valid identifier (UUID-shaped lowercase
// hex with dashes, by enforcement of the validator below). Treated as
// a programming error, not a user error.
var ErrInvalidSandboxID = errors.New("blobstore: invalid sandbox id")

// ErrInvalidHash is returned when a caller passes a hash that is not a
// 64-char lowercase hex string (i.e. not the output of sha256.Sum256
// formatted by encoding/hex).
var ErrInvalidHash = errors.New("blobstore: invalid sha256 hash")

// sandboxIDPattern is a strict allowlist for sandbox ID path segments.
// Sandbox IDs are UUIDs in canonical form (8-4-4-4-12 lowercase hex).
// Rejecting anything else here closes the path-traversal door before
// the value reaches filepath.Join.
var sandboxIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// hashPattern is a strict allowlist for SHA-256 hex strings.
var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// PutResult summarizes a successful Put.
type PutResult struct {
	// SHA256Hex is the lowercase hex SHA-256 of the uncompressed content.
	SHA256Hex string
	// SizeOnDisk is the number of bytes the gzip-compressed blob occupies.
	SizeOnDisk int64
	// SizeUncompressed is the original input length.
	SizeUncompressed int64
}

// BlobStore is the abstraction Phase 2 (snapshot service) and Phase 3
// (diff endpoint) consume. The interface lets tests substitute an
// in-memory or fault-injecting implementation without touching the
// real storage tree.
type BlobStore interface {
	// Put writes content for sandboxID and returns its content-address.
	// Put is idempotent: writing the same content twice yields the same
	// hash and overwrites the on-disk blob with byte-identical contents.
	Put(ctx context.Context, sandboxID string, content []byte) (PutResult, error)

	// Get returns the uncompressed content for sha256Hex under sandboxID,
	// or ErrNotFound when no such blob exists.
	Get(ctx context.Context, sandboxID, sha256Hex string) ([]byte, error)

	// Stat reports the on-disk (compressed) size and whether the blob
	// exists. Returns (0, false, nil) for a missing blob; non-nil
	// errors signal IO problems (permission, corruption) and never
	// "missing".
	Stat(ctx context.Context, sandboxID, sha256Hex string) (sizeOnDisk int64, exists bool, err error)

	// DeleteSandbox removes every blob for sandboxID. Idempotent:
	// removing a sandbox that has no blobs returns nil. Used both by
	// retention (eviction) and by Service.snapshotDiff's rollback path
	// (best-effort cleanup of partial blobs).
	DeleteSandbox(ctx context.Context, sandboxID string) error
}

// Store is the production BlobStore backed by api-core/storage.
type Store struct {
	resolver *storage.Resolver
}

// New constructs a Store that resolves paths through resolver. The
// resolver MUST be configured with the same AppID the rest of the API
// uses (typically "vrooli") so blobs share the storage tree with the
// SQLite database that records them.
func New(resolver *storage.Resolver) (*Store, error) {
	if resolver == nil {
		return nil, errors.New("blobstore: resolver is nil")
	}
	return &Store{resolver: resolver}, nil
}

// Put writes content as a gzipped blob, content-addressed by the
// SHA-256 of the **uncompressed** input. The compressed bytes are
// written atomically (temp file + fsync + rename) via
// storage.WriteFileAtomic.
func (s *Store) Put(_ context.Context, sandboxID string, content []byte) (PutResult, error) {
	if !sandboxIDPattern.MatchString(sandboxID) {
		return PutResult{}, fmt.Errorf("%w: %q", ErrInvalidSandboxID, sandboxID)
	}

	sum := sha256.Sum256(content)
	hashHex := hex.EncodeToString(sum[:])

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(content); err != nil {
		return PutResult{}, fmt.Errorf("blobstore: gzip write: %w", err)
	}
	if err := gw.Close(); err != nil {
		return PutResult{}, fmt.Errorf("blobstore: gzip close: %w", err)
	}

	path, err := s.blobPath(sandboxID, hashHex)
	if err != nil {
		return PutResult{}, err
	}
	if err := storage.WriteFileAtomic(path, buf.Bytes(), 0o644); err != nil {
		return PutResult{}, fmt.Errorf("blobstore: write blob: %w", err)
	}

	return PutResult{
		SHA256Hex:        hashHex,
		SizeOnDisk:       int64(buf.Len()),
		SizeUncompressed: int64(len(content)),
	}, nil
}

// Get reads the gzipped blob at sandboxID/sha256Hex and returns its
// decompressed content. Returns ErrNotFound when the blob does not
// exist; any other error is an IO or corruption failure.
func (s *Store) Get(_ context.Context, sandboxID, sha256Hex string) ([]byte, error) {
	if !sandboxIDPattern.MatchString(sandboxID) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidSandboxID, sandboxID)
	}
	if !hashPattern.MatchString(sha256Hex) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidHash, sha256Hex)
	}

	path, err := s.blobPath(sandboxID, sha256Hex)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path) // #nosec G304 -- path built from validated allowlist
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("blobstore: open blob: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("blobstore: gzip reader (%s): %w", path, err)
	}
	defer gr.Close()

	out, err := io.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("blobstore: gzip read (%s): %w", path, err)
	}

	// Verify content-address: a corrupted blob whose hash no longer
	// matches its filename is an integrity failure, not a 404. We
	// surface it as a distinct error so callers can log/alert.
	verify := sha256.Sum256(out)
	if hex.EncodeToString(verify[:]) != sha256Hex {
		return nil, fmt.Errorf("blobstore: hash mismatch for %s/%s (file is corrupted)", sandboxID, sha256Hex)
	}

	return out, nil
}

// Stat reports whether a blob exists and its on-disk size. A missing
// blob is reported as (0, false, nil) — not an error.
func (s *Store) Stat(_ context.Context, sandboxID, sha256Hex string) (int64, bool, error) {
	if !sandboxIDPattern.MatchString(sandboxID) {
		return 0, false, fmt.Errorf("%w: %q", ErrInvalidSandboxID, sandboxID)
	}
	if !hashPattern.MatchString(sha256Hex) {
		return 0, false, fmt.Errorf("%w: %q", ErrInvalidHash, sha256Hex)
	}

	path, err := s.blobPath(sandboxID, sha256Hex)
	if err != nil {
		return 0, false, err
	}

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("blobstore: stat blob: %w", err)
	}
	return info.Size(), true, nil
}

// DeleteSandbox removes the per-sandbox blob directory and everything
// below it. Idempotent — succeeds with nil error if the directory is
// already absent.
func (s *Store) DeleteSandbox(_ context.Context, sandboxID string) error {
	if !sandboxIDPattern.MatchString(sandboxID) {
		return fmt.Errorf("%w: %q", ErrInvalidSandboxID, sandboxID)
	}
	dir, err := s.sandboxDir(sandboxID)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("blobstore: remove sandbox dir %q: %w", dir, err)
	}
	return nil
}

// sandboxDir returns the absolute path of the per-sandbox blob
// directory. The Resolver enforces that the relative segment cannot
// escape the class root; we additionally pre-validated sandboxID.
func (s *Store) sandboxDir(sandboxID string) (string, error) {
	rel := filepath.Join(archivesSubdir, sandboxID)
	abs, err := s.resolver.Path(
		storage.Options{ScenarioID: ScenarioID},
		storage.ClassData,
		rel,
	)
	if err != nil {
		return "", fmt.Errorf("blobstore: resolve %q: %w", rel, err)
	}
	return abs, nil
}

// blobPath returns the absolute path of a single blob.
func (s *Store) blobPath(sandboxID, sha256Hex string) (string, error) {
	rel := filepath.Join(archivesSubdir, sandboxID, sha256Hex+blobSuffix)
	abs, err := s.resolver.Path(
		storage.Options{ScenarioID: ScenarioID},
		storage.ClassData,
		rel,
	)
	if err != nil {
		return "", fmt.Errorf("blobstore: resolve %q: %w", rel, err)
	}
	return abs, nil
}
