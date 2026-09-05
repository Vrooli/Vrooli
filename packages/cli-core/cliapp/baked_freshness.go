package cliapp

import "strings"

// BakedFreshnessInputs is overwritten at link time by cli-installer with the
// exact freshness-input list the binary was built against. It is a single
// string with comma (",") separators — no current resource/scenario
// freshness input contains a comma (they are path globs and filenames),
// and avoiding whitespace keeps the value transparent through the Go
// linker's -ldflags parsing.
//
// Why this exists: the install-time fingerprint and the run-time stale
// check must walk the same file set. Resources/scenarios that customize
// `cli.freshness.inputs` in their manifest used to silently disagree —
// the installer read the manifest list, the runtime hardcoded a default
// — so the BuildFingerprint baked into the binary never matched what the
// runtime computed, and every invocation triggered a rebuild loop.
//
// By baking the inputs the installer used into the same binary, we
// guarantee that NewResourceApp / NewScenarioApp default to the very
// list the fingerprint was built from. Per-call `opts.FreshnessInputs`
// still wins when explicitly set; the bake is only consulted when the
// caller didn't override.
//
// Empty (the linker did not write a value, e.g. plain `go build` for
// development) means "fall back to the package's hardcoded defaults",
// preserving the original behavior for non-installer builds.
var BakedFreshnessInputs string

// resolveFreshnessInputs returns the canonical freshness-input list for
// a CLI in priority order:
//  1. caller-provided opts (always wins),
//  2. linker-baked BakedFreshnessInputs (set by cli-installer),
//  3. the package's hardcoded fallback.
func resolveFreshnessInputs(callerInputs, fallback []string) []string {
	if len(callerInputs) > 0 {
		return append([]string(nil), callerInputs...)
	}
	if baked := parseBakedFreshnessInputs(BakedFreshnessInputs); len(baked) > 0 {
		return baked
	}
	return append([]string(nil), fallback...)
}

// parseBakedFreshnessInputs splits the linker-baked string into trimmed,
// non-empty entries. Comma-separated; blank entries are dropped so a
// stray trailing "," never produces an empty glob match.
func parseBakedFreshnessInputs(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
