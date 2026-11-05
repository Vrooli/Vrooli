package apply

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// fsWorkspace is the production Workspace: it reads and writes files inside
// scenario directories under a single scenarios root (e.g. the repo's
// `scenarios/` directory). Every method confirms the resolved path stays under
// the target scenario's directory, so a crafted scenario name or relative path
// can never escape the tree.
type fsWorkspace struct {
	root string
}

// NewFSWorkspace constructs the production Workspace rooted at scenariosRoot —
// the directory that contains one subdirectory per scenario.
func NewFSWorkspace(scenariosRoot string) Workspace {
	return &fsWorkspace{root: filepath.Clean(scenariosRoot)}
}

var _ Workspace = (*fsWorkspace)(nil)

func (w *fsWorkspace) ScenarioExists(_ context.Context, scenario string) (bool, error) {
	dir, err := w.scenarioDir(scenario)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat scenario dir: %w", err)
	}
	return info.IsDir(), nil
}

func (w *fsWorkspace) ReadFile(_ context.Context, scenario, rel string) ([]byte, error) {
	full, err := w.resolve(scenario, rel)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(full)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read scenario file: %w", err)
	}
	return data, nil
}

func (w *fsWorkspace) WriteFile(_ context.Context, scenario, rel string, data []byte) error {
	full, err := w.resolve(scenario, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return fmt.Errorf("create scenario dir: %w", err)
	}
	if err := writeFileAtomic(full, data); err != nil {
		return fmt.Errorf("write scenario file: %w", err)
	}
	return nil
}

// scenarioDir resolves and validates the directory for a scenario, rejecting a
// name that contains path separators or traversal segments.
func (w *fsWorkspace) scenarioDir(scenario string) (string, error) {
	clean := strings.TrimSpace(scenario)
	if clean == "" {
		return "", ErrInvalidApply{Field: "scenario_name", Reason: "required"}
	}
	if clean != filepath.Base(clean) || clean == "." || clean == ".." {
		return "", ErrInvalidApply{Field: "scenario_name", Reason: "must be a bare scenario name"}
	}
	return filepath.Join(w.root, clean), nil
}

// resolve joins a scenario-relative path onto the scenario dir and confirms the
// result stays within that dir.
func (w *fsWorkspace) resolve(scenario, rel string) (string, error) {
	dir, err := w.scenarioDir(scenario)
	if err != nil {
		return "", err
	}
	full := filepath.Clean(filepath.Join(dir, rel))
	if full != dir && !strings.HasPrefix(full, dir+string(os.PathSeparator)) {
		return "", ErrInvalidApply{Field: "file", Reason: "path escapes scenario directory"}
	}
	return full, nil
}

// writeFileAtomic writes data to a temp file in the destination directory and
// renames it into place, so a crash mid-write never leaves a truncated file.
func writeFileAtomic(dest string, data []byte) error {
	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".apply-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o640); err != nil {
		return err
	}
	return os.Rename(tmpName, dest)
}
