package runtimeapp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
	"github.com/vrooli/vrooli/internal/rootauthority"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/stateformat"
)

// ForceReceipt is the durable record of one forced transition. It is written
// before any effect and is never rewritten: a later run starts a new receipt.
type ForceReceipt struct {
	Format        int                    `json:"format"`
	ReceivedAt    time.Time              `json:"received_at"`
	Authority     rootauthority.Decision `json:"authority"`
	Reason        string                 `json:"reason"`
	ServingSHA    string                 `json:"serving_sha256"`
	CandidateSHA  string                 `json:"candidate_sha256"`
	Controls      []string               `json:"controls"`
	SnapshotDir   string                 `json:"snapshot_dir"`
	SnapshotFiles []ForceSnapshotFile    `json:"snapshot_files"`
	SnapshotSHA   string                 `json:"snapshot_sha256"`
	// UnreadableStateFiles are declared state files that exist but cannot be
	// read by this user. The transition proceeds and the receipt names them, so
	// "restored from the snapshot" never silently omits a file the snapshot
	// could not see.
	UnreadableStateFiles []string `json:"unreadable_state_files,omitempty"`
	// StoresWithState and StoresTotal report how much of the compiled store
	// set the snapshot actually covers. A store that declares no state paths
	// contributes no files, so the receipt states the gap instead of implying
	// that "every declared path" means "all state".
	StoresWithState    int `json:"stores_with_state"`
	StoresContractOnly int `json:"stores_contract_only"`
	StoresDiscardable  int `json:"stores_discardable"`
	StoresTotal        int `json:"stores_total"`
}

// uniqueStamp names one attempt's snapshot and receipt. It is second-resolution
// plus random, so two overlapping attempts cannot share a directory.
func uniqueStamp() (string, error) {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", fmt.Errorf("name the forced-transition snapshot: %w", err)
	}
	return time.Now().UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(suffix[:]), nil
}

// ForceSnapshotFile is one captured state file.
type ForceSnapshotFile struct {
	Store string `json:"store"`
	Path  string `json:"path"`
	// Capture is the file's location inside the snapshot directory, relative to
	// the store's own directory. It preserves the runtime-home layout so two
	// matched files with the same base name cannot overwrite each other.
	// Receipts written before this field existed leave it empty and restore
	// from the base name, which is how they were written.
	Capture string `json:"capture,omitempty"`
	Digest  string `json:"sha256"`
	Size    int64  `json:"size"`
	// Mode is the permission bits of the captured file. A restore uses them so
	// it puts back the file it captured rather than a 0600 approximation of it.
	// Receipts written before this field existed leave it zero and restore 0600.
	Mode uint32 `json:"mode,omitempty"`
}

// ForceControls is what a forced transition must be able to show before it may
// replace the serving image: a consistent copy of every declared state file,
// a digest over that copy, and a receipt naming who authorized it.
type ForceControls struct {
	Dir       string
	Receipt   ForceReceipt
	ReceiptAt string
}

// ForceRequest names the authority and the images behind one forced
// transition. Nothing here grants anything; the authority was verified by the
// canonical evaluator before this is called.
type ForceRequest struct {
	Authority    rootauthority.Decision
	Reason       string
	ServingSHA   string
	CandidateSHA string
}

