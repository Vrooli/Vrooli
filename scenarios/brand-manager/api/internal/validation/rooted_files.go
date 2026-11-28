package validation

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// scenarioPath opens a path beneath the selected scenario and rejects every
// symlink component. The os.Root also keeps a concurrent symlink replacement
// from redirecting a fixer outside the scenario while it writes or removes.
func scenarioPath(root, rel string) (*os.Root, string, error) {
	path := filepath.Clean(filepath.FromSlash(rel))
	if rel == "" || filepath.IsAbs(path) || path == "." || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return nil, "", fmt.Errorf("unsafe scenario path %q", rel)
	}
	fsRoot, err := os.OpenRoot(root)
	if err != nil {
		return nil, "", fmt.Errorf("open scenario root: %w", err)
	}
	if err := rejectSymlinkComponents(fsRoot, path); err != nil {
		_ = fsRoot.Close()
		return nil, "", err
	}
	return fsRoot, path, nil
}

func rejectSymlinkComponents(root *os.Root, rel string) error {
	current := ""
	components := strings.Split(rel, string(filepath.Separator))
	for i, component := range components {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, err := root.Lstat(current)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("inspect scenario path %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse symlink component %q in scenario path %q", current, rel)
		}
		if i < len(components)-1 && !info.IsDir() {
			return fmt.Errorf("non-directory component %q in scenario path %q", current, rel)
		}
	}
	return nil
}

func writeScenarioFile(root, rel string, content []byte, mode fs.FileMode) error {
	fsRoot, path, err := scenarioPath(root, rel)
	if err != nil {
		return err
	}
	defer fsRoot.Close()
	parent := filepath.Dir(path)
	if parent != "." {
		if err := fsRoot.MkdirAll(parent, 0o750); err != nil {
			return fmt.Errorf("create scenario path %q: %w", parent, err)
		}
	}
	if err := rejectSymlinkComponents(fsRoot, path); err != nil {
		return err
	}
	if err := fsRoot.WriteFile(path, content, mode); err != nil {
		return fmt.Errorf("write scenario path %q: %w", path, err)
	}
	return nil
}

func validateScenarioPath(root, rel string) error {
	fsRoot, _, err := scenarioPath(root, rel)
	if err != nil {
		return err
	}
	return fsRoot.Close()
}

func scenarioFileExists(root, rel string) (bool, error) {
	fsRoot, path, err := scenarioPath(root, rel)
	if err != nil {
		return false, err
	}
	defer fsRoot.Close()
	_, err = fsRoot.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func removeScenarioFile(root, rel string) error {
	fsRoot, path, err := scenarioPath(root, rel)
	if err != nil {
		return err
	}
	defer fsRoot.Close()
	return fsRoot.Remove(path)
}
