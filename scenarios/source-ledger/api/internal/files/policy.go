// Package files searches a file-authoritative corpus such as the research
// home. Files are the authority. The SQLite index is a rebuildable cache that
// every read first syncs against the files (size, modification time and
// inode change time, then content hash), so a hit cites the text on disk now.
//
// What gets indexed is data: built-in defaults, overridden by an optional
// `feeds.json` at the corpus root and at each `topics/<topic>/`. The research
// skills' file fallback (`find.py`) reads the same files with the same rules,
// so the two paths agree on what exists.
package files

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// PolicyFile is the optional per-folder override.
const PolicyFile = "feeds.json"

// Kind decides how a file is chunked and which end of a large file is kept.
type Kind string

const (
	KindMarkdown Kind = "markdown"
	KindText     Kind = "text"
	KindJSON     Kind = "json"
	KindJSONL    Kind = "jsonl"
	KindCSV      Kind = "csv"
	KindTSV      Kind = "tsv"
)

var kindByExt = map[string]Kind{
	".md": KindMarkdown, ".markdown": KindMarkdown,
	".txt": KindText, ".yaml": KindText, ".yml": KindText,
	".json":  KindJSON,
	".jsonl": KindJSONL, ".ndjson": KindJSONL,
	".csv": KindCSV, ".tsv": KindTSV,
}

// tail reports whether a large file's newest content is at its end
// (append-only logs and tables). Markdown ledgers here are newest-first.
func (k Kind) tail() bool { return k == KindJSONL || k == KindCSV || k == KindTSV }

// Policy is the effective indexing policy for one folder.
type Policy struct {
	// RunDepth is how many folder levels below runs/<run>/ are indexed in full;
	// deeper folders contribute only their README/INDEX/SUMMARY files.
	RunDepth int `json:"run_depth"`
	// DatasetFiles marks a folder holding more indexable files than this as a
	// generated dataset: only its README/INDEX/SUMMARY files are indexed.
	DatasetFiles int `json:"dataset_files"`
	// MaxFileKB caps the bytes read from one file (head, or tail for logs).
	MaxFileKB int `json:"max_file_kb"`
	// MaxJSONKB skips larger .json files outright: past this size they are
	// machine evidence, not prose, and only add noise and index weight.
	MaxJSONKB int `json:"max_json_kb"`
	// Exclude and Include are globs relative to the folder holding the
	// feeds.json (`**` spans folders). Include re-admits paths the depth,
	// dataset or code-tree rules would summarise.
	Exclude []string `json:"exclude"`
	Include []string `json:"include"`
	// Feeds names groups of files; unnamed files belong to their top folder.
	Feeds map[string][]string `json:"feeds"`
	// Private lists globs that are never indexed or served: operator
	// conversations and personal data. Unset inherits the corpus value (by
	// default DefaultPrivate); a topic may set its own list, even [].
	Private *[]string `json:"private"`
}

// DefaultPrivate keeps operator conversations and personal data out of the
// index unless a feeds.json says otherwise. Research reads these files
// directly when it needs them; search never serves them to other agents.
var DefaultPrivate = []string{
	"**/OPERATOR.md", "**/OPERATOR-*.md", "**/STANCE-MESSAGE.txt", "**/OPERATOR-WORDS*",
	"**/*resume*", "**/*Resume*", "**/personal/**", "**/private/**", "**/*.private.*",
}

// DefaultPolicy is used when no feeds.json overrides a value.
func DefaultPolicy() Policy {
	return Policy{RunDepth: 2, DatasetFiles: 200, MaxFileKB: 256, MaxJSONKB: 64}
}

// prunedDirs are never entered: dependencies, caches, VCS and service data.
var prunedDirs = map[string]bool{
	"node_modules": true, ".git": true, ".venv": true, "venv": true, "__pycache__": true,
	"dist": true, "build": true, "target": true, ".next": true, "persist": true,
	"vendor": true, "site-packages": true, ".cache": true, "coverage": true, ".pytest_cache": true,
}

// codeMarkers identify a checked-out source tree; code is Code Facts' corpus,
// so such a folder contributes only its README.
var codeMarkers = []string{".git", "go.mod", "package.json", "pyproject.toml", "Cargo.toml"}

func summaryFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "readme") || strings.HasPrefix(lower, "index") || strings.HasPrefix(lower, "summary")
}

// limits copies only the numeric limits, which nested folders inherit.
func (p Policy) limits() Policy {
	return Policy{RunDepth: p.RunDepth, DatasetFiles: p.DatasetFiles, MaxFileKB: p.MaxFileKB, MaxJSONKB: p.MaxJSONKB}
}

// loadPolicy reads dir/feeds.json over base's numeric limits (globs and
// feeds never inherit: they are relative to the folder that declares them).
// A missing file returns base's limits; a malformed one is an error the
// caller reports (as a search degradation) and handles conservatively.
func loadPolicy(dir string, base Policy) (Policy, error) {
	next := base.limits()
	raw, err := os.ReadFile(filepath.Join(dir, PolicyFile))
	if errors.Is(err, fs.ErrNotExist) {
		return next, nil
	}
	if err != nil {
		return next, err
	}
	var o Policy
	if err := json.Unmarshal(raw, &o); err != nil {
		return next, fmt.Errorf("%s: %w", filepath.Join(dir, PolicyFile), err)
	}
	for _, f := range []struct {
		dst *int
		v   int
	}{{&next.RunDepth, o.RunDepth}, {&next.DatasetFiles, o.DatasetFiles}, {&next.MaxFileKB, o.MaxFileKB}, {&next.MaxJSONKB, o.MaxJSONKB}} {
		if f.v > 0 {
			*f.dst = f.v
		}
	}
	next.Exclude, next.Include, next.Feeds, next.Private = o.Exclude, o.Include, o.Feeds, o.Private
	return next, nil
}

// match reports whether rel (slash-separated) matches glob. `**` matches any
// number of segments; other segments use path.Match. The Python fallback
// implements the same rule.
func match(glob, rel string) bool {
	return matchSegments(strings.Split(strings.Trim(glob, "/"), "/"), strings.Split(rel, "/"))
}

func matchSegments(glob, parts []string) bool {
	for len(glob) > 0 {
		if glob[0] == "**" {
			for i := 0; i <= len(parts); i++ {
				if matchSegments(glob[1:], parts[i:]) {
					return true
				}
			}
			return false
		}
		if len(parts) == 0 {
			return false
		}
		if ok, _ := path.Match(glob[0], parts[0]); !ok {
			return false
		}
		glob, parts = glob[1:], parts[1:]
	}
	return len(parts) == 0
}

// mayContain reports whether some path below dir could match glob, so a
// summarised folder is still entered for an Include.
func mayContain(glob, dir string) bool {
	g := strings.Split(strings.Trim(glob, "/"), "/")
	for i, part := range strings.Split(dir, "/") {
		if i >= len(g) {
			return false
		}
		if g[i] == "**" {
			return true
		}
		if ok, _ := path.Match(g[i], part); !ok {
			return false
		}
	}
	return true
}

func anyMatch(globs []string, rel string) bool {
	for _, g := range globs {
		if match(g, rel) || match(g+"/**", rel) {
			return true
		}
	}
	return false
}
