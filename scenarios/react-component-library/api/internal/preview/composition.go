package preview

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"react-component-library/internal/components"
)

// Composition is shared by preview and generated scenario layout. Slot paths
// name actual template props (for example data.inspector), never DOM selectors.
// LocalSubmitPolicyMarker versions form-event behavior inside immutable render bytes.
const LocalSubmitPolicyMarker = `<meta name="rcl-preview-form-policy" content="local-submit-v1">`

type Composition struct {
	Revision string              `json:"revision"`
	Template CompositionAsset    `json:"template"`
	Regions  []CompositionRegion `json:"regions"`
}
type CompositionAsset struct {
	CatalogID string `json:"catalogId"`
	Version   string `json:"version"`
	Export    string `json:"export"`
}
type CompositionRegion struct {
	Parent   string            `json:"parent,omitempty"`
	ID       string            `json:"id"`
	Asset    *CompositionAsset `json:"asset,omitempty"`
	Slot     []string          `json:"slot"`
	Required bool              `json:"required"`
}
type CompositionGap struct {
	Region, Code, Message string
	Required              bool
}
type CompositionBundle struct {
	Bundle
	Source   string
	Revision string
	Gaps     []CompositionGap
}
type CompositionService interface {
	GetCompositionBundle(context.Context, Composition) (CompositionBundle, error)
}

type compositionImport struct {
	Asset CompositionAsset
	Slug  string
}

