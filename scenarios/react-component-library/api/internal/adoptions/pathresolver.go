package adoptions

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"react-component-library/internal/uimanifest"
)

// ResolveSource enumerates how the resolver arrived at a path. The UI shows
// this as a badge so users know whether to trust the pre-filled value.
type ResolveSource string

const (
	SourceExplicit         ResolveSource = "explicit"
	SourceTemplateManifest ResolveSource = "template-manifest"
	SourceHeuristic        ResolveSource = "heuristic"
	SourceFallback         ResolveSource = "fallback"
)

// ResolveInput is what callers (handler, CLI) supply.
type ResolveInput struct {
	// ComponentSlot is the slot declared by the component's library entry
	// (component.json `slot` field). Empty means "use the manifest default".
	ComponentSlot string
	// ComponentName is the component's display name in PascalCase
	// (e.g. "SidebarShell"). Used to substitute path-pattern tokens.
	ComponentName string
	// Scenario is the target scenario folder name under scenarios/.
	Scenario string
	// Override, when non-empty, short-circuits the entire resolution.
	Override string
	// Feature is the optional feature folder name; required when the slot's
	// requiresFeature is true.
	Feature string
}

// ResolveResult is what the resolver returns. Path is relative to the
// scenario root.
type ResolveResult struct {
	Path     string
	Source   ResolveSource
	Slot     string
	Warnings []string
}

// Resolver resolves an adoption path. The Loader seam is injectable so tests
// supply an in-memory fake instead of touching the filesystem.
type Resolver struct {
	Loader   uimanifest.Loader
	RepoRoot string
}

// NewResolver constructs a resolver. RepoRoot is needed for the heuristic +
// fallback branches which inspect the scenario directory.
func NewResolver(loader uimanifest.Loader, repoRoot string) *Resolver {
	return &Resolver{Loader: loader, RepoRoot: repoRoot}
}

// Resolve runs the four-stage resolution order described in
// docs/concepts/UI-ARCHITECTURE.md:
//
//  1. Explicit override.
//  2. Template manifest lookup with token substitution.
//  3. Heuristic: scan the scenario's ui/src/ for a directory matching the slot.
//  4. Fallback: ui/src/components/<ComponentName>.tsx.
func (r *Resolver) Resolve(in ResolveInput) (ResolveResult, error) {
	if in.ComponentName == "" {
		return ResolveResult{}, errors.New("resolve: componentName is required")
	}
	if in.Scenario == "" {
		return ResolveResult{}, errors.New("resolve: scenario is required")
	}

	// 1. Explicit override.
	if in.Override != "" {
		cleaned, err := safeRelPath(in.Override)
		if err != nil {
			return ResolveResult{}, err
		}
		return ResolveResult{Path: cleaned, Source: SourceExplicit, Slot: in.ComponentSlot}, nil
	}

	// 2. Template manifest lookup.
	mf, mfErr := r.Loader.Load(in.Scenario)
	if mfErr == nil {
		slotName, slot, ok := pickSlot(mf, in.ComponentSlot)
		if ok {
			if slot.RequiresFeature && in.Feature == "" {
				return ResolveResult{}, fmt.Errorf("resolve: slot %q requires a feature folder", slotName)
			}
			path, err := substitutePattern(slot, in)
			if err != nil {
				return ResolveResult{}, err
			}
			if err := guardWithinScenario(path); err != nil {
				return ResolveResult{}, err
			}
			return ResolveResult{Path: path, Source: SourceTemplateManifest, Slot: slotName}, nil
		}
	}

	// 3. Heuristic — manifest absent OR slot not in manifest; scan scenario
	// ui/src for a directory matching the slot's expected name. Only attempt
	// when we actually have a slot hint to match against.
	if in.ComponentSlot != "" && r.RepoRoot != "" {
		dir, ok := heuristicDir(r.RepoRoot, in.Scenario, in.ComponentSlot)
		if ok {
			path := filepath.ToSlash(filepath.Join(dir, in.ComponentName+".tsx"))
			warnings := []string{
				fmt.Sprintf("template manifest unavailable; using heuristic match for slot %q", in.ComponentSlot),
			}
			if mfErr != nil {
				warnings = append(warnings, "manifest load error: "+mfErr.Error())
			}
			return ResolveResult{
				Path:     path,
				Source:   SourceHeuristic,
				Slot:     in.ComponentSlot,
				Warnings: warnings,
			}, nil
		}
	}

	// 4. Fallback.
	path := "ui/src/components/" + in.ComponentName + ".tsx"
	warnings := []string{"no template manifest or matching directory; using fallback path"}
	if mfErr != nil {
		warnings = append(warnings, "manifest load error: "+mfErr.Error())
	}
	return ResolveResult{
		Path:     path,
		Source:   SourceFallback,
		Slot:     in.ComponentSlot,
		Warnings: warnings,
	}, nil
}

