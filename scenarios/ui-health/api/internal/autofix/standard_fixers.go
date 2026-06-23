package autofix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Net-new UI-standard rule codes ui-health can auto-remediate. Only the *safe*,
// fully-mechanical subset is autofixed; the remaining standards
// (standard_no_raw_hex, standard_a11y_harness, standard_pwa_manifest,
// standard_eslint_stability) are detection-only by nature (a hex→token mapping,
// authoring a test, or rewriting hand-tuned config is not a safe mechanical
// transform). Keep this list in lockstep with FixClassFor and the maturity.json
// declarations — the ConsistencyWarnings check enforces it.
const (
	// RuleStandardTSConfigStrict flips an explicit "strict": false to true in
	// ui/tsconfig.json. Pure token substitution, format-preserving, idempotent.
	// When the flag is absent entirely the fixer makes no change (inserting into
	// JSONC with comments is not a safe mechanical edit), so AutofixAvailable is
	// false for that finding and it stays a report-only detection.
	RuleStandardTSConfigStrict = "standard_tsconfig_strict"
	// RuleStandardI18nLocaleParity scaffolds keys present in en.json but missing
	// from a sibling locale catalog, copying the English value as a translation
	// placeholder. Additive and idempotent; orphan (extra) keys are left for a
	// human (deleting a translation is not a safe mechanical edit).
	RuleStandardI18nLocaleParity = "standard_i18n_locale_parity"
)

// strictFalseRewrite matches "strict": false (any whitespace) for the flip.
var strictFalseRewrite = regexp.MustCompile(`"strict"\s*:\s*false`)

// standardRewriteFixers returns the net-new-standard fixers that are pure
// whole-file content rewrites (driven off the same RunAll authority that detects
// the violation, via the shared previewInterop/canFixInterop machinery).
func (f *Fixer) standardRewriteFixers() []interopFixerSpec {
	return []interopFixerSpec{
		{ruleID: RuleStandardTSConfigStrict, rewrite: rewriteTSConfigStrict},
	}
}

// rewriteTSConfigStrict flips an explicit "strict": false to true. It returns
// changed=false when there is no explicit false to flip (a missing flag is not
// safely insertable), so the fix is a no-op rather than a risky edit.
func rewriteTSConfigStrict(content string) (string, bool) {
	if !strictFalseRewrite.MatchString(content) {
		return content, false
	}
	out := strictFalseRewrite.ReplaceAllString(content, `"strict": true`)
	return out, out != content
}

// previewI18nLocaleParity scaffolds missing locale keys. It reads the reference
// en.json, then for each sibling locale catalog adds any *non-plural* English
// leaf whose base key is absent from the locale, copying the English value as a
// translation placeholder. Driving off the on-disk catalogs (the same authority
// the rule reads) makes the fix idempotent: once every base key is present the
// merge produces no change and the next preview yields nothing.
//
// Plural base keys (i18next CLDR variants like items_one / items_other) are
// deliberately NOT synthesized — the correct plural categories depend on the
// target language's CLDR rules, which is a translation-judgment call, not a safe
// mechanical edit. A locale missing an entire plural concept is still flagged by
// the rule but left for a human (CanFix reports false for it).
func (f *Fixer) previewI18nLocaleParity(root string) ([]Candidate, error) {
	localesDir := filepath.Join(root, filepath.FromSlash("ui/src/i18n/locales"))
	enDoc, ok := readLocaleDoc(filepath.Join(localesDir, "en.json"))
	if !ok {
		return nil, nil
	}
	enLeaves := flattenLeaves("", enDoc)
	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return nil, nil
	}
	var out []Candidate
	for _, e := range entries {
		if e.IsDir() || e.Name() == "en.json" || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		abs := filepath.Join(localesDir, e.Name())
		data, readErr := os.ReadFile(abs)
		if readErr != nil {
			continue
		}
		var doc map[string]any
		if json.Unmarshal(data, &doc) != nil {
			continue
		}
		targetBases := localeBaseKeys(doc)
		merged, changed := deepCopyDoc(doc)
		if !changed {
			continue // unmarshalable copy; skip rather than risk a bad write
		}
		added := false
		for _, leaf := range enLeaves {
			if isPluralKey(leaf.path) {
				continue // do not synthesize plural variants
			}
			if _, covered := targetBases[stripPluralSuffix(leaf.path)]; covered {
				continue
			}
			setNested(merged, leaf.path, leaf.value)
			added = true
		}
		if !added {
			continue
		}
		after, marshalErr := marshalLocale(merged)
		if marshalErr != nil {
			continue
		}
		out = append(out, Candidate{
			RuleID:      RuleStandardI18nLocaleParity,
			FilePath:    abs,
			Description: "Scaffold missing locale keys in " + e.Name() + " from en.json (English value as a translation placeholder).",
			Before:      string(data),
			After:       after,
		})
	}
	return out, nil
}

