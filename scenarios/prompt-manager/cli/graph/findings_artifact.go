// Findings artifact: stable on-disk JSON shape produced by
// `prompt-manager graph topics --findings-out=<path>`. Consumed by CI for
// diff-against-previous-run telemetry on three-pillar topic validation.
//
// CONTRACT (do not break):
//
//   - Field set is additive. New fields may be appended at the end of the
//     struct without bumping schema_version. Renaming or removing a field
//     requires a bump and a coordinated consumer migration.
//   - findings[] is always present (never null) so a clean run produces an
//     empty array, not a null. CI diff scripts can rely on this.
//   - Findings are sorted by the API's validation pipeline (rule, member,
//     prefix); the artifact preserves that order.
//   - generated_at is RFC 3339 UTC. Time is injected at the call site so
//     tests can pin a deterministic value.
//   - team_filter mirrors the --team flag (empty when not set). Diff
//     scripts that compare two artifacts must check team_filter matches
//     before treating findings sets as comparable.
//   - File writes are atomic (temp file + rename). A killed process
//     leaves either the previous file or no file — never a half-written
//     file CI would mis-parse.
//
// DOC: docs/agent-system/RUNTIME_ATTRIBUTION.md (telemetry artifact).
// DOC: docs/agent-system/PRIMITIVES.md ("Three Pillars" section).
package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// findingsArtifactSchemaVersion is the on-disk shape version. Bump only
// when removing or renaming a field; additive changes do not require a
// bump.
const findingsArtifactSchemaVersion = 1

// findingsArtifact is the stable JSON shape written to --findings-out.
//
// See package-level CONTRACT comment for the additive-fields rule.
type findingsArtifact struct {
	SchemaVersion int            `json:"schema_version"`
	GeneratedAt   string         `json:"generated_at"`
	TeamFilter    string         `json:"team_filter"`
	Errors        int            `json:"errors"`
	Warnings      int            `json:"warnings"`
	Findings      []topicFinding `json:"findings"`
}

// buildFindingsArtifact materializes the validation result + context into
// the stable on-disk shape. Pure: no I/O, no globals, no time.Now() —
// caller injects now so tests can pin it.
//
// Findings is normalized to a non-nil slice so the marshalled JSON has
// "findings": [] rather than "findings": null on a clean run.
func buildFindingsArtifact(resp topicsGraphResponse, team string, now time.Time) findingsArtifact {
	findings := resp.Validation.Findings
	if findings == nil {
		findings = []topicFinding{}
	}
	return findingsArtifact{
		SchemaVersion: findingsArtifactSchemaVersion,
		GeneratedAt:   now.UTC().Format(time.RFC3339),
		TeamFilter:    team,
		Errors:        resp.Validation.Errors,
		Warnings:      resp.Validation.Warnings,
		Findings:      findings,
	}
}

// writeFindingsArtifact materializes a topics-graph response into the
// stable on-disk shape and writes it atomically to path.
//
// Empty path is a documented no-op so the caller can pass the flag
// value verbatim without branching on its emptiness.
//
// Atomicity: writes go to a sibling temp file (CreateTemp in the
// destination's directory) and are renamed into place. A signal between
// Marshal and Rename leaves either no temp file (Marshal failure) or
// only the temp file (Rename failure); the destination path is never
// observed in a half-written state. Callers may surface a non-fatal
// warning on error — the artifact is telemetry, not the primary
// validation result.
//
// path is interpreted relative to CWD when not absolute. The destination
// directory must already exist; this function does not mkdir, so a
// missing intermediate dir is a hard error rather than silent
// auto-creation (silent mkdir would mask a misspelt CI path).
func writeFindingsArtifact(path string, resp topicsGraphResponse, team string, now time.Time) error {
	if path == "" {
		return nil
	}

	art := buildFindingsArtifact(resp, team, now)
	raw, err := json.MarshalIndent(art, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal findings artifact: %w", err)
	}
	// Trailing newline matches the convention used elsewhere in the
	// store (team-corpus entries, taxonomies, topics.json) so the file
	// composes cleanly with line-oriented tools (diff, grep, etc.).
	raw = append(raw, '\n')

	dir := filepath.Dir(path)
	if dir == "" {
		dir = "."
	}
	tmp, err := os.CreateTemp(dir, ".findings-*.json")
	if err != nil {
		return fmt.Errorf("create temp file in %q: %w", dir, err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write findings artifact: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close findings artifact temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename findings artifact to %q: %w", path, err)
	}
	return nil
}
