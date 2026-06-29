package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"brand-manager/internal/brandsurface"
)

// This file holds the /public/ convention fixer: it relocates the branding
// assets served at the site root into the /public/ source location
// (ui/public/public) and repoints the index.html links + manifest references at
// /public/. It is gated on the rule firing (ruleFiresIsolated) so the advertised
// AutofixAvailable flag matches what it can deterministically do, and it is
// idempotent — a re-run finds nothing at the root and proposes no candidate.

// fixPublicAssetConvention is the deterministic remediation for the
// public-asset-convention rule.
func fixPublicAssetConvention(root string, apply bool) (Candidate, bool, error) {
	if !ruleFiresIsolated(root, "public-asset-convention") {
		return Candidate{}, false, nil
	}
	o := detectPublicConventionOffenders(root)
	if o.empty() {
		return Candidate{}, false, nil
	}

	var beforeLines, afterLines []string
	for _, rel := range o.rootFiles {
		beforeLines = append(beforeLines, rel)
		afterLines = append(afterLines, publicSourcePath(rel))
	}
	for _, ref := range o.rootRefs {
		beforeLines = append(beforeLines, "ref "+ref)
		afterLines = append(afterLines, "ref "+brandsurface.PublicURLForRootRef(ref))
	}
	cand := Candidate{
		FilePath:    brandsurface.PublicAssetSourceDir,
		Description: "Relocate public branding assets under /public/ and repoint the index.html + manifest references.",
		Before:      strings.Join(beforeLines, "\n"),
		After:       strings.Join(afterLines, "\n"),
	}
	if !apply {
		return cand, true, nil
	}

	// 1) Relocate the asset files into ui/public/public.
	for _, rel := range o.rootFiles {
		if err := relocateUnderPublic(root, rel); err != nil {
			return Candidate{}, false, err
		}
	}
	// 2) Repoint the index.html references.
	if err := repointIndexHTMLPublic(root, o.rootRefs); err != nil {
		return Candidate{}, false, err
	}
	// 3) Repoint the (now relocated) manifest's own references.
	if err := repointManifestPublic(root); err != nil {
		return Candidate{}, false, err
	}
	cand.Applied = true
	return cand, true, nil
}

// publicSourcePath maps a root-served asset path (ui/public/<base>) to its
// /public/ source location (ui/public/public/<base>).
func publicSourcePath(rel string) string {
	return brandsurface.PublicAssetSourceDir + "/" + filepath.Base(rel)
}

// relocateUnderPublic moves root/<rel> (a file directly under the publicDir
// root) into the /public/ source directory, preserving its bytes. It is
// idempotent: a missing source or an already-relocated file is a no-op.
func relocateUnderPublic(root, rel string) error {
	src := filepath.Join(root, filepath.FromSlash(rel))
	dst := filepath.Join(root, filepath.FromSlash(publicSourcePath(rel)))
	if src == dst {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", rel, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("create public dir: %w", err)
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return os.Remove(src)
}

// repointIndexHTMLPublic rewrites each offending root-absolute branding
// reference in ui/index.html to its /public/ equivalent (href="..." and
// content="..." occurrences), leaving the rest of the document untouched.
func repointIndexHTMLPublic(root string, refs []string) error {
	if len(refs) == 0 {
		return nil
	}
	content, ok := loadIndexHTML(root)
	if !ok {
		return nil
	}
	updated := content
	for _, ref := range refs {
		want := brandsurface.PublicURLForRootRef(ref)
		for _, attr := range []string{"href", "content"} {
			updated = strings.ReplaceAll(updated, attr+`="`+ref+`"`, attr+`="`+want+`"`)
		}
	}
	if updated == content {
		return nil
	}
	return writeIndexHTML(root, updated)
}

// repointManifestPublic repoints the active web manifest's own references after
// it has been relocated under /public/: root-absolute icon srcs gain the
// /public/ prefix, and start_url/scope are pinned to the app root ("/") so the
// installed PWA launches into the app, not the /public/ asset folder (the
// manifest now lives under /public/, so a relative start_url would resolve
// there). Relative icon srcs are left untouched — they resolve beside the
// relocated manifest.
func repointManifestPublic(root string) error {
	rel, obj, present := loadManifestForFix(root)
	if !present {
		return nil
	}
	// Only adjust a manifest that now lives under the /public/ source directory.
	if !strings.HasPrefix(filepath.ToSlash(rel), brandsurface.PublicAssetSourceDir+"/") {
		return nil
	}
	changed := false
	if icons, ok := obj["icons"].([]any); ok {
		for _, ic := range icons {
			m, ok := ic.(map[string]any)
			if !ok {
				continue
			}
			if src, _ := m["src"].(string); isRootAbsoluteRef(src) && !brandsurface.IsUnderPublicPrefix(src) {
				m["src"] = brandsurface.PublicURLForRootRef(src)
				changed = true
			}
		}
	}
	// A relative or /public-rooted launch target would open the asset folder now
	// that the manifest lives under /public/; pin both to the app root.
	for _, key := range []string{"start_url", "scope"} {
		if v, _ := obj[key].(string); v != "/" {
			if v == "" || !strings.HasPrefix(v, "/") || strings.HasPrefix(v, brandsurface.PublicURLPrefix) {
				obj[key] = "/"
				changed = true
			}
		}
	}
	if !changed {
		return nil
	}
	return writeManifest(root, rel, obj)
}
