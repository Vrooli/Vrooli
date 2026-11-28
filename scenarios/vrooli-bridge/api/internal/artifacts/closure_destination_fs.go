package artifacts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DirectoryClosureDestination is the local destination adapter used by
// service-boundary integration tests and by a future agent RPC adapter. Each
// revision has a private tree; the active revision pointer is switched
// atomically, and paths outside this adapter's root are never removed.
type DirectoryClosureDestination struct{ Root string }

func (d DirectoryClosureDestination) CheckRevision(_ context.Context, x ClosureShipment) error {
	if strings.TrimSpace(x.NewRevision) == "" || x.NewRevision == x.PriorRevision {
		return errors.New("new closure revision must differ from prior revision")
	}
	base, err := filepath.Abs(d.Root)
	if err != nil {
		return err
	}
	current, err := readRevision(filepath.Join(base, ".bridge-closures", safeSegment(x.TargetID), safeSegment(x.Destination), "current"))
	if err != nil {
		return err
	}
	if current != x.PriorRevision {
		return fmt.Errorf("stale closure shipment: expected prior revision %q, found %q", x.PriorRevision, current)
	}
	return nil
}

func (d DirectoryClosureDestination) Stage(_ context.Context, x ClosureShipment, files []ClosureFile) (string, error) {
	root, err := filepath.Abs(d.Root)
	if err != nil {
		return "", err
	}
	parent := filepath.Join(root, ".bridge-closures", safeSegment(x.TargetID), safeSegment(x.Destination), "revisions")
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return "", err
	}
	base := filepath.Join(parent, safeSegment(x.NewRevision)+".staging")
	if _, err := os.Stat(base); err == nil {
		return filepath.Rel(root, base)
	}
	tmp := base + ".tmp." + safeSegment(x.ID)
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o750); err != nil {
		return "", err
	}
	for _, f := range files {
		name := filepath.Join(tmp, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(name), 0o750); err != nil {
			_ = os.RemoveAll(tmp)
			return "", err
		}
		mode := os.FileMode(normalizedMode(f.Mode))
		if err := os.WriteFile(name, f.Data, mode); err != nil {
			_ = os.RemoveAll(tmp)
			return "", fmt.Errorf("stage %q: %w", f.Path, err)
		}
	}
	if err := os.Rename(tmp, base); err != nil {
		_ = os.RemoveAll(tmp)
		return "", err
	}
	return filepath.Rel(root, base)
}

func (d DirectoryClosureDestination) Inspect(_ context.Context, stageRef string) ([]StagedFile, error) {
	base, err := d.resolve(stageRef)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(base); os.IsNotExist(err) {
		base = strings.TrimSuffix(base, ".staging")
	}
	var files []StagedFile
	err = filepath.Walk(base, func(name string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(base, name)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		files = append(files, StagedFile{Path: filepath.ToSlash(rel), SHA256: hex.EncodeToString(sum[:]), Mode: uint32(info.Mode().Perm()), Size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func (d DirectoryClosureDestination) Activate(_ context.Context, x ClosureShipment, stageRef string) error {
	stage, err := d.resolve(stageRef)
	if err != nil {
		return err
	}
	base, err := filepath.Abs(d.Root)
	if err != nil {
		return err
	}
	stateDir := filepath.Join(base, ".bridge-closures", safeSegment(x.TargetID), safeSegment(x.Destination))
	if err := os.MkdirAll(stateDir, 0o750); err != nil {
		return err
	}
	currentPath := filepath.Join(stateDir, "current")
	current, err := readRevision(currentPath)
	if err != nil {
		return err
	}
	if current == x.NewRevision {
		return nil
	}
	if current != x.PriorRevision && current != x.NewRevision {
		return fmt.Errorf("stale closure shipment: expected prior revision %q, found %q", x.PriorRevision, current)
	}
	final := filepath.Join(stateDir, "revisions", safeSegment(x.NewRevision))
	if _, err := os.Stat(final); os.IsNotExist(err) {
		if err := os.Rename(stage, final); err != nil {
			return fmt.Errorf("promote verified staged closure: %w", err)
		}
	}
	return writeRevision(currentPath, x.NewRevision)
}

func (d DirectoryClosureDestination) Rollback(_ context.Context, x ClosureShipment, stageRef string) error {
	if _, err := d.resolve(stageRef); err != nil {
		return err
	}
	base, err := filepath.Abs(d.Root)
	if err != nil {
		return err
	}
	currentPath := filepath.Join(base, ".bridge-closures", safeSegment(x.TargetID), safeSegment(x.Destination), "current")
	current, err := readRevision(currentPath)
	if err != nil {
		return err
	}
	if current == x.PriorRevision {
		return nil
	}
	if current != x.NewRevision {
		return errors.New("cannot roll back closure shipment after a newer revision activated")
	}
	return writeRevision(currentPath, x.PriorRevision)
}

func (d DirectoryClosureDestination) Discard(_ context.Context, stageRef string) error {
	stage, err := d.resolve(stageRef)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(stage, ".staging") {
		return errors.New("refusing to discard an activated closure revision")
	}
	return os.RemoveAll(stage)
}

func (d DirectoryClosureDestination) resolve(ref string) (string, error) {
	root, err := filepath.Abs(d.Root)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(ref) {
		return "", errors.New("stage reference must be relative")
	}
	clean := filepath.Clean(ref)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("stage reference escapes destination root")
	}
	joined := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, joined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("stage reference escapes destination root")
	}
	return joined, nil
}
func safeSegment(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	if s == "" || s == "." || s == ".." {
		return "_"
	}
	return s
}
func readRevision(path string) (string, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	return strings.TrimSpace(string(b)), err
}
func writeRevision(path, value string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(value+"\n"), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
