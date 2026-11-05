package evidenceretention

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// References is the set of evidence paths that work records name. Work
// records (delivery LOG/STATUS/epochs, goal homes, requirements, scenario
// assets, plan artifacts, research notes) cite evidence by absolute or
// home-relative path containing ".vrooli/evidence/".
type References struct {
	// paths maps a cleaned evidence-relative path to one citing file.
	paths map[string]string
}

// evidenceMarker is what every citation of the evidence home contains.
var evidenceMarker = []byte(".vrooli/evidence/")

var referencePattern = regexp.MustCompile(`\.vrooli/evidence/([^\s"'` + "`" + `<>()\[\]{},;|*]+)`)

// skipDirs are generated or bulky trees that never hold work records.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "dist": true, "build": true, "coverage": true,
	"vendor": true, "target": true, ".next": true, "__pycache__": true, ".cache": true,
	".venv": true, "venv": true, "tmp": true, "gen": true, "data": true,
}

var textExt = map[string]bool{
	".md": true, ".json": true, ".jsonl": true, ".yaml": true, ".yml": true, ".txt": true,
	".go": true, ".ts": true, ".tsx": true, ".js": true, ".html": true, ".toml": true, ".sh": true, ".csv": true,
}

// maxReferenceFileBytes bounds one scanned work record.
const maxReferenceFileBytes = 4 << 20

// NewReferences builds a reference set from explicit paths (tests and
// callers that already know the citations).
func NewReferences(paths map[string]string) *References {
	r := &References{paths: map[string]string{}}
	for p, src := range paths {
		r.add(p, src)
	}
	return r
}

func (r *References) add(raw, source string) {
	p := strings.TrimRight(raw, ".:/'\"")
	if p == "" {
		return
	}
	p = path.Clean(p)
	if p == "." || strings.HasPrefix(p, "..") {
		return
	}
	if _, ok := r.paths[p]; !ok {
		r.paths[p] = source
	}
}

// Len reports how many distinct evidence paths are cited.
func (r *References) Len() int {
	if r == nil {
		return 0
	}
	return len(r.paths)
}

// Protects returns the citing file when a work record names the candidate
// itself or anything inside it. A citation of an ancestor directory (an
// effort root or a whole checkpoint) does not protect a regenerable
// intermediate: the class already proves it is not the cited evidence.
func (r *References) Protects(candidate string) string {
	if r == nil {
		return ""
	}
	if src, ok := r.paths[candidate]; ok {
		return src
	}
	prefix := candidate + "/"
	keys := r.sortedKeys()
	i := sort.SearchStrings(keys, prefix)
	if i < len(keys) && strings.HasPrefix(keys[i], prefix) {
		return r.paths[keys[i]]
	}
	return ""
}

// ReferencesRetained reports whether a retained file is cited directly or
// sits under a cited directory deeper than the effort root.
func (r *References) ReferencesRetained(rel string) bool {
	if r == nil {
		return false
	}
	for p := rel; strings.Contains(p, "/"); p = path.Dir(p) {
		if _, ok := r.paths[p]; ok {
			return true
		}
	}
	return false
}

func (r *References) sortedKeys() []string {
	keys := make([]string, 0, len(r.paths))
	for k := range r.paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// CollectReferences scans work-record roots for citations of the evidence
// home. Missing roots are skipped; the scan never follows symlinks and never
// reads generated or dependency trees.
func CollectReferences(ctx context.Context, roots []string, evidenceRoot string) (*References, error) {
	r := &References{paths: map[string]string{}}
	evidenceRoot = filepath.Clean(evidenceRoot)
	for _, root := range roots {
		if root == "" {
			continue
		}
		if _, err := os.Stat(root); err != nil {
			continue
		}
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			if d.IsDir() {
				if p != root && (skipDirs[d.Name()] || filepath.Clean(p) == evidenceRoot) {
					return fs.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() || !textExt[strings.ToLower(filepath.Ext(d.Name()))] {
				return nil
			}
			info, err := d.Info()
			if err != nil || info.Size() > maxReferenceFileBytes {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil || !bytes.Contains(data, evidenceMarker) {
				return nil
			}
			for _, m := range referencePattern.FindAllSubmatch(data, -1) {
				r.add(string(m[1]), p)
			}
			return nil
		})
		if err != nil {
			return r, err
		}
	}
	return r, nil
}
