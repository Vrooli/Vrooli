package blenderrenderer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
)

const pythonCacheRepairSchemaVersion = 1

// PythonCacheRepairReceipt describes only the narrowly quarantined cache
// directories. It deliberately contains no executable contents or host paths.
type PythonCacheRepairReceipt struct {
	SchemaVersion    int      `json:"schema_version"`
	ProfileID        string   `json:"profile_id"`
	ExpectedTreeSHA  string   `json:"expected_tree_sha256"`
	FilteredTreeSHA  string   `json:"filtered_tree_sha256"`
	CacheDirectories []string `json:"cache_directories,omitempty"`
	QuarantineID     string   `json:"quarantine_id,omitempty"`
	DryRun           bool     `json:"dry_run"`
	Repaired         bool     `json:"repaired"`
	ObservedAt       string   `json:"observed_at"`
	Detail           string   `json:"detail"`
}

// RepairPythonCache removes only __pycache__ directories containing regular
// .pyc files, and only after a filtered hardlink/symlink tree has proven equal
// to the exact pinned archive digest. The caller owns install-ledger checks
// and must hold the shared tool-install lock for the duration of this call.
func RepairPythonCache(ctx context.Context, optDir, quarantineRoot string, artifact Artifact, dryRun bool) (PythonCacheRepairReceipt, error) {
	if ctx == nil {
		return PythonCacheRepairReceipt{}, errors.New("Blender cache repair context is nil")
	}
	if err := validateRepairPath(optDir, true); err != nil {
		return PythonCacheRepairReceipt{}, fmt.Errorf("validate managed Blender root: %w", err)
	}
	if err := validateRepairPath(quarantineRoot, false); err != nil {
		return PythonCacheRepairReceipt{}, fmt.Errorf("validate Blender cache quarantine root: %w", err)
	}
	if artifact.ProfileID != ProfileID {
		return PythonCacheRepairReceipt{}, errors.New("Blender cache repair artifact is not the pinned profile")
	}
	if len(artifact.TreeSHA256) != 64 {
		return PythonCacheRepairReceipt{}, errors.New("Blender cache repair tree digest must be a SHA-256 hex digest")
	}
	if _, err := hex.DecodeString(artifact.TreeSHA256); err != nil {
		return PythonCacheRepairReceipt{}, errors.New("Blender cache repair tree digest must be a SHA-256 hex digest")
	}
	relative, err := filepath.Rel(filepath.Clean(optDir), filepath.Clean(quarantineRoot))
	if err != nil {
		return PythonCacheRepairReceipt{}, fmt.Errorf("compare Blender cache repair roots: %w", err)
	}
	if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return PythonCacheRepairReceipt{}, errors.New("Blender cache quarantine must be outside the managed tool root")
	}
	rootInfo, err := os.Lstat(optDir)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return PythonCacheRepairReceipt{}, errors.New("managed Blender root is missing or not a regular directory")
	}
	tree, caches, err := filteredTreeDigest(ctx, optDir)
	if err != nil {
		return PythonCacheRepairReceipt{}, err
	}
	receipt := PythonCacheRepairReceipt{
		SchemaVersion: pythonCacheRepairSchemaVersion, ProfileID: artifact.ProfileID,
		ExpectedTreeSHA: artifact.TreeSHA256, FilteredTreeSHA: tree,
		CacheDirectories: append([]string(nil), caches...), DryRun: dryRun,
		ObservedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if tree != artifact.TreeSHA256 {
		return receipt, fmt.Errorf("filtered managed Blender tree %s does not match pinned tree %s; refusing cache removal", tree, artifact.TreeSHA256)
	}
	if len(caches) == 0 {
		return receipt, errors.New("no removable Blender Python cache directories were found")
	}
	if dryRun {
		receipt.Detail = "filtered tree matches the pinned archive; listed Python caches are eligible for quarantine"
		return receipt, nil
	}
	if err := os.MkdirAll(quarantineRoot, 0o700); err != nil {
		return receipt, fmt.Errorf("create Blender cache quarantine root: %w", err)
	}
	if err := validateRepairPath(quarantineRoot, true); err != nil {
		return receipt, fmt.Errorf("revalidate Blender cache quarantine root: %w", err)
	}
	quarantineRootInfo, err := os.Lstat(quarantineRoot)
	if err != nil || !quarantineRootInfo.IsDir() || quarantineRootInfo.Mode()&os.ModeSymlink != 0 {
		return receipt, errors.New("Blender cache quarantine root is not a regular directory")
	}
	quarantine, err := os.MkdirTemp(quarantineRoot, "python-cache-")
	if err != nil {
		return receipt, fmt.Errorf("create Blender Python cache quarantine: %w", err)
	}
	receipt.QuarantineID = filepath.Base(quarantine)
	type movedDir struct{ source, target string }
	moved := make([]movedDir, 0, len(caches))
	rollback := func() error {
		var rollbackErr error
		for i := len(moved) - 1; i >= 0; i-- {
			item := moved[i]
			if _, statErr := os.Lstat(item.source); statErr == nil {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("cache path reappeared during rollback: %s", filepath.Base(item.source)))
				continue
			} else if !errors.Is(statErr, os.ErrNotExist) {
				rollbackErr = errors.Join(rollbackErr, statErr)
				continue
			}
			if err := os.MkdirAll(filepath.Dir(item.source), 0o755); err != nil {
				rollbackErr = errors.Join(rollbackErr, err)
				continue
			}
			if err := os.Rename(item.target, item.source); err != nil {
				rollbackErr = errors.Join(rollbackErr, err)
			}
		}
		return rollbackErr
	}
	for _, relative := range caches {
		if err := ctx.Err(); err != nil {
			return receipt, errors.Join(err, rollback())
		}
		source := filepath.Join(optDir, filepath.FromSlash(relative))
		if ok, checkErr := removablePythonCache(source); checkErr != nil || !ok {
			if checkErr == nil {
				checkErr = errors.New("cache directory changed after verification")
			}
			return receipt, errors.Join(checkErr, rollback())
		}
		target := filepath.Join(quarantine, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return receipt, errors.Join(err, rollback())
		}
		if err := os.Rename(source, target); err != nil {
			return receipt, errors.Join(fmt.Errorf("quarantine Blender Python cache: %w", err), rollback())
		}
		moved = append(moved, movedDir{source: source, target: target})
	}
	actualTree, err := binaryfetch.TreeDigestContext(ctx, optDir)
	if err != nil || actualTree != artifact.TreeSHA256 {
		if err == nil {
			err = fmt.Errorf("post-repair tree %s does not match pinned tree %s", actualTree, artifact.TreeSHA256)
		}
		return receipt, errors.Join(err, rollback())
	}
	receipt.Repaired = true
	receipt.Detail = "Python cache directories were quarantined; the remaining managed tree matches the pinned archive"
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		receipt.Repaired = false
		receipt.Detail = "cache quarantine receipt encoding failed; rollback was attempted (see returned errors)"
		return receipt, errors.Join(err, rollback())
	}
	if err := writeReceiptAtomic(filepath.Join(quarantine, "repair-receipt.json"), append(data, '\n')); err != nil {
		receipt.Repaired = false
		receipt.Detail = "durable cache quarantine receipt failed; rollback was attempted (see returned errors)"
		return receipt, errors.Join(err, rollback())
	}
	return receipt, nil
}