var (
	compositionRegionID   = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	compositionIdentifier = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
	compositionVersion    = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?(?:\+[A-Za-z0-9.-]+)?$`)
)

func validateComposition(c Composition) error {
	if c.Revision == "" {
		return fmt.Errorf("composition revision is required")
	}
	if len(c.Regions) > 64 {
		return fmt.Errorf("composition exceeds 64 regions")
	}
	ids := map[string]bool{}
	paths := []string{}
	for _, r := range c.Regions {
		if !compositionRegionID.MatchString(r.ID) || ids[r.ID] {
			return fmt.Errorf("empty, reserved, or duplicate region %q", r.ID)
		}
		ids[r.ID] = true
		if len(r.Slot) == 0 || len(r.Slot) > 4 {
			return fmt.Errorf("region %s requires a slot path of one to four props", r.ID)
		}
		for _, part := range r.Slot {
			if !compositionIdentifier.MatchString(part) || part == "__proto__" || part == "constructor" || part == "prototype" {
				return fmt.Errorf("region %s has an invalid slot prop", r.ID)
			}
		}
		path := r.Parent + "/" + strings.Join(r.Slot, ".")
		for _, other := range paths {
			if path == other || strings.HasPrefix(path, other+".") || strings.HasPrefix(other, path+".") {
				return fmt.Errorf("overlapping composition slots %s and %s", path, other)
			}
		}
		paths = append(paths, path)
	}
	parents := map[string]string{}
	for _, r := range c.Regions {
		if r.Parent != "" && !ids[r.Parent] {
			return fmt.Errorf("region %s has undeclared parent %s", r.ID, r.Parent)
		}
		parents[r.ID] = r.Parent
	}
	for _, r := range c.Regions {
		seen := map[string]bool{r.ID: true}
		for parent := r.Parent; parent != ""; parent = parents[parent] {
			if seen[parent] {
				return fmt.Errorf("composition region parent cycle at %s", parent)
			}
			seen[parent] = true
		}
	}
	return nil
}

func (s *service) GetCompositionBundle(ctx context.Context, c Composition) (CompositionBundle, error) {
	if err := validateComposition(c); err != nil {
		return CompositionBundle{}, err
	}
	imports := map[string]compositionImport{}
	var gaps []CompositionGap
	var declarations []components.ResolvedAsset
	sourceHashes := map[string]string{}
	resolve := func(key string, ref CompositionAsset) error {
		if ref.CatalogID == "" || !compositionVersion.MatchString(ref.Version) || !compositionIdentifier.MatchString(ref.Export) {
			return fmt.Errorf("%s requires catalog identity, exact version, and named export", key)
		}
		asset, err := s.components.Get(ctx, ref.CatalogID)
		if err != nil {
			return err
		}
		if asset.CatalogID != ref.CatalogID || !compositionIdentifier.MatchString(asset.Slug) {
			return fmt.Errorf("asset identity does not resolve to a published module")
		}
		closure, err := components.ResolveDependencyClosure(ctx, s.components, asset.ID, ref.Version)
		if err != nil {
			return err
		}
		for _, entry := range closure {
			v := entry.Version
			sourceHashes[entry.Asset.ID+"@"+v.Version] = v.ContentSHA256
			if (v.Status != components.VersionStatusReleased && v.Status != components.VersionStatusDeprecated && v.Status != components.VersionStatusArchived) || !v.DependencyLockPresent || v.ComponentID != entry.Asset.ID || (entry.Asset.ID == asset.ID && v.Version != ref.Version) {
				return fmt.Errorf("%s@%s lacks published dependency evidence", entry.Asset.LibraryID, v.Version)
			}
			if v.ContentSHA256 == "" || digest(v.Content) != v.ContentSHA256 {
				return fmt.Errorf("%s@%s entry digest differs from published source", entry.Asset.LibraryID, v.Version)
			}
			for _, file := range v.Files {
				sourceHashes[entry.Asset.ID+"@"+v.Version+"/"+file.Path] = file.ContentSHA256
				if file.ContentSHA256 == "" || digest(file.Content) != file.ContentSHA256 {
					return fmt.Errorf("%s@%s companion digest differs from published source", entry.Asset.LibraryID, v.Version)
				}
			}
			if err := s.verifyCompositionMaterialization(entry); err != nil {
				return err
			}
		}
		for _, entry := range closure {
			module := "@vrooli/react-component-library/" + entry.Asset.Slug + "/" + entry.Version.Version
			if _, ok := resolveLocalVrooliPackage(module, filepath.Join(s.repoRoot, "scenarios", "react-component-library", "library")); !ok {
				return fmt.Errorf("published module %s is not materialized for preview", module)
			}
		}
		selector := fmt.Sprintf("import { %s as Selected } from %s; export const selected = Selected;", ref.Export, jsonString("@vrooli/react-component-library/"+asset.Slug+"/"+ref.Version))
		if _, _, err := s.bundler.BuildBundle(ctx, selector, filepath.Join(s.repoRoot, "scenarios", "react-component-library", "library", "CompositionExport.generated.tsx")); err != nil {
			return fmt.Errorf("selected export is unavailable: %w", err)
		}
		declarations = append(declarations, closure...)
		imports[key] = compositionImport{Asset: ref, Slug: asset.Slug}
		return nil
	}
	if err := resolve("$template", c.Template); err != nil {
		return CompositionBundle{}, fmt.Errorf("composition template: %w", err)
	}
	for _, r := range c.Regions {
		if err := ctx.Err(); err != nil {
			return CompositionBundle{}, err
		}
		if r.Asset == nil {
			gaps = append(gaps, CompositionGap{r.ID, "asset_not_selected", "No published region asset is selected", r.Required})
			continue
		}
		if err := resolve(r.ID, *r.Asset); err != nil {
			gaps = append(gaps, CompositionGap{r.ID, "asset_unresolved", err.Error(), r.Required})
		}
	}
	parents := map[string]string{}
	for _, r := range c.Regions {
		parents[r.ID] = r.Parent
	}
	for _, r := range c.Regions {
		for parent := r.Parent; parent != ""; parent = parents[parent] {
			if _, ok := imports[parent]; !ok {
				gaps = append(gaps, CompositionGap{r.ID, "parent_unresolved", "Containing region " + parent + " is not renderable", r.Required})
				break
			}
		}
	}
	source, err := lowerComposition(c, imports)
	if err != nil {
		return CompositionBundle{}, err
	}
	provenance, err := compositionProvenance(c, declarations, gaps)
	if err != nil {
		return CompositionBundle{}, err
	}
	source += "\nexport const compositionProvenance = " + string(provenance) + " as const;\n"
	// Existing Preview bundling owns module resolution, dependency imports, and
	// export validation. The source is also the production layout artifact.
	sourcePath := filepath.Join(s.repoRoot, "scenarios", "react-component-library", "library", "Composition.generated.tsx")
	js, warnings, err := s.bundler.BuildBundle(ctx, source, sourcePath)
	if err != nil {
		return CompositionBundle{}, err
	}
	if err := ctx.Err(); err != nil {
		return CompositionBundle{}, err
	}
	// Source can change while esbuild runs. Recheck the successfully selected
	// closure before returning evidence associated with its indexed digests.
	for _, entry := range declarations {
		if err := s.verifyCompositionMaterialization(entry); err != nil {
			return CompositionBundle{}, err
		}
	}
	if len(js) > 8*1024*1024 {
		return CompositionBundle{}, fmt.Errorf("composition bundle exceeds 8 MiB")
	}
	inputs, err := json.Marshal(struct {
		Composition  Composition
		Imports      map[string]compositionImport
		Gaps         []CompositionGap
		SourceHashes map[string]string
	}{c, imports, gaps, sourceHashes})
	if err != nil {
		return CompositionBundle{}, err
	}
	out := CompositionBundle{Source: source, Revision: c.Revision, Gaps: gaps, Bundle: Bundle{JS: js, SHA256: digest(string(inputs) + "\n" + js), Warnings: warnings, SourcePath: sourcePath}}
	if s.deps != nil {
		seen := map[string]bool{}
		for _, entry := range declarations {
			key := entry.Asset.ID + "@" + entry.Version.Version
			if seen[key] {
				continue
			}
			seen[key] = true
			entryPath := entry.Version.SourcePath
			if filepath.IsAbs(entryPath) {
				entryPath, err = filepath.Rel(filepath.Join(s.repoRoot, "scenarios", "react-component-library", "library"), entryPath)
				if err != nil {
					return CompositionBundle{}, err
				}
			}
			files := append([]components.ComponentVersionFile(nil), entry.Version.Files...)
			files = append(files, components.ComponentVersionFile{Path: filepath.Base(entryPath), Content: entry.Version.Content})
			deps, err := s.previewDeclarations(ctx, entry.Asset.ID, entry.Version.Version, files, entryPath)
			if err != nil {
				return CompositionBundle{}, err
			}
			out.Dependencies = appendUniqueDeclarations(out.Dependencies, deps)
		}
	}
	return out, nil
}

// The index can retain the last valid release when a later refresh rejects
// changed files. Its internally consistent digest is therefore insufficient
// evidence for the source currently available to the filesystem bundler.
func (s *service) verifyCompositionMaterialization(entry components.ResolvedAsset) error {
	root := filepath.Join(s.repoRoot, "scenarios", "react-component-library", "library")
	path := entry.Version.SourcePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	verify := func(path, expected string) error {
		rel, err := filepath.Rel(root, path)
		if err != nil || !filepath.IsLocal(rel) {
			return fmt.Errorf("%s@%s materialized source is outside the library", entry.Asset.LibraryID, entry.Version.Version)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s@%s materialized source cannot be read (%s): %w", entry.Asset.LibraryID, entry.Version.Version, rel, err)
		}
		if digest(string(body)) != expected {
			return fmt.Errorf("%s@%s materialized source differs from published digest (%s)", entry.Asset.LibraryID, entry.Version.Version, rel)
		}
		return nil
	}
	if err := verify(path, entry.Version.ContentSHA256); err != nil {
		return err
	}
	for _, file := range entry.Version.Files {
		if !filepath.IsLocal(file.Path) {
			return fmt.Errorf("%s@%s published companion path is invalid", entry.Asset.LibraryID, entry.Version.Version)
		}
		if err := verify(filepath.Join(filepath.Dir(path), file.Path), file.ContentSHA256); err != nil {
			return err
		}
	}
	return nil
}

func lowerComposition(c Composition, imports map[string]compositionImport) (string, error) {
	if err := validateComposition(c); err != nil {
		return "", err
	}
	if _, ok := imports["$template"]; !ok {
		return "", fmt.Errorf("resolved template is required")
	}
	var source strings.Builder
	source.WriteString("// Generated composition. Supply business bindings from an authored module.\nimport * as React from 'react';\n")
	keys := []string{"$template"}
	for _, r := range c.Regions {
		if _, ok := imports[r.ID]; ok {
			keys = append(keys, r.ID)
		}
	}
	aliases := map[string]string{}
	for i, key := range keys {
		ref := imports[key]
		if !compositionIdentifier.MatchString(ref.Slug) || !compositionIdentifier.MatchString(ref.Asset.Export) || !compositionVersion.MatchString(ref.Asset.Version) {
			return "", fmt.Errorf("invalid resolved composition module")
		}
		alias := fmt.Sprintf("Asset%d", i)
		aliases[key] = alias
		fmt.Fprintf(&source, "import { %s as %s } from %s;\n", ref.Asset.Export, alias, jsonString("@vrooli/react-component-library/"+ref.Slug+"/"+ref.Asset.Version))
	}
	trees := map[string]*slotTree{"": {children: map[string]*slotTree{}}}
	children := map[string][]CompositionRegion{}
	for _, r := range c.Regions {
		children[r.Parent] = append(children[r.Parent], r)
		if trees[r.Parent] == nil {
			trees[r.Parent] = &slotTree{children: map[string]*slotTree{}}
		}
		node := trees[r.Parent]
		for _, part := range r.Slot {
			if node.children[part] == nil {
				node.children[part] = &slotTree{children: map[string]*slotTree{}}
			}
			node = node.children[part]
		}
		node.region = r.ID
	}
	source.WriteString(`type CompositionSlotInput<T, K extends keyof T, V> = Omit<T, K> &
 ({} extends V ? { [P in K]?: V } : { [P in K]: V });
`)
	source.WriteString("export type CompositionBindings = {\n  '$labels': { missing: string; failed: string };\n")
	for _, key := range keys {
		props := "React.ComponentProps<typeof " + aliases[key] + ">"
		parent := key
		if key == "$template" {
			parent = ""
		}
		if tree := trees[parent]; tree != nil {
			props = tree.inputType(props)
		}
		fmt.Fprintf(&source, "  %s: %s;\n", jsonString(key), props)
	}
	source.WriteString("};\n")
	source.WriteString(`class RegionBoundary extends React.Component<{region:string; failed:string; children:React.ReactNode},{failed:boolean}> {
 state = {failed:false};
 static getDerivedStateFromError() { return {failed:true}; }
 render() { return this.state.failed ? <div role="alert" data-design-region={this.props.region} data-design-render="failed">{this.props.failed}</div> : this.props.children; }
}
export function Composition({bindings}:{bindings:CompositionBindings}) {
 const regions: Record<string,React.ReactNode> = {};
`)
	var emit func(string)
	emit = func(parent string) {
		for _, r := range children[parent] {
			emit(r.ID)
			props := "bindings[" + jsonString(r.ID) + "]"
			if tree := trees[r.ID]; tree != nil {
				props = tree.expression(props)
			}
			if alias, ok := aliases[r.ID]; ok {
				fmt.Fprintf(&source, " regions[%s] = <RegionBoundary key=%s region=%s failed={bindings.$labels.failed}><div data-design-region=%s data-design-render=\"asset\"><%s {...%s} /></div></RegionBoundary>;\n", jsonString(r.ID), jsonString(r.ID), jsonString(r.ID), jsonString(r.ID), alias, props)
			} else {
				fmt.Fprintf(&source, " regions[%s] = <div role=\"status\" data-design-region=%s data-design-render=\"wireframe\">{bindings.$labels.missing}</div>;\n", jsonString(r.ID), jsonString(r.ID))
			}
		}
	}
	emit("")
	tree := trees[""]
	fmt.Fprintf(&source, " const templateProps = %s;\n return <RegionBoundary region=\"$template\" failed={bindings.$labels.failed}><Asset0 {...templateProps} /></RegionBoundary>;\n}\nexport default Composition;\n", tree.expression("bindings[\"$template\"]"))
	return source.String(), nil
}

type slotTree struct {
	region   string
	children map[string]*slotTree
}

// inputType removes compiler-owned leaves while retaining business props and
// required siblings. An ancestor becomes optional only when its remaining
// input can be empty. Generated children instantiate even optional ancestors.
func (t *slotTree) inputType(base string) string {
	keys := make([]string, 0, len(t.children))
	for key := range t.children {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := base
	for _, key := range keys {
		child := t.children[key]
		if child.region != "" {
			result = "Omit<" + result + ", " + jsonString(key) + ">"
		} else {
			nested := child.inputType("NonNullable<" + base + "[" + jsonString(key) + "]>")
			result = "CompositionSlotInput<" + result + ", " + jsonString(key) + ", " + nested + ">"
		}
	}
	return result
}

func (t *slotTree) expression(base string) string {
	if t.region != "" {
		return "regions[" + jsonString(t.region) + "]"
	}
	keys := make([]string, 0, len(t.children))
	for key := range t.children {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := []string{"...(" + base + " ?? {})"}
	for _, key := range keys {
		fields = append(fields, jsonString(key)+": "+t.children[key].expression(base+"?.["+jsonString(key)+"]"))
	}
	return "{" + strings.Join(fields, ", ") + "}"
}
func jsonString(s string) string { raw, _ := json.Marshal(s); return string(raw) }

// CompositionGeneratorVersion identifies the source-lowering contract, including
// nested slot ownership. Change it when emitted production semantics change.
const CompositionGeneratorVersion = "composition/2"

type compositionAssetRecord struct {
	LibraryID    string                        `json:"libraryId"`
	Version      string                        `json:"version"`
	EntryHash    string                        `json:"entryHash"`
	Files        map[string]string             `json:"files"`
	Dependencies []compositionDependencyRecord `json:"dependencies"`
}
type compositionDependencyRecord struct {
	LibraryID string `json:"libraryId"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
}

type compositionGapRecord struct {
	Region   string `json:"region"`
	Code     string `json:"code"`
	Required bool   `json:"required"`
}

func compositionProvenance(c Composition, declarations []components.ResolvedAsset, gaps []CompositionGap) ([]byte, error) {
	records := map[string]compositionAssetRecord{}
	for _, entry := range declarations {
		record := compositionAssetRecord{LibraryID: entry.Asset.LibraryID, Version: entry.Version.Version, EntryHash: entry.Version.ContentSHA256, Files: map[string]string{}, Dependencies: []compositionDependencyRecord{}}
		for _, file := range entry.Version.Files {
			record.Files[file.Path] = file.ContentSHA256
		}
		for _, dep := range entry.Version.Dependencies {
			record.Dependencies = append(record.Dependencies, compositionDependencyRecord{dep.LibraryID, dep.Version, string(dep.Kind)})
		}
		sort.Slice(record.Dependencies, func(i, j int) bool {
			a, b := record.Dependencies[i], record.Dependencies[j]
			return a.LibraryID+"@"+a.Version+":"+a.Kind < b.LibraryID+"@"+b.Version+":"+b.Kind
		})
		key := record.LibraryID + "@" + record.Version
		if previous, ok := records[key]; ok {
			a, _ := json.Marshal(previous)
			b, _ := json.Marshal(record)
			if string(a) != string(b) {
				return nil, fmt.Errorf("conflicting provenance for %s", key)
			}
		}
		records[key] = record
	}
	keys := make([]string, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	assets := make([]compositionAssetRecord, 0, len(keys))
	for _, key := range keys {
		assets = append(assets, records[key])
	}
	input, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	unresolved := make([]compositionGapRecord, 0, len(gaps))
	for _, gap := range gaps {
		unresolved = append(unresolved, compositionGapRecord{gap.Region, gap.Code, gap.Required})
	}
	// No host paths, timestamps, or preview fixtures: this describes generated
	// layout/source provenance, not behavior, appearance, or acceptance evidence.
	return json.Marshal(struct {
		SchemaVersion    int                      `json:"schemaVersion"`
		GeneratorVersion string                   `json:"generatorVersion"`
		Revision         string                   `json:"revision"`
		CompositionHash  string                   `json:"compositionHash"`
		Composition      Composition              `json:"composition"`
		Assets           []compositionAssetRecord `json:"assets"`
		Unresolved       []compositionGapRecord   `json:"unresolved"`
	}{1, CompositionGeneratorVersion, c.Revision, digest(string(input)), c, assets, unresolved})
}
