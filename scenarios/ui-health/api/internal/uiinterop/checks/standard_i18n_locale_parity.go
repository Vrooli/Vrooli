/*
Rule: i18n Locale Parity
ID: standard_i18n_locale_parity
Description: When a UI ships locale catalogs under ui/src/i18n/locales, every
  non-English catalog must define exactly the same key set as en.json. Missing
  keys fall back to English at runtime (untranslated strings); extra keys are
  dead translations that drift out of sync.
Why: The reference catalog (en.json) is the source of truth for the string
  registry. A locale missing a key renders English in an otherwise-translated
  UI (a visible inconsistency); a locale with an orphan key signals a removed
  string the translator never cleaned up. Parity keeps every shipped locale a
  complete, drift-free translation of the same surface.
Category: i18n
Severity: medium
Slot: [D]
SlotFile: ui/src/i18n/locales
TechStack: React
Recommendation: Add the missing keys to each lagging locale (copy the English
  value as a placeholder for translation) and remove orphan keys. The
  deterministic fixer scaffolds missing keys for you.
Standard: vrooli-ui-i18n-v1

GoodExample:
    en.json: {"greeting":"Hello","bye":"Goodbye"}
    ja.json: {"greeting":"こんにちは","bye":"さようなら"}

BadExample:
    en.json: {"greeting":"Hello","bye":"Goodbye"}
    ja.json: {"greeting":"こんにちは"}   // missing "bye"

<test-case id="i18n-parity-ok" should-fail="false">
  <description>All locale catalogs share the same key set</description>
  <input>
    [ui/src/i18n/locales/en.json]
    { "greeting": "Hello", "nav": { "home": "Home" } }
    [ui/src/i18n/locales/ja.json]
    { "greeting": "こんにちは", "nav": { "home": "ホーム" } }
  </input>
</test-case>

<test-case id="i18n-parity-plural-variants" should-fail="false">
  <description>CLDR plural variants differ by locale but the base keys match (no drift)</description>
  <input>
    [ui/src/i18n/locales/en.json]
    { "items_one": "1 item", "items_other": "{{count}} items" }
    [ui/src/i18n/locales/ja.json]
    { "items_other": "{{count}}個" }
  </input>
</test-case>

<test-case id="i18n-parity-no-locales" should-fail="false">
  <description>No locale catalogs present; nothing to check</description>
  <input>
    [ui/src/App.tsx]
    export function App() { return null; }
  </input>
</test-case>

<test-case id="i18n-parity-missing-key" should-fail="true">
  <description>ja.json is missing a key present in en.json</description>
  <input>
    [ui/src/i18n/locales/en.json]
    { "greeting": "Hello", "bye": "Goodbye" }
    [ui/src/i18n/locales/ja.json]
    { "greeting": "こんにちは" }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>missing</expected-message>
</test-case>
*/

package checks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("standard_i18n_locale_parity", checkI18nLocaleParity)
}

const i18nLocalesDir = "ui/src/i18n/locales"

// referenceLocale is the source-of-truth catalog every other locale mirrors.
const referenceLocale = "en.json"

func checkI18nLocaleParity(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_i18n_locale_parity"

	localesDir := filepath.Join(ctx.ScenarioRoot, filepath.FromSlash(i18nLocalesDir))
	refKeys, ok := readLocaleKeys(filepath.Join(localesDir, referenceLocale))
	if !ok {
		// No reference catalog ⇒ the scenario does not ship i18n catalogs here;
		// parity is not applicable.
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no ui/src/i18n/locales/en.json reference catalog",
			Message:    "no reference locale catalog; skipping i18n parity",
		}
	}

	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "locales directory unreadable",
			Message:    "i18n locales directory unreadable; skipping",
		}
	}

	var violations []uiinterop.Violation
	for _, e := range entries {
		if e.IsDir() || e.Name() == referenceLocale || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		rel := i18nLocalesDir + "/" + e.Name()
		keys, ok := readLocaleKeys(filepath.Join(localesDir, e.Name()))
		if !ok {
			violations = append(violations, uiinterop.Violation{
				RuleID:         ruleID,
				Severity:       "medium",
				Title:          "Unparseable locale catalog",
				Description:    rel + " could not be parsed as JSON",
				FilePath:       rel,
				Recommendation: "Fix the JSON syntax in " + rel,
			})
			continue
		}
		missing := keySetDiff(refKeys, keys) // in en, not in locale
		extra := keySetDiff(keys, refKeys)   // in locale, not in en
		if len(missing) == 0 && len(extra) == 0 {
			continue
		}
		violations = append(violations, uiinterop.Violation{
			RuleID:         ruleID,
			Severity:       "medium",
			Title:          "Locale catalog out of parity with en.json",
			Description:    localeParityDescription(rel, missing, extra),
			FilePath:       rel,
			Recommendation: "Add the missing keys (placeholder = the English value) and remove orphan keys so " + e.Name() + " matches en.json",
		})
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "one or more locale catalogs are out of parity with en.json",
			Violations: violations,
		}
	}
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "all locale catalogs share the reference key set",
	}
}

// localeParityDescription renders a stable, human-readable summary of the diff.
func localeParityDescription(rel string, missing, extra []string) string {
	var b strings.Builder
	b.WriteString(rel + " is out of parity with en.json")
	if len(missing) > 0 {
		b.WriteString(fmt.Sprintf("; missing %d key(s): %s", len(missing), strings.Join(missing, ", ")))
	}
	if len(extra) > 0 {
		b.WriteString(fmt.Sprintf("; %d orphan key(s): %s", len(extra), strings.Join(extra, ", ")))
	}
	return b.String()
}

// readLocaleKeys parses a JSON catalog and returns its flattened (dotted) key
// set. ok is false when the file does not exist or is not valid JSON.
func readLocaleKeys(path string) (map[string]struct{}, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, false
	}
	keys := map[string]struct{}{}
	flattenLocaleKeys("", doc, keys)
	return keys, true
}

// flattenLocaleKeys walks a parsed catalog, recording the dotted path of every
// leaf (and intermediate object) key so nested structures compare correctly.
// The last path segment is normalized through stripPluralSuffix so i18next CLDR
// plural variants do not register as drift: English declares greeting_one /
// greeting_other, Japanese only greeting_other, Arabic all six — all collapse to
// the base key "greeting", which is what parity actually requires.
func flattenLocaleKeys(prefix string, node map[string]any, out map[string]struct{}) {
	for k, v := range node {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		out[stripPluralSuffix(key)] = struct{}{}
		// Recurse with the un-stripped key so nested paths stay accurate.
		if child, ok := v.(map[string]any); ok {
			flattenLocaleKeys(key, child, out)
		}
	}
}

// cldrPluralSuffixes are the i18next plural-form suffixes (CLDR plural
// categories). A locale only carries the categories its language uses, so these
// must be stripped before comparing key sets across locales.
var cldrPluralSuffixes = []string{"_zero", "_one", "_two", "_few", "_many", "_other"}

// stripPluralSuffix removes a trailing CLDR plural suffix from the final segment
// of a dotted key (greeting_one → greeting; nav.count_other → nav.count). Keys
// without a plural suffix are returned unchanged.
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

// keySetDiff returns the sorted keys present in a but absent from b.
func keySetDiff(a, b map[string]struct{}) []string {
	var diff []string
	for k := range a {
		if _, ok := b[k]; !ok {
			diff = append(diff, k)
		}
	}
	sort.Strings(diff)
	return diff
}
