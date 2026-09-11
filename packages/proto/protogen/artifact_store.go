package protogen

// This file owns the runtime-home publication boundary for generated Proto.
// The repository's packages/proto/gen tree remains a collaboration/compatibility
// view; lifecycle readers use an immutable snapshot selected by this store.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vrooli/platform-go"
)

const (
	artifactMetadataSchemaVersion  = 1
	artifactSelectionSchemaVersion = 1
	artifactLeaseSchemaVersion     = 1
	defaultReaderLeaseTTL          = 2 * time.Minute
	defaultSnapshotRetention       = 24 * time.Hour
)

var (
	ErrArtifactNotFound       = errors.New("proto artifact not found")
	ErrArtifactIncomplete     = errors.New("proto artifact incomplete")
	ErrArtifactDigestMismatch = errors.New("proto artifact digest mismatch")
	ErrArtifactRecordCorrupt  = errors.New("proto artifact selection record corrupt")
	ErrArtifactPublication    = errors.New("proto artifact publication failed")
	ErrArtifactPathEscape     = errors.New("proto artifact path escapes store")
	ErrArtifactSourceDrift    = errors.New("proto artifact source changed during generation")
)

const (
	ArtifactErrorSourceInvalid        = "source-invalid"
	ArtifactErrorToolFailed           = "tool-failed"
	ArtifactErrorLockTimeout          = "lock-timeout"
	ArtifactErrorCandidateInvalid     = "candidate-invalid"
	ArtifactErrorPublicationFailed    = "publication-failed"
	ArtifactErrorActiveRecordCorrupt  = "active-record-corrupt"
	ArtifactErrorMissingLastGood      = "missing-last-good"
	ArtifactErrorConsumerIncompatible = "consumer-incompatible"
)

// ArtifactErrorCode maps public storage errors to stable operator-facing
// categories. Callers should log the code instead of parsing error text.
func ArtifactErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrArtifactSourceDrift):
		return ArtifactErrorSourceInvalid
	case errors.Is(err, ErrArtifactPublication):
		return ArtifactErrorPublicationFailed
	case errors.Is(err, ErrArtifactRecordCorrupt):
		return ArtifactErrorActiveRecordCorrupt
	case errors.Is(err, ErrArtifactNotFound):
		return ArtifactErrorMissingLastGood
	case errors.Is(err, ErrArtifactIncomplete), errors.Is(err, ErrArtifactDigestMismatch), errors.Is(err, ErrArtifactPathEscape):
		return ArtifactErrorCandidateInvalid
	default:
		return ArtifactErrorToolFailed
	}
}

var leaseSequence atomic.Uint64

// ArtifactOutput is a generated file relative to the snapshot's gen directory.
// Paths use slash separators in metadata on every host.
type ArtifactOutput struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

// ArtifactMetadata identifies one complete generated result. It is deliberately
// self-describing: a reader never needs the mutable source tree to validate it.
type ArtifactMetadata struct {
	SchemaVersion    int              `json:"schema_version"`
	ArtifactID       string           `json:"artifact_id"`
	SourceDigest     string           `json:"source_digest"`
	GeneratorDigest  string           `json:"generator_digest"`
	GeneratorVersion string           `json:"generator_version,omitempty"`
	ToolIdentity     string           `json:"tool_identity,omitempty"`
	GeneratedAt      time.Time        `json:"generated_at"`
	InputFiles       []string         `json:"input_files,omitempty"`
	Scope            []string         `json:"scope,omitempty"`
	ParentArtifactID string           `json:"parent_artifact_id,omitempty"`
	Outputs          []ArtifactOutput `json:"outputs"`
	Descriptor       string           `json:"descriptor_digest,omitempty"`
	Manifest         string           `json:"manifest_digest,omitempty"`
	ValidationRef    string           `json:"validation_ref,omitempty"`
	ValidationStatus string           `json:"validation_status,omitempty"`
}

type selectionRecord struct {
	SchemaVersion int       `json:"schema_version"`
	ArtifactID    string    `json:"artifact_id"`
	Snapshot      string    `json:"snapshot"`
	Metadata      string    `json:"metadata_digest"`
	SelectedAt    time.Time `json:"selected_at"`
}