func filteredTreeDigest(ctx context.Context, root string) (string, []string, error) {
	parent := filepath.Dir(filepath.Clean(root))
	scratch, err := os.MkdirTemp(parent, ".blender-filtered-tree-")
	if err != nil {
		return "", nil, fmt.Errorf("create filtered Blender verification tree: %w", err)
	}
	defer os.RemoveAll(scratch)
	var caches []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(scratch, relative)
		if entry.IsDir() {
			if filepath.Base(path) == "__pycache__" {
				removable, checkErr := removablePythonCache(path)
				if checkErr != nil {
					return checkErr
				}
				if removable {
					caches = append(caches, filepath.ToSlash(relative))
					return filepath.SkipDir
				}
			}
			return os.MkdirAll(destination, 0o755)
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if filepath.Base(path) == "__pycache__" {
				return errors.New("refusing symlink Blender __pycache__ directory")
			}
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
				return err
			}
			return os.Symlink(link, destination)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("managed Blender tree contains unsupported entry %s", filepath.ToSlash(relative))
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		if err := os.Link(path, destination); err != nil {
			return fmt.Errorf("hardlink Blender verification entry %s: %w", filepath.ToSlash(relative), err)
		}
		return nil
	})
	if err != nil {
		return "", nil, fmt.Errorf("build filtered Blender verification tree: %w", err)
	}
	digest, err := binaryfetch.TreeDigestContext(ctx, scratch)
	return digest, caches, err
}

func removablePythonCache(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, fmt.Errorf("inspect Blender Python cache path: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false, errors.New("refusing non-directory or symlink Blender __pycache__")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, fmt.Errorf("inspect Blender Python cache directory: %w", err)
	}
	if len(entries) == 0 {
		return false, nil
	}
	pycCount := 0
	for _, entry := range entries {
		info, err := os.Lstat(filepath.Join(path, entry.Name()))
		if err != nil {
			return false, err
		}
		if !info.Mode().IsRegular() {
			return false, errors.New("refusing unknown or symlink content in Blender __pycache__")
		}
		if filepath.Ext(entry.Name()) == ".pyc" {
			pycCount++
		}
	}
	if pycCount == 0 {
		return false, nil
	}
	if pycCount != len(entries) {
		return false, errors.New("refusing mixed content in Blender __pycache__")
	}
	return true, nil
}

func writeReceiptAtomic(path string, data []byte) error {
	return writeReceiptAtomicWith(path, data, syncDirectory)
}

func writeReceiptAtomicWith(path string, data []byte, syncDir func(string) error) error {
	temp, err := os.OpenFile(path+".tmp", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	tempName := temp.Name()
	cleanup := func() { _ = temp.Close(); _ = os.Remove(tempName) }
	if _, err := temp.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := temp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempName)
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		_ = os.Remove(tempName)
		return err
	}
	if err := syncDir(filepath.Dir(path)); err != nil {
		_ = os.Remove(path)
		_ = syncDir(filepath.Dir(path))
		return err
	}
	return nil
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func validateRepairPath(path string, requireDirectory bool) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("path must be absolute and clean")
	}
	volume := filepath.VolumeName(path)
	remainder := strings.TrimPrefix(path, volume+string(filepath.Separator))
	current := volume + string(filepath.Separator)
	if volume == "" {
		current = string(filepath.Separator)
	}
	if info, err := os.Lstat(current); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("path root is not a regular directory")
	}
	for _, part := range strings.Split(remainder, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) && !requireDirectory {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path contains a symlink component %s", filepath.Base(current))
		}
		if current != path && !info.IsDir() {
			return fmt.Errorf("path component %s is not a directory", filepath.Base(current))
		}
	}
	if requireDirectory {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("path is not a regular directory")
		}
	}
	return nil
}
