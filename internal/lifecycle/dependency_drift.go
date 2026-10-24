package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const dependencySpecifierMatchParts = 3

// checkNodeLockfileDrift is deliberately read-only. It runs immediately
// before a JavaScript component install/build and makes a stale lockfile a
// typed lifecycle failure instead of an opaque package-manager error.
func checkNodeLockfileDrift(root string) ([]string, error) {
	manifest, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return nil, err
	}
	var raw struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Optional        map[string]string `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(manifest, &raw); err != nil {
		return nil, fmt.Errorf("decode package.json: %w", err)
	}
	lock, err := os.ReadFile(filepath.Join(root, "pnpm-lock.yaml"))
	if err != nil {
		return nil, nil
	}
	text := string(lock)
	if !strings.Contains(text, "specifiers:") {
		return nil, nil
	}
	declared := map[string]string{}
	for _, group := range []map[string]string{raw.Dependencies, raw.DevDependencies, raw.Optional} {
		for name, spec := range group {
			declared[name] = spec
		}
	}
	found := map[string]string{}
	linePattern := regexp.MustCompile(`^\s{2}([^:#]+):\s*(\S+)\s*$`)
	inSpecifiers := false
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "specifiers:" {
			inSpecifiers = true
			continue
		}
		if inSpecifiers && len(line) > 0 && line[0] != ' ' {
			break
		}
		if inSpecifiers {
			match := linePattern.FindStringSubmatch(line)
			if len(match) == dependencySpecifierMatchParts {
				found[strings.TrimSpace(match[1])] = strings.TrimSpace(match[2])
			}
		}
	}
	var drift []string
	for name, spec := range declared {
		if found[name] != spec {
			drift = append(drift, name)
		}
	}
	for name := range found {
		if _, ok := declared[name]; !ok {
			drift = append(drift, name)
		}
	}
	sort.Strings(drift)
	return drift, nil
}
