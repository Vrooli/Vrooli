// Package intents measures discoverability: for each natural request an
// operator might type, does the search a cold agent runs surface a leaf (skill,
// program, command) of the scenario that can actually do it?
//
// This is the cross-provider counterpart of `evals`. Provider suites grade one
// corpus's retrieval; the router suite grades provider selection; intents grade
// the end result an agent sees across `--type library,skill,command`. A
// scenario can be perfect in every suite and still be undiscoverable because
// its descriptions share no vocabulary with how requests are phrased.
package intents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultTypes is the leaf-type set a cold agent is told to search first.
const DefaultTypes = "library,skill,command"

// DefaultLimit is the display depth an agent reads (the top-5 contract).
const DefaultLimit = 5

// corpusFile is one authored intent corpus. File-level Scenario is the owner of
// every case; the value "*" means each case names its own owner.
type corpusFile struct {
	Scenario    string `json:"scenario"`
	Description string `json:"description"`
	// ExpectTypes is the default leaf-type filter for cases that set none.
	ExpectTypes []string     `json:"expect_types,omitempty"`
	Cases       []intentCase `json:"cases"`
}

// intentCase is one reviewed natural request and what must surface.
type intentCase struct {
	ID    string `json:"id"`
	Query string `json:"query"`
	// Owner overrides the file's scenario (required when the file scenario is "*").
	Owner string `json:"owner,omitempty"`
	// Types overrides DefaultTypes (comma-separated leaf types).
	Types string `json:"types,omitempty"`
	// ExpectTypes limits which owned leaves count, e.g. ["skill"]. Empty means
	// the file default; empty there too means any leaf type counts.
	ExpectTypes []string `json:"expect_types,omitempty"`
	// ExpectIDs, when set, requires one of these exact leaf ids.
	ExpectIDs []string `json:"expect_ids,omitempty"`
	Tags      []string `json:"tags,omitempty"`

	source string // file the case came from, for error messages
}

// owner returns the scenario that must surface for the case.
func (c intentCase) owner(file corpusFile) string {
	if c.Owner != "" {
		return c.Owner
	}
	return file.Scenario
}

// loadedCase is a validated case joined with its resolved owner.
type loadedCase struct {
	intentCase
	Scenario string
}

// corpusSources lists every authored corpus under root: the shared search-hub
// corpus directory plus scenario-owned `.vrooli/intents.json` files, so a
// scenario can declare its own intents without editing Search Hub.
func corpusSources(root string) ([]string, error) {
	var paths []string
	for _, pattern := range []string{
		filepath.Join(root, "scenarios", "search-hub", "intents", "*.json"),
		filepath.Join(root, "scenarios", "*", ".vrooli", "intents.json"),
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)
	return paths, nil
}

// loadCorpus reads and validates the given corpus files. Validation is strict
// (unknown fields, missing ids/queries/owners, duplicate ids) because a silent
// skip would turn a broken corpus into a passing measurement.
func loadCorpus(paths []string) ([]loadedCase, error) {
	var out []loadedCase
	seen := make(map[string]string)
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read intent corpus %s: %w", path, err)
		}
		var file corpusFile
		decoder := json.NewDecoder(strings.NewReader(string(raw)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&file); err != nil {
			return nil, fmt.Errorf("parse intent corpus %s: %w", path, err)
		}
		if strings.TrimSpace(file.Scenario) == "" {
			return nil, fmt.Errorf("intent corpus %s: scenario is required (use \"*\" when each case names its owner)", path)
		}
		for _, c := range file.Cases {
			c.source = path
			owner := c.owner(file)
			switch {
			case strings.TrimSpace(c.ID) == "":
				return nil, fmt.Errorf("intent corpus %s: a case has no id", path)
			case strings.TrimSpace(c.Query) == "":
				return nil, fmt.Errorf("intent corpus %s: case %q has no query", path, c.ID)
			case owner == "" || owner == "*":
				return nil, fmt.Errorf("intent corpus %s: case %q has no owner scenario", path, c.ID)
			}
			if len(c.ExpectTypes) == 0 {
				c.ExpectTypes = file.ExpectTypes
			}
			key := owner + "/" + c.ID
			if prior, dup := seen[key]; dup {
				return nil, fmt.Errorf("intent corpus %s: case %q duplicates one in %s", path, key, prior)
			}
			seen[key] = path
			out = append(out, loadedCase{intentCase: c, Scenario: owner})
		}
	}
	return out, nil
}
