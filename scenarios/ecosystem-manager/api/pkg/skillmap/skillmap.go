// Package skillmap is the controller's skill → dimension capability map. It
// reads each steer skill's declared target dimensions (sourced from
// prompt-manager's catalog) and builds the dimension → [eligible skills] index
// that selection consults: given the heaviest open dimension and a profile's
// allowed-skill set, which skills can close it?
//
// The vocabulary itself is owned by the shared maturity-go/dimensions package;
// this package validates
// declarations against it (out-of-vocabulary dimensions are dropped and warned,
// skills left with no valid dimension are excluded and warned).
package skillmap

import (
	"log"
	"sort"

	"github.com/vrooli/maturity-go/dimensions"
)

// SkillDeclaration is one skill's declared dimension coverage, as read from the
// catalog.
type SkillDeclaration struct {
	ID         string
	Dimensions []string
}

// CatalogSource is the controller's read boundary to the skill catalog.
//
// seam: CatalogSource supplies skill dimension declarations. Production wires a
// prompt-manager-backed source (autosteer.promptLoaderCatalog); tests wire
// FakeCatalog.
type CatalogSource interface {
	Skills() []SkillDeclaration
}

// Resolver is an immutable snapshot of the capability map.
type Resolver struct {
	// index maps a dimension to the skill IDs that target it (sorted).
	index map[dimensions.Dimension][]string
	// dimsBySkill maps a skill ID to its valid declared dimensions.
	dimsBySkill map[string][]dimensions.Dimension
	// excluded lists skill IDs that declared no in-vocabulary dimension.
	excluded []string
}

// Warner receives diagnostic warnings during resolution. Defaults to the
// standard logger; injectable for tests.
type Warner func(format string, args ...any)

// NewResolver builds a Resolver from a catalog source, validating declarations
// against the dimension vocabulary. Warnings about dropped dimensions and
// excluded skills are emitted via the standard logger.
func NewResolver(src CatalogSource) *Resolver {
	return NewResolverWithWarner(src, func(f string, a ...any) { log.Printf(f, a...) })
}

// NewResolverWithWarner is NewResolver with an injectable warning sink.
func NewResolverWithWarner(src CatalogSource, warn Warner) *Resolver {
	r := &Resolver{
		index:       make(map[dimensions.Dimension][]string),
		dimsBySkill: make(map[string][]dimensions.Dimension),
	}
	if warn == nil {
		warn = func(string, ...any) {}
	}

	var decls []SkillDeclaration
	if src != nil {
		decls = src.Skills()
	}
	for _, d := range decls {
		valid := make([]dimensions.Dimension, 0, len(d.Dimensions))
		seen := make(map[dimensions.Dimension]struct{})
		for _, raw := range d.Dimensions {
			dim := dimensions.Dimension(raw)
			if !dimensions.IsValid(dim) {
				warn("skillmap: skill %q declares unknown dimension %q; dropping", d.ID, raw)
				continue
			}
			if _, dup := seen[dim]; dup {
				continue
			}
			seen[dim] = struct{}{}
			valid = append(valid, dim)
		}
		if len(valid) == 0 {
			warn("skillmap: skill %q declares no valid dimensions; excluded from selection", d.ID)
			r.excluded = append(r.excluded, d.ID)
			continue
		}
		r.dimsBySkill[d.ID] = valid
		for _, dim := range valid {
			r.index[dim] = append(r.index[dim], d.ID)
		}
	}

	for dim := range r.index {
		sort.Strings(r.index[dim])
	}
	sort.Strings(r.excluded)
	return r
}

// SkillsForDimension returns all skill IDs that target a dimension, sorted.
func (r *Resolver) SkillsForDimension(dim dimensions.Dimension) []string {
	return append([]string(nil), r.index[dim]...)
}

// EligibleSkills returns the skills that target dim AND are in the allow-set,
// in deterministic (allow-set) order. An empty or nil allow-set means "no
// restriction" — all skills targeting the dimension are eligible.
func (r *Resolver) EligibleSkills(dim dimensions.Dimension, allow []string) []string {
	targeting := r.index[dim]
	if len(targeting) == 0 {
		return nil
	}
	targetingSet := make(map[string]struct{}, len(targeting))
	for _, id := range targeting {
		targetingSet[id] = struct{}{}
	}

	if len(allow) == 0 {
		return append([]string(nil), targeting...)
	}

	out := make([]string, 0, len(allow))
	emitted := make(map[string]struct{})
	for _, id := range allow {
		if _, ok := targetingSet[id]; !ok {
			continue
		}
		if _, dup := emitted[id]; dup {
			continue
		}
		emitted[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// DimensionsForSkill returns a skill's validated declared dimensions.
func (r *Resolver) DimensionsForSkill(skillID string) []dimensions.Dimension {
	return append([]dimensions.Dimension(nil), r.dimsBySkill[skillID]...)
}

// AllSkills returns every selectable skill ID in deterministic order.
func (r *Resolver) AllSkills() []string {
	out := make([]string, 0, len(r.dimsBySkill))
	for id := range r.dimsBySkill {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Excluded returns the skill IDs dropped for declaring no valid dimension.
func (r *Resolver) Excluded() []string {
	return append([]string(nil), r.excluded...)
}

// FakeCatalog is the test double for CatalogSource.
type FakeCatalog struct {
	Declarations []SkillDeclaration
}

var _ CatalogSource = (*FakeCatalog)(nil)

func (f *FakeCatalog) Skills() []SkillDeclaration { return f.Declarations }