// PrepareForceControls captures the compensating controls. It refuses when the
// snapshot cannot be taken or the receipt cannot be written, which keeps a
// forced transition recoverable by construction: there is no path that changes
// state without a copy of the state as it was.
func PrepareForceControls(ctx context.Context, home string, request ForceRequest) (ForceControls, error) {
	if err := ctx.Err(); err != nil {
		return ForceControls{}, err
	}
	// A grant names an approver and an ID; operator authority names the terminal
	// and has no grant ID. Both must identify who authorized the transition.
	if strings.TrimSpace(request.Authority.Target) == "" || strings.TrimSpace(request.Authority.ApprovedBy) == "" {
		return ForceControls{}, fmt.Errorf("forced transition requires a verified authority decision")
	}
	// The receipt is the durable record of why a transition that could not be
	// rolled back automatically was taken. One without a reason records the act
	// and not the justification, which is the part a later reader needs.
	if strings.TrimSpace(request.Reason) == "" {
		return ForceControls{}, fmt.Errorf("forced transition requires a reason; it is recorded in the receipt")
	}
	manifest, err := NativeStateFormatManifest()
	if err != nil {
		return ForceControls{}, fmt.Errorf("read the state format before a forced transition: %w", err)
	}
	coverage := manifest.Coverage()
	// A store path is relative to the runtime home, not to the OS home the
	// caller resolved. Joining it against $HOME would snapshot nothing and
	// write the receipt outside the runtime home.
	runtimeHome, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return ForceControls{}, fmt.Errorf("resolve the runtime home for a forced transition: %w", err)
	}
	// The transition lock is taken after the controls are prepared, so two
	// attempts can overlap here. A second-resolution stamp would let the second
	// capture into the first's snapshot directory and leave a receipt claiming a
	// mixture of two moments. The random suffix keeps each attempt whole.
	stamp, err := uniqueStamp()
	if err != nil {
		return ForceControls{}, err
	}
	root := filepath.Join(runtimeHome, "root-rebaseline")
	// Every forced transition keeps a full copy of the registry, which is the
	// largest store on the host. Without retirement those copies accumulate
	// until a host that transitions often cannot afford one more. Pruning runs
	// before this attempt captures, and refuses to touch anything recent or
	// anything an unfinished transition could still fall back to.
	pruneRootRebaselineSnapshots(runtimeHome, root)
	snapshotDir := filepath.Join(root, "snapshots", stamp)
	if _, err := config.EnsureOwnedDir(snapshotDir); err != nil {
		return ForceControls{}, fmt.Errorf("prepare the state snapshot directory: %w", err)
	}
	// Nothing has changed on the host yet, so a failure between here and the
	// receipt write must not leave a partial snapshot behind: an abandoned
	// directory would look like a usable recovery copy to the next reader.
	prepared := false
	defer func() {
		if !prepared {
			_ = os.RemoveAll(snapshotDir)
		}
	}()
	files, unreadable, err := captureStoreState(ctx, runtimeHome, snapshotDir, manifest.Stores)
	if err != nil {
		return ForceControls{}, err
	}
	digest, err := digestSnapshot(files)
	if err != nil {
		return ForceControls{}, err
	}
	// A receipt names both images. The candidate is known to the caller; the
	// serving image is the installed root CLI at the moment before effects, so
	// resolve it here rather than trusting a caller to remember.
	servingSHA := request.ServingSHA
	if servingSHA == "" {
		installed := filepath.Join(runtimeHome, "bin", "vrooli")
		if info, statErr := os.Stat(installed); statErr == nil && !info.IsDir() {
			if digest, _, digestErr := digestFile(installed); digestErr == nil {
				servingSHA = digest
			}
		}
	}
	receipt := ForceReceipt{
		Format: 1, ReceivedAt: time.Now().UTC(), Authority: request.Authority, Reason: strings.TrimSpace(request.Reason),
		ServingSHA: servingSHA, CandidateSHA: request.CandidateSHA,
		Controls: []string{
			"state snapshot of every declared store path",
			"prior image retained for rollback by the ordinary transition",
			"restore on a failed post-activation verification",
			"receipt naming the actor, authority and images",
		},
		SnapshotDir: snapshotDir, SnapshotFiles: files, SnapshotSHA: digest, UnreadableStateFiles: unreadable,
		StoresWithState: coverage.Declared, StoresContractOnly: coverage.ContractOnly, StoresDiscardable: coverage.Discardable, StoresTotal: coverage.Total,
	}
	receiptPath := filepath.Join(root, "receipts", stamp+".json")
	if _, err := config.EnsureOwnedDir(filepath.Dir(receiptPath)); err != nil {
		return ForceControls{}, fmt.Errorf("prepare the receipt directory: %w", err)
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return ForceControls{}, err
	}
	if err := writeFileExclusive(receiptPath, append(data, '\n')); err != nil {
		return ForceControls{}, fmt.Errorf("write the forced-transition receipt: %w", err)
	}
	prepared = true
	return ForceControls{Dir: snapshotDir, Receipt: receipt, ReceiptAt: receiptPath}, nil
}

