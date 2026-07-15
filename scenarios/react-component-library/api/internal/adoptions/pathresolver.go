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

// SlotSource enumerates how the resolver decided a file's slot. Distinct from
// ResolveSource (which explains how the PATH was resolved once the slot is
// known). The UI shows both so a reviewer can trust a placement.
type SlotSource string

const (
	// SlotSourceExplicit — the file carried authored slot metadata (component
	// version file `slot` column / component.json fileSlots).
	SlotSourceExplicit SlotSource = "explicit"
	// SlotSourceHeuristic — the slot came from a filename/extension rule
	// (e.g. use*.ts -> hook, *.css -> theme-token).
	SlotSourceHeuristic SlotSource = "heuristic"
	// SlotSourceComponent — a non-entry file with no better signal inherited
	// the component's declared slot.
	SlotSourceComponent SlotSource = "component"
	// SlotSourceEntry — the entry file, placed at the component's declared slot.
	SlotSourceEntry SlotSource = "entry"
)

// FileInput is one member of a version unit the caller wants placed.
type FileInput struct {
	// Path is the basename within the version unit (e.g. "useFocusTrap.ts").
	Path string
	// IsEntry marks the version's renderable entry file.
	IsEntry bool
	// Slot is explicit authored slot metadata; empty means "derive".
	Slot string
}

// VersionResolveInput resolves an entire version unit's file set.
type VersionResolveInput struct {
	// ComponentName is the entry component's PascalCase name.
	ComponentName string
	// ComponentSlot is the component's declared slot (drives entry placement).
	ComponentSlot string
	// Scenario chases service.json -> template manifest. Ignored when Template
	// is set; still echoed for labeling.
	Scenario string
	// Template, when set, resolves placement directly against the named
	// template's manifest (scenario-agnostic).
	Template string
	// Feature is the optional feature folder name.
	Feature string
	// Files is the full version unit. Exactly one entry expected.
	Files []FileInput
}

// ResolvedFile is one placed file with full provenance.
type ResolvedFile struct {
	LibraryPath string
	TargetPath  string
	Slot        string
	Source      ResolveSource
	SlotSource  SlotSource
	IsEntry     bool
	Warnings    []string
}

// VersionResolveResult is the placed file set plus manifest provenance.
type VersionResolveResult struct {
	// Template whose manifest drove placement; empty when none resolved.
	Template string
	// ManifestResolved is true when a template UI manifest authoritatively
	// placed the files (the UI trusts the tree; otherwise it renders flat).
	ManifestResolved bool
	Files            []ResolvedFile
	Warnings         []string
}

