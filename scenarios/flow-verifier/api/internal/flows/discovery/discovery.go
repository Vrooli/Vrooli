// Package discovery walks a scenario root for flow contracts and
// dispatches each to its registered Kind for parsing and validation.
//
// On-disk convention: any *.json file inside a directory named "flow/"
// is a flow contract. The required top-level "kind" field selects the
// Kind that owns it.
package discovery

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"flow-verifier/internal/flows/kind"
)

// ignoredDirs mirrors fsadapter's ignore list so a recursive walk does
// not descend into VCS, build, or dependency caches.
var ignoredDirs = map[string]bool{
	".git":          true,
	"node_modules":  true,
	"dist":          true,
	"build":         true,
	"coverage":      true,
	"_apalache-out": true,
}

// FindContracts walks root, locates every *.json file inside a flow/
// directory, peeks the "kind" field, and dispatches to the matching
// registered Kind.Load. Returns Specs sorted by FlowID for stable output.
func FindContracts(root string) ([]kind.Spec, error) {
	paths, err := findContractFiles(root)
	if err != nil {
		return nil, err
	}
	specs := make([]kind.Spec, 0, len(paths))
	for _, rel := range paths {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(abs)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		kindName, err := peekKind(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		k, ok := kind.Get(kindName)
		if !ok {
			return nil, fmt.Errorf("%s: unknown kind %q (registered kinds: %v)", rel, kindName, kind.Names())
		}
		spec, err := k.Load(data, rel)
		if err != nil {
			return nil, fmt.Errorf("invalid %s contract %s:\n%s", kindName, rel, err)
		}
		specs = append(specs, spec)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].FlowID() < specs[j].FlowID() })
	return specs, nil
}

// Filter returns the subset of specs whose FlowID matches flowID. An
// empty flowID returns all specs unchanged.
func Filter(specs []kind.Spec, flowID string) []kind.Spec {
	if flowID == "" {
		return specs
	}
	out := make([]kind.Spec, 0, len(specs))
	for _, s := range specs {
		if s.FlowID() == flowID {
			out = append(out, s)
		}
	}
	return out
}

// findContractFiles walks root and returns repo-relative paths of every
// *.json file living inside a directory named "flow". Ignored
// directories are skipped.
func findContractFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if entry != nil && entry.IsDir() && os.IsPermission(err) {
				return filepath.SkipDir
			}
			return err
		}
		if entry.IsDir() {
			if ignoredDirs[entry.Name()] && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) != ".json" {
			return nil
		}
		if filepath.Base(filepath.Dir(path)) != "flow" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// peekKind extracts the top-level "kind" field from a contract's raw
// bytes without parsing the rest. Returns an error if the field is
// missing or empty — every contract must declare its kind.
func peekKind(data []byte) (string, error) {
	var probe struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return "", fmt.Errorf("parse top-level kind field: %w", err)
	}
	if probe.Kind == "" {
		return "", fmt.Errorf("contract is missing required top-level \"kind\" field")
	}
	return probe.Kind, nil
}