// retainedRebaselineSnapshots is how many forced-transition snapshots a host
// keeps. One is the current safety net; the second covers the transition that
// just completed, which is the one an operator is most likely to inspect.
const retainedRebaselineSnapshots = 2

// rebaselineSnapshotMinimumAge keeps a snapshot that another attempt may still
// be writing. Controls are prepared before the transition lock, so an attempt
// that started moments ago is not yet protected by the lock.
const rebaselineSnapshotMinimumAge = time.Hour

// pruneRootRebaselineSnapshots retires old forced-transition snapshots. It is
// deliberately conservative: an unfinished transition disables pruning, recent
// snapshots survive, and receipts are never deleted because they are the
// durable record of why a transition happened, not a copy that can be re-taken.
func pruneRootRebaselineSnapshots(runtimeHome, root string) {
	progress, err := rebaselineprogress.Read(filepath.Join(runtimeHome, "state", "root-rebaseline", "progress.json"))
	if err != nil || (progress != nil && !progress.Complete) {
		return
	}
	entries, err := os.ReadDir(filepath.Join(root, "snapshots"))
	if err != nil {
		return
	}
	candidates := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			candidates = append(candidates, entry.Name())
		}
	}
	// Stamps begin with a UTC timestamp, so lexical order is age order.
	sort.Sort(sort.Reverse(sort.StringSlice(candidates)))
	cutoff := time.Now().Add(-rebaselineSnapshotMinimumAge)
	kept := 0
	for _, name := range candidates {
		info, statErr := os.Stat(filepath.Join(root, "snapshots", name))
		if statErr != nil {
			continue
		}
		// A recent snapshot is kept unconditionally. Old snapshots are kept
		// newest-first up to the retention count, so a recent copy never uses
		// up the historical slots.
		if info.ModTime().After(cutoff) {
			continue
		}
		if kept < retainedRebaselineSnapshots {
			kept++
			continue
		}
		_ = os.RemoveAll(filepath.Join(root, "snapshots", name))
	}
}

// RestoreForceControls copies the snapshot back over the declared store paths.
// It is the failure path of a forced transition: the image rollback and the
// state rollback must describe the same moment.
func RestoreForceControls(ctx context.Context, home string, controls ForceControls) error {
	if len(controls.Receipt.SnapshotFiles) == 0 {
		return nil
	}
	runtimeHome, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return fmt.Errorf("resolve the runtime home while restoring forced-transition state: %w", err)
	}
	var failures []string
	for _, file := range controls.Receipt.SnapshotFiles {
		if err := ctx.Err(); err != nil {
			return err
		}
		captured := filepath.Join(controls.Dir, file.Store, snapshotRelative(file))
		target := filepath.Join(runtimeHome, filepath.FromSlash(file.Path))
		if err := verifySnapshotFile(captured, file); err != nil {
			// Restoring bytes that no longer match the receipt would replace a
			// working store with a corrupt one while claiming the rollback
			// succeeded. Refuse this file and name it.
			failures = append(failures, fmt.Sprintf("%s: %v", file.Path, err))
			continue
		}
		if err := restoreOne(captured, target, file.Mode); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", file.Path, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("restore state from %s: %s", controls.Dir, strings.Join(failures, "; "))
	}
	return nil
}

