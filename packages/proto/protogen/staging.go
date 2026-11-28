package protogen

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/platform-go"
)

// Staging trees.
//
// Every protogen run builds its output in a private staging tree and publishes
// from there, so the generated tree is never observed half-written. A full tree
// is a copy of the whole generated fleet (about 100 MB), which makes two
// properties load-bearing:
//
//   - Staging lives outside the repository. A tree under packages/ is seen by
//     every repository walker (seam scanners, file watchers, freshness
//     fingerprints) and races them while it is created and removed. Trees are
//     staged beside the runtime artifact store instead, and fall back to the
//     historical in-tree locations only when that store is on another
//     filesystem, because publication renames files out of the stage.
//   - A tree is owned by the process that created it. The owner holds an
//     advisory lock on the tree's owner.lock for as long as the tree exists and
//     removes the tree on every exit path. The lock dies with the process, so a
//     tree whose lock can be taken has no live owner and is swept by the next
//     run, whatever interrupted the previous one (error, signal, kill, crash).

const (
	generationStagePrefix    = ".proto-gen-stage-"
	verificationStagePrefix  = ".proto-gen-verify-"
	descriptorStagePrefix    = ".proto-descriptor-"
	compatibilityStagePrefix = ".proto-compatibility-"

	stageOwnerLockName = "owner.lock"

	// unownedStageGrace protects a tree that has no owner lock yet. A run
	// creates the lock immediately after the directory, so only trees written
	// before owner locks existed stay lockless for long; the grace keeps the
	// sweep from racing the microseconds between those two steps.
	unownedStageGrace = 10 * time.Minute
)

// stagePrefixes names every staging kind. Sweeping keys on the whole set so a
// new staging kind cannot silently escape cleanup.
var stagePrefixes = []string{generationStagePrefix, verificationStagePrefix, descriptorStagePrefix, compatibilityStagePrefix}

func hasStagePrefix(name string) bool {
	for _, prefix := range stagePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// StagingRoot is the runtime directory staging trees are created in: beside
// the artifact store the staged output is published to, outside the repository.
func StagingRoot(artifactRoot string) string {
	return filepath.Join(filepath.Clean(artifactRoot), "staging")
}

// InTreeStageParent is the fallback staging parent for generation,
// verification and descriptor trees when the runtime staging root is on a
// different filesystem from the generated tree. It is also where every tree
// was created before staging moved out of the repository.
func InTreeStageParent(protoRoot string) string {
	return filepath.Dir(filepath.Clean(protoRoot))
}

// InTreeStageParents lists every in-repository directory protogen may stage
// into: generation trees beside packages/proto, compatibility trees inside it.
func InTreeStageParents(protoRoot string) []string {
	return []string{InTreeStageParent(protoRoot), filepath.Clean(protoRoot)}
}

// resolveStageParent prefers runtimeRoot and falls back to fallback when the
// two are on different filesystems: published files are renamed out of the
// stage into publishDir, and a rename cannot cross filesystems.
func resolveStageParent(runtimeRoot, publishDir, fallback string) string {
	if err := os.MkdirAll(runtimeRoot, 0o755); err != nil {
		return fallback
	}
	if same, err := sameFilesystem(runtimeRoot, publishDir); err != nil || !same {
		return fallback
	}
	return runtimeRoot
}

// stage is one owned staging tree.
type stage struct {
	Dir     string
	release func()
}

// createStage creates a staging tree under parent and takes its owner lock.
// Callers must defer Remove immediately.
func createStage(parent, prefix string) (*stage, error) {
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, fmt.Errorf("create staging parent %s: %w", parent, err)
	}
	dir, err := os.MkdirTemp(parent, prefix)
	if err != nil {
		return nil, fmt.Errorf("create staging tree: %w", err)
	}
	lock, err := platform.OpenLockFile(filepath.Join(dir, stageOwnerLockName), 0o600)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("create staging owner lock: %w", err)
	}
	release, err := platform.LockFile(lock, true)
	if err != nil {
		_ = lock.Close()
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("take staging owner lock: %w", err)
	}
	var once sync.Once
	return &stage{Dir: dir, release: func() {
		once.Do(func() {
			release()
			_ = lock.Close()
		})
	}}, nil
}

// Remove releases ownership and deletes the tree. The lock is released first
// because an open file cannot be deleted on every platform; a sweep that wins
// the lock in between only deletes the same garbage.
func (s *stage) Remove() error {
	if s == nil {
		return nil
	}
	s.release()
	return os.RemoveAll(s.Dir)
}

