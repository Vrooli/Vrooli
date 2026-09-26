// Package spacedoc parses a coverage-space denominator markdown document into
// the cross-scenario `space-definition/v1` JSON contract.
//
// Three scenarios own a projection denominator, each authored as a markdown
// "space doc" with a "## This Space" metadata table and a "## Coverage Grid"
// section of pipe tables:
//
//   - search-hub   docs/spaces/answer-space.md    (Answer projection)
//   - test-genie   docs/spaces/validate-space.md  (Validate projection)
//   - prompt-manager docs/spaces/guide-space.md   (Guide projection)
//
// The three docs use different column headers and section shapes, but share a
// common conceptual model (cells with an id, a question, an owner, a status,
// and qualitative notes). This package normalizes all three into a single
// SpaceDefinition that serializes to .vrooli/schemas/space-definition.schema.json.
//
// It parses the *denominator* (the curated intended space) only — it never
// computes or carries the numerator (live coverage). meta-optimization-manager
// joins the numerator live; this package is the read contract both the owner's
// `space` verb and the aggregator's space-reader client share.
//
// The parser is intentionally tolerant of authoring drift (header order,
// qualifier suffixes like "IN-REACH (gap stub)", combined statuses like
// "NOW (UI) / IN-REACH (API)", category-only rows) because the markdown doc is
// the human-authored source of truth; the verb's job is a faithful projection,
// not validation. Base-document integrity (stale refs, etc.) is a separate
// concern owned by meta-optimization-manager's ValidateBaseDocs.
package spacedoc
