package config

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// docPath is the canonical reference doc that the meta-test asserts
// stays in lockstep with the env vars referenced in config.go.
//
// The doc's location is relative to the api/ directory (where Go runs
// `go test`); we walk up to the scenario root then back down. This
// keeps the test working regardless of which package directory the
// test invocation starts from.
const docPath = "../../../docs/reference/configuration.md"

// envVarRe matches the canonical workspace-sandbox env var prefix
// plus the four well-known external env vars the API consumes.
//
// Why these: the parity check is about *operator-tunable knobs* —
// not about every transient env var the API ever reads (e.g.,
// VROOLI_ROOT during root discovery or $HOME for path validation). Those
// are service-runtime inputs rather than workspace-sandbox storage knobs;
// duplicating them here would invite drift without operator value.
var envVarRe = regexp.MustCompile(`\b(WORKSPACE_SANDBOX_[A-Z0-9_]+|API_PORT|PROJECT_ROOT)\b`)

// readFile is a convenience wrapper that fails the test on read
// errors instead of forcing every caller to handle the error.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// extractEnvVars pulls the de-duplicated set of operator-knob env vars
// from a chunk of source text.
func extractEnvVars(text string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, m := range envVarRe.FindAllString(text, -1) {
		out[m] = struct{}{}
	}
	return out
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestExposedKnobs_DocumentationParity asserts that every env var
// referenced as a string literal in config.go has a documented
// section in docs/reference/configuration.md, and vice versa. A new
// knob added to config.go without a doc entry (or removed from
// config.go without removing its doc) fails this test.
//
// Round 4 Phase 8 (2026-04-29): introduced as the loud-failure guard
// against doc drift.
func TestExposedKnobs_DocumentationParity(t *testing.T) {
	configSrc := readFile(t, "config.go")
	docSrc := readFile(t, filepath.Clean(docPath))

	// Strip the doc's "convention" prose from the env-var literal
	// extraction — it references types like `time.ParseDuration`
	// which (correctly) do not appear in the env-var set.

	configVars := extractEnvVars(configSrc)
	docVars := extractEnvVars(docSrc)

	// Drop env vars that LoadFromEnv consumes only as keys for the
	// internal envInt/envBool/envDuration test helpers. None of these
	// pollute the canonical regex (it only matches the known
	// prefixes), so configVars is already clean.

	// Calculate set differences in both directions.
	missingFromDoc := map[string]struct{}{}
	for k := range configVars {
		if _, ok := docVars[k]; !ok {
			missingFromDoc[k] = struct{}{}
		}
	}
	missingFromConfig := map[string]struct{}{}
	for k := range docVars {
		if _, ok := configVars[k]; !ok {
			missingFromConfig[k] = struct{}{}
		}
	}

	if len(missingFromDoc) > 0 {
		t.Errorf("env vars referenced in config.go but missing from %s:\n  %s",
			docPath, strings.Join(sortedKeys(missingFromDoc), "\n  "))
	}
	if len(missingFromConfig) > 0 {
		t.Errorf("env vars documented in %s but not referenced in config.go:\n  %s",
			docPath, strings.Join(sortedKeys(missingFromConfig), "\n  "))
	}
}
