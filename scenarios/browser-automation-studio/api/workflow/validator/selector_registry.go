package validator

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func loadSelectorManifest(selectorFile string) (map[string]string, string, error) {
	if manifest, source, err := loadSelectorManifestViaNode(selectorFile); err == nil && len(manifest) > 0 {
		return manifest, source, nil
	}
	return loadSelectorManifestFromFile(selectorFile)
}

func loadSelectorManifestFromFile(selectorFile string) (map[string]string, string, error) {
	resolved := strings.TrimSpace(selectorFile)
	if resolved == "" {
		resolved = detectSelectorFile()
	}
	if resolved == "" {
		return nil, "", errors.New("selector registry not found")
	}
	content, err := os.ReadFile(resolved)
	if err != nil {
		return nil, resolved, err
	}
	manifest, parseErr := parseSelectorManifest(content)
	return manifest, resolved, parseErr
}

func parseSelectorManifest(content []byte) (map[string]string, error) {
	manifest := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(content))
	inside := false
	entryPattern := regexp.MustCompile(`^([a-zA-Z0-9_]+)\s*:\s*['"]([^'"]+)['"]\s*,?\s*$`)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !inside {
			if strings.HasPrefix(line, "export const testIds") {
				inside = true
			}
			continue
		}
		if strings.HasPrefix(line, "}") {
			break
		}
		matches := entryPattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			manifest[matches[2]] = matches[1]
		}
	}
	return manifest, scanner.Err()
}

func detectSelectorFile() string {
	if env := strings.TrimSpace(os.Getenv("BAS_SELECTOR_FILE")); env != "" {
		if info, err := os.Stat(env); err == nil && !info.IsDir() {
			return env
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		if candidate := searchSelectorFileFrom(cwd); candidate != "" {
			return candidate
		}
	}
	if exe, err := os.Executable(); err == nil {
		if candidate := searchSelectorFileFrom(filepath.Dir(exe)); candidate != "" {
			return candidate
		}
	}
	return ""
}

func searchSelectorFileFrom(start string) string {
	if start == "" {
		return ""
	}
	visited := map[string]struct{}{}
	for dir := start; dir != ""; dir = filepath.Dir(dir) {
		if _, ok := visited[dir]; ok {
			break
		}
		visited[dir] = struct{}{}
		candidate := filepath.Join(dir, "ui", "src", "consts", "selectors.ts")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		scenarioCandidate := filepath.Join(dir, "scenarios", "browser-automation-studio", "ui", "src", "consts", "selectors.ts")
		if info, err := os.Stat(scenarioCandidate); err == nil && !info.IsDir() {
			return scenarioCandidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return ""
}

func loadSelectorManifestViaNode(selectorFile string) (map[string]string, string, error) {
	resolved := strings.TrimSpace(selectorFile)
	if resolved == "" {
		resolved = detectSelectorFile()
	}
	if resolved == "" {
		return nil, "", errors.New("selector registry not found")
	}
	scenarioDir := scenarioDirFromSelectors(resolved)
	if scenarioDir == "" {
		return nil, resolved, errors.New("unable to detect scenario root for selector registry")
	}
	scriptPath := detectSelectorRegistryScript(scenarioDir)
	if scriptPath == "" {
		return nil, resolved, errors.New("selector registry script not found")
	}
	cmd := exec.Command("node", scriptPath, "--scenario", scenarioDir)
	output, err := cmd.Output()
	if err != nil {
		return nil, resolved, err
	}
	var payload struct {
		TestIDs map[string]string `json:"testIds"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, resolved, err
	}
	if len(payload.TestIDs) == 0 {
		return nil, resolved, errors.New("selector registry is empty")
	}
	manifest := make(map[string]string, len(payload.TestIDs))
	for key, value := range payload.TestIDs {
		manifest[value] = key
	}
	return manifest, resolved, nil
}

func scenarioDirFromSelectors(selectorsPath string) string {
	dir := filepath.Clean(selectorsPath)
	for i := 0; i < 4; i++ {
		dir = filepath.Dir(dir)
		if dir == "." || dir == "" {
			return ""
		}
	}
	return dir
}

func detectSelectorRegistryScript(start string) string {
	if start == "" {
		return ""
	}
	for dir := start; dir != ""; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "scripts", "scenarios", "testing", "playbooks", "selector-registry.js")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return ""
}
