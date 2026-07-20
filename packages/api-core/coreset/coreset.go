// Package coreset is the single source of truth for Vrooli's reflexive "core
// set" seed and the irreducible "trusted base" subset that the Baseline Modes
// self-improvement machinery depends on.
//
// The live core set is computed by scenario-dependency-analyzer as
// seed ∪ transitive Required-closure (fresh from disk). This package owns only
// the two hardcoded authorities that must never drift across the analyzer and
// every consumer (the Baseline Modes decision tree, P2):
//
//   - the always-included seed (CoreSeedScenarios), and
//   - the trusted-base subset (TrustedBaseScenarios) that can never be shadowed.
//
// It also exposes DefaultFallbackCoreSet, the safe over-inclusive answer a
// consumer uses when the analyzer is unreachable (e.g. postgres down): over-
// inclusion is safe (more scenarios get careful live-mode treatment), under-
// inclusion is dangerous (a core scenario gets shadow-broken).
//
// This package depends on the standard library only, so any layer — platform
// internal/, cli-core, api-core, or a scenario — can import it cheaply without
// pulling transitive weight.
package coreset

import (
	"sort"
	"strings"
)

// coreSeedScenarios is the always-included authority for the reflexive set: the
// scenarios the self-improvement loop depends on to run, validate, and now to
// safely promote. The computed core set is this seed UNIONED with the fresh
// transitive Required closure, so unreliable `required` flags in service.json
// (e.g. git-control-tower declares all its scenario deps required=false) can
// only ever ADD members, never drop one.
//
// Vrooli Events belongs here because its shared receipt/policy substrate is
// consumed across scenarios and must be available to preserve the platform's
// attributable event trail. It is core, but not trusted-base: Baseline Modes
// can still validate a shadow Vrooli Events instance.
//
// Kept private and exposed only through accessors that return copies, so a
// consumer can never mutate the shared authority.
var coreSeedScenarios = []string{
	"agent-manager",
	"data-backup-manager",
	"git-control-tower",
	"prompt-manager",
	"scenario-dependency-analyzer",
	"swarm-manager",
	"test-genie",
	"vrooli-events",
	"workspace-sandbox",
}

// trustedBaseScenarios is the irreducible subset of the seed that the Baseline
// Modes machinery itself runs ON during an engagement, and therefore can never
// be validated in a shadow by that same machinery — they are always developed
// in live mode (the decision tree hard-routes them; self-validation passes
// --instance live):
//
//   - git-control-tower owns the `baseline` engagement verbs; a shadow GCT
//     cannot promote itself, and the recovery floor must be able to recover a
//     broken GCT.
//   - test-genie is the validator; a shadow test-genie validating itself is not
//     trustworthy, so its own `baseline check` routes to the live instance.
//   - data-backup-manager is the documented trusted-base member — it takes the
//     pre-promote snapshots and performs restores that promote/abandon rely on.
//
// The rest of the seed (the executor pair agent-manager/workspace-sandbox, the
// orchestrators swarm-manager/swarm-manager, prompt-manager, and the
// analyzer) are reflexive but DO get shadow mode — they are exactly the kernel
// scenarios Baseline Modes is built to make shadow-safe. Self-promote of those
// is handled by P6's drain + self-identity guard + external one-shot, not by
// this trusted-base gate. NOTE: P2's decision tree consumes this subset; if it
// needs to refine membership (e.g. hard-route the analyzer to live while
// postgres is its only data source), do it here so the authority stays single.
var trustedBaseScenarios = []string{
	"data-backup-manager",
	"git-control-tower",
	"test-genie",
}

// CoreSeedScenarios returns a fresh copy of the always-included core-set seed
// (sorted, lower-case). Callers may mutate the result freely.
func CoreSeedScenarios() []string {
	return cloneSorted(coreSeedScenarios)
}

// TrustedBaseScenarios returns a fresh copy of the trusted-base subset (the
// seed members that are never shadowed). Callers may mutate the result freely.
func TrustedBaseScenarios() []string {
	return cloneSorted(trustedBaseScenarios)
}

// DefaultFallbackCoreSet is the safe answer a consumer uses when the live core
// set cannot be computed (analyzer unreachable / postgres down / disk error):
// the 9-seed, which is always over-inclusive and therefore safe. It is
// identical to CoreSeedScenarios; the distinct name documents the intent at
// call sites that are handling a failure rather than reading the seed.
func DefaultFallbackCoreSet() []string {
	return cloneSorted(coreSeedScenarios)
}

// IsCoreSeed reports whether name is one of the hardcoded seed scenarios.
// Comparison is case-insensitive and whitespace-trimmed. This checks seed
// membership only; the full computed core set (seed ∪ closure) is owned by
// scenario-dependency-analyzer.
func IsCoreSeed(name string) bool {
	return contains(coreSeedScenarios, name)
}

// IsTrustedBase reports whether name is a trusted-base scenario that must never
// be shadowed. Comparison is case-insensitive and whitespace-trimmed.
func IsTrustedBase(name string) bool {
	return contains(trustedBaseScenarios, name)
}

// normalize lower-cases and trims a scenario name for consistent comparison.
func normalize(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

func contains(set []string, name string) bool {
	target := normalize(name)
	if target == "" {
		return false
	}
	for _, candidate := range set {
		if normalize(candidate) == target {
			return true
		}
	}
	return false
}

// cloneSorted returns a normalized, lexically sorted copy of the input. The
// seed slices are already canonical, so this is defensive: it guarantees every
// accessor returns the same stable, mutation-safe ordering.
func cloneSorted(set []string) []string {
	out := make([]string, len(set))
	for i, value := range set {
		out[i] = normalize(value)
	}
	sort.Strings(out)
	return out
}
