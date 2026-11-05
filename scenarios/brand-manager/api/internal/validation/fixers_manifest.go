package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// This file holds the manifest-writing self-contained fixers. The active
// manifest is the first existing webManifestCandidate; when none exists the
// canonical ui/public/site.webmanifest is created. JSON is re-marshalled with
// sorted keys, so applies are deterministic and idempotent.

// loadManifestForFix reads the active manifest fresh from disk for a fixer.
func loadManifestForFix(root string) (rel string, obj map[string]any, present bool) {
	for _, cand := range webManifestCandidates {
		if content, ok := readFile(root, cand); ok {
			m := map[string]any{}
			_ = json.Unmarshal([]byte(content), &m)
			return cand, m, true
		}
	}
	return webManifestCandidates[0], map[string]any{}, false
}

func writeManifest(root, rel string, obj map[string]any) error {
	abs := filepath.Join(root, filepath.FromSlash(rel))
	out, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return err
	}
	return os.WriteFile(abs, append(out, '\n'), 0o600)
}

// alignManifestName sets manifest name/short_name to want when an existing
// manifest disagrees. It reports whether a change is needed and only writes when
// apply is true. A missing manifest is left to fixManifestCompleteness.
func alignManifestName(root, want string, apply bool) (bool, error) {
	rel, obj, present := loadManifestForFix(root)
	if !present || strings.TrimSpace(want) == "" {
		return false, nil
	}
	changed := false
	for _, key := range []string{"name", "short_name"} {
		if v, ok := obj[key].(string); ok && strings.TrimSpace(v) != "" && !strings.EqualFold(strings.TrimSpace(v), want) {
			obj[key] = want
			changed = true
		}
	}
	if changed && apply {
		if err := writeManifest(root, rel, obj); err != nil {
			return false, err
		}
	}
	return changed, nil
}

// fixThemeColorConsistency aligns the manifest theme_color to the page's
// <meta theme-color> (the source of truth).
func fixThemeColorConsistency(root string, apply bool) (Candidate, bool, error) {
	if !ruleFiresIsolated(root, "theme-color-consistency") {
		return Candidate{}, false, nil
	}
	content, ok := loadIndexHTML(root)
	if !ok {
		return Candidate{}, false, nil
	}
	meta, hasMeta := parseHead(content, true).metaByName("theme-color")
	if !hasMeta {
		return Candidate{}, false, nil
	}
	rel, obj, present := loadManifestForFix(root)
	if !present {
		return Candidate{}, false, nil
	}
	before, _ := obj["theme_color"].(string)
	obj["theme_color"] = strings.TrimSpace(meta)
	cand := Candidate{
		FilePath:    rel,
		Description: "Set the manifest theme_color to match the page <meta theme-color>.",
		Before:      "theme_color: " + before,
		After:       "theme_color: " + strings.TrimSpace(meta),
	}
	if apply {
		if err := writeManifest(root, rel, obj); err != nil {
			return Candidate{}, false, err
		}
		cand.Applied = true
	}
	return cand, true, nil
}

// fixManifestCompleteness fills the missing scalar manifest fields from the
// scenario identity + house defaults (and theme/background color from the page
// theme-color when present). Icon entries require real image assets and are left
// to a human / brand-projection path, so a manifest missing only icons is not
// auto-fixed.
func fixManifestCompleteness(root string, apply bool) (Candidate, bool, error) {
	if !ruleFiresIsolated(root, "manifest-completeness") {
		return Candidate{}, false, nil
	}
	rel, obj, _ := loadManifestForFix(root)
	c := newScanContext("", root)

	fillable := map[string]string{}
	for k, v := range c.identity().ManifestScalars() {
		fillable[k] = v
	}
	if content, ok := loadIndexHTML(root); ok {
		if tc, has := parseHead(content, true).metaByName("theme-color"); has && strings.TrimSpace(tc) != "" {
			fillable["theme_color"] = strings.TrimSpace(tc)
			fillable["background_color"] = strings.TrimSpace(tc)
		}
	}

	changed := false
	for key, val := range fillable {
		if strings.TrimSpace(val) == "" {
			continue
		}
		if !manifestHasKey(obj, key) {
			obj[key] = val
			changed = true
		}
	}
	if !changed {
		return Candidate{}, false, nil // only icons missing — not deterministically fixable
	}
	out, _ := json.MarshalIndent(obj, "", "  ")
	cand := Candidate{
		FilePath:    rel,
		Description: "Fill the missing scalar manifest fields from the scenario identity and house defaults (icons still require a logo).",
		Before:      "(incomplete manifest)",
		After:       string(out),
	}
	if apply {
		if err := writeManifest(root, rel, obj); err != nil {
			return Candidate{}, false, err
		}
		cand.Applied = true
	}
	return cand, true, nil
}