// captureStoreState snapshots every declared store file that exists. A store
// that declares no state paths has nothing to recover, so a forced transition
// is only as safe as the descriptors it was built from.
func captureStoreState(ctx context.Context, home, snapshotDir string, stores []stateformat.Store) ([]ForceSnapshotFile, []string, error) {
	var captured []ForceSnapshotFile
	var unreadable []string
	// Two stores may declare the same file (a shared envelope, for example).
	// Capturing it twice would take two different moments of one file and make
	// the receipt describe a state that never existed. Each physical file is
	// captured once and attributed to the first store that declared it.
	seen := map[string]bool{}
	for _, store := range stores {
		// A discardable store is derived; there is nothing worth preserving.
		// Validation already forbids it from declaring paths, so this is a
		// second, independent guard against snapshotting derived bytes.
		if store.Discardable || len(store.StatePaths) == 0 {
			continue
		}
		matches, err := expandStatePaths(home, store.StatePaths)
		if err != nil {
			return nil, nil, fmt.Errorf("expand the state paths for %s: %w", store.Name, err)
		}
		for _, relative := range matches {
			if err := ctx.Err(); err != nil {
				return nil, nil, err
			}
			if seen[relative] {
				continue
			}
			source := filepath.Join(home, filepath.FromSlash(relative))
			info, err := os.Stat(source)
			if os.IsNotExist(err) {
				// A wildcard can match and then lose its file to a concurrent
				// writer; the store simply has nothing to capture there.
				continue
			}
			if err != nil {
				if os.IsPermission(err) {
					// Another identity owns the file (a root-run installer wrote
					// it). The transition proceeds and the receipt names it: a
					// blocked transition helps nobody, but silence would claim a
					// recovery the snapshot cannot deliver.
					unreadable = append(unreadable, relative)
					continue
				}
				return nil, nil, fmt.Errorf("inspect %s for the snapshot: %w", relative, err)
			}
			if info.IsDir() {
				return nil, nil, fmt.Errorf("state path %s is a directory; a store must declare its files", relative)
			}
			dest := filepath.Join(snapshotDir, store.Name, filepath.FromSlash(relative))
			if _, err := config.EnsureOwnedDir(filepath.Dir(dest)); err != nil {
				return nil, nil, fmt.Errorf("prepare snapshot for %s: %w", store.Name, err)
			}
			if path.Ext(relative) == ".db" {
				if err := scenarioruntime.SnapshotDatabase(ctx, source, dest); err != nil {
					return nil, nil, fmt.Errorf("snapshot %s: %w", relative, err)
				}
			} else if err := copyFile(source, dest); err != nil {
				if os.IsPermission(err) {
					unreadable = append(unreadable, relative)
					continue
				}
				return nil, nil, fmt.Errorf("snapshot %s: %w", relative, err)
			}
			digest, size, err := digestFile(dest)
			if err != nil {
				return nil, nil, err
			}
			seen[relative] = true
			captured = append(captured, ForceSnapshotFile{Store: store.Name, Path: relative, Capture: relative, Digest: digest, Size: size, Mode: uint32(info.Mode().Perm())})
		}
	}
	sortSnapshotFiles(captured)
	sort.Strings(unreadable)
	return captured, unreadable, nil
}

// expandStatePaths turns declared paths into concrete files. A path without a
// wildcard is itself; a path with "*" segments is expanded against the runtime
// home. Expansion happens at snapshot time because the member names — one
// directory per scenario, for example — are not known when the binary is
// compiled.
func expandStatePaths(home string, patterns []string) ([]string, error) {
	seen := map[string]bool{}
	expanded := make([]string, 0, len(patterns))
	add := func(relative string) {
		if seen[relative] {
			return
		}
		seen[relative] = true
		expanded = append(expanded, relative)
	}
	for _, pattern := range patterns {
		if !strings.Contains(pattern, "*") {
			add(pattern)
			continue
		}
		matches, err := filepath.Glob(filepath.Join(home, filepath.FromSlash(pattern)))
		if err != nil {
			return nil, fmt.Errorf("%s is not a valid state path: %w", pattern, err)
		}
		for _, match := range matches {
			relative, err := filepath.Rel(home, match)
			if err != nil {
				return nil, err
			}
			add(filepath.ToSlash(relative))
		}
	}
	sort.Strings(expanded)
	return expanded, nil
}