// ResolveVersion places every file of a version unit at its slot-derived
// target path. The entry file uses the component's declared slot; companions
// derive their slot from explicit metadata (wins), then a filename heuristic,
// then the component slot. Each file records both how its slot was chosen
// (SlotSource) and how its path was resolved (Source).
func (r *Resolver) ResolveVersion(in VersionResolveInput) (VersionResolveResult, error) {
	if in.ComponentName == "" {
		return VersionResolveResult{}, errors.New("resolveVersion: componentName is required")
	}
	if in.Scenario == "" && in.Template == "" {
		return VersionResolveResult{}, errors.New("resolveVersion: scenario or template is required")
	}

	// Load the placement manifest. Template wins (scenario-agnostic preview);
	// otherwise chase the scenario's service.json.
	var (
		mf         uimanifest.Manifest
		mfErr      error
		templateID = in.Template
	)
	if in.Template != "" {
		mf, mfErr = r.Loader.LoadTemplate(in.Template)
	} else {
		mf, mfErr = r.Loader.Load(in.Scenario)
	}
	manifestOK := mfErr == nil
	if manifestOK {
		if mf.Contract.Template != "" {
			templateID = mf.Contract.Template
		}
	} else {
		templateID = ""
	}

	files := in.Files
	if len(files) == 0 {
		// No indexed files — synthesize a single entry from the component name
		// so callers still get the entry placement.
		files = []FileInput{{Path: in.ComponentName + ".tsx", IsEntry: true}}
	}

	res := VersionResolveResult{Template: templateID, ManifestResolved: manifestOK}
	if !manifestOK && mfErr != nil {
		res.Warnings = append(res.Warnings, "template manifest unavailable; using flat fallback placement: "+mfErr.Error())
	}

	for _, f := range files {
		slotName, slotSrc := r.deriveSlot(f, in.ComponentSlot)
		rf := ResolvedFile{LibraryPath: f.Path, Slot: slotName, SlotSource: slotSrc, IsEntry: f.IsEntry}

		// Name token for path substitution: entry uses the component name;
		// companions use their own basename (minus extension) so hooks land as
		// {dir}/useFocusTrap.ts rather than {dir}/DrawerShell.ts.
		nameToken := in.ComponentName
		if !f.IsEntry {
			nameToken = baseName(f.Path)
		}

		if manifestOK {
			name, slot, ok := pickSlot(mf, slotName)
			if ok {
				sub := ResolveInput{ComponentName: nameToken, Feature: in.Feature}
				path, err := substitutePattern(slot, sub)
				if err == nil {
					if gerr := guardWithinScenario(path); gerr == nil {
						rf.TargetPath = preserveExtension(path, f.Path)
						rf.Source = SourceTemplateManifest
						rf.Slot = name
						res.Files = append(res.Files, rf)
						continue
					}
				}
				rf.Warnings = append(rf.Warnings, fmt.Sprintf("slot %q pattern did not substitute cleanly; using fallback", name))
			}
		}

		// Fallback placement — manifest absent or slot unusable.
		rf.TargetPath = fallbackPath(slotName, f)
		rf.Source = SourceFallback
		res.Files = append(res.Files, rf)
	}

	return res, nil
}

// deriveSlot picks a file's slot: explicit authored metadata wins, then a
// filename heuristic, then (entry) the component slot, then component slot for
// companions with no signal.
func (r *Resolver) deriveSlot(f FileInput, componentSlot string) (string, SlotSource) {
	if s := strings.TrimSpace(f.Slot); s != "" {
		return s, SlotSourceExplicit
	}
	if s, ok := heuristicSlot(f.Path); ok {
		return s, SlotSourceHeuristic
	}
	if f.IsEntry {
		return componentSlot, SlotSourceEntry
	}
	return componentSlot, SlotSourceComponent
}

// heuristicSlot infers a slot from a filename. Conservative: only the
// unambiguous conventions. Returns false when nothing matches.
func heuristicSlot(basename string) (string, bool) {
	name := baseName(basename)
	switch {
	case strings.HasPrefix(name, "use") && len(name) > 3 && unicode.IsUpper([]rune(name)[3]) &&
		(strings.HasSuffix(basename, ".ts") || strings.HasSuffix(basename, ".tsx")):
		// use<Upper>… .ts(x) — a React hook.
		return "hook", true
	case strings.HasSuffix(basename, ".css"):
		return "theme-token", true
	}
	return "", false
}

// fallbackPath places a file when no manifest resolved. Hooks land under
// ui/src/hooks; everything else under ui/src/components, preserving the
// library basename.
func fallbackPath(slotName string, f FileInput) string {
	switch slotName {
	case "hook":
		return "ui/src/hooks/" + f.Path
	case "theme-token":
		return "ui/src/theme/" + f.Path
	case "api-client":
		return "ui/src/api/" + f.Path
	case "lib-util":
		return "ui/src/lib/" + f.Path
	}
	return "ui/src/components/" + f.Path
}

// baseName strips the file extension from a basename.
func baseName(p string) string {
	b := filepath.Base(p)
	if ext := filepath.Ext(b); ext != "" {
		return strings.TrimSuffix(b, ext)
	}
	return b
}

// preserveExtension keeps the resolved path's directory + stem but restores the
// library file's extension when the slot pattern assumed a different one. This
// matters for companions whose pattern is {dir}/{camelName}.ts but whose source
// is .tsx (or vice versa) — the placed file must keep its real extension.
func preserveExtension(resolved, libraryPath string) string {
	libExt := filepath.Ext(libraryPath)
	resExt := filepath.Ext(resolved)
	if libExt == "" || libExt == resExt {
		return resolved
	}
	return strings.TrimSuffix(resolved, resExt) + libExt
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
