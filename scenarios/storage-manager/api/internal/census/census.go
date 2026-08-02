// Package census measures the storage surface without mutating it.
package census

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Entry struct {
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Bytes    int64  `json:"bytes"`
	Declared bool   `json:"declared"`
}

type Report struct {
	Root              string   `json:"root"`
	MeasuredBytes     int64    `json:"measured_bytes"`
	AttributedBytes   int64    `json:"attributed_bytes"`
	UnattributedBytes int64    `json:"unattributed_bytes"`
	Closed            bool     `json:"closed"`
	UnreadablePaths   []string `json:"unreadable_paths,omitempty"`
	Entries           []Entry  `json:"entries"`
}

func (r Report) MarshalJSON() ([]byte, error) { type alias Report; return json.Marshal(alias(r)) }

// Scan walks root and attributes bytes to matching declared storage paths.
// It never creates, removes, renames, or opens a database for writing.
func Scan(root string, manifests map[string][]Declaration) (Report, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Report{}, err
	}
	var files []string
	var unreadable []string
	var measured int64
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// A census is still useful when a host-managed subtree is not
			// readable. Record the gap explicitly and keep the scan read-only.
			unreadable = append(unreadable, path)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return statErr
		}
		measured += info.Size()
		files = append(files, path)
		return nil
	})
	if err != nil {
		return Report{}, fmt.Errorf("scan %s: %w", root, err)
	}
	var entries []Entry
	var attributed int64
	for owner, declarations := range manifests {
		for _, declaration := range declarations {
			path := declaration.Path
			if !filepath.IsAbs(path) {
				path = filepath.Join(root, path)
			}
			var bytes int64
			for _, file := range files {
				if sameOrBelow(path, file) {
					info, e := os.Stat(file)
					if e == nil {
						bytes += info.Size()
					}
				}
			}
			if bytes == 0 && declaration.Budgeted {
				continue
			}
			entries = append(entries, Entry{Owner: owner, Name: declaration.Name, Path: path, Bytes: bytes, Declared: true})
			attributed += bytes
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Owner != entries[j].Owner {
			return entries[i].Owner < entries[j].Owner
		}
		return entries[i].Name < entries[j].Name
	})
	sort.Strings(unreadable)
	return Report{Root: root, MeasuredBytes: measured, AttributedBytes: attributed, UnattributedBytes: measured - attributed, Closed: attributed <= measured && len(unreadable) == 0, UnreadablePaths: unreadable, Entries: entries}, nil
}

type Declaration struct {
	Name     string
	Path     string
	Budgeted bool
}

func sameOrBelow(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
