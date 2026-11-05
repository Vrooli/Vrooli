package gates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func ValidateLifecycle(scope Scope) (Result, error) {
	root := scope.Root
	result := Result{}
	paths, err := activeLibrarySources(scope)
	if err != nil {
		return Result{}, err
	}
	for _, path := range paths {
		if !lifecycleSourceApplies(root, path) {
			continue
		}
		// Stories are browser-only specimens, not released runtime. Including
		// them here makes the lifecycle gate report demo timers and AbortSignal
		// listeners as component defects.
		if isStorySource(path) || isTestSource(path) {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		text := string(data)
		if strings.Contains(text, "addEventListener") && !strings.Contains(text, "removeEventListener") {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.lifecycle_cleanup", AssetID: implementationName(path), File: repoRel(root, path), Line: lineOf(data, "addEventListener"),
				Message:     "adds an event listener with no matching removeEventListener anywhere in the file",
				Remediation: "Return a cleanup function from the effect that registered this listener, calling removeEventListener with the same target, type, and handler reference. Without it the handler outlives the component: every mount adds another subscription, so the work done per event grows with the number of times the user has visited the surface.",
				DocsRef:     "docs/internal/SEAMS.md",
			})
		}
		if strings.Contains(text, "new MutationObserver") && !strings.Contains(text, ".disconnect(") {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.lifecycle_cleanup", AssetID: implementationName(path), File: repoRel(root, path), Line: lineOf(data, "new MutationObserver"),
				Message:     "constructs a MutationObserver with no .disconnect() anywhere in the file",
				Remediation: "Call .disconnect() in the cleanup of the effect that constructed this observer. An undisconnected observer keeps its target subtree alive and keeps firing after unmount, which shows up as work attributed to whatever surface is mounted next rather than to this one.",
				DocsRef:     "docs/internal/SEAMS.md",
			})
		}
		if hasBrowserAccessOutsideEffects(text) {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.lifecycle_ssr", AssetID: implementationName(path), File: repoRel(root, path),
				Message:     "reads a browser global during render or module scope, where no SSR guard applies",
				Remediation: "Move the access into useEffect/useLayoutEffect, or guard it with a typeof window !== \"undefined\" check. Render and module scope both execute during server rendering, so a bare window/document reference throws there — and because it throws at import time it takes down the whole route, not just this component.",
				DocsRef:     "docs/internal/SEAMS.md",
			})
		}
	}
	return nonEmpty(result, "lifecycle"), nil
}

func lifecycleSourceApplies(root, path string) bool {
	assetDir := filepath.Dir(filepath.Dir(filepath.Dir(path)))
	data, err := os.ReadFile(filepath.Join(assetDir, "component.json"))
	if err != nil {
		// Isolated fixtures do not need a generated manifest and retain the
		// historical source-only contract used by the unit tests.
		return true
	}
	var manifest struct {
		AssetKind string `json:"assetKind"`
	}
	if json.Unmarshal(data, &manifest) != nil || manifest.AssetKind == "" {
		return true
	}
	switch manifest.AssetKind {
	case "runtime-hook", "runtime-service", "adapter", "generator":
		return true
	default:
		return false
	}
}
