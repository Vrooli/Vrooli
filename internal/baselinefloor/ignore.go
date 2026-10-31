package baselinefloor

import (
	"bufio"
	"bytes"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// Ignorer decides whether a path inside a copied tree is skipped. rel is the
// slash-separated path relative to the copy root; isDir reports a directory.
type Ignorer interface {
	Ignored(rel string, isDir bool) bool
}

// floorIgnorePatterns are the restore-point exclusions that hold even where no
// .gitignore exists (fixtures, a --source outside the repository). They cover
// runtime state a code restore point must never carry: a project's .vrooli
// secrets, locks, local overrides and runtime state, and database files with
// their journals. The scenario's own .vrooli configuration (service.json,
// testing.json, ...) is NOT excluded: builds read it.
var floorIgnorePatterns = []string{
	"**/.vrooli/secrets.json",
	"**/.vrooli/*.lock",
	"**/.vrooli/*.local.json",
	"**/.vrooli/generated-service.json",
	"**/.vrooli/operator-state.json",
	"**/.vrooli/state/",
	"**/.vrooli/runtime/",
	"*.db",
	"*.db-wal",
	"*.db-shm",
	"*.db-journal",
	"*.sqlite",
	"*.sqlite-wal",
	"*.sqlite-shm",
	"*.sqlite3",
	"*.sqlite3-journal",
}

// FloorIgnorePatterns returns a copy of the floor's always-on exclusion
// patterns (gitignore syntax).
func FloorIgnorePatterns() []string {
	return append([]string(nil), floorIgnorePatterns...)
}

// IgnoreSpec configures an IgnoreMatcher.
type IgnoreSpec struct {
	// RepoRoot is the repository whose .gitignore files apply. Empty disables
	// .gitignore loading; only Patterns apply.
	RepoRoot string
	// TreeRoot is where the copied tree sits in the repository (for example
	// <repo>/scenarios/<name>). The .gitignore files of its ancestors up to
	// RepoRoot apply with their own anchors. A TreeRoot outside RepoRoot gets
	// no ancestor rules.
	TreeRoot string
	// NestedRoot is the directory nested .gitignore files are read from: the
	// copy source for a capture, the working tree for a restore. Empty uses
	// TreeRoot.
	NestedRoot string
	// Patterns are extra gitignore-syntax rules anchored at the tree root
	// (declared build outputs, data directories, floor rules). They are
	// evaluated before every .gitignore, so a .gitignore negation can
	// re-include a path they exclude, as git would.
	Patterns []string
}

// IgnoreMatcher applies .gitignore semantics without git: patterns, not the
// index, decide. Rules come from the extra patterns, then each ancestor
// .gitignore from RepoRoot down, then nested .gitignore files inside the
// tree; the last matching rule wins, and "!" re-includes. A path below an
// excluded directory is never evaluated (the copy walk skips the directory),
// which matches git's "cannot re-include a file under an excluded parent".
// A file that is tracked even though a pattern matches it (git add -f) is
// excluded here; the restore point is a code undo, so such a file is
// regenerated or left in place by an overlay restore.
type IgnoreMatcher struct {
	spec      IgnoreSpec
	treeVRel  string // TreeRoot relative to RepoRoot, slash-separated; "" if outside
	inRepo    bool
	baseRules []ignoreRuleSet

	mu     sync.Mutex
	nested map[string]*ignoreRuleSet // tree-relative dir -> rules (nil if none)
}

type ignoreRule struct {
	pattern  string
	negate   bool
	dirOnly  bool
	anchored bool
	segments []string
}

type ignoreRuleSet struct {
	// base is the rule set's directory in tree-virtual coordinates: the
	// slash path relative to RepoRoot (or to the tree root when the tree is
	// outside the repository).
	base  string
	rules []ignoreRule
}

// NewIgnoreMatcher builds a matcher for one tree copy.
func NewIgnoreMatcher(spec IgnoreSpec) *IgnoreMatcher {
	m := &IgnoreMatcher{spec: spec, nested: map[string]*ignoreRuleSet{}}
	if strings.TrimSpace(m.spec.NestedRoot) == "" {
		m.spec.NestedRoot = m.spec.TreeRoot
	}
	if repo := strings.TrimSpace(spec.RepoRoot); repo != "" && strings.TrimSpace(spec.TreeRoot) != "" {
		if rel, err := filepath.Rel(filepath.Clean(repo), filepath.Clean(spec.TreeRoot)); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			m.inRepo = true
			m.treeVRel = filepath.ToSlash(rel)
			if m.treeVRel == "." {
				m.treeVRel = ""
			}
		}
	}
	if len(spec.Patterns) > 0 {
		m.baseRules = append(m.baseRules, ignoreRuleSet{base: m.treeVRel, rules: parseIgnoreRules(spec.Patterns)})
	}
	if m.inRepo {
		// Ancestor .gitignore files, repo root first. The tree root's own
		// .gitignore is loaded as a nested file (tree dir "").
		dirs := []string{""}
		if m.treeVRel != "" {
			parts := strings.Split(m.treeVRel, "/")
			for i := 1; i < len(parts); i++ {
				dirs = append(dirs, strings.Join(parts[:i], "/"))
			}
		}
		for _, dir := range dirs {
			if set := readIgnoreFile(filepath.Join(spec.RepoRoot, filepath.FromSlash(dir), ".gitignore"), dir); set != nil {
				m.baseRules = append(m.baseRules, *set)
			}
		}
	}
	return m
}