// sweepStages removes every staging tree under parent that no live process
// owns, and reports what it removed. A live tree is never touched.
func sweepStages(parent string, now time.Time) ([]string, error) {
	entries, err := os.ReadDir(parent)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var removed []string
	var failures []error
	for _, entry := range entries {
		if !entry.IsDir() || !hasStagePrefix(entry.Name()) {
			continue
		}
		path := filepath.Join(parent, entry.Name())
		orphaned, err := stageOrphaned(path, now)
		if err != nil {
			failures = append(failures, fmt.Errorf("inspect staging tree %s: %w", path, err))
			continue
		}
		if !orphaned {
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			failures = append(failures, fmt.Errorf("remove staging tree %s: %w", path, err))
			continue
		}
		removed = append(removed, path)
	}
	return removed, errors.Join(failures...)
}

// sweepStagesLogged is the run-start form of sweepStages: stale staging output
// is garbage, never a reason to fail the run that found it.
func sweepStagesLogged(parent string, logger io.Writer) {
	removed, err := sweepStages(parent, time.Now())
	for _, path := range removed {
		logf(logger, "protogen: removed orphaned staging tree %s", path)
	}
	if err != nil {
		logf(logger, "protogen: staging sweep incomplete: %v", err)
	}
}

// stageOrphaned reports whether no live process owns the tree at path.
func stageOrphaned(path string, now time.Time) (bool, error) {
	lockPath := filepath.Join(path, stageOwnerLockName)
	if _, err := os.Lstat(lockPath); errors.Is(err, fs.ErrNotExist) {
		info, err := os.Stat(path)
		if err != nil {
			return false, err
		}
		return now.Sub(info.ModTime()) >= unownedStageGrace, nil
	} else if err != nil {
		return false, err
	}
	lock, err := platform.OpenLockFile(lockPath, 0o600)
	if err != nil {
		return false, err
	}
	defer lock.Close()
	release, err := platform.LockFile(lock, true)
	if errors.Is(err, platform.ErrLockUnavailable) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	release()
	return true, nil
}

// CleanStages removes every unowned staging tree under each parent. It is the
// explicit `protogen clean` form; live trees of a concurrent run survive.
func CleanStages(parents ...string) error {
	var failures []error
	for _, parent := range parents {
		if _, err := sweepStages(parent, time.Now()); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

// OrphanedStages lists the unowned staging trees under each parent without
// removing them, for reviewable cleanup plans.
func OrphanedStages(parents []string, now time.Time) ([]string, error) {
	var out []string
	for _, parent := range parents {
		entries, err := os.ReadDir(parent)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() || !hasStagePrefix(entry.Name()) {
				continue
			}
			path := filepath.Join(parent, entry.Name())
			orphaned, err := stageOrphaned(path, now)
			if err != nil {
				return nil, fmt.Errorf("inspect staging tree %s: %w", path, err)
			}
			if orphaned {
				out = append(out, path)
			}
		}
	}
	return out, nil
}

func compatibilityBackupPath(target string) string { return target + ".previous" }

// IsDefaultStagingOrBackupPath identifies work trees the default repository
// generator and compatibility publisher may create inside the repository: the
// in-tree staging fallback and the compatibility view's transient backup. The
// active gen projection remains a source input. Custom staging locations are
// not guessed here.
func IsDefaultStagingOrBackupPath(repoRoot, candidate string) bool {
	protoRoot := DefaultConfig(repoRoot).ProtoRoot
	if child, ok := firstChild(InTreeStageParent(protoRoot), candidate); ok && hasStagePrefix(child) && !strings.HasPrefix(child, compatibilityStagePrefix) {
		return true
	}
	if child, ok := firstChild(protoRoot, candidate); ok && strings.HasPrefix(child, compatibilityStagePrefix) {
		return true
	}
	backup := compatibilityBackupPath(filepath.Join(protoRoot, "gen"))
	return filepath.Clean(candidate) == backup || isWithinDirectory(backup, candidate)
}

func firstChild(parent, candidate string) (string, bool) {
	rel, err := filepath.Rel(parent, candidate)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return strings.SplitN(rel, string(filepath.Separator), 2)[0], true
}

func isWithinDirectory(parent, candidate string) bool {
	_, ok := firstChild(parent, candidate)
	return ok
}
