package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// fsScanner is the production Scanner: it reads files inside scenario directories
// under a single scenarios root (e.g. the repo's `scenarios/` directory). Every
// method confirms the resolved path stays under the target scenario's directory,
// so a crafted scenario name or relative path can never escape the tree. The
// scanner is read-only — discovery never writes through it.
type fsScanner struct {
	root string
}

// NewFSScanner constructs the production Scanner rooted at scenariosRoot — the
// directory that contains one subdirectory per scenario.
func NewFSScanner(scenariosRoot string) Scanner {
	return &fsScanner{root: filepath.Clean(scenariosRoot)}
}

var _ Scanner = (*fsScanner)(nil)

func (s *fsScanner) ScenarioExists(_ context.Context, scenario string) (bool, error) {
	dir, err := s.scenarioDir(scenario)
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

func (s *fsScanner) ReadFile(_ context.Context, scenario, rel string) ([]byte, error) {
	full, err := s.resolve(scenario, rel)
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

func (s *fsScanner) ListDir(_ context.Context, scenario, rel string) ([]string, error) {
	full, err := s.resolve(scenario, rel)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(full)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read scenario dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

// scenarioDir resolves and validates the directory for a scenario, rejecting a
// name that contains path separators or traversal segments.
func (s *fsScanner) scenarioDir(scenario string) (string, error) {
	clean := strings.TrimSpace(scenario)
	if clean == "" {
		return "", ErrInvalidDiscovery{Field: "scenario_name", Reason: "required"}
	}
	if clean != filepath.Base(clean) || clean == "." || clean == ".." {
		return "", ErrInvalidDiscovery{Field: "scenario_name", Reason: "must be a bare scenario name"}
	}
	return filepath.Join(s.root, clean), nil
}

// resolve joins a scenario-relative path onto the scenario dir and confirms the
// result stays within that dir.
func (s *fsScanner) resolve(scenario, rel string) (string, error) {
	dir, err := s.scenarioDir(scenario)
	if err != nil {
		return "", err
	}
	full := filepath.Clean(filepath.Join(dir, rel))
	if full != dir && !strings.HasPrefix(full, dir+string(os.PathSeparator)) {
		return "", ErrInvalidDiscovery{Field: "path", Reason: "path escapes scenario directory"}
	}
	return full, nil
}
