// Package staleness derives manifest-vs-current drift over the
// (manifest, golden, skill_catalog) state (OT-P0-007). Pure
// composition — owns no storage of its own. The manifest domain owns
// the manifest_stale_overrides table that suppresses drift after a
// manual clear.
package staleness

import "fmt"

// StaleKind classifies why a tuple is stale.
type StaleKind int

const (
	StaleKindUnspecified   StaleKind = 0
	StaleKindTemplateDrift StaleKind = 1
	StaleKindSkillDrift    StaleKind = 2
	StaleKindBoth          StaleKind = 3
)

// Entry is the domain shape for one stale tuple.
type Entry struct {
	SkillID    string
	GoldenSlug string
	Kind       StaleKind

	ManifestTemplateVersionPinned string
	ManifestSkillVersionPinned    string
	GoldenTemplateVersionCurrent  string
	SkillVersionCurrent           string
}

// ErrInvalidStaleness is the typed sentinel for input validation
// failures (currently unused, reserved for future ClearStale handlers
// that might land in this domain).
type ErrInvalidStaleness struct {
	Field  string
	Reason string
}

func (e ErrInvalidStaleness) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