// pickSlot resolves the slot name to use: caller-supplied first, then the
// manifest default, then the conventional "shared-component" fallback.
func pickSlot(mf uimanifest.Manifest, hint string) (string, uimanifest.Slot, bool) {
	if hint != "" {
		if s, ok := mf.LookupSlot(hint); ok {
			return hint, s, true
		}
	}
	if mf.Defaults.Slot != "" {
		if s, ok := mf.LookupSlot(mf.Defaults.Slot); ok {
			return mf.Defaults.Slot, s, true
		}
	}
	if s, ok := mf.LookupSlot("shared-component"); ok {
		return "shared-component", s, true
	}
	return "", uimanifest.Slot{}, false
}

// substitutePattern fills the slot's path pattern. Defaults to
// {dir}/{ComponentName}.tsx when no pattern is declared.
func substitutePattern(slot uimanifest.Slot, in ResolveInput) (string, error) {
	pattern := slot.PathPattern
	if pattern == "" {
		pattern = "{dir}/{ComponentName}.tsx"
	}
	dir := slot.Dir
	if slot.RequiresFeature {
		dir = strings.ReplaceAll(dir, "{feature}", in.Feature)
	}
	out := pattern
	out = strings.ReplaceAll(out, "{dir}", dir)
	out = strings.ReplaceAll(out, "{ComponentName}", in.ComponentName)
	out = strings.ReplaceAll(out, "{componentName}", toCamel(in.ComponentName))
	out = strings.ReplaceAll(out, "{camelName}", toCamel(in.ComponentName))
	out = strings.ReplaceAll(out, "{kebab-name}", toKebab(in.ComponentName))
	if in.Feature != "" {
		out = strings.ReplaceAll(out, "{feature}", in.Feature)
	}
	// {locale} is intentionally left for the i18n-strings slot — callers
	// using that slot will pass a feature-style override.
	if strings.Contains(out, "{") {
		return "", fmt.Errorf("resolve: unsubstituted token in path %q", out)
	}
	return filepath.ToSlash(out), nil
}

// heuristicDir looks for an obvious directory name match inside the scenario's
// ui/src/ tree. We don't try to be clever — exact lowercase match on the slot
// name's trailing token (e.g. "layout-nav" -> "layout", "page" -> "pages").
func heuristicDir(repoRoot, scenario, slotName string) (string, bool) {
	candidates := slotDirCandidates(slotName)
	uiSrc := filepath.Join(repoRoot, "scenarios", scenario, "ui", "src")
	for _, c := range candidates {
		abs := filepath.Join(uiSrc, c)
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			return filepath.ToSlash(filepath.Join("ui/src", c)), true
		}
	}
	return "", false
}

// slotDirCandidates lists directory names to try for a given slot. Conservative
// — empty when we don't have a sensible guess.
func slotDirCandidates(slotName string) []string {
	switch slotName {
	case "ui-primitive":
		return []string{"components/ui"}
	case "shared-component":
		return []string{"components"}
	case "layout-shell", "layout-nav":
		return []string{"layout"}
	case "page":
		return []string{"pages"}
	case "hook":
		return []string{"hooks"}
	case "api-client":
		return []string{"api"}
	case "lib-util":
		return []string{"lib"}
	case "consts":
		return []string{"consts"}
	case "theme-token":
		return []string{"theme"}
	case "test-util":
		return []string{"test-utils"}
	}
	return nil
}

// safeRelPath cleans a user-supplied override and rejects absolute / escaping
// paths. Paths must stay below the scenario root.
func safeRelPath(p string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(p))
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("resolve: path %q must be scenario-relative", p)
	}
	if err := guardWithinScenario(clean); err != nil {
		return "", err
	}
	return clean, nil
}

// guardWithinScenario rejects parent-traversal paths (../).
func guardWithinScenario(p string) error {
	clean := filepath.ToSlash(filepath.Clean(p))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("resolve: path %q escapes scenario root", p)
	}
	return nil
}

// toCamel converts a PascalCase name (e.g. "SidebarShell") to camelCase
// ("sidebarShell").
func toCamel(name string) string {
	if name == "" {
		return ""
	}
	runes := []rune(name)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// toKebab converts a PascalCase name (e.g. "SidebarShell") to kebab-case
// ("sidebar-shell").
func toKebab(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			b.WriteByte('-')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