// canFixI18nLocaleParity scopes the preview to a single finding path (the locale
// catalog the finding points at), so AutofixAvailable never claims a no-op (e.g.
// a locale whose only drift is orphan keys, which this fixer does not touch).
func (f *Fixer) canFixI18nLocaleParity(root, findingPath string) bool {
	candidates, err := f.previewI18nLocaleParity(root)
	if err != nil || len(candidates) == 0 {
		return false
	}
	if findingPath == "" {
		return true
	}
	for _, c := range candidates {
		if c.FilePath == findingPath {
			return true
		}
	}
	return false
}

// readLocaleDoc parses a JSON catalog into a generic map. ok is false when the
// file is absent or not valid JSON.
func readLocaleDoc(path string) (map[string]any, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var doc map[string]any
	if json.Unmarshal(data, &doc) != nil {
		return nil, false
	}
	return doc, true
}

// localeLeaf is a single non-object value in a catalog at a dotted path.
type localeLeaf struct {
	path  string
	value any
}

// flattenLeaves returns every non-object leaf (dotted path + value) in a parsed
// catalog. Object nodes are recursed into; scalars, arrays, and null are leaves.
func flattenLeaves(prefix string, node map[string]any) []localeLeaf {
	var out []localeLeaf
	for k, v := range node {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if child, ok := v.(map[string]any); ok {
			out = append(out, flattenLeaves(key, child)...)
			continue
		}
		out = append(out, localeLeaf{path: key, value: v})
	}
	return out
}

// localeBaseKeys returns the set of base keys (plural-suffix-stripped) present in
// a catalog, including intermediate object paths, so a missing-key check can ask
// "does this locale already carry this concept?".
func localeBaseKeys(doc map[string]any) map[string]struct{} {
	out := map[string]struct{}{}
	collectBaseKeys("", doc, out)
	return out
}

func collectBaseKeys(prefix string, node map[string]any, out map[string]struct{}) {
	for k, v := range node {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		out[stripPluralSuffix(key)] = struct{}{}
		if child, ok := v.(map[string]any); ok {
			collectBaseKeys(key, child, out)
		}
	}
}

// setNested writes value at the dotted path within doc, creating intermediate
// object maps as needed. An existing non-object node along the path is replaced
// with an object (the path the en reference dictates wins for a missing key).
func setNested(doc map[string]any, dottedPath string, value any) {
	parts := strings.Split(dottedPath, ".")
	cur := doc
	for i := 0; i < len(parts)-1; i++ {
		child, ok := cur[parts[i]].(map[string]any)
		if !ok {
			child = map[string]any{}
			cur[parts[i]] = child
		}
		cur = child
	}
	cur[parts[len(parts)-1]] = value
}

// deepCopyDoc clones a parsed catalog via a JSON round-trip so edits never mutate
// the caller's map. ok is false only if the (already-parsed) doc fails to
// re-marshal, which should not happen for JSON-derived values.
func deepCopyDoc(doc map[string]any) (map[string]any, bool) {
	b, err := json.Marshal(doc)
	if err != nil {
		return nil, false
	}
	var out map[string]any
	if json.Unmarshal(b, &out) != nil {
		return nil, false
	}
	return out, true
}

// cldrPluralSuffixes are the i18next plural-form suffixes (CLDR plural
// categories). Mirror of the rule-side list (checks package) — kept local to
// avoid the autofix package depending on the rules package.
var cldrPluralSuffixes = []string{"_zero", "_one", "_two", "_few", "_many", "_other"}

// stripPluralSuffix removes a trailing CLDR plural suffix from the final segment
// of a dotted key (greeting_one → greeting). Keys without one are unchanged.
func stripPluralSuffix(key string) string {
	prefix, last := "", key
	if idx := strings.LastIndex(key, "."); idx >= 0 {
		prefix, last = key[:idx+1], key[idx+1:]
	}
	for _, suf := range cldrPluralSuffixes {
		if len(last) > len(suf) && strings.HasSuffix(last, suf) {
			return prefix + strings.TrimSuffix(last, suf)
		}
	}
	return key
}

// isPluralKey reports whether a dotted key carries a CLDR plural suffix.
func isPluralKey(key string) bool { return stripPluralSuffix(key) != key }

// marshalLocale renders a catalog with 2-space indentation and a trailing
// newline (the repo's JSON catalog convention). Go marshals map keys sorted, so
// the output is deterministic — idempotent across repeated applies.
func marshalLocale(doc map[string]any) (string, error) {
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}
