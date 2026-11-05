package uiinterop

import (
	"io/fs"
	"sort"
	"strings"
	"sync"
)

var (
	mu        sync.Mutex
	checkFns  = map[string]CheckFunc{}
	embedFSes = map[string]fs.FS{}
	built     []Rule
	buildOnce sync.Once
)

// Register is called from each rule's init() to register id + check function.
func Register(id string, fn CheckFunc) {
	mu.Lock()
	defer mu.Unlock()
	checkFns[id] = fn
}

// RegisterEmbedFS is called from each category's init() to register the
// embedded source files for metadata parsing.
func RegisterEmbedFS(category string, fsys fs.FS) {
	mu.Lock()
	defer mu.Unlock()
	embedFSes[category] = fsys
}

// build parses all registered embed.FS files and marries metadata with CheckFuncs.
func build() {
	mu.Lock()
	defer mu.Unlock()

	built = make([]Rule, 0, len(checkFns))

	// Parse metadata from all embedded source files.
	defs := map[string]RuleDef{}
	for _, fsys := range embedFSes {
		_ = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := fs.ReadFile(fsys, path)
			if err != nil {
				return nil
			}
			def, ok := ParseRuleDoc(string(data))
			if !ok {
				return nil
			}
			def.Enabled = true
			defs[def.ID] = def
			return nil
		})
	}

	for id, fn := range checkFns {
		def, ok := defs[id]
		if !ok {
			// Rule registered without metadata — create a minimal def.
			def = RuleDef{ID: id, Enabled: true}
		}
		built = append(built, Rule{Def: def, Check: fn})
	}

	sort.Slice(built, func(i, j int) bool {
		return built[i].Def.ID < built[j].Def.ID
	})
}

// All returns all registered rules sorted by ID.
func All() []Rule {
	buildOnce.Do(build)
	out := make([]Rule, len(built))
	copy(out, built)
	return out
}

// AllDefs returns just the metadata for all rules (for API responses).
func AllDefs() []RuleDef {
	rules := All()
	defs := make([]RuleDef, len(rules))
	for i, r := range rules {
		defs[i] = r.Def
	}
	return defs
}

// EmbedFSes returns the registered embedded filesystems (used by test harness).
func EmbedFSes() map[string]fs.FS {
	mu.Lock()
	defer mu.Unlock()
	out := make(map[string]fs.FS, len(embedFSes))
	for k, v := range embedFSes {
		out[k] = v
	}
	return out
}

// severityOrder defines priority for sorting rules by severity.
var severityOrder = map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}

// SortDefsBySeverity sorts rule definitions in-place by severity priority (critical first),
// with stable ordering by ID within the same severity.
func SortDefsBySeverity(defs []RuleDef) {
	sort.SliceStable(defs, func(i, j int) bool {
		oi := severityOrder[strings.ToLower(defs[i].Severity)]
		oj := severityOrder[strings.ToLower(defs[j].Severity)]
		if oi != oj {
			return oi < oj
		}
		return defs[i].ID < defs[j].ID
	})
}

// DefsForTechStack returns rule definitions matching the given tech stack, sorted by severity.
func DefsForTechStack(stack []string) []RuleDef {
	matched := ForTechStack(stack)
	defs := make([]RuleDef, len(matched))
	for i, r := range matched {
		defs[i] = r.Def
	}
	SortDefsBySeverity(defs)
	return defs
}

// ForTechStack returns rules whose TechStack matches the scenario's detected stack.
// A rule matches if any of its required components appear in the stack, or if
// its TechStack is empty (meaning it always runs).
func ForTechStack(stack []string) []Rule {
	all := All()
	stackSet := map[string]bool{}
	for _, s := range stack {
		stackSet[strings.ToLower(s)] = true
	}

	out := make([]Rule, 0, len(all))
	for _, r := range all {
		if len(r.Def.TechStack) == 0 {
			out = append(out, r)
			continue
		}
		// A single "*" means always run.
		for _, ts := range r.Def.TechStack {
			if ts == "*" {
				out = append(out, r)
				break
			}
			if stackSet[strings.ToLower(ts)] {
				out = append(out, r)
				break
			}
		}
	}
	return out
}
