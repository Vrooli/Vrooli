package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/fsx"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/tuning"
)

const defaultVrooliDirPerm = tuning.PermDir

// WriteOwnedJSON encodes one managed JSON document and commits it through the
// invoking-user-owned atomic write path.
func WriteOwnedJSON(path string, value any, perm os.FileMode) error {
	data, err := fsx.MarshalJSON(value)
	if err != nil {
		return err
	}
	return WriteOwnedFileAtomic(path, data, perm)
}

// RepairIdentity returns the only identity that a managed runtime-home repair
// may target: the sudo invoking user when present, otherwise the current
// process identity.
func RepairIdentity() (uint32, uint32) {
	if uid, gid, ok := invokingRepairIdentity(); ok {
		return uid, gid
	}
	return currentRepairIdentity()
}

// EnsureOwnedDir creates dir (and any missing ancestors) and — when the process
// is root via sudo — chowns exactly the components this call created back to the
// invoking user. Components that already existed are left untouched, so a
// sudo'd vrooli never leaves root-owned directories in the operator's home and
// never disturbs pre-existing ownership. Returns the directory path.
func EnsureOwnedDir(dir string) (string, error) {
	return EnsureOwnedDirMode(dir, defaultVrooliDirPerm)
}

// EnsureOwnedDirMode gives a sensitive owner an explicit creation mode while
// preserving the canonical invoking-user ownership of newly created ancestors.
// Existing directories are not chmodded or reassigned by this generic helper.
func EnsureOwnedDirMode(dir string, mode os.FileMode) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("config: empty directory path")
	}
	created := missingAncestors(dir)
	if err := os.MkdirAll(dir, mode); err != nil {
		return "", err
	}
	if err := chownCreatedToInvokingUser(created); err != nil {
		return "", err
	}
	return dir, nil
}

// WriteOwnedFile ensures the parent directory exists (owned), writes the file,
// then chowns the file to the invoking user under sudo.
func WriteOwnedFile(path string, data []byte, perm os.FileMode) error {
	return WriteOwnedFileAtomic(path, data, perm)
}

// WriteOwnedFileAtomic replaces a managed file through a same-directory
// temporary file. The temporary file is explicitly chmod/chowned before the
// rename, and the final path is rechecked after replacement. This keeps setup
// sudo from leaving either a root-owned temporary or a partially-written file.
func WriteOwnedFileAtomic(path string, data []byte, perm os.FileMode) error {
	return writeOwnedFileAtomicStream(path, perm, func(dst io.Writer) error {
		_, err := io.Copy(dst, bytes.NewReader(data))
		return err
	})
}

func writeOwnedFileAtomicStream(path string, perm os.FileMode, write func(io.Writer) error) error {
	if path == "" {
		return fmt.Errorf("config: empty file path")
	}
	if _, err := EnsureOwnedDir(filepath.Dir(path)); err != nil {
		return err
	}
	release, err := acquireOwnedWriteLock(path)
	if err != nil {
		return err
	}
	defer release()
	tmp, err := os.CreateTemp(filepath.Dir(path), ".vrooli-owned-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := write(tmp); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := chownPathToInvokingUser(tmpPath); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	expected, err := tmp.Stat()
	if err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	actual, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !os.SameFile(expected, actual) {
		return fmt.Errorf("config: published owned file identity changed")
	}
	return nil
}

// acquireOwnedWriteLock serializes publishers targeting the same file across
// processes. Atomic rename alone is insufficient: a concurrent publisher can
// replace the just-published inode before its writer verifies identity. The
// lock lives in the runtime home's owned-write lock directory, so publishing
// does not add artifacts to the target directory. Its key uses the resolved
// parent directory so aliases to the same target share one lock namespace.
func acquireOwnedWriteLock(path string) (func(), error) {
	absolute, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("config: resolve owned file path: %w", err)
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return nil, fmt.Errorf("config: resolve owned file parent: %w", err)
	}
	lockDir, err := VrooliPath("state", "locks", "owned-writes")
	if err != nil {
		return nil, fmt.Errorf("config: resolve owned write lock directory: %w", err)
	}
	if _, err := EnsureOwnedDir(lockDir); err != nil {
		return nil, err
	}
	key := sha256.Sum256([]byte(filepath.Join(parent, filepath.Base(absolute))))
	lockPath := filepath.Join(lockDir, hex.EncodeToString(key[:])+".lock")
	release, err := platform.AcquireFileLock(lockPath)
	if err != nil {
		return nil, fmt.Errorf("config: acquire owned file lock: %w", err)
	}
	if err := ChownToInvokingUser(lockPath); err != nil {
		release()
		return nil, fmt.Errorf("config: assign owned file lock: %w", err)
	}
	return release, nil
}

