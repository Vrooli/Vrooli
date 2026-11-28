package policybridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corestorage "github.com/vrooli/api-core/storage"
)

// DisposableFile is the repository's declaration of leftovers that tools and
// tests make for themselves and never clean up. Each entry names the owner that
// makes them and the exact generated names, so an agent may delete one without
// asking while everything else under the same folder keeps its disposition.
// It is read by storage-manager only; the agent policy runtime receives the
// result as path rules with a name match.
const DisposableFile = ".vrooli/disposable-leftovers.json"

// Leftover is one declared kind of disposable leftover.
type Leftover struct {
	ID    string `json:"id"`
	Owner string `json:"owner"`
	// Root is the folder the leftovers appear in, with governed variables
	// ($REPO_ROOT, $VROOLI_HOME, $USER_HOME). Match is a name glob, never a path.
	Root  string `json:"root"`
	Match string `json:"match"`
	// Only names the files a leftover may hold. A leftover holding anything
	// else is not disposable and the deletion asks, so a name that happens to
	// match real data is not opened. Empty means the tool's whole folder.
	Only      []string `json:"only,omitempty"`
	Rationale string   `json:"rationale"`
}

type disposableDocument struct {
	Version   int        `json:"version"`
	Leftovers []Leftover `json:"leftovers"`
}

// ValidateLeftover enforces what the declaration must say: who makes it, what
// it matches, and why it is safe. A match that is only wildcards, or that names
// a path, is refused, since it would open far more than a tool's own leftovers.
func ValidateLeftover(l Leftover) error {
	switch {
	case strings.TrimSpace(l.ID) == "" || strings.TrimSpace(l.Owner) == "":
		return fmt.Errorf("leftover %q: id and owner are required", l.ID)
	case strings.TrimSpace(l.Rationale) == "":
		return fmt.Errorf("leftover %q: a rationale is required", l.ID)
	case strings.TrimSpace(l.Root) == "":
		return fmt.Errorf("leftover %q: root is required", l.ID)
	case strings.TrimSpace(l.Match) == "" || strings.ContainsAny(l.Match, `/\`):
		return fmt.Errorf("leftover %q: match must be a name glob, not a path", l.ID)
	case strings.Trim(l.Match, "*?[]") == "":
		return fmt.Errorf("leftover %q: match %q names nothing specific", l.ID, l.Match)
	}
	if _, err := filepath.Match(l.Match, "x"); err != nil {
		return fmt.Errorf("leftover %q: match: %w", l.ID, err)
	}
	for _, glob := range l.Only {
		if _, err := filepath.Match(glob, "x"); err != nil {
			return fmt.Errorf("leftover %q: only %q: %w", l.ID, glob, err)
		}
	}
	return nil
}

// LoadLeftovers reads the declarations. A repository without the file declares
// none; a file that cannot be read or holds an invalid entry is an error, so a
// typo never silently drops or widens a declaration.
func LoadLeftovers(repoRoot string) ([]Leftover, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(DisposableFile)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc disposableDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", DisposableFile, err)
	}
	if doc.Version != 1 {
		return nil, fmt.Errorf("%s: version must be 1", DisposableFile)
	}
	seen := map[string]bool{}
	for _, l := range doc.Leftovers {
		if err := ValidateLeftover(l); err != nil {
			return nil, fmt.Errorf("%s: %w", DisposableFile, err)
		}
		if seen[l.ID] {
			return nil, fmt.Errorf("%s: duplicate id %q", DisposableFile, l.ID)
		}
		seen[l.ID] = true
	}
	return doc.Leftovers, nil
}

// disposableRules turns the declarations into allow rules that match generated
// names. They outrank nothing by priority: a rule on a name inside a protected
// parent is more specific than the parent, which is how the runtime lets it
// open one declared kind of folder and nothing else.
func disposableRules(repoRoot string, inputs corestorage.GovernedRootInputs, warnings *[]string) ([]PathRule, error) {
	leftovers, err := LoadLeftovers(repoRoot)
	if err != nil {
		return nil, err
	}
	var rules []PathRule
	for _, l := range leftovers {
		root, err := corestorage.ResolveGovernedRootStrict(l.Root, inputs)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("disposable leftover %s: root %q does not resolve on this host", l.ID, l.Root))
			continue
		}
		rules = append(rules, PathRule{
			Root: root, Match: l.Match, Only: append([]string(nil), l.Only...), Action: "allow",
			Reason: "disposable leftover declared by " + l.Owner + ": " + l.Rationale,
			Source: "disposable-leftovers/" + l.ID, Owner: l.Owner, priority: priorityStorageRoot + 1,
		})
	}
	return rules, nil
}