// Ignored reports whether rel (relative to the copy root) is excluded.
func (m *IgnoreMatcher) Ignored(rel string, isDir bool) bool {
	if m == nil {
		return false
	}
	rel = strings.Trim(filepath.ToSlash(rel), "/")
	if rel == "" || rel == "." {
		return false
	}
	vpath := joinSlash(m.treeVRel, rel)
	ignored := false
	apply := func(set ignoreRuleSet) {
		sub, ok := relativeSlash(set.base, vpath)
		if !ok {
			return
		}
		for _, rule := range set.rules {
			if rule.matches(sub, isDir) {
				ignored = !rule.negate
			}
		}
	}
	for _, set := range m.baseRules {
		apply(set)
	}
	// Nested .gitignore files from the tree root down to rel's parent.
	parent := path.Dir(rel)
	dirs := []string{""}
	if parent != "." {
		parts := strings.Split(parent, "/")
		for i := 1; i <= len(parts); i++ {
			dirs = append(dirs, strings.Join(parts[:i], "/"))
		}
	}
	for _, dir := range dirs {
		if set := m.nestedRules(dir); set != nil {
			apply(*set)
		}
	}
	return ignored
}

func (m *IgnoreMatcher) nestedRules(treeDir string) *ignoreRuleSet {
	m.mu.Lock()
	defer m.mu.Unlock()
	if set, ok := m.nested[treeDir]; ok {
		return set
	}
	var set *ignoreRuleSet
	if root := strings.TrimSpace(m.spec.NestedRoot); root != "" {
		set = readIgnoreFile(filepath.Join(root, filepath.FromSlash(treeDir), ".gitignore"), joinSlash(m.treeVRel, treeDir))
	}
	m.nested[treeDir] = set
	return set
}

func readIgnoreFile(file, base string) *ignoreRuleSet {
	data, err := os.ReadFile(file)
	if err != nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	rules := parseIgnoreRules(lines)
	if len(rules) == 0 {
		return nil
	}
	return &ignoreRuleSet{base: base, rules: rules}
}

func parseIgnoreRules(lines []string) []ignoreRule {
	rules := make([]ignoreRule, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rule := ignoreRule{}
		switch {
		case strings.HasPrefix(line, "!"):
			rule.negate = true
			line = line[1:]
		case strings.HasPrefix(line, `\!`), strings.HasPrefix(line, `\#`):
			line = line[1:]
		}
		if strings.HasSuffix(line, "/") {
			rule.dirOnly = true
			line = strings.TrimRight(line, "/")
		}
		if line == "" {
			continue
		}
		// A slash anywhere but the end anchors the pattern to its file's
		// directory; otherwise it matches a name at any depth.
		rule.anchored = strings.Contains(line, "/")
		line = strings.TrimPrefix(line, "/")
		rule.pattern = line
		rule.segments = strings.Split(line, "/")
		rules = append(rules, rule)
	}
	return rules
}

func (r ignoreRule) matches(rel string, isDir bool) bool {
	if r.dirOnly && !isDir {
		return false
	}
	if !r.anchored {
		ok, _ := path.Match(r.pattern, path.Base(rel))
		return ok
	}
	return matchIgnoreSegments(r.segments, strings.Split(rel, "/"))
}

// matchIgnoreSegments matches slash segments where "**" spans zero or more
// whole segments, as in .gitignore.
func matchIgnoreSegments(pattern, name []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			rest := pattern[1:]
			if len(rest) == 0 {
				// "x/**" matches everything inside x, not x itself.
				return len(name) > 0
			}
			for i := 0; i <= len(name); i++ {
				if matchIgnoreSegments(rest, name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		if ok, _ := path.Match(pattern[0], name[0]); !ok {
			return false
		}
		pattern, name = pattern[1:], name[1:]
	}
	return len(name) == 0
}

func joinSlash(base, rel string) string {
	switch {
	case base == "":
		return rel
	case rel == "":
		return base
	default:
		return base + "/" + rel
	}
}

// relativeSlash returns target relative to base when target lies below it.
func relativeSlash(base, target string) (string, bool) {
	if base == "" {
		return target, target != ""
	}
	if !strings.HasPrefix(target, base+"/") {
		return "", false
	}
	return target[len(base)+1:], true
}
