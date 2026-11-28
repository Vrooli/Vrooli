package fetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Acquisition staging.
//
// Every download lands in a staging directory and is published by rename only
// after verification. Staging lives in its own top-level directory of the data
// root (<root>/staging/<models|adapters>/), a sibling of the weights it
// publishes into so the rename stays on one filesystem. It is deliberately not
// inside models/ or adapters/: those are declared durable, and a deny above a
// path governs everything beneath it, so staging there could never be declared
// reclaimable. A staging directory is named
// from the destination and the fetch identity (repository revision or asset
// URLs and checksums), so a later acquisition of the same capability resumes a
// partial snapshot that a crash or restart left behind instead of starting
// over: huggingface_hub continues its `.incomplete` files in a local_dir, and
// the verification that gates publication hashes the finished tree either way.
// A different revision or source hashes to a different directory, so leftovers
// are never mixed across identities.
//
// Ownership is in-process: a directory is owned while an Acquire (or a sweep)
// holds it in the registry below. Anything else with the staging prefix is a
// leftover; internal/modelstorage decides when a leftover is abandoned.

// StagingDirName is the data-root directory holding all acquisition staging.
const StagingDirName = "staging"

const (
	// StagePrefix starts every acquisition staging directory name.
	StagePrefix = ".acquiring-"
	// PreviousSuffix marks the replaced installation set aside while a verified
	// replacement is published.
	PreviousSuffix = ".previous"
)

// ErrStageBusy is returned when another acquisition already owns the staging
// directory for the same destination and source.
var ErrStageBusy = errors.New("fetch: staging directory is owned by another acquisition")

var stageRegistry = struct {
	mu   sync.Mutex
	held map[string]struct{}
}{held: map[string]struct{}{}}

// HoldStage marks path as owned. It reports false when it is already owned.
func HoldStage(path string) bool {
	path = filepath.Clean(path)
	stageRegistry.mu.Lock()
	defer stageRegistry.mu.Unlock()
	if _, taken := stageRegistry.held[path]; taken {
		return false
	}
	stageRegistry.held[path] = struct{}{}
	return true
}

// ReleaseStage ends ownership taken by HoldStage.
func ReleaseStage(path string) {
	stageRegistry.mu.Lock()
	delete(stageRegistry.held, filepath.Clean(path))
	stageRegistry.mu.Unlock()
}

// StageHeld reports whether an acquisition or sweep currently owns path.
func StageHeld(path string) bool {
	stageRegistry.mu.Lock()
	defer stageRegistry.mu.Unlock()
	_, held := stageRegistry.held[filepath.Clean(path)]
	return held
}

// stageIdentity is every input that decides which bytes a fetch produces.
func stageIdentity(source Source) string {
	identity := struct {
		DownloadURL string   `json:"download_url,omitempty"`
		Checksum    string   `json:"checksum,omitempty"`
		Assets      []Asset  `json:"assets,omitempty"`
		Repo        RepoSpec `json:"repo"`
	}{source.DownloadURL, source.Checksum.Value, source.Assets, source.Repo}
	raw, _ := json.Marshal(identity)
	return string(raw)
}

// StagingRoot is the directory acquisitions into destination's collection
// (models, adapters) stage in.
func StagingRoot(root, destination string) string {
	return filepath.Join(root, StagingDirName, filepath.Base(filepath.Dir(destination)))
}

// StageDir is the deterministic staging directory for destination and source.
func StageDir(root, destination string, source Source) string {
	sum := sha256.Sum256([]byte(stageIdentity(source)))
	name := StagePrefix + filepath.Base(destination) + "-" + hex.EncodeToString(sum[:6])
	return filepath.Join(StagingRoot(root, destination), name)
}

var stageNamePattern = regexp.MustCompile(`^` + regexp.QuoteMeta(StagePrefix) + `(.+)-[0-9a-f]{12}$`)

// StageDestination maps a staging directory (or its ".previous" form) under
// <root>/staging/<collection>/ to the destination it publishes to. It is false
// for anything else, including the random names older releases created beside
// the weights, whose destination is unknown.
func StageDestination(path string) (string, bool) {
	match := stageNamePattern.FindStringSubmatch(strings.TrimSuffix(filepath.Base(path), PreviousSuffix))
	collection := filepath.Dir(path)
	stagingDir := filepath.Dir(collection)
	if match == nil || filepath.Base(stagingDir) != StagingDirName {
		return "", false
	}
	return filepath.Join(filepath.Dir(stagingDir), filepath.Base(collection), match[1]), true
}

// IsStageName reports whether name is an acquisition staging entry of any
// release, including a ".previous" set-aside.
func IsStageName(name string) bool { return strings.HasPrefix(name, StagePrefix) }

// IsPreviousName reports whether name is a replaced-installation set-aside.
func IsPreviousName(name string) bool { return strings.HasSuffix(name, PreviousSuffix) }

// claimStage takes ownership of the staging directory for destination,
// waiting for a concurrent sweep to finish with it, and returns it ready for
// writing together with whether leftovers from an earlier attempt were found.
func claimStage(ctx context.Context, root, destination string, source Source) (stage string, resumed bool, err error) {
	stage = StageDir(root, destination, source)
	for !HoldStage(stage) {
		select {
		case <-ctx.Done():
			return "", false, fmt.Errorf("%w: %s: %w", ErrStageBusy, stage, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	defer func() {
		if err != nil {
			ReleaseStage(stage)
		}
	}()
	if err := recoverPrevious(destination, stage+PreviousSuffix); err != nil {
		return "", false, err
	}
	info, statErr := os.Lstat(stage)
	switch {
	case statErr == nil && info.IsDir():
		resumed = true
	case statErr == nil:
		if err := os.RemoveAll(stage); err != nil {
			return "", false, err
		}
	case !os.IsNotExist(statErr):
		return "", false, statErr
	}
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return "", false, err
	}
	// Resuming must not leave the directory looking untouched to a sweep.
	now := time.Now()
	_ = os.Chtimes(stage, now, now)
	return stage, resumed, nil
}

// recoverPrevious settles a set-aside left by a crash between publication
// renames: it restores the replaced installation when the destination is gone
// and otherwise discards it, because a verified replacement already landed.
func recoverPrevious(destination, previous string) error {
	if _, err := os.Lstat(previous); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if _, err := os.Lstat(destination); os.IsNotExist(err) {
		return os.Rename(previous, destination)
	} else if err != nil {
		return err
	}
	return os.RemoveAll(previous)
}
