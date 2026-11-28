// Package discovery walks a scenario root for flow contracts and
// dispatches each to its registered Kind for parsing and validation.
//
// On-disk convention: any *.json file inside a directory named "flow/"
// is a flow contract. The required top-level "kind" field selects the
// Kind that owns it.
package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"flow-verifier/internal/flows/kind"
)

var ErrFlowNotFound = errors.New("flow not found")

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
		spec, err := loadContract(root, rel)
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].FlowID() < specs[j].FlowID() })
	return specs, nil
}

// FindContract locates and validates only the requested flow. A targeted
// operation must not be blocked by an unrelated malformed contract elsewhere
// in the repository; global inventory still uses FindContracts and reports all
// such failures.
func FindContract(root, flowID string) (kind.Spec, error) {
	if flowID == "" {
		return nil, fmt.Errorf("flow id is required")
	}
	paths, err := findContractFiles(root)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, rel := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		id, err := peekFlowID(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		if id == flowID {
			matches = append(matches, rel)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrFlowNotFound, flowID)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("flow id %s is declared by multiple contracts: %v", flowID, matches)
	}
	return loadContract(root, matches[0])
}

// FindContractsFiltered validates the whole inventory when flowID is empty,
// or validates only the requested flow when the caller names one.
func FindContractsFiltered(root, flowID string) ([]kind.Spec, error) {
	if flowID == "" {
		return FindContracts(root)
	}
	spec, err := FindContract(root, flowID)
	if err != nil {
		return nil, err
	}
	return []kind.Spec{spec}, nil
}

func loadContract(root, rel string) (kind.Spec, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
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
	return spec, nil
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

func peekFlowID(data []byte) (string, error) {
	var probe struct {
		FlowID string `json:"flowId"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return "", fmt.Errorf("parse top-level flowId field: %w", err)
	}
	if probe.FlowID == "" {
		return "", fmt.Errorf("contract is missing required top-level \"flowId\" field")
	}
	return probe.FlowID, nil
}
