package gates

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"react-component-library/internal/librarywalk"
)

func ValidateOverlaySurfaceComposition(scope Scope) (Result, error) {
	root := scope.Root
	factsIndex, factsErr := readSourceFactsIndex(root, scope)
	if factsErr != nil && !os.IsNotExist(factsErr) {
		return Result{}, factsErr
	}
	result := Result{}
	paths, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "components", "*", "component.json"))
	if err != nil {
		return Result{}, err
	}
	for _, manifest := range paths {
		data, err := os.ReadFile(manifest)
		if err != nil {
			return Result{}, err
		}
		var doc struct {
			LibraryID                  string `json:"libraryId"`
			CatalogID                  string `json:"catalogId"`
			Category                   string `json:"category"`
			Latest                     string `json:"latest"`
			Draft                      string `json:"draft"`
			OverlaySurfaceOptOutReason string `json:"overlaySurfaceOptOutReason"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			return Result{}, err
		}
		if doc.Category != "overlays" && !strings.HasPrefix(doc.CatalogID, "overlays.") {
			continue
		}
		assetID := strings.TrimSpace(doc.CatalogID)
		if assetID == "" {
			assetID = doc.LibraryID
		}
		for _, version := range []string{doc.Latest, doc.Draft} {
			version = strings.TrimSpace(version)
			if version == "" {
				continue
			}
			versionDir := filepath.Join(filepath.Dir(manifest), "versions", version)
			entries, readErr := os.ReadDir(versionDir)
			if readErr != nil {
				return Result{}, readErr
			}
			for _, entry := range entries {
				if entry.IsDir() || !(strings.HasSuffix(entry.Name(), ".tsx") || strings.HasSuffix(entry.Name(), ".ts")) {
					continue
				}
				path := filepath.Join(versionDir, entry.Name())
				source, readErr := os.ReadFile(path)
				if readErr != nil {
					return Result{}, readErr
				}
				result.Inspected++
				overlayRole := sourceHasOverlayRoleFromFacts(factsIndex, path)
				if factsErr != nil {
					overlayRole = sourceContainsOverlayRole(source)
				}
				if overlayRole && !bytes.Contains(source, []byte("useOverlaySurface")) && strings.TrimSpace(doc.OverlaySurfaceOptOutReason) == "" {
					result.Findings = append(result.Findings, Finding{Code: "catalog.overlay_surface_composition", AssetID: assetID, File: repoRel(root, path), Message: "overlay role is implemented without useOverlaySurface", Remediation: "Compose useOverlaySurface, or add overlaySurfaceOptOutReason with a concrete non-overlay rationale.", DocsRef: "docs/reference/overlay-contract.md"})
				}
			}
		}
	}
	return nonEmpty(result, "overlay-surface-composition"), nil
}

var (
	styleTagRE             = regexp.MustCompile(`(?s)<style\b`)
	sharedMotionRE         = regexp.MustCompile(`(?is)@media\s*\(\s*prefers-reduced-motion\s*:`)
	sharedForcedColorsRE   = regexp.MustCompile(`(?is)@media\s*\(\s*forced-colors\s*:\s*active\s*\)`)
	sharedFocusRE          = regexp.MustCompile(`(?is)[^{}]*:focus-visible[^{}]*\{`)
	sharedVisuallyHiddenRE = regexp.MustCompile(`(?is)clip-path\s*:\s*inset\s*\(\s*50%`)
	componentSourceRE      = regexp.MustCompile(`(?m)@vrooliComponentSource\s+([^*\s]+)`)
)

// ValidateSharedStyleOwnership keeps cross-cutting rules in BaseStyles. An
// asset-local copy is not harmless duplication: it gives render order control
// over focus, motion, and forced-colors behavior.
