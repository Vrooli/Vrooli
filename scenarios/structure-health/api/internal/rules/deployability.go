package rules

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	deployability "github.com/vrooli/vrooli/packages/deployability"
)

const deployabilityInstanceRule = "DEPLOYABILITY_INSTANCE_IDENTIFIER"

// deployabilityInstanceRules registers the shared no-instance-identifier
// policy with structure-health. The instance vocabulary is loaded from the
// manifests; structure-health owns enforcement, while deployability owns the
// AST predicate.
func deployabilityInstanceRules(in Input) []Finding {
	repoRoot := findRepoRoot(in.Model.RootPath)
	if repoRoot == "" {
		return nil
	}
	known := declaredInstanceNames(repoRoot)
	if len(known) == 0 {
		return nil
	}
	deployabilityRoot := filepath.Join(repoRoot, "internal", "deployability")
	var findings []Finding
	_ = filepath.WalkDir(deployabilityRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		source, readErr := os.ReadFile(path) // #nosec G304 -- path is constrained under the repository deployability root.
		if readErr != nil {
			return nil
		}
		literals, parseErr := deployability.FindInstanceLiterals(path, source, known)
		if parseErr != nil {
			return nil
		}
		for _, literal := range literals {
			findings = append(findings, Finding{
				Code:        deployabilityInstanceRule,
				Severity:    sevError,
				Title:       "Deployability decision code names a concrete instance",
				Message:     fmt.Sprintf("string literal %q names a declared fleet object; load names at the manifest boundary", literal.Value),
				Location:    filepath.ToSlash(literal.Path) + fmt.Sprintf(":%d", literal.Line),
				Remediation: "Remove the instance literal and pass manifest declarations into the pure resolver.",
			})
		}
		return nil
	})
	sort.Slice(findings, func(i, j int) bool { return findings[i].Location < findings[j].Location })
	return findings
}

func findRepoRoot(start string) string {
	current, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	for {
		if info, statErr := os.Stat(filepath.Join(current, "internal", "deployability")); statErr == nil && info.IsDir() {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func declaredInstanceNames(repoRoot string) []string {
	seen := map[string]struct{}{}
	add := func(name string) {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" {
			seen[name] = struct{}{}
		}
	}
	for _, pattern := range []string{
		filepath.Join(repoRoot, "resources", "*", "resource.json"),
		filepath.Join(repoRoot, "internal", "tools", "*", "tool.json"),
		filepath.Join(repoRoot, "internal", "safeguards", "*", "safeguard.json"),
	} {
		matches, _ := filepath.Glob(pattern)
		for _, path := range matches {
			var manifest struct {
				Name string `json:"name"`
			}
			raw, err := os.ReadFile(path) // #nosec G304 -- paths are generated from repository-local glob patterns.
			if err == nil && json.Unmarshal(raw, &manifest) == nil {
				add(manifest.Name)
			}
		}
	}
	for _, pattern := range []string{filepath.Join(repoRoot, "scenarios", "*")} {
		matches, _ := filepath.Glob(pattern)
		for _, path := range matches {
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				add(filepath.Base(path))
			}
		}
	}
	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
