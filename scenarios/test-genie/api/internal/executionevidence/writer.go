package executionevidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const ManifestFile = "evidence-manifest.json"

const maxManifestBytes = 2 * 1024 * 1024

// Writer is the one write seam for run-owned evidence. It writes immutable
// files by temporary-file replacement and never returns artifact bytes to the
// caller, which keeps binary evidence out of terminal run state.
type Writer struct {
	runRoot string
}

func NewWriter(runRoot string) (*Writer, error) {
	if runRoot == "" {
		return nil, fmt.Errorf("evidence run root is required")
	}
	abs, err := filepath.Abs(runRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve evidence run root: %w", err)
	}
	return &Writer{runRoot: abs}, nil
}

// StreamArtifact copies one artifact through a size-limited stream and returns
// only its digest-backed reference. The destination is relative to this run;
// traversal and partially written oversized assets are rejected and removed.
func (w *Writer) StreamArtifact(ctx context.Context, id, kind, relativePath, contentType, phase string, source io.Reader, maxBytes int64) (ArtifactRef, error) {
	if w == nil || source == nil || maxBytes < 1 {
		return ArtifactRef{}, fmt.Errorf("%w: writer, source, and positive budget are required", ErrCorruptEvidence)
	}
	ref := ArtifactRef{ID: id, Kind: kind, RelativePath: relativePath, ContentType: contentType, Phase: phase}
	if err := ref.validateForWrite(); err != nil {
		return ArtifactRef{}, err
	}
	destination, err := w.resolve(ref.RelativePath)
	if err != nil {
		return ArtifactRef{}, err
	}
	hash := sha256.New()
	limited := &contextReader{ctx: ctx, source: io.LimitReader(source, maxBytes+1)}
	var written int64
	if err := writeTempAndPublish(destination, ".evidence-*.tmp", func(target io.Writer) error {
		var copyErr error
		written, copyErr = io.Copy(io.MultiWriter(target, hash), limited)
		if copyErr != nil {
			return fmt.Errorf("stream evidence artifact: %w", copyErr)
		}
		if written > maxBytes {
			return fmt.Errorf("%w: %s is %d bytes (limit %d)", ErrArtifactTooLarge, ref.ID, written, maxBytes)
		}
		return nil
	}, func(tempPath, destination string) error {
		return publishImmutableArtifact(tempPath, destination, written, hex.EncodeToString(hash.Sum(nil)))
	}); err != nil {
		return ArtifactRef{}, err
	}
	ref.SizeBytes = written
	ref.SHA256 = hex.EncodeToString(hash.Sum(nil))
	return ref, nil
}

// ReferenceExisting verifies a file written by a producer that already owns
// its streaming implementation, then returns its immutable reference without
// loading the file into memory or copying it into another store.
func (w *Writer) ReferenceExisting(id, kind, relativePath, contentType, phase string) (ArtifactRef, error) {
	if w == nil {
		return ArtifactRef{}, fmt.Errorf("evidence writer is required")
	}
	ref := ArtifactRef{ID: id, Kind: kind, RelativePath: relativePath, ContentType: contentType, Phase: phase}
	if err := ref.validateForWrite(); err != nil {
		return ArtifactRef{}, err
	}
	path, err := w.resolve(relativePath)
	if err != nil {
		return ArtifactRef{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return ArtifactRef{}, fmt.Errorf("open evidence artifact: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return ArtifactRef{}, fmt.Errorf("digest evidence artifact: %w", err)
	}
	ref.SizeBytes = size
	ref.SHA256 = hex.EncodeToString(hash.Sum(nil))
	return ref, nil
}

// WriteManifest publishes the canonical run manifest only after callers have
// completed all detailed writes referenced by it.
func (w *Writer) WriteManifest(manifest Manifest) error {
	if w == nil {
		return fmt.Errorf("evidence writer is required")
	}
	if err := manifest.Validate(); err != nil {
		return err
	}
	return w.writeJSONAtomically(ManifestFile, manifest)
}

// ManifestPath returns the manifest path under this writer's run root.
func (w *Writer) ManifestPath() string {
	if w == nil {
		return ""
	}
	return filepath.Join(w.runRoot, ManifestFile)
}

// ReadManifest reads the bounded summary index without touching the detailed
// findings payload. Callers that need an artifact's bytes must request that
// artifact explicitly through its digest-backed reference.
func ReadManifest(runRoot string) (Manifest, error) {
	file, err := os.Open(filepath.Join(runRoot, ManifestFile))
	if err != nil {
		return Manifest{}, fmt.Errorf("open evidence manifest: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, maxManifestBytes+1))
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode evidence manifest: %w", err)
	}
	if extra, err := decoder.Token(); err != io.EOF || extra != nil {
		if err == nil {
			return Manifest{}, fmt.Errorf("%w: manifest has trailing data", ErrCorruptEvidence)
		}
		return Manifest{}, fmt.Errorf("decode evidence manifest tail: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (w *Writer) writeJSONAtomically(relativePath string, value any) error {
	destination, err := w.resolve(relativePath)
	if err != nil {
		return err
	}
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal evidence manifest: %w", err)
	}
	return writeTempAndPublish(destination, ".manifest-*.tmp", func(target io.Writer) error {
		if _, err := target.Write(payload); err != nil {
			return fmt.Errorf("write evidence manifest: %w", err)
		}
		return nil
	}, func(tempPath, destination string) error {
		if err := os.Rename(tempPath, destination); err != nil {
			return fmt.Errorf("publish evidence manifest: %w", err)
		}
		return nil
	})
}

// writeTempAndPublish centralizes the temporary-file lifecycle used by both
// immutable artifacts and the replaceable manifest index. Publication policy
// stays with each caller while cleanup and close ordering remain consistent.
func writeTempAndPublish(destination, pattern string, write func(io.Writer) error, publish func(string, string) error) error {
	dir := filepath.Dir(destination)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create evidence directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return fmt.Errorf("create evidence temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	writeErr := write(tmp)
	closeErr := tmp.Close()
	if writeErr != nil {
		return errors.Join(writeErr, closeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close evidence temp file: %w", closeErr)
	}
	return publish(tmpPath, destination)
}

func publishImmutableArtifact(tempPath, destination string, size int64, digest string) error {
	if err := os.Link(tempPath, destination); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("publish immutable evidence artifact: %w", err)
	}
	existing, err := os.Open(destination)
	if err != nil {
		return fmt.Errorf("inspect existing immutable evidence artifact: %w", err)
	}
	hash := sha256.New()
	existingSize, readErr := io.Copy(hash, existing)
	closeErr := existing.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return fmt.Errorf("verify existing immutable evidence artifact: %w", err)
	}
	if existingSize != size || hex.EncodeToString(hash.Sum(nil)) != digest {
		return ErrArtifactConflict
	}
	return nil
}

func (w *Writer) resolve(relativePath string) (string, error) {
	if filepath.IsAbs(relativePath) || relativePath == "" {
		return "", fmt.Errorf("%w: invalid evidence path", ErrCorruptEvidence)
	}
	clean := filepath.Clean(relativePath)
	if clean == "." || clean == ".." || len(clean) >= 3 && clean[:3] == ".."+string(filepath.Separator) {
		return "", fmt.Errorf("%w: artifact path escapes run root", ErrCorruptEvidence)
	}
	return filepath.Join(w.runRoot, clean), nil
}

func (r ArtifactRef) validateForWrite() error {
	if r.ID == "" || r.Kind == "" || r.RelativePath == "" {
		return fmt.Errorf("%w: id, kind, and relative path are required", ErrCorruptEvidence)
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	source io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.source.Read(p)
	}
}
