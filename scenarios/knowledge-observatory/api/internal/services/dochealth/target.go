package dochealth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DOC: docs/internal/SEAMS.md#dochealth
// Target resolution. A DocHealth run targets either a scenario (all checks) or
// a project-level documentation path (generic checks only). A path that
// resolves inside a scenario is promoted to that scenario so its contract
// checks still apply — matching the operator rule "a path in a scenario
// includes everything".

type docTarget struct {
	root         string // directory or file to scan, or scenario root
	isScenario   bool
	scenarioName string // populated for scenario targets
	label        string // echoed back as the response scenario_name
}

// repoRoot is the parent of the scenarios root (the Vrooli repo root), used to
// anchor relative --path values.
func (s *Service) repoRoot() string {
	return filepath.Dir(s.scenariosRoot)
}

// resolveTarget converts (scenarioName, opts) into a concrete target.
func (s *Service) resolveTarget(scenarioName string, opts DocHealthOptions) (docTarget, error) {
	scope := strings.ToLower(strings.TrimSpace(opts.Scope))
	pathArg := strings.TrimSpace(opts.Path)
	scenarioName = strings.TrimSpace(scenarioName)

	usePath := scope == "path" || (scope == "" && scenarioName == "" && pathArg != "")
	if usePath {
		if pathArg == "" {
			return docTarget{}, fmt.Errorf("%w: scope=path requires --path", ErrScenarioNameInvalid)
		}
		abs := pathArg
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(s.repoRoot(), abs)
		}
		abs = filepath.Clean(abs)
		if _, err := os.Stat(abs); err != nil {
			return docTarget{}, fmt.Errorf("%w: %s", ErrScenarioNotFound, pathArg)
		}
		if name, ok := s.scenarioForPath(abs); ok {
			return docTarget{
				root:         filepath.Join(s.scenariosRoot, name),
				isScenario:   true,
				scenarioName: name,
				label:        name,
			}, nil
		}
		return docTarget{root: abs, isScenario: false, label: s.relLabel(abs)}, nil
	}

	// Scenario mode (default) — preserves the original byte-for-byte behavior.
	root, err := s.scenarioPath(scenarioName)
	if err != nil {
		return docTarget{}, err
	}
	return docTarget{root: root, isScenario: true, scenarioName: scenarioName, label: scenarioName}, nil
}

// scenarioForPath reports the scenario a path belongs to, if it resolves inside
// the scenarios root under a real scenario directory.
func (s *Service) scenarioForPath(abs string) (string, bool) {
	rel, err := filepath.Rel(s.scenariosRoot, abs)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	name := strings.SplitN(rel, "/", 2)[0]
	info, statErr := os.Stat(filepath.Join(s.scenariosRoot, name))
	if statErr != nil || !info.IsDir() {
		return "", false
	}
	return name, true
}

// relLabel renders a repo-relative label for a generic path, falling back to
// the absolute path when it lies outside the repo.
func (s *Service) relLabel(abs string) string {
	if rel, err := filepath.Rel(s.repoRoot(), abs); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return abs
}
