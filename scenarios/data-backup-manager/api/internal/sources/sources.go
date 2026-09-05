// Package sources is the strategy/adapter layer that turns a registered
// target's source into a consistent, snapshottable artifact (and applies a
// restored artifact back). It owns one Capturer per source kind and a Registry
// that runs and restores consume to dispatch by kind. It holds no durable
// product data.
//
// Capture/restore mechanics live behind the Capturer seam so the runs and
// restores domains depend on the kind-neutral interface and substitute
// mocks.FakeCapturer in unit tests. The six concrete kinds (filesystem,
// sqlite, postgres, redis, qdrant, object-storage) are implemented in this
// package and exercised by integration tests gated behind KOPIA_INTEGRATION /
// the source resources.
package sources

import (
	"context"
	"fmt"
)

// SourceKind is the canonical domain vocabulary for a source kind. The handler
// layer translates the proto SourceKind enum to/from these string values so
// domain code never imports the generated proto types.
type SourceKind string

const (
	KindFilesystem    SourceKind = "filesystem"
	KindSQLite        SourceKind = "sqlite"
	KindPostgres      SourceKind = "postgres"
	KindRedis         SourceKind = "redis"
	KindQdrant        SourceKind = "qdrant"
	KindObjectStorage SourceKind = "object-storage"
)

// Valid reports whether k is one of the six supported source kinds.
func (k SourceKind) Valid() bool {
	switch k {
	case KindFilesystem, KindSQLite, KindPostgres, KindRedis, KindQdrant, KindObjectStorage:
		return true
	default:
		return false
	}
}

// CaptureSpec describes a single source to capture.
type CaptureSpec struct {
	// Locator is interpreted per source kind (a path/glob, a db file, a
	// database/schema name, a redis key prefix, a qdrant collection, or a
	// bucket/prefix). Never holds secrets.
	Locator string
	// StageDir is a scratch directory the capturer writes its artifact into;
	// the returned Artifact.Path lives under it. The caller owns cleanup.
	StageDir string
}

// Artifact is the result of a capture: a filesystem path handed to
// KopiaEngine.SnapshotCreate.
type Artifact struct {
	// Path is the file or directory kopia should snapshot.
	Path string
	// Bytes is the on-disk size of the captured artifact (best-effort).
	Bytes int64
}

// RestoreSpec describes applying a kopia-restored artifact back to a source.
type RestoreSpec struct {
	// Locator identifies the live source to restore into (same interpretation
	// as CaptureSpec.Locator).
	Locator string
	// ArtifactPath is the filesystem path kopia restored the snapshot into.
	ArtifactPath string
	// Target is the destination path/namespace for the restore (e.g. a scratch
	// directory in verify mode, or a chosen restore location).
	Target string
}

// Capturer turns a source into a snapshottable artifact and applies a restored
// artifact back to a source. One implementation per source kind.
//
// seam: Capturer is the per-source-kind capture/restore boundary. Production
// wires the concrete kind impls (fs.go, sqlite.go, …) via NewRegistry; tests
// wire mocks.FakeCapturer.
type Capturer interface {
	// Kind reports which source kind this capturer handles.
	Kind() SourceKind
	// Capture reads the source described by spec and writes a snapshottable
	// artifact under spec.StageDir, returning its path and size.
	Capture(ctx context.Context, spec CaptureSpec) (Artifact, error)
	// Restore applies a previously-restored artifact back to the source.
	Restore(ctx context.Context, spec RestoreSpec) error
}

// Registry resolves a Capturer by source kind. runs and restores hold a
// Registry and dispatch each target to the matching capturer.
type Registry struct {
	byKind map[SourceKind]Capturer
}

// NewRegistry builds a Registry from a set of capturers, keyed by Kind().
// Duplicate kinds are a programming error and panic at construction.
func NewRegistry(capturers ...Capturer) *Registry {
	byKind := make(map[SourceKind]Capturer, len(capturers))
	for _, c := range capturers {
		if _, dup := byKind[c.Kind()]; dup {
			panic(fmt.Sprintf("sources: duplicate capturer for kind %q", c.Kind()))
		}
		byKind[c.Kind()] = c
	}
	return &Registry{byKind: byKind}
}

// Capturer returns the capturer for kind, or an error if no capturer is
// registered (an unsupported or not-yet-wired source kind).
func (r *Registry) Capturer(kind SourceKind) (Capturer, error) {
	c, ok := r.byKind[kind]
	if !ok {
		return nil, fmt.Errorf("sources: no capturer registered for kind %q", kind)
	}
	return c, nil
}

// Kinds returns the registered source kinds (for diagnostics).
func (r *Registry) Kinds() []SourceKind {
	kinds := make([]SourceKind, 0, len(r.byKind))
	for k := range r.byKind {
		kinds = append(kinds, k)
	}
	return kinds
}