// WriteOwnedFileDurable commits an owned file and syncs the rename and its
// complete ancestor chain. Lifecycle intent must use this stronger
// boundary before an irreversible effect. An error may follow a visible rename:
// callers must reconcile the same request rather than assume nothing committed.
// A filesystem/platform that cannot sync directories returns an error; it does
// not silently downgrade a durable intent to an atomic-only write.
func WriteOwnedFileDurable(path string, data []byte, perm os.FileMode) error {
	return writeOwnedFileDurableWith(path, data, perm, WriteOwnedFileAtomic, syncOwnedDirectory)
}

// WriteOwnedFileDurableStream publishes a streamed payload through the same
// owned atomic/durable boundary. The writer may validate a checksum and return
// an error before publication. It must not close or retain the supplied writer.
// This avoids buffering an entire executable or archive in memory.
func WriteOwnedFileDurableStream(path string, perm os.FileMode, write func(io.Writer) error) error {
	if write == nil {
		return fmt.Errorf("config: stream writer is required")
	}
	return writeOwnedFileDurableWith(path, nil, perm, func(path string, _ []byte, perm os.FileMode) error {
		return writeOwnedFileAtomicStream(path, perm, write)
	}, syncOwnedDirectory)
}

func writeOwnedFileDurableWith(path string, data []byte, perm os.FileMode, write func(string, []byte, os.FileMode) error, syncDir func(string) error) error {
	if path == "" {
		return fmt.Errorf("config: empty file path")
	}
	parent := filepath.Dir(path)
	if err := write(path, data, perm); err != nil {
		return err
	}
	// Visible ancestors may be from an unconfirmed prior write. Resolve both
	// directory aliases and their physical ancestors; a symlink's parent is
	// not the directory that persists its target's own entry.
	directories, err := ownedDurableDirectories(parent)
	if err != nil {
		return fmt.Errorf("config: commit visible but directory durability unconfirmed: %w", err)
	}
	for _, dir := range directories {
		if err := syncDir(dir); err != nil {
			return fmt.Errorf("config: commit visible but directory durability unconfirmed: %w", err)
		}
	}
	return nil
}

func ownedDurableDirectories(parent string) ([]string, error) {
	logical, err := filepath.Abs(parent)
	if err != nil {
		return nil, err
	}
	depths := map[string]int{}
	for {
		physical, err := filepath.EvalSymlinks(logical)
		if err != nil {
			return nil, err
		}
		var chain []string
		for dir := physical; ; dir = filepath.Dir(dir) {
			chain = append(chain, dir)
			if filepath.Dir(dir) == dir {
				break
			}
		}
		for i, dir := range chain {
			depths[dir] = len(chain) - i
		}
		if filepath.Dir(logical) == logical {
			break
		}
		logical = filepath.Dir(logical)
	}
	directories := make([]string, 0, len(depths))
	for dir := range depths {
		directories = append(directories, dir)
	}
	sort.Slice(directories, func(i, j int) bool {
		if depths[directories[i]] != depths[directories[j]] {
			return depths[directories[i]] > depths[directories[j]]
		}
		return directories[i] < directories[j]
	})
	return directories, nil
}

func syncOwnedDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// InstallExecutableAtomic copies a prepared executable into its live path
// through the canonical owned-write seam, then syncs the destination
// directory so the rename is durable. Callers own any required process lock;
// keeping this primitive lock-free lets an already-serialized rebuild reuse
// the exact same commit path as a normal CLI install.
func InstallExecutableAtomic(source, destination string) error {
	if source == "" {
		return fmt.Errorf("config: empty executable source path")
	}
	if destination == "" {
		return fmt.Errorf("config: empty executable destination path")
	}
	if err := os.Chmod(source, tuning.PermExecutable); err != nil {
		return err
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := WriteOwnedFileAtomic(destination, data, tuning.PermExecutable); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(destination))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// OpenOwnedFile opens a managed runtime file with an explicit mode and repairs
// ownership when setup is running for an invoking user. It is the append/lock
// counterpart to WriteOwnedFileAtomic for lifecycle records and logs.
func OpenOwnedFile(path string, flags int, perm os.FileMode) (*os.File, error) {
	if path == "" {
		return nil, fmt.Errorf("config: empty file path")
	}
	if _, err := EnsureOwnedDir(filepath.Dir(path)); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, flags, perm)
	if err != nil {
		return nil, err
	}
	if err := chownPathToInvokingUser(path); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

// EnsureVrooliDir resolves a runtime-home entry (repocontract.HomeKey*, plus any
// consumer sub-components) and ensures it exists with correct ownership.
func EnsureVrooliDir(key string, sub ...string) (string, error) {
	dir, err := VrooliPath(key, sub...)
	if err != nil {
		return "", err
	}
	return EnsureOwnedDir(dir)
}

// WriteVrooliFile resolves a runtime-home entry path (repocontract.HomeKey*,
// plus any consumer sub-components) and writes it with correct ownership.
func WriteVrooliFile(data []byte, perm os.FileMode, key string, sub ...string) error {
	path, err := VrooliPath(key, sub...)
	if err != nil {
		return err
	}
	return WriteOwnedFile(path, data, perm)
}

// ReconcileVrooliOwnership heals pre-existing strays: when the process is root
// via sudo, it walks the resolved runtime home and reclaims every entry still
// owned by root to the invoking user. It only ever touches root-owned entries
// (never reassigns a non-root owner, so multi-user hosts are safe), never
// follows symlinks, and never escapes the home root. Returns the number of
// entries re-owned. A no-op (0, nil) when not running root-via-sudo or when the
// home does not exist. This is what cleans up the root-owned files a sudo'd
// vrooli left behind before the owned-write seam existed.
func ReconcileVrooliOwnership() (int, error) {
	uid, gid, ok := hostreqkit.InvokingUserIDs()
	if !ok {
		return 0, nil
	}
	home, err := VrooliHome()
	if err != nil {
		return 0, err
	}
	if _, statErr := os.Lstat(home); statErr != nil {
		if os.IsNotExist(statErr) {
			return 0, nil
		}
		return 0, statErr
	}
	return reconcileHomeOwnership(home, uid, gid)
}

// ChownToInvokingUser chowns a single existing path to the invoking (sudo) user
// when it lies within the resolved runtime home. It is a no-op when the process
// is not root-via-sudo. Use this for home writes that don't flow through
// WriteOwnedFile — e.g. atomic temp+rename or O_EXCL lock files — so a sudo'd
// vrooli never leaves them root-owned.
func ChownToInvokingUser(path string) error {
	return chownPathToInvokingUser(path)
}

// ChownFileToInvokingUser repairs an already-admitted runtime-home inode.
// It never reopens or follows the file's pathname; the caller owns admission.
func ChownFileToInvokingUser(file *os.File) error {
	if file == nil {
		return fmt.Errorf("config: nil owned file")
	}
	return chownFileToInvokingUser(file)
}

// missingAncestors returns the path and each ancestor that does not yet exist,
// deepest-first, stopping at the first component that already exists. This is
// exactly the set a subsequent MkdirAll will create, and therefore the set we
// must chown. Pure (no syscalls beyond Lstat, no chown) so the
// "which-ancestors-were-created" logic is unit-testable without root.
func missingAncestors(dir string) []string {
	dir = filepath.Clean(dir)
	var missing []string
	for p := dir; ; {
		if _, err := os.Lstat(p); err == nil {
			break
		}
		missing = append(missing, p)
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
		p = parent
	}
	return missing
}

// CommitOwnedPathAtomic moves a same-directory staging path over its final
// name, repairing ownership on both sides.
//
// It is the link-shaped counterpart to WriteOwnedFileAtomic. That function
// commits managed *content*, and cannot express committing a symlink or a hard
// link, because it only knows how to write bytes. Both belong here for the same
// reason: replacing a managed path is where a sudo'd vrooli most easily leaves
// a root-owned artifact in the operator's home, and where a non-atomic
// implementation most easily leaves nothing at all.
//
// Ownership is repaired before the rename so the path is never briefly visible
// under the wrong owner, and again afterwards because the destination inode is
// what the operator's own later commands must be able to replace. Lchown is
// used throughout, so a symlink's own ownership is fixed rather than its
// target's.
//
// staging must already exist and sit in the same directory as path; a rename
// across filesystems is not atomic and is not what callers want here.
func CommitOwnedPathAtomic(staging, path string) error {
	if filepath.Dir(filepath.Clean(staging)) != filepath.Dir(filepath.Clean(path)) {
		return fmt.Errorf("config: staging path %s must sit beside %s", staging, path)
	}
	if _, err := os.Lstat(staging); err != nil {
		return fmt.Errorf("config: staging path unusable: %w", err)
	}
	if err := chownPathToInvokingUser(staging); err != nil {
		return err
	}
	if err := os.Rename(staging, path); err != nil {
		return err
	}
	return chownPathToInvokingUser(path)
}
