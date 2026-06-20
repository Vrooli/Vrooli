// Package scan builds the per-rule content inputs the migrated scenario-auditor
// rule packs expect, replicating the auditor's standards-scan file feeding so
// the default profile reproduces today's verdicts byte-for-byte.
//
// The auditor fed rules three ways:
//   - structure rules: one JSON {scenario, files:[...]} payload + the scenario
//     root, run once per scenario;
//   - service_json config rules: the .vrooli/service.json content, run once;
//   - api/cli/ui/test rules: each matching source file's content, run per file.
//
// The classification logic (classifyFileTargets, allowedExtensions, the binary
// guard, the size guard, the directory skip-set) is preserved here verbatim.
package scan

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/vrooli/api-core/pathfilter"
)

// Target identifies which rule pack a file feeds, mirroring the auditor targets.
const (
	TargetAPI         = "api"
	TargetMainGo      = "main_go"
	TargetUI          = "ui"
	TargetCLI         = "cli"
	TargetTest        = "test"
	TargetServiceJSON = "service_json"
	TargetMakefile    = "makefile"
	TargetStructure   = "structure"
	TargetDocs        = "documentation"
)

// maxFileSizeBytes is the auditor's 1 MiB per-file guardrail.
const maxFileSizeBytes = 1 << 20

// allowedExtensions mirrors the auditor's scannable-file allowlist.
var allowedExtensions = map[string]struct{}{
	".go": {}, ".ts": {}, ".tsx": {}, ".js": {}, ".jsx": {}, ".sh": {},
	".json": {}, ".yaml": {}, ".yml": {}, ".html": {}, ".css": {},
	".py": {}, ".java": {}, ".md": {},
}

// File is one scannable file: its scenario-relative slash path, absolute path,
// content, and the targets it feeds.
type File struct {
	Relative string
	AbsPath  string
	Content  string
	Targets  []string
}

// Context is a scenario's scanned content, indexed by target.
type Context struct {
	Scenario string
	RootPath string
	// AllFiles is every non-skipped, non-symlink, non-oversize file's relative
	// path (binaries included) — the auditor's structure file-list.
	AllFiles []string
	// byTarget maps a target to the files that feed it (content-bearing,
	// text, allowed-extension files only).
	byTarget map[string][]File
}

// Build walks the scenario root and classifies its files, replicating the
// auditor's standards-scan traversal.
func Build(scenario, rootPath string) (*Context, error) {
	ctx := &Context{
		Scenario: scenario,
		RootPath: rootPath,
		byTarget: map[string][]File{},
	}
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			if pathfilter.SkipDir(filepath.Base(path)) {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if info.Size() > maxFileSizeBytes {
			return nil
		}

		rel, err := filepath.Rel(rootPath, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "" || rel == "." {
			return nil
		}
		// Every visited file joins the structure file-list, binaries included
		// (the auditor adds to structureData before the binary guard).
		ctx.AllFiles = append(ctx.AllFiles, rel)

		targets := classifyTargets(rel)
		if len(targets) == 0 {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if isBinaryContent(content) {
			return nil
		}
		f := File{Relative: rel, AbsPath: path, Content: string(content), Targets: targets}
		for _, t := range targets {
			ctx.byTarget[t] = append(ctx.byTarget[t], f)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(ctx.AllFiles)
	for t := range ctx.byTarget {
		sort.Slice(ctx.byTarget[t], func(i, j int) bool {
			return ctx.byTarget[t][i].Relative < ctx.byTarget[t][j].Relative
		})
	}
	return ctx, nil
}

// FilesForTarget returns the content-bearing files feeding the given target.
func (c *Context) FilesForTarget(target string) []File {
	return c.byTarget[target]
}

// ServiceJSON returns the .vrooli/service.json file (content + abs path) and
// whether it was found among the scannable files.
func (c *Context) ServiceJSON() (File, bool) {
	for _, f := range c.byTarget[TargetServiceJSON] {
		return f, true
	}
	return File{}, false
}

// StructurePayload returns the JSON {scenario, files} payload the structure
// rule pack expects as its content argument.
func (c *Context) StructurePayload() string {
	payload := struct {
		Scenario string   `json:"scenario"`
		Files    []string `json:"files"`
	}{Scenario: c.Scenario, Files: c.AllFiles}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

// classifyTargets mirrors the auditor's classifyFileTargets for a
// scenario-relative slash path.
func classifyTargets(rel string) []string {
	if rel == "" {
		return nil
	}
	if rel == ".vrooli/service.json" {
		return []string{TargetServiceJSON}
	}
	base := filepath.Base(rel)
	if strings.HasSuffix(base, "-api") {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(rel))
	if ext != "" {
		if _, ok := allowedExtensions[ext]; !ok {
			return nil
		}
	}

	var targets []string
	if rel == "Makefile" {
		targets = append(targets, TargetMakefile)
	}
	lower := strings.ToLower(rel)
	if strings.EqualFold(rel, "PRD.md") || strings.EqualFold(rel, "README.md") {
		targets = append(targets, TargetDocs)
	}
	if strings.HasPrefix(lower, "docs/") {
		targets = append(targets, TargetDocs)
	}
	if strings.HasPrefix(rel, "api/") {
		targets = append(targets, TargetAPI)
	}
	if strings.HasPrefix(rel, "cli/") {
		targets = append(targets, TargetCLI)
	}
	if strings.HasPrefix(rel, "ui/") {
		targets = append(targets, TargetUI)
	}
	if strings.HasPrefix(rel, "test/") {
		targets = append(targets, TargetTest)
	}
	if strings.HasSuffix(rel, "main.go") {
		targets = append(targets, TargetMainGo)
	}
	return targets
}

func isBinaryContent(content []byte) bool {
	if len(content) == 0 {
		return false
	}
	if bytes.IndexByte(content, 0) >= 0 {
		return true
	}
	return !utf8.Valid(content)
}
