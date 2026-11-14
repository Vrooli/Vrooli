package validation

import (
	"bytes"
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
	for _, fix := range append(append([]publicRefFix(nil), o.headRefs...), o.manifestIcons...) {
		beforeLines = append(beforeLines, "ref "+fix.From)
		afterLines = append(afterLines, "ref "+fix.To)
	}
	if o.manifestLaunch {
		beforeLines = append(beforeLines, "manifest start_url/scope")
		afterLines = append(afterLines, "manifest start_url/scope "+brandsurface.ManifestAppRoot)
	}
	workers := planServiceWorkerRepoint(root, o.rootFiles)
	for _, fix := range workers.refs {
		beforeLines = append(beforeLines, "sw "+fix.From)
		afterLines = append(afterLines, "sw "+fix.To)
	}
	description := "Relocate public branding assets under /public/, repoint the index.html, manifest and service-worker references to their canonical public form, and launch the manifest at the app root."
	if workers.refusal != "" {
		beforeLines = append(beforeLines, "refused: "+workers.refusal)
		description = "Refused: " + workers.refusal
	}
	cand := Candidate{
		FilePath:    brandsurface.PublicAssetSourceDir,
		Description: description,
		Before:      strings.Join(beforeLines, "\n"),
		After:       strings.Join(afterLines, "\n"),
	}
	if !apply {
		return cand, true, nil
	}

	// 0) Refuse before changing anything when a root copy would replace a
	// different /public/ file, or a service worker names a moved file in a
	// form the fixer cannot rewrite, so a refused apply leaves the tree
	// untouched.
	if workers.refusal != "" {
		return Candidate{}, false, fmt.Errorf("public-asset-convention refused: %s", workers.refusal)
	}
	for _, rel := range o.rootFiles {
		if err := relocationConflict(root, rel); err != nil {
			return Candidate{}, false, err
		}
	}
	// 1) Relocate the asset files into ui/public/public.
	for _, rel := range o.rootFiles {
		if err := relocateUnderPublic(root, rel); err != nil {
			return Candidate{}, false, err
		}
	}
	// 2) Repoint the index.html references.
	if err := repointIndexHTMLPublic(root, o.headRefs); err != nil {
		return Candidate{}, false, err
	}
	// 3) Repoint the (now relocated) manifest's own references.
	if err := repointManifestPublic(root); err != nil {
		return Candidate{}, false, err
	}
	// 4) Repoint service-worker precache entries at the moved files.
	for rel, content := range workers.content {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			return Candidate{}, false, fmt.Errorf("write %s: %w", rel, err)
		}
	}
	cand.Applied = true
	return cand, true, nil
}

// relocationConflict reports a root asset whose /public/ destination already
// exists with different content.
func relocationConflict(root, rel string) error {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil
	}
	existing, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(publicSourcePath(rel))))
	if err != nil {
		return nil
	}
	if !bytes.Equal(existing, data) {
		return fmt.Errorf("%s and %s both exist with different content; remove the stale copy before relocating", rel, publicSourcePath(rel))
	}
	return nil
}

// publicSourcePath maps a root-served asset path (ui/public/<base>) to its
// /public/ source location (ui/public/public/<base>).
func publicSourcePath(rel string) string {
	return brandsurface.PublicAssetSourceDir + "/" + filepath.Base(rel)
}

// relocateUnderPublic moves root/<rel> (a file directly under the publicDir
// root) into the /public/ source directory, preserving its bytes. It is
// idempotent: a missing source or an already-relocated file is a no-op. It
// never overwrites: a root copy identical to the /public/ file is redundant
// and removed, and a different one is refused, because the /public/ file is
// the one being served and a stale root copy must not replace it.
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
	if err := relocationConflict(root, rel); err != nil {
		return err
	}
	if _, statErr := os.Stat(dst); statErr == nil {
		return os.Remove(src) // identical to the served copy: redundant
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("create public dir: %w", err)
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return os.Remove(src)
}

// repointIndexHTMLPublic rewrites each non-canonical branding reference in
// ui/index.html (href="..." and content="..." occurrences) to its canonical
// public form, leaving the rest of the document untouched.
func repointIndexHTMLPublic(root string, fixes []publicRefFix) error {
	if len(fixes) == 0 {
		return nil
	}
	content, ok := loadIndexHTML(root)
	if !ok {
		return nil
	}
	updated := content
	for _, fix := range fixes {
		updated = strings.ReplaceAll(updated, fix.Attr+`="`+fix.From+`"`, fix.Attr+`="`+fix.To+`"`)
	}
	if updated == content {
		return nil
	}
	return writeIndexHTML(root, updated)
}

// repointManifestPublic repoints the active web manifest once it lives under
// /public/: root-absolute icon srcs become relative to the manifest, which
// sits beside its icons, and start_url/scope become "../", the app root
// relative to the manifest, so the installed PWA launches into the app rather
// than the asset folder and the manifest stays correct behind a path-prefix
// proxy.
func repointManifestPublic(root string) error {
	rel, obj, present := loadManifestForFix(root)
	if !present {
		return nil
	}
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
			if src, _ := m["src"].(string); isRootAbsoluteRef(src) {
				m["src"] = brandsurface.PublicManifestIconSrc(src)
				changed = true
			}
		}
	}
	for _, key := range []string{"start_url", "scope"} {
		if v, _ := obj[key].(string); v != brandsurface.ManifestAppRoot {
			obj[key] = brandsurface.ManifestAppRoot
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return writeManifest(root, rel, obj)
}
