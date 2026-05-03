// Taxonomy loader: reads docs/<domain>/<id>.json sidecar files that own the
// signal taxonomy + dispatch + evidence rules + destination schemas for a
// domain. The heartbeat builder consumes these to render the generated Inbox
// Flow section; the validator consumes them to enforce
// `unknown_taxonomy` and `missing_destination_schema`.
//
// DOC: docs/agent-system/drafts/inbox-flow-refactor-plan.md §6.3
package memberflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Taxonomy is the parsed JSON sidecar that owns one domain's signal
// vocabulary, dispatch table, evidence rules, and destination schemas.
//
// One id per file; the file basename matches the id (e.g.
// `marketing-research` -> `signal-taxonomy.json` linked from the registry,
// resolved by id, not by basename — see TaxonomyRegistry below).
type Taxonomy struct {
	SchemaVersion int                       `json:"schemaVersion"`
	ID            string                    `json:"id"`
	DisplayName   string                    `json:"displayName"`
	OwnerTeam     string                    `json:"owner_team,omitempty"`
	PoRPath       string                    `json:"porPath,omitempty"`
	SignalTypes   []TaxonomySignalType      `json:"signalTypes,omitempty"`
	EvidenceRules []string                  `json:"evidenceRules,omitempty"`
	ActionSelect  map[string]string         `json:"actionSelection,omitempty"`
	Schemas       map[string]TaxonomySchema `json:"schemas,omitempty"`
	HonestyFlags  []string                  `json:"honestyFlags,omitempty"`
	// PendingMethodSkills lists default-method ids that the taxonomy
	// references but that do not yet exist in the skill registry.
	// taxonomy_authoring_test verifies that any signalType.defaultMethod
	// missing from the registry is listed here, so the test does not fail
	// while the method skill is still being authored.
	PendingMethodSkills []string `json:"pendingMethodSkills,omitempty"`

	// SourcePath is the absolute path the taxonomy was loaded from. Set by
	// LoadTaxonomy; ignored on marshal.
	SourcePath string `json:"-"`
}

// TaxonomySignalType is one row of the dispatch table.
type TaxonomySignalType struct {
	ID                       string `json:"id"`
	Definition               string `json:"definition,omitempty"`
	DefaultMethod            string `json:"defaultMethod,omitempty"`
	DefaultDestinationPrefix string `json:"defaultDestinationPrefix,omitempty"`
	EvidenceMinimum          string `json:"evidenceMinimum,omitempty"`
}

// TaxonomySchema is one destination front-matter shape declared by the
// taxonomy. The validator looks up `output[].schema` ids here.
type TaxonomySchema struct {
	FrontMatter          map[string]string `json:"frontMatter,omitempty"`
	BodyRequiredSections []string          `json:"bodyRequiredSections,omitempty"`
}

// TaxonomyRegistry indexes loaded taxonomies by id.
type TaxonomyRegistry map[string]*Taxonomy

// taxonomyGlobs are the filename patterns the loader treats as taxonomy
// sidecars. Any *.json file under docs/ matching one of these glob suffixes
// is parsed; non-taxonomy JSON files (e.g. fixture data) should not match.
var taxonomyGlobs = []string{
	"*-taxonomy.json",
	"signal-taxonomy.json",
	"opportunity-taxonomy.json",
	"validation-taxonomy.json",
}

// LoadTaxonomy reads a single taxonomy file by id from the docs registry
// rooted at repoRoot. Returns nil + error when the id cannot be resolved.
//
// Resolution: the loader walks docs/ looking for *.json files whose contents
// declare {"id": "<id>"}; first match wins. This is O(N) over a small set
// (<10) of taxonomies, acceptable for heartbeat-build and CLI use.
func LoadTaxonomy(repoRoot, id string) (*Taxonomy, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("taxonomy id is required")
	}
	registry, err := LoadAllTaxonomies(repoRoot)
	if err != nil {
		return nil, err
	}
	if tx, ok := registry[id]; ok {
		return tx, nil
	}
	return nil, fmt.Errorf("taxonomy %q not found under %s/docs", id, repoRoot)
}

// LoadAllTaxonomies walks docs/ under repoRoot and returns every parseable
// taxonomy sidecar, indexed by id.
//
// Returns an empty registry (no error) when docs/ is absent — callers that
// need taxonomies should treat empty as "skip the check," matching the
// LoadSkillIDs convention.
func LoadAllTaxonomies(repoRoot string) (TaxonomyRegistry, error) {
	out := make(TaxonomyRegistry)
	if strings.TrimSpace(repoRoot) == "" {
		return out, nil
	}
	docsDir := filepath.Join(repoRoot, "docs")
	if _, err := os.Stat(docsDir); err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, fmt.Errorf("memberflow: stat docs %q: %w", docsDir, err)
	}

	walkErr := filepath.Walk(docsDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if !matchesTaxonomyName(base) {
			return nil
		}
		tx, err := parseTaxonomyFile(path)
		if err != nil {
			return fmt.Errorf("taxonomy %q: %w", path, err)
		}
		if tx == nil {
			return nil
		}
		if existing, ok := out[tx.ID]; ok {
			return fmt.Errorf("duplicate taxonomy id %q in %q and %q", tx.ID, existing.SourcePath, tx.SourcePath)
		}
		out[tx.ID] = tx
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return out, nil
}

func matchesTaxonomyName(base string) bool {
	for _, g := range taxonomyGlobs {
		if matched, _ := filepath.Match(g, base); matched {
			return true
		}
	}
	return false
}

func parseTaxonomyFile(path string) (*Taxonomy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var tx Taxonomy
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if strings.TrimSpace(tx.ID) == "" {
		return nil, fmt.Errorf("missing id field")
	}
	tx.SourcePath = path
	return &tx, nil
}

// SignalTypeIDs returns the taxonomy's signal-type ids in declaration order.
func (t *Taxonomy) SignalTypeIDs() []string {
	if t == nil {
		return nil
	}
	out := make([]string, 0, len(t.SignalTypes))
	for _, s := range t.SignalTypes {
		out = append(out, s.ID)
	}
	return out
}

// SchemaIDs returns the schema ids in lexical order.
func (t *Taxonomy) SchemaIDs() []string {
	if t == nil {
		return nil
	}
	ids := make([]string, 0, len(t.Schemas))
	for id := range t.Schemas {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// HasSchema reports whether the taxonomy declares the given schema id.
func (t *Taxonomy) HasSchema(id string) bool {
	if t == nil || strings.TrimSpace(id) == "" {
		return false
	}
	_, ok := t.Schemas[id]
	return ok
}

// IDs returns the registry's taxonomy ids in lexical order.
func (r TaxonomyRegistry) IDs() []string {
	ids := make([]string, 0, len(r))
	for id := range r {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// HasSchema reports whether *any* taxonomy in the registry declares the
// given schema id. Used by ruleMissingDestinationSchema.
func (r TaxonomyRegistry) HasSchema(id string) bool {
	if strings.TrimSpace(id) == "" {
		return false
	}
	for _, tx := range r {
		if tx.HasSchema(id) {
			return true
		}
	}
	return false
}
