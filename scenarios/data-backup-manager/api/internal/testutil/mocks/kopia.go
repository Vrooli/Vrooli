package mocks

import (
	"context"
	"fmt"
	"sync"

	"data-backup-manager/internal/engine"
)

// FakeKopiaEngine satisfies engine.KopiaEngine for tests that must not touch
// kopia or the network. Each method has an overridable func field so a test
// can program exactly the behavior it needs (canned stats, an injected error,
// a checksum-mismatch on verify); when a func field is nil the fake records
// the call and returns a sensible default. Every call is recorded in Calls so
// tests can assert engine interactions after the fact.
//
// The defaults model a minimally-working engine: RepoCreate succeeds and
// remembers the repo; SnapshotCreate returns a deterministic incrementing id;
// RepoStatus reports encryption on. Tests that need failure or specific values
// set the matching func field.
type FakeKopiaEngine struct {
	mu sync.Mutex

	// Programmable behavior. Leave nil to use the default.
	RepoCreateFn      func(ctx context.Context, spec engine.RepoSpec) error
	RepoStatusFn      func(ctx context.Context, repo string) (engine.RepoStatus, error)
	PassphraseRefFn   func(repo string) string
	RepoStatsFn       func(ctx context.Context, repo string) (engine.RepoStats, error)
	RepoDeleteFn      func(ctx context.Context, repo string) error
	SnapshotCreateFn  func(ctx context.Context, repo, path string, meta engine.SnapshotMetadata) (engine.Snapshot, error)
	SnapshotListFn    func(ctx context.Context, repo, path string) ([]engine.Snapshot, error)
	SnapshotRestoreFn func(ctx context.Context, repo, snapshotID, target string) error
	SnapshotVerifyFn  func(ctx context.Context, repo, snapshotID string, verifyPercent int) error
	BrowseSnapshotFn  func(ctx context.Context, repo, snapshotID, path string) ([]engine.SnapshotEntry, error)
	PolicySetFn       func(ctx context.Context, repo, path string, keepLatest int) error

	// Recorded state.
	Calls         []string
	Repos         []engine.RepoSpec
	SnapshotMetas []engine.SnapshotMetadata
	snapshotSeq   int
}

// Compile-time guarantee.
var _ engine.KopiaEngine = (*FakeKopiaEngine)(nil)

func (f *FakeKopiaEngine) record(format string, args ...any) {
	f.Calls = append(f.Calls, fmt.Sprintf(format, args...))
}

func (f *FakeKopiaEngine) RepoCreate(ctx context.Context, spec engine.RepoSpec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("RepoCreate(%s,%s)", spec.Name, spec.Backend)
	if f.RepoCreateFn != nil {
		return f.RepoCreateFn(ctx, spec)
	}
	f.Repos = append(f.Repos, spec)
	return nil
}

func (f *FakeKopiaEngine) RepoStatus(ctx context.Context, repo string) (engine.RepoStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("RepoStatus(%s)", repo)
	if f.RepoStatusFn != nil {
		return f.RepoStatusFn(ctx, repo)
	}
	return engine.RepoStatus{EncryptionAlgorithm: "AES256-GCM-HMAC-SHA256", Connected: true}, nil
}

// PassphraseRef mirrors the production convention by default so tests can
// assert the bundle carries the real reference path; override PassphraseRefFn
// to inject a custom value.
func (f *FakeKopiaEngine) PassphraseRef(repo string) string {
	if f.PassphraseRefFn != nil {
		return f.PassphraseRefFn(repo)
	}
	return "secret/resources/kopia/repo/" + repo + "/passphrase"
}

func (f *FakeKopiaEngine) RepoStats(ctx context.Context, repo string) (engine.RepoStats, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("RepoStats(%s)", repo)
	if f.RepoStatsFn != nil {
		return f.RepoStatsFn(ctx, repo)
	}
	return engine.RepoStats{}, nil
}

func (f *FakeKopiaEngine) RepoDelete(ctx context.Context, repo string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("RepoDelete(%s)", repo)
	if f.RepoDeleteFn != nil {
		return f.RepoDeleteFn(ctx, repo)
	}
	return nil
}

func (f *FakeKopiaEngine) SnapshotCreate(ctx context.Context, repo, path string, meta engine.SnapshotMetadata) (engine.Snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("SnapshotCreate(%s,%s)", repo, path)
	f.SnapshotMetas = append(f.SnapshotMetas, meta)
	if f.SnapshotCreateFn != nil {
		return f.SnapshotCreateFn(ctx, repo, path, meta)
	}
	f.snapshotSeq++
	return engine.Snapshot{ID: fmt.Sprintf("snap-%d", f.snapshotSeq), Path: path}, nil
}

func (f *FakeKopiaEngine) SnapshotList(ctx context.Context, repo, path string) ([]engine.Snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("SnapshotList(%s,%s)", repo, path)
	if f.SnapshotListFn != nil {
		return f.SnapshotListFn(ctx, repo, path)
	}
	return nil, nil
}

func (f *FakeKopiaEngine) SnapshotRestore(ctx context.Context, repo, snapshotID, target string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("SnapshotRestore(%s,%s,%s)", repo, snapshotID, target)
	if f.SnapshotRestoreFn != nil {
		return f.SnapshotRestoreFn(ctx, repo, snapshotID, target)
	}
	return nil
}

func (f *FakeKopiaEngine) SnapshotVerify(ctx context.Context, repo, snapshotID string, verifyPercent int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("SnapshotVerify(%s,%s,%d)", repo, snapshotID, verifyPercent)
	if f.SnapshotVerifyFn != nil {
		return f.SnapshotVerifyFn(ctx, repo, snapshotID, verifyPercent)
	}
	return nil
}

func (f *FakeKopiaEngine) BrowseSnapshot(ctx context.Context, repo, snapshotID, path string) ([]engine.SnapshotEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("BrowseSnapshot(%s,%s,%s)", repo, snapshotID, path)
	if f.BrowseSnapshotFn != nil {
		return f.BrowseSnapshotFn(ctx, repo, snapshotID, path)
	}
	return nil, nil
}

func (f *FakeKopiaEngine) PolicySet(ctx context.Context, repo, path string, keepLatest int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record("PolicySet(%s,%s,%d)", repo, path, keepLatest)
	if f.PolicySetFn != nil {
		return f.PolicySetFn(ctx, repo, path, keepLatest)
	}
	return nil
}