// snapshotRelative is where a captured file lives inside its store's snapshot
// directory. Receipts written before Capture existed stored files flat.
func snapshotRelative(file ForceSnapshotFile) string {
	if file.Capture != "" {
		return filepath.FromSlash(file.Capture)
	}
	return filepath.Base(file.Path)
}

// sortSnapshotFiles orders captures for a deterministic receipt digest. Two
// stores may legitimately declare the same file, so path alone is not a total
// order: their relative order would be unspecified.
func sortSnapshotFiles(files []ForceSnapshotFile) {
	sort.Slice(files, func(i, j int) bool {
		if files[i].Path != files[j].Path {
			return files[i].Path < files[j].Path
		}
		return files[i].Store < files[j].Store
	})
}

func digestSnapshot(files []ForceSnapshotFile) (string, error) {
	canonical := make([]ForceSnapshotFile, len(files))
	copy(canonical, files)
	for i := range canonical {
		canonical[i].Size = 0
	}
	data, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func digestFile(source string) (string, int64, error) {
	file, err := os.Open(source)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

func copyFile(source, dest string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(dest)
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		_ = os.Remove(dest)
		return err
	}
	return out.Close()
}

func verifySnapshotFile(captured string, file ForceSnapshotFile) error {
	if file.Digest == "" {
		// Receipts written before digests were recorded cannot be verified;
		// their capture was already integrity-checked when it was written.
		return nil
	}
	digest, _, err := digestFile(captured)
	if err != nil {
		return fmt.Errorf("read the snapshot copy: %w", err)
	}
	if digest != file.Digest {
		return fmt.Errorf("snapshot copy does not match its recorded digest (recorded %s, found %s)", file.Digest, digest)
	}
	return nil
}

func restoreOne(captured, target string, mode uint32) error {
	// A restored store file may live in a directory the candidate removed, so
	// the restore creates it rather than failing on a missing parent.
	if _, err := config.EnsureOwnedDir(filepath.Dir(target)); err != nil {
		return err
	}
	// Stage the restored bytes first so a failed copy cannot destroy the file
	// it was meant to replace, then replace it in one step.
	staged, err := stageCopy(captured, filepath.Dir(target))
	if err != nil {
		return err
	}
	defer os.Remove(staged)
	if mode != 0 {
		if err := os.Chmod(staged, os.FileMode(mode)); err != nil {
			return err
		}
	}
	if path.Ext(target) == ".db" {
		// A surviving write-ahead log or rollback journal would be replayed
		// over the restored database, so neither may outlive the main file.
		// The journal is the older mechanism but still appears after a crash
		// mid-transaction.
		for _, suffix := range []string{"-wal", "-shm", "-journal"} {
			os.RemoveAll(target + suffix)
		}
	}
	if err := platform.AtomicReplace(staged, target); err != nil {
		return err
	}
	return nil
}

// stageCopy writes source's bytes into a new temporary file in dir and returns
// its path. The caller publishes it with an atomic replace.
func stageCopy(source, dir string) (string, error) {
	staged, err := os.CreateTemp(dir, ".restore-*")
	if err != nil {
		return "", err
	}
	name := staged.Name()
	fail := func(err error) (string, error) {
		staged.Close()
		os.Remove(name)
		return "", err
	}
	in, err := os.Open(source)
	if err != nil {
		return fail(err)
	}
	defer in.Close()
	if _, err := io.Copy(staged, in); err != nil {
		return fail(err)
	}
	if err := staged.Chmod(0o600); err != nil {
		return fail(err)
	}
	if err := staged.Sync(); err != nil {
		return fail(err)
	}
	if err := staged.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

func writeFileExclusive(name string, data []byte) error {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