type leaseRecord struct {
	SchemaVersion int       `json:"schema_version"`
	LeaseID       string    `json:"lease_id"`
	ArtifactID    string    `json:"artifact_id"`
	Owner         string    `json:"owner"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// Candidate is the output directory produced by a generator. SourceRoot must
// contain generated files directly (go/, python/, typescript/, descriptor/,
// manifests/); it is copied into a private snapshot and never published in
// place.
type Candidate struct {
	Metadata          ArtifactMetadata
	SourceRoot        string
	ValidationReceipt *ArtifactValidationReceipt
}

// ArtifactValidationReceipt records the checks that authorized publication.
// Detailed tool output remains in the owning validation system; this compact
// receipt travels with the immutable snapshot.
type ArtifactValidationReceipt struct {
	SchemaVersion int       `json:"schema_version"`
	ArtifactID    string    `json:"artifact_id"`
	SourceDigest  string    `json:"source_digest"`
	Status        string    `json:"status"`
	Checks        []string  `json:"checks"`
	RecordedAt    time.Time `json:"recorded_at"`
}

type ArtifactSelectionStatus struct {
	Path     string           `json:"path"`
	Artifact string           `json:"artifact_id,omitempty"`
	Snapshot string           `json:"snapshot,omitempty"`
	Metadata ArtifactMetadata `json:"metadata,omitempty"`
	Valid    bool             `json:"valid"`
	Error    string           `json:"error,omitempty"`
}

type ArtifactStoreStatus struct {
	Root          string                  `json:"root"`
	Active        ArtifactSelectionStatus `json:"active"`
	LastKnownGood ArtifactSelectionStatus `json:"last_known_good"`
	Snapshots     []string                `json:"snapshots"`
	Leases        []leaseRecord           `json:"leases,omitempty"`
}

// BuildArtifactMetadata derives a complete output inventory from a generated
// tree. Callers use this only after generation and consumer checks have
// succeeded; Publish repeats the file-level validation before activation.
func BuildArtifactMetadata(root, sourceDigest, generatorDigest string, inputFiles, scope []string, parentArtifactID string) (ArtifactMetadata, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || root == "" {
		return ArtifactMetadata{}, fmt.Errorf("generated root is required")
	}
	info, err := os.Stat(root)
	if err != nil {
		return ArtifactMetadata{}, err
	}
	if !info.IsDir() {
		return ArtifactMetadata{}, fmt.Errorf("generated root is not a directory")
	}
	var outputs []ArtifactOutput
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported generated entry %s", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		digest, err := artifactFileDigest(path)
		if err != nil {
			return err
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		outputs = append(outputs, ArtifactOutput{Path: filepath.ToSlash(rel), Digest: digest, Size: fileInfo.Size()})
		return nil
	})
	if err != nil {
		return ArtifactMetadata{}, err
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Path < outputs[j].Path })
	metadata := ArtifactMetadata{
		SchemaVersion:    artifactMetadataSchemaVersion,
		SourceDigest:     strings.TrimSpace(sourceDigest),
		GeneratorDigest:  strings.TrimSpace(generatorDigest),
		GeneratedAt:      time.Now().UTC(),
		InputFiles:       append([]string(nil), inputFiles...),
		Scope:            append([]string(nil), scope...),
		ParentArtifactID: strings.TrimSpace(parentArtifactID),
		Outputs:          outputs,
	}
	metadata.ArtifactID = artifactID(metadata)
	if err := validateMetadata(metadata); err != nil {
		return ArtifactMetadata{}, fmt.Errorf("%w: %v", ErrArtifactIncomplete, err)
	}
	return metadata, nil
}

// PublicationHooks are test seams for the durable publication boundaries. A
// production caller leaves them nil. Returning an error simulates a crash or
// interrupted filesystem operation before the selection rename.
type PublicationHooks struct {
	BeforeSnapshotRename func() error
	BeforeActiveRename   func() error
	BeforeLastGoodRename func() error
}

// Store manages one artifact namespace under runtime home. The root is not
// inside the repository so a shared-worktree edit cannot partially overwrite
// the artifact a running scenario uses.
type Store struct {
	Root      string
	LeaseTTL  time.Duration
	Retention time.Duration
	Hooks     PublicationHooks
	Now       func() time.Time

	mu sync.Mutex
}

func DefaultArtifactRoot(home string) string {
	if strings.TrimSpace(home) == "" {
		if resolved, err := platform.ResolveHomePath("", filepath.Join(".vrooli", "artifacts", "proto")); err == nil {
			return resolved
		}
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".vrooli", "artifacts", "proto")
}

func NewArtifactStore(root string) (*Store, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || root == "" {
		return nil, fmt.Errorf("artifact root is required")
	}
	return &Store{Root: root, LeaseTTL: defaultReaderLeaseTTL, Retention: defaultSnapshotRetention, Now: time.Now}, nil
}

// Status is intentionally observational: malformed pointers are returned as
// data so an operator can diagnose them, while cleanup and resolution remain
// fail-closed.
func (s *Store) Status(ctx context.Context) (ArtifactStoreStatus, error) {
	if err := ctx.Err(); err != nil {
		return ArtifactStoreStatus{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	status := ArtifactStoreStatus{Root: s.Root}
	status.Active = s.selectionStatus(s.activePath())
	status.LastKnownGood = s.selectionStatus(s.lastGoodPath())
	entries, err := os.ReadDir(s.snapshotsRoot())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return ArtifactStoreStatus{}, err
	}
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			status.Snapshots = append(status.Snapshots, entry.Name())
		}
	}
	sort.Strings(status.Snapshots)
	_, _, leaseRoot := s.directories()
	leaseEntries, err := os.ReadDir(leaseRoot)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return ArtifactStoreStatus{}, err
	}
	for _, entry := range leaseEntries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if lease, leaseErr := readLease(filepath.Join(leaseRoot, entry.Name())); leaseErr == nil {
			status.Leases = append(status.Leases, lease)
		}
	}
	return status, nil
}

func (s *Store) selectionStatus(path string) ArtifactSelectionStatus {
	status := ArtifactSelectionStatus{Path: path}
	record, err := s.readSelection(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			status.Error = err.Error()
		}
		return status
	}
	status.Artifact = record.ArtifactID
	status.Snapshot = record.Snapshot
	snapshot, err := s.snapshotForSelectionLocked(record)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	status.Metadata = snapshot.Metadata
	status.Valid = true
	_ = snapshot.Close()
	return status
}

// Promote selects an already validated snapshot. It is used by explicit
// operator promotion and rollback commands; it never constructs or modifies
// generated output.
func (s *Store) Promote(ctx context.Context, artifactID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" || strings.ContainsAny(artifactID, `/\\`) || artifactID == "." || artifactID == ".." {
		return fmt.Errorf("invalid artifact id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.snapshotsRoot(), artifactID)
	metadata, err := readMetadata(filepath.Join(path, "metadata.json"))
	if err != nil {
		return fmt.Errorf("validate promotion target: %w: %v", ErrArtifactIncomplete, err)
	}
	if err := validateSnapshot(path, metadata); err != nil {
		return fmt.Errorf("validate promotion target: %w", err)
	}
	record := selectionRecord{SchemaVersion: artifactSelectionSchemaVersion, ArtifactID: metadata.ArtifactID, Snapshot: filepath.ToSlash(filepath.Join("snapshots", metadata.ArtifactID)), Metadata: metadataDigest(metadata), SelectedAt: s.now()}
	if err := s.publishSelection(s.activePath(), record, nil); err != nil {
		return err
	}
	return s.publishSelection(s.lastGoodPath(), record, nil)
}

func (s *Store) directories() (snapshots, candidates, leases string) {
	return filepath.Join(s.Root, "snapshots"), filepath.Join(s.Root, "candidates"), filepath.Join(s.Root, "leases")
}

func (s *Store) activePath() string   { return filepath.Join(s.Root, "active.json") }
func (s *Store) lastGoodPath() string { return filepath.Join(s.Root, "last-known-good.json") }

// Publish copies and validates a candidate, then exposes it through an atomic
// selection record. A failed copy, validation, or pointer write leaves any old
// active and last-known-good records untouched.
func (s *Store) Publish(ctx context.Context, candidate Candidate) (ArtifactMetadata, error) {
	if err := ctx.Err(); err != nil {
		return ArtifactMetadata{}, err
	}
	metadata := candidate.Metadata
	if metadata.GeneratedAt.IsZero() {
		metadata.GeneratedAt = s.now()
	}
	if metadata.ArtifactID == "" {
		metadata.ArtifactID = artifactID(metadata)
	}
	if err := validateMetadata(metadata); err != nil {
		return ArtifactMetadata{}, fmt.Errorf("%w: %v", ErrArtifactIncomplete, err)
	}
	if strings.TrimSpace(metadata.ValidationRef) != "" && candidate.ValidationReceipt == nil {
		return ArtifactMetadata{}, fmt.Errorf("%w: validation receipt %q is required", ErrArtifactIncomplete, metadata.ValidationRef)
	}
	if err := validateSourceRoot(candidate.SourceRoot, metadata); err != nil {
		return ArtifactMetadata{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	snapshots, candidatesRoot, leases := s.directories()
	if err := os.MkdirAll(snapshots, 0o755); err != nil {
		return ArtifactMetadata{}, fmt.Errorf("create snapshots directory: %w", err)
	}
	if err := os.MkdirAll(candidatesRoot, 0o755); err != nil {
		return ArtifactMetadata{}, fmt.Errorf("create candidates directory: %w", err)
	}
	if err := os.MkdirAll(leases, 0o755); err != nil {
		return ArtifactMetadata{}, fmt.Errorf("create leases directory: %w", err)
	}

	metadata.GeneratedAt = metadata.GeneratedAt.UTC()
	tmp, err := os.MkdirTemp(candidatesRoot, ".candidate-")
	if err != nil {
		return ArtifactMetadata{}, fmt.Errorf("create candidate: %w", err)
	}
	keepCandidate := false
	defer func() {
		if !keepCandidate {
			_ = os.RemoveAll(tmp)
		}
	}()

	genRoot := filepath.Join(tmp, "gen")
	if err := copyTree(ctx, candidate.SourceRoot, genRoot); err != nil {
		return ArtifactMetadata{}, fmt.Errorf("copy candidate: %w", err)
	}
	if err := writeJSON(filepath.Join(tmp, "metadata.json"), metadata); err != nil {
		return ArtifactMetadata{}, fmt.Errorf("write artifact metadata: %w", err)
	}
	if candidate.ValidationReceipt != nil {
		receipt := *candidate.ValidationReceipt
		if receipt.SchemaVersion == 0 {
			receipt.SchemaVersion = artifactMetadataSchemaVersion
		}
		if receipt.ArtifactID == "" {
			receipt.ArtifactID = metadata.ArtifactID
		}
		if receipt.SourceDigest == "" {
			receipt.SourceDigest = metadata.SourceDigest
		}
		if receipt.RecordedAt.IsZero() {
			receipt.RecordedAt = now
		}
		if err := writeJSON(filepath.Join(tmp, "validation-receipt.json"), receipt); err != nil {
			return ArtifactMetadata{}, fmt.Errorf("write validation receipt: %w", err)
		}
	}
	if err := validateSnapshot(tmp, metadata); err != nil {
		return ArtifactMetadata{}, err
	}

	snapshotTmp := filepath.Join(snapshots, ".snapshot-"+metadata.ArtifactID)
	_ = os.RemoveAll(snapshotTmp)
	if err := os.Rename(tmp, snapshotTmp); err != nil {
		return ArtifactMetadata{}, fmt.Errorf("stage snapshot: %w", err)
	}
	keepCandidate = true
	snapshotPath := filepath.Join(snapshots, metadata.ArtifactID)
	if existing, err := os.Stat(snapshotPath); err == nil && existing.IsDir() {
		// Content-addressed publication is idempotent. Validate the existing
		// object before reusing it; never replace a known object with unverified
		// bytes.
		existingMetadata, err := readMetadata(filepath.Join(snapshotPath, "metadata.json"))
		if err != nil {
			return ArtifactMetadata{}, fmt.Errorf("existing snapshot %s: %w", metadata.ArtifactID, err)
		}
		if err := validateSnapshot(snapshotPath, metadata); err != nil {
			_ = os.RemoveAll(snapshotTmp)
			return ArtifactMetadata{}, fmt.Errorf("existing snapshot %s: %w", metadata.ArtifactID, err)
		}
		// The artifact ID intentionally excludes volatile metadata such as
		// GeneratedAt. Reuse the immutable snapshot metadata for the selection
		// record; otherwise a repeat publication would point at this snapshot
		// with a digest for a different, never-published metadata value.
		metadata = existingMetadata
		_ = os.RemoveAll(snapshotTmp)
	} else {
		if s.Hooks.BeforeSnapshotRename != nil {
			if err := s.Hooks.BeforeSnapshotRename(); err != nil {
				_ = os.RemoveAll(snapshotTmp)
				return ArtifactMetadata{}, fmt.Errorf("%w: snapshot: %v", ErrArtifactPublication, err)
			}
		}
		if err := os.Rename(snapshotTmp, snapshotPath); err != nil {
			_ = os.RemoveAll(snapshotTmp)
			return ArtifactMetadata{}, fmt.Errorf("%w: install snapshot: %v", ErrArtifactPublication, err)
		}
		if err := syncDirectory(snapshots); err != nil {
			return ArtifactMetadata{}, fmt.Errorf("durably publish snapshot: %w", err)
		}
	}
	if err := makeReadOnly(snapshotPath); err != nil {
		return ArtifactMetadata{}, fmt.Errorf("make snapshot immutable: %w", err)
	}

	previous, previousErr := s.readSelection(s.activePath())
	if previousErr != nil && !errors.Is(previousErr, fs.ErrNotExist) {
		return ArtifactMetadata{}, previousErr
	}
	active := selectionRecord{SchemaVersion: artifactSelectionSchemaVersion, ArtifactID: metadata.ArtifactID, Snapshot: filepath.ToSlash(filepath.Join("snapshots", metadata.ArtifactID)), Metadata: metadataDigest(metadata), SelectedAt: now}
	if err := s.publishSelection(s.activePath(), active, s.Hooks.BeforeActiveRename); err != nil {
		return ArtifactMetadata{}, err
	}
	// last-known-good is updated only after active has become readable. If this
	// second pointer fails, active and the previous last-good are both valid.
	lastGood := active
	if previousErr == nil {
		if previous.ArtifactID == metadata.ArtifactID {
			lastGood = active
		}
	}
	if err := s.publishSelection(s.lastGoodPath(), lastGood, s.Hooks.BeforeLastGoodRename); err != nil {
		return ArtifactMetadata{}, err
	}
	return metadata, nil
}

func (s *Store) publishSelection(path string, record selectionRecord, beforeRename func() error) error {
	if err := validateSelection(record); err != nil {
		return fmt.Errorf("%w: %v", ErrArtifactRecordCorrupt, err)
	}
	tmp := path + fmt.Sprintf(".tmp-%d-%d", os.Getpid(), leaseSequence.Add(1))
	if err := writeJSON(tmp, record); err != nil {
		return fmt.Errorf("%w: write %s: %v", ErrArtifactPublication, filepath.Base(path), err)
	}
	defer os.Remove(tmp)
	if beforeRename != nil {
		if err := beforeRename(); err != nil {
			return fmt.Errorf("%w: %s: %v", ErrArtifactPublication, filepath.Base(path), err)
		}
	}
	f, err := os.OpenFile(tmp, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("%w: open %s: %v", ErrArtifactPublication, filepath.Base(path), err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("%w: sync %s: %v", ErrArtifactPublication, filepath.Base(path), err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("%w: close %s: %v", ErrArtifactPublication, filepath.Base(path), err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("%w: rename %s: %v", ErrArtifactPublication, filepath.Base(path), err)
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return fmt.Errorf("%w: sync selection directory: %v", ErrArtifactPublication, err)
	}
	return nil
}

// Resolve returns a validated snapshot and a lease. The lease must be closed
// by the caller after the consumer no longer reads generated files.
func (s *Store) Resolve(ctx context.Context) (*Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.readSelection(s.activePath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return s.resolveLastGoodLocked()
		}
		if fallback, fallbackErr := s.readSelection(s.lastGoodPath()); fallbackErr == nil {
			fallbackSnapshot, fallbackSnapshotErr := s.snapshotForSelectionLocked(fallback)
			if fallbackSnapshotErr == nil {
				fallbackSnapshot.Selection = "last-known-good"
				fallbackSnapshot.Degraded = true
			}
			return fallbackSnapshot, fallbackSnapshotErr
		}
		return nil, fmt.Errorf("%w: active: %v", ErrArtifactRecordCorrupt, err)
	}
	snapshot, snapshotErr := s.snapshotForSelectionLocked(record)
	if snapshotErr == nil {
		snapshot.Selection = "active"
		return snapshot, nil
	}
	// A syntactically valid active record can still name a missing, incomplete,
	// or tampered snapshot. Treat that as an invalid active selection and use
	// last-known-good after independently validating it.
	if fallback, fallbackErr := s.readSelection(s.lastGoodPath()); fallbackErr == nil && fallback.ArtifactID != record.ArtifactID {
		if fallbackSnapshot, fallbackSnapshotErr := s.snapshotForSelectionLocked(fallback); fallbackSnapshotErr == nil {
			fallbackSnapshot.Selection = "last-known-good"
			fallbackSnapshot.Degraded = true
			return fallbackSnapshot, nil
		}
	}
	return nil, snapshotErr
}

func (s *Store) resolveLastGoodLocked() (*Snapshot, error) {
	record, err := s.readSelection(s.lastGoodPath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrArtifactNotFound
		}
		return nil, fmt.Errorf("%w: last-known-good: %v", ErrArtifactRecordCorrupt, err)
	}
	snapshot, err := s.snapshotForSelectionLocked(record)
	if err == nil {
		snapshot.Selection = "last-known-good"
		snapshot.Degraded = true
	}
	return snapshot, err
}

func (s *Store) snapshotForSelectionLocked(record selectionRecord) (*Snapshot, error) {
	if err := validateSelection(record); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrArtifactRecordCorrupt, err)
	}
	path, err := secureJoin(s.Root, record.Snapshot)
	if err != nil {
		return nil, err
	}
	metadata, err := readMetadata(filepath.Join(path, "metadata.json"))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrArtifactIncomplete, err)
	}
	if metadata.ArtifactID != record.ArtifactID || metadataDigest(metadata) != record.Metadata {
		return nil, fmt.Errorf("%w: selection metadata does not match snapshot", ErrArtifactDigestMismatch)
	}
	if err := validateSnapshot(path, metadata); err != nil {
		return nil, err
	}
	lease, err := s.acquireLeaseLocked(metadata.ArtifactID)
	if err != nil {
		return nil, err
	}
	return &Snapshot{ArtifactID: metadata.ArtifactID, Root: filepath.Join(path, "gen"), Metadata: metadata, lease: lease}, nil
}

type Snapshot struct {
	ArtifactID string
	Root       string
	Metadata   ArtifactMetadata
	Selection  string
	Degraded   bool
	lease      *ReaderLease
}

// MaterializeCompatibilityView atomically replaces a legacy package output
// directory with the contents of a resolved snapshot. The view exists only
// for consumers whose language toolchain still requires a fixed local module
// path; lifecycle holds the shared Proto lock across setup and build. A
// previous view is retained until the replacement is installed so an
// interrupted replacement can be repaired on the next call.
func (s *Store) MaterializeCompatibilityView(ctx context.Context, snapshot *Snapshot, target string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if snapshot == nil || strings.TrimSpace(snapshot.GenRoot()) == "" {
		return fmt.Errorf("snapshot is required")
	}
	target = filepath.Clean(strings.TrimSpace(target))
	if target == "." || target == "" {
		return fmt.Errorf("compatibility target is required")
	}
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	if err := recoverCompatibilityView(target); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, ".proto-compatibility-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := copyTree(ctx, snapshot.GenRoot(), tmp); err != nil {
		return fmt.Errorf("copy selected Proto snapshot: %w", err)
	}
	backup := target + ".previous"
	_ = os.RemoveAll(backup)
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, backup); err != nil {
			return fmt.Errorf("stage compatibility view replacement: %w", err)
		}
	}
	if err := os.Rename(tmp, target); err != nil {
		if _, restoreErr := os.Stat(backup); restoreErr == nil {
			_ = os.Rename(backup, target)
		}
		return fmt.Errorf("install compatibility view: %w", err)
	}
	if err := syncDirectory(parent); err != nil {
		return fmt.Errorf("durably install compatibility view: %w", err)
	}
	if err := os.RemoveAll(backup); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove previous compatibility view: %w", err)
	}
	return syncDirectory(parent)
}

func recoverCompatibilityView(target string) error {
	backup := target + ".previous"
	_, targetErr := os.Stat(target)
	_, backupErr := os.Stat(backup)
	if targetErr == nil {
		if backupErr == nil {
			return os.RemoveAll(backup)
		}
		if !errors.Is(backupErr, fs.ErrNotExist) {
			return backupErr
		}
		return nil
	}
	if !errors.Is(targetErr, fs.ErrNotExist) {
		return targetErr
	}
	if backupErr == nil {
		if err := os.Rename(backup, target); err != nil {
			return fmt.Errorf("recover compatibility view: %w", err)
		}
		return syncDirectory(filepath.Dir(target))
	}
	if !errors.Is(backupErr, fs.ErrNotExist) {
		return backupErr
	}
	return nil
}

func (s *Snapshot) GenRoot() string {
	if s == nil {
		return ""
	}
	return s.Root
}

func (s *Snapshot) Close() error {
	if s == nil || s.lease == nil {
		return nil
	}
	return s.lease.Close()
}

type ReaderLease struct {
	store  *Store
	record leaseRecord
	path   string
	once   sync.Once
}

func (s *Store) acquireLeaseLocked(artifactID string) (*ReaderLease, error) {
	_, _, leases := s.directories()
	if err := os.MkdirAll(leases, 0o755); err != nil {
		return nil, fmt.Errorf("create reader leases: %w", err)
	}
	now := s.now()
	leaseID := fmt.Sprintf("%d-%d-%d", now.UnixNano(), os.Getpid(), leaseSequence.Add(1))
	record := leaseRecord{SchemaVersion: artifactLeaseSchemaVersion, LeaseID: leaseID, ArtifactID: artifactID, Owner: fmt.Sprintf("pid:%d", os.Getpid()), CreatedAt: now, ExpiresAt: now.Add(s.leaseTTL())}
	path := filepath.Join(leases, leaseID+".json")
	if err := writeJSON(path, record); err != nil {
		return nil, fmt.Errorf("create reader lease: %w", err)
	}
	return &ReaderLease{store: s, record: record, path: path}, nil
}

func (l *ReaderLease) Close() error {
	if l == nil {
		return nil
	}
	var err error
	l.once.Do(func() { err = os.Remove(l.path) })
	return err
}

func (s *Store) Collect(ctx context.Context, now time.Time) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	active, activeErr := s.readSelection(s.activePath())
	lastGood, lastGoodErr := s.readSelection(s.lastGoodPath())
	if activeErr != nil && !errors.Is(activeErr, fs.ErrNotExist) {
		return 0, fmt.Errorf("active selection is corrupt; refusing cleanup: %w", activeErr)
	}
	if lastGoodErr != nil && !errors.Is(lastGoodErr, fs.ErrNotExist) {
		return 0, fmt.Errorf("last-known-good selection is corrupt; refusing cleanup: %w", lastGoodErr)
	}
	protected := map[string]bool{}
	if activeErr == nil {
		protected[active.ArtifactID] = true
	}
	if lastGoodErr == nil {
		protected[lastGood.ArtifactID] = true
	}
	leases := map[string]bool{}
	_, _, leaseRoot := s.directories()
	entries, err := os.ReadDir(leaseRoot)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return 0, err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		record, readErr := readLease(filepath.Join(leaseRoot, entry.Name()))
		if readErr == nil && record.ExpiresAt.After(now) {
			leases[record.ArtifactID] = true
		}
	}
	entries, err = os.ReadDir(s.snapshotsRoot())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || protected[entry.Name()] || leases[entry.Name()] {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil || now.Sub(info.ModTime()) < s.retention() {
			continue
		}
		if err := os.RemoveAll(filepath.Join(s.snapshotsRoot(), entry.Name())); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func (s *Store) snapshotsRoot() string { snapshots, _, _ := s.directories(); return snapshots }

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Store) leaseTTL() time.Duration {
	if s.LeaseTTL > 0 {
		return s.LeaseTTL
	}
	return defaultReaderLeaseTTL
}

func (s *Store) retention() time.Duration {
	if s.Retention > 0 {
		return s.Retention
	}
	return defaultSnapshotRetention
}

func validateMetadata(metadata ArtifactMetadata) error {
	if metadata.SchemaVersion != artifactMetadataSchemaVersion {
		return fmt.Errorf("unsupported metadata schema %d", metadata.SchemaVersion)
	}
	if strings.TrimSpace(metadata.ArtifactID) == "" {
		return fmt.Errorf("artifact id is empty")
	}
	if strings.TrimSpace(metadata.SourceDigest) == "" {
		return fmt.Errorf("source digest is empty")
	}
	if len(metadata.Outputs) == 0 {
		return fmt.Errorf("outputs are empty")
	}
	seen := map[string]bool{}
	for _, output := range metadata.Outputs {
		path := filepath.ToSlash(filepath.Clean(output.Path))
		if path == "." || strings.HasPrefix(path, "../") || strings.Contains(path, "/../") || filepath.IsAbs(path) || seen[path] {
			return fmt.Errorf("invalid or duplicate output path %q", output.Path)
		}
		if !strings.HasPrefix(output.Digest, "sha256:") || output.Size < 0 {
			return fmt.Errorf("invalid output metadata for %q", output.Path)
		}
		seen[path] = true
	}
	for _, input := range metadata.InputFiles {
		input = filepath.ToSlash(filepath.Clean(input))
		if input == "." || strings.HasPrefix(input, "../") || strings.Contains(input, "/../") || filepath.IsAbs(input) {
			return fmt.Errorf("invalid input path %q", input)
		}
	}
	return nil
}

func validateSelection(record selectionRecord) error {
	if record.SchemaVersion != artifactSelectionSchemaVersion || strings.TrimSpace(record.ArtifactID) == "" || strings.TrimSpace(record.Metadata) == "" {
		return fmt.Errorf("invalid selection fields")
	}
	if filepath.IsAbs(record.Snapshot) || strings.HasPrefix(filepath.ToSlash(record.Snapshot), "../") {
		return ErrArtifactPathEscape
	}
	return nil
}

func validateSourceRoot(root string, metadata ArtifactMetadata) error {
	if strings.TrimSpace(root) == "" {
		return fmt.Errorf("candidate source root is empty")
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return fmt.Errorf("candidate source root is not a directory: %w", err)
	}
	for _, output := range metadata.Outputs {
		path, err := secureJoin(root, output.Path)
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("%w: missing output %s", ErrArtifactIncomplete, output.Path)
		}
		if info.Size() != output.Size {
			return fmt.Errorf("%w: size mismatch for %s", ErrArtifactDigestMismatch, output.Path)
		}
		digest, err := artifactFileDigest(path)
		if err != nil {
			return err
		}
		if digest != output.Digest {
			return fmt.Errorf("%w: %s", ErrArtifactDigestMismatch, output.Path)
		}
	}
	return nil
}

func validateSnapshot(root string, metadata ArtifactMetadata) error {
	if err := validateMetadata(metadata); err != nil {
		return fmt.Errorf("%w: %v", ErrArtifactIncomplete, err)
	}
	return validateSourceRoot(filepath.Join(root, "gen"), metadata)
}

func (s *Store) readSelection(path string) (selectionRecord, error) {
	var record selectionRecord
	if err := readJSON(path, &record); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return record, fmt.Errorf("%w: %v", ErrArtifactRecordCorrupt, err)
		}
		return record, err
	}
	if err := validateSelection(record); err != nil {
		return record, fmt.Errorf("%w: %v", ErrArtifactRecordCorrupt, err)
	}
	return record, nil
}

func readMetadata(path string) (ArtifactMetadata, error) {
	var metadata ArtifactMetadata
	if err := readJSON(path, &metadata); err != nil {
		return metadata, err
	}
	if err := validateMetadata(metadata); err != nil {
		return metadata, err
	}
	return metadata, nil
}

func readLease(path string) (leaseRecord, error) {
	var record leaseRecord
	if err := readJSON(path, &record); err != nil {
		return record, err
	}
	if record.SchemaVersion != artifactLeaseSchemaVersion || record.ArtifactID == "" || record.ExpiresAt.IsZero() {
		return record, fmt.Errorf("invalid lease")
	}
	return record, nil
}

func secureJoin(root, rel string) (string, error) {
	root = filepath.Clean(root)
	joined := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	relative, err := filepath.Rel(root, joined)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", fmt.Errorf("%w: %s", ErrArtifactPathEscape, rel)
	}
	return joined, nil
}

func artifactID(metadata ArtifactMetadata) string {
	encoded, _ := json.Marshal(struct {
		Source, Generator string
		Outputs           []ArtifactOutput
	}{metadata.SourceDigest, metadata.GeneratorDigest, metadata.Outputs})
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func metadataDigest(metadata ArtifactMetadata) string {
	encoded, _ := json.Marshal(metadata)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func artifactFileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func syncDirectory(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func copyTree(ctx context.Context, source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported candidate entry %s", rel)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			_ = in.Close()
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			_ = in.Close()
			return err
		}
		_, err = io.Copy(out, in)
		if closeErr := in.Close(); err == nil {
			err = closeErr
		}
		if closeErr := out.Close(); err == nil {
			err = closeErr
		}
		return err
	})
}

func makeReadOnly(root string) error {
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err == nil {
			paths = append(paths, path)
		}
		return err
	}); err != nil {
		return err
	}
	sort.Slice(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		// Keep directory write permission so the owner can garbage-collect an
		// immutable snapshot. Files are read-only; digest validation remains the
		// correctness control even on filesystems that ignore chmod semantics.
		if !info.IsDir() {
			mode &^= 0o222
		}
		if err := os.Chmod(path, mode); err != nil {
			return err
		}
	}
	return nil
}
