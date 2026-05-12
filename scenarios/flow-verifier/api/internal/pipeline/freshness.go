// freshness.go owns the typed substrate the verifications recorder uses
// to distinguish "missing artifacts" from "stale artifacts" from other
// pipeline failures. AssertFresh returns a *FreshnessError when an
// artifact is absent or its on-disk content drifts from the canonical
// rendering; callers up the chain (recordPerFlow) unwrap to populate
// RunEntry.FailureReason and MissingArtifacts on the persisted run.
package pipeline

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// FreshnessKind narrows the failure into the two states the UI's
// "Needs generate" affordance recovers from. Other pipeline failures
// (counterexample, lint, quint exec) are represented by ordinary errors.
type FreshnessKind string

const (
	FreshnessMissing FreshnessKind = "missing"
	FreshnessStale   FreshnessKind = "stale"
)

// FreshnessError is the typed result of AssertFresh. Missing and Stale
// hold the relative artifact paths (e.g. "api/.../generated/runtime.go")
// so the recorder can persist a structured list, not a parsed string.
type FreshnessError struct {
	FlowID  string
	Kind    FreshnessKind
	Missing []string
	Stale   []string
}

// Paths returns the union of Missing+Stale, sorted, with duplicates
// removed. Convenience for the recorder which stores both lists merged.
func (e *FreshnessError) Paths() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(e.Missing)+len(e.Stale))
	for _, p := range e.Missing {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, p := range e.Stale {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func (e *FreshnessError) Error() string {
	parts := []string{}
	if len(e.Missing) > 0 {
		parts = append(parts, fmt.Sprintf("missing: %s", strings.Join(e.Missing, ", ")))
	}
	if len(e.Stale) > 0 {
		parts = append(parts, fmt.Sprintf("stale: %s", strings.Join(e.Stale, ", ")))
	}
	hint := ""
	if e.FlowID != "" {
		hint = fmt.Sprintf(". Run: flow-verifier verify run --flow %s", e.FlowID)
	}
	return fmt.Sprintf("generated artifacts not fresh (%s)%s", strings.Join(parts, "; "), hint)
}

// AsFreshnessError unwraps the chain looking for a *FreshnessError. It
// exists so callers (recordPerFlow, handlers) can keep error wrapping
// idiomatic while still inspecting the typed payload.
func AsFreshnessError(err error) (*FreshnessError, bool) {
	var fe *FreshnessError
	if errors.As(err, &fe) {
		return fe, true
	}
	return nil, false
}

// normaliseArtifactPath strips the leading root from an absolute target
// path so the persisted list is reproducible across machines. Falls back
// to filepath.Base if root isn't a prefix.
func normaliseArtifactPath(root, target string) string {
	rel, err := filepath.Rel(root, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(filepath.Base(target))
	}
	return filepath.ToSlash(rel)
}
