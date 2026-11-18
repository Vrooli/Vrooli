package runtimeapp

import (
	"context"
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
}

// ForceSnapshotFile is one captured state file.
type ForceSnapshotFile struct {
	Store  string `json:"store"`
	Path   string `json:"path"`
	Digest string `json:"sha256"`
	Size   int64  `json:"size"`
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
	if strings.TrimSpace(request.Authority.Target) == "" || strings.TrimSpace(request.Authority.GrantID) == "" {
		return ForceControls{}, fmt.Errorf("forced transition requires a verified authority decision")
	}
	manifest, err := NativeStateFormatManifest()
	if err != nil {
		return ForceControls{}, fmt.Errorf("read the state format before a forced transition: %w", err)
	}
	// A store path is relative to the runtime home, not to the OS home the
	// caller resolved. Joining it against $HOME would snapshot nothing and
	// write the receipt outside the runtime home.
	runtimeHome, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return ForceControls{}, fmt.Errorf("resolve the runtime home for a forced transition: %w", err)
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	root := filepath.Join(runtimeHome, "root-rebaseline")
	snapshotDir := filepath.Join(root, "snapshots", stamp)
	if _, err := config.EnsureOwnedDir(snapshotDir); err != nil {
		return ForceControls{}, fmt.Errorf("prepare the state snapshot directory: %w", err)
	}
	files, err := captureStoreState(ctx, runtimeHome, snapshotDir, manifest.Stores)
	if err != nil {
		return ForceControls{}, err
	}
	digest, err := digestSnapshot(files)
	if err != nil {
		return ForceControls{}, err
	}
	receipt := ForceReceipt{
		Format: 1, ReceivedAt: time.Now().UTC(), Authority: request.Authority, Reason: strings.TrimSpace(request.Reason),
		ServingSHA: request.ServingSHA, CandidateSHA: request.CandidateSHA,
		Controls: []string{
			"state snapshot of every declared store path",
			"prior image retained for rollback by the ordinary transition",
			"restore on a failed post-activation verification",
			"receipt naming the actor, authority and images",
		},
		SnapshotDir: snapshotDir, SnapshotFiles: files, SnapshotSHA: digest,
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
	return ForceControls{Dir: snapshotDir, Receipt: receipt, ReceiptAt: receiptPath}, nil
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
		captured := filepath.Join(controls.Dir, file.Store, filepath.Base(file.Path))
		target := filepath.Join(runtimeHome, file.Path)
		if err := restoreOne(captured, target); err != nil {
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
func captureStoreState(ctx context.Context, home, snapshotDir string, stores []stateformat.Store) ([]ForceSnapshotFile, error) {
	var captured []ForceSnapshotFile
	for _, store := range stores {
		if len(store.StatePaths) == 0 {
			continue
		}
		for _, relative := range store.StatePaths {
			source := filepath.Join(home, relative)
			info, err := os.Stat(source)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("inspect %s for the snapshot: %w", relative, err)
			}
			if info.IsDir() {
				return nil, fmt.Errorf("state path %s is a directory; a store must declare its files", relative)
			}
			destDir := filepath.Join(snapshotDir, store.Name)
			if _, err := config.EnsureOwnedDir(destDir); err != nil {
				return nil, fmt.Errorf("prepare snapshot for %s: %w", store.Name, err)
			}
			dest := filepath.Join(destDir, filepath.Base(relative))
			if path.Ext(relative) == ".db" {
				if err := scenarioruntime.SnapshotDatabase(ctx, source, dest); err != nil {
					return nil, fmt.Errorf("snapshot %s: %w", relative, err)
				}
			} else if err := copyFile(source, dest); err != nil {
				return nil, fmt.Errorf("snapshot %s: %w", relative, err)
			}
			digest, size, err := digestFile(dest)
			if err != nil {
				return nil, err
			}
			captured = append(captured, ForceSnapshotFile{Store: store.Name, Path: relative, Digest: digest, Size: size})
		}
	}
	sort.Slice(captured, func(i, j int) bool { return captured[i].Path < captured[j].Path })
	return captured, nil
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

func restoreOne(captured, target string) error {
	// Stage the restored bytes first so a failed copy cannot destroy the file
	// it was meant to replace, then replace it in one step.
	staged, err := stageCopy(captured, filepath.Dir(target))
	if err != nil {
		return err
	}
	defer os.Remove(staged)
	if path.Ext(target) == ".db" {
		// A surviving write-ahead log would be replayed over the restored
		// database, so it must not outlive the main file.
		os.RemoveAll(target + "-wal")
		os.RemoveAll(target + "-shm")
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
