package calibration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadCases validates fixture metadata and paths, not parser syntax: malformed
// source is an intentional calibration input. Missing/unreadable source must be
// modeled explicitly by an evaluator input seam, not a accidentally absent file.
func LoadCases(root, manifest, partition string) ([]Case, error) {
	data, err := os.ReadFile(filepath.Join(root, manifest))
	if err != nil {
		return nil, err
	}
	var cases []Case
	if err := json.Unmarshal(data, &cases); err != nil {
		var envelope struct {
			Cases []Case `json:"cases"`
		}
		if envelopeErr := json.Unmarshal(data, &envelope); envelopeErr != nil {
			return nil, err
		}
		cases = envelope.Cases
	}
	if err := ValidateCases(cases, partition); err != nil {
		return nil, err
	}
	for _, c := range cases {
		for _, file := range c.Input.Files {
			if !filepath.IsLocal(c.Input.Root) || !filepath.IsLocal(file) {
				return nil, fmt.Errorf("case %s: non-local source path", c.Input.ID)
			}
			path := filepath.Join(root, c.Input.Root, file)
			if _, err := os.ReadFile(path); err != nil {
				return nil, fmt.Errorf("case %s source: %w", c.Input.ID, err)
			}
		}
		for _, e := range c.Expected {
			found := false
			for _, f := range c.Input.Files {
				if f == e.File {
					found = true
				}
			}
			if !found {
				return nil, fmt.Errorf("case %s expectation names an unlisted file", c.Input.ID)
			}
		}
	}
	return cases, nil
}

type Specification struct {
	Status string `json:"status"`
	Cases  []struct {
		ID            string `json:"id"`
		Group         string `json:"group"`
		Name          string `json:"name"`
		Expected      string `json:"expected"`
		Rationale     string `json:"rationale"`
		RetiredReason string `json:"retired_reason,omitempty"`
	} `json:"cases"`
}

type InventoryEntry struct {
	ID          string `json:"id"`
	Group       string `json:"group"`
	Expected    string `json:"retainedExpectation"`
	Disposition string `json:"disposition"`
	Reason      string `json:"reason"`
}

// Inventory preserves every retained requirement, including cases not yet
// materialized. Pending fixtures are not silently counted as unknown passes.
func Inventory(root string, development []Case) ([]InventoryEntry, error) {
	data, err := os.ReadFile(filepath.Join(root, "case-specification.json"))
	if err != nil {
		return nil, err
	}
	var spec Specification
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, err
	}
	if len(spec.Cases) == 0 {
		return nil, fmt.Errorf("empty retained specification")
	}
	materialized := map[string]bool{}
	for _, c := range development {
		materialized[c.Input.ID] = true
	}
	native, err := LoadNativeSpecification(root)
	if err != nil {
		return nil, err
	}
	nativeIDs := map[string]bool{}
	for _, c := range native.Cases {
		nativeIDs[c.CaseID] = true
	}
	contextCases, err := LoadContextCases(root)
	if err != nil {
		return nil, err
	}
	contextIDs := map[string]string{}
	for _, c := range contextCases {
		contextIDs[c.ID] = c.ExpectedOutcome
	}
	seen := map[string]bool{}
	out := []InventoryEntry{}
	for _, c := range spec.Cases {
		if c.ID == "" || seen[c.ID] || strings.TrimSpace(c.Expected) == "" || c.Rationale == "" {
			return nil, fmt.Errorf("invalid retained case %q", c.ID)
		}
		seen[c.ID] = true
		row := InventoryEntry{ID: c.ID, Group: c.Group, Expected: c.Expected, Disposition: "pending-fixture", Reason: "Required case remains unfinished; not counted as calibration success."}
		if materialized[c.ID] {
			row.Disposition = "development-fixture"
			row.Reason = "Expectations authored; adapter execution and calibration proof remain required."
		}
		if nativeIDs[c.ID] {
			row.Disposition = "native-fixture"
			if materialized[c.ID] {
				row.Disposition = "development-and-native-fixture"
			}
			row.Reason = "Native fixture materialized; version-specific observations must be compared independently."
		}
		if expected, exists := contextIDs[c.ID]; exists {
			if expected != c.Expected {
				return nil, fmt.Errorf("context fixture %s changed retained expected outcome", c.ID)
			}
			row.Disposition = "context-fixture"
			row.Reason = "Structured input and expected outcome authored; owner evaluation and behavioral evidence remain required."
		}
		out = append(out, row)
	}
	for id := range materialized {
		if !seen[id] {
			return nil, fmt.Errorf("fixture %s has no retained specification", id)
		}
	}
	for id := range nativeIDs {
		if !seen[id] {
			return nil, fmt.Errorf("native case %s has no retained specification", id)
		}
	}
	for id := range contextIDs {
		if !seen[id] {
			return nil, fmt.Errorf("context fixture %s has no retained specification", id)
		}
	}
	return out, nil
}
