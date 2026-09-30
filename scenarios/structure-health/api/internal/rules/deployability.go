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

// EvaluateControlPlane runs the rules that inspect control-plane source rather
// than a scenario tree. It is called only for the control-plane target kind.
//
// This rule used to run in Evaluate, on every scenario. Because it always walks
// <repo>/internal/deployability regardless of the target, every scenario report
// carried an identical set of errors located outside the scenario, which nobody
// validating a scenario could act on and which left no scenario able to pass.
func EvaluateControlPlane(in Input) []Finding {
	return deployabilityInstanceRules(in)
}

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
		// A test has to name a concrete object to exercise the resolver at all;
		// that is the fixture doing its job, not decision code carrying an
		// instance. The remediation ("pass manifest declarations into the pure
		// resolver") has no meaning for a fixture, so flagging one is noise.
		if strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		// cmd/ is the wiring boundary. The remediation for this rule is to
		// "pass manifest declarations into the pure resolver" — which names the
		// resolver, not the command that supplies it. A command dialling a
		// service by name is the boundary doing its job, and it is accepted
		// practice elsewhere in the control plane (see
		// internal/cli/scenariohandlers/tool_runtime.go). Decision code under
		// internal/deployability is still fully scanned.
		if isUnderCmd(path, deployabilityRoot) {
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
	// Tools are deliberately excluded. The policy is that deployability
	// decision code must not name a concrete object it deploys; a tool is an
	// ambient host binary that this code legitimately invokes. Including them
	// made every exec of "go", "git" or "docker" a violation, and the tool
	// vocabulary is full of generic words that collide with ordinary strings.
	for _, pattern := range []string{
		filepath.Join(repoRoot, "resources", "*", "resource.json"),
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
	// A scenario is named by its manifest, not by its directory. Using the
	// basename added non-scenario directories such as scenarios/scenarios to
	// the vocabulary, so the generic word "scenarios" became a violation.
	for _, path := range globDirs(filepath.Join(repoRoot, "scenarios", "*")) {
		var manifest struct {
			Service struct {
				Name string `json:"name"`
			} `json:"service"`
		}
		raw, err := os.ReadFile(filepath.Join(path, ".vrooli", "service.json")) // #nosec G304 -- repository-local glob.
		if err == nil && json.Unmarshal(raw, &manifest) == nil {
			add(manifest.Service.Name)
		}
	}
	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// globDirs returns only the directories matching pattern.
func globDirs(pattern string) []string {
	matches, _ := filepath.Glob(pattern)
	dirs := make([]string, 0, len(matches))
	for _, path := range matches {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			dirs = append(dirs, path)
		}
	}
	return dirs
}

// isUnderCmd reports whether path sits beneath a cmd/ directory within root.
func isUnderCmd(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	for _, segment := range strings.Split(filepath.ToSlash(rel), "/") {
		if segment == "cmd" {
			return true
		}
	}
	return false
}
