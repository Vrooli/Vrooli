package analysis

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// SourceLocator resolves a React component name to its definition site by
// scanning a scenario's UI source tree (ui/src) for the component's declaration.
// It is deterministic — pure file/text scanning, NO AI. It recognizes the common
// React component declaration forms:
//
//	function Foo(            export function Foo(
//	const Foo = (           export const Foo = (
//	const Foo: React.FC     class Foo extends
//	export default function Foo(
//
// The returned definition is "ui/src/relative/path.tsx:line" (repo-relative to
// the scenario root). Returns ok=false when the component can't be located.
type SourceLocator struct {
	// RepoRoot is the Vrooli repository root; scenarios live under
	// <RepoRoot>/scenarios/<scenario>/ui/src.
	RepoRoot string
	// ScenarioRoot, when set, overrides RepoRoot+scenario resolution (used by
	// tests and when the analysis runs against an explicit path).
	ScenarioRoot string
}

// Locate finds the file:line where the named component is defined.
func (l SourceLocator) Locate(scenario, component string) (string, bool) {
	root := l.uiSrcRoot(scenario)
	if root == "" {
		return "", false
	}
	pattern := declPattern(component)
	if pattern == nil {
		return "", false
	}

	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // best-effort scan; skip unreadable entries
		}
		if found != "" {
			return filepath.SkipAll
		}
		if d.IsDir() {
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isTSX(path) {
			return nil
		}
		if line, ok := scanFile(path, pattern); ok {
			rel := relTo(l.scenarioRoot(scenario), path)
			found = rel + ":" + strconv.Itoa(line)
			return filepath.SkipAll
		}
		return nil
	})
	if found == "" {
		return "", false
	}
	return found, true
}

// uiSrcRoot resolves the ui/src directory for a scenario.
func (l SourceLocator) uiSrcRoot(scenario string) string {
	base := l.scenarioRoot(scenario)
	if base == "" {
		return ""
	}
	src := filepath.Join(base, "ui", "src")
	if info, err := os.Stat(src); err == nil && info.IsDir() {
		return src
	}
	return ""
}

func (l SourceLocator) scenarioRoot(scenario string) string {
	if strings.TrimSpace(l.ScenarioRoot) != "" {
		return l.ScenarioRoot
	}
	if strings.TrimSpace(l.RepoRoot) == "" || strings.TrimSpace(scenario) == "" {
		return ""
	}
	return filepath.Join(l.RepoRoot, "scenarios", scenario)
}

// declPattern builds the regexp matching any declaration form for the named
// component. The name is anchored on a word boundary so "List" doesn't match
// "ListItem".
func declPattern(component string) *regexp.Regexp {
	name := strings.TrimSpace(component)
	if name == "" || !isIdentifier(name) {
		return nil
	}
	q := regexp.QuoteMeta(name)
	// Common React component declaration forms.
	return regexp.MustCompile(
		`(?:^|\b)(?:export\s+)?(?:default\s+)?` +
			`(?:function\s+` + q + `\b` +
			`|(?:const|let|var)\s+` + q + `\b\s*(?::[^=]+)?=\s*(?:\(|[A-Za-z_$]+\s*=>|React\.|memo\(|forwardRef\()` +
			`|class\s+` + q + `\b)`,
	)
}

// scanFile returns the 1-based line where pattern first matches.
func scanFile(path string, pattern *regexp.Regexp) (int, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	line := 0
	for sc.Scan() {
		line++
		if pattern.MatchString(sc.Text()) {
			return line, true
		}
	}
	return 0, false
}

func relTo(scenarioRoot, path string) string {
	rel, err := filepath.Rel(scenarioRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func isTSX(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".tsx", ".jsx", ".ts", ".js":
		return true
	}
	return false
}

func skipDir(name string) bool {
	switch name {
	case "node_modules", "dist", "build", ".vite", "coverage", "__tests__":
		return true
	}
	return false
}

func isIdentifier(s string) bool {
	for i, r := range s {
		if r == '_' || r == '$' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}
