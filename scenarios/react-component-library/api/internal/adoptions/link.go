package adoptions

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"react-component-library/internal/components"
)

const (
	linkedPackageName    = "@vrooli/react-component-library"
	linkedPackagePath    = "file:../../../packages/react-component-library"
	selectorBridgeStart  = "// vrooli:library-selectors start"
	stringsProviderStart = "// vrooli:library-strings-provider start"
)

// Link records a package-backed adoption and updates only the adopter's
// package manifest. The library source is never copied into the scenario.
func (s *service) Link(ctx context.Context, in LinkInput) (LinkResult, error) {
	in.ComponentID = strings.TrimSpace(in.ComponentID)
	in.Scenario = strings.TrimSpace(in.Scenario)
	if in.ComponentID == "" || in.Scenario == "" {
		return LinkResult{}, ErrInvalidAdoption{Field: "component_id/scenario", Reason: "required"}
	}
	component, err := s.library.Get(ctx, in.ComponentID)
	if err != nil {
		return LinkResult{}, err
	}
	version := strings.TrimSpace(in.Version)
	if version == "" {
		version = firstNonEmpty(component.LatestVersion, component.Version)
	}
	if version == "" {
		return LinkResult{}, ErrInvalidAdoption{Field: "version", Reason: "component has no latest version"}
	}
	if err := s.ensureVersionMaterialized(ctx, component.ID, version); err != nil {
		return LinkResult{}, err
	}
	versionInfo, err := s.library.GetVersion(ctx, component.ID, version)
	if err != nil {
		return LinkResult{}, err
	}
	importSubpath := strings.TrimSpace(in.ImportSubpath)
	if importSubpath == "" {
		importSubpath = canonicalImportSubpath(component, version)
	}
	if !strings.HasPrefix(importSubpath, "./") {
		return LinkResult{}, ErrInvalidAdoption{Field: "import_subpath", Reason: "must be a package-relative export beginning with ./"}
	}

	packageBytes, err := s.files.Read(ctx, in.Scenario, "ui/package.json")
	if err != nil {
		return LinkResult{}, fmt.Errorf("read adopter package manifest: %w", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(packageBytes, &manifest); err != nil {
		return LinkResult{}, fmt.Errorf("decode adopter package manifest: %w", err)
	}
	dependencies, _ := manifest["dependencies"].(map[string]any)
	if dependencies == nil {
		dependencies = map[string]any{}
		manifest["dependencies"] = dependencies
	}
	if existing, ok := dependencies[linkedPackageName]; ok && existing != linkedPackagePath && !in.ConfirmExisting {
		return LinkResult{}, ErrInvalidAdoption{Field: "confirm_existing", Reason: "adopter already declares a different react component library dependency"}
	}
	dependencies[linkedPackageName] = linkedPackagePath
	updatedPackage, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return LinkResult{}, fmt.Errorf("encode adopter package manifest: %w", err)
	}
	updatedPackage = append(updatedPackage, '\n')
	if _, err := s.files.Write(ctx, in.Scenario, "ui/package.json", updatedPackage); err != nil {
		return LinkResult{}, fmt.Errorf("write adopter package manifest: %w", err)
	}
	updatedFiles := []string{"ui/package.json"}
	versionSource := versionInfo.Content
	for _, file := range versionInfo.Files {
		versionSource += "\n" + file.Content
	}
	if reader, ok := s.library.(VersionFileReader); ok {
		entry := strings.TrimSuffix(filepath.Base(versionInfo.SourcePath), filepath.Ext(versionInfo.SourcePath))
		for _, candidate := range []string{entry + ".strings.ts", component.Slug + ".strings.ts"} {
			content, readErr := reader.GetVersionContentAt(ctx, component.ID, version, candidate)
			if readErr == nil {
				versionSource += "\n" + content.Body
				break
			}
		}
	}
	if err := mergeLocaleCatalog(ctx, s.files, in.Scenario, derivedLocaleEntries(versionSource)); err != nil {
		return LinkResult{}, err
	}
	if _, err := s.files.Exists(ctx, in.Scenario, "ui/src/i18n/locales/en.json"); err == nil {
		updatedFiles = append(updatedFiles, "ui/src/i18n/locales/en.json")
	}

	selectorPath := "ui/src/consts/selectors.library.ts"
	selectorExists, err := s.files.Exists(ctx, in.Scenario, selectorPath)
	if err != nil {
		return LinkResult{}, fmt.Errorf("check adopter selector registry: %w", err)
	}
	var selectorSource []byte
	if selectorExists {
		selectorSource, err = s.files.Read(ctx, in.Scenario, selectorPath)
		if err != nil {
			return LinkResult{}, fmt.Errorf("read adopter selector registry: %w", err)
		}
	}
	selectorKey := firstNonEmpty(component.CatalogID, strings.ToLower(component.Slug))
	selectorText, selectorChanged := mergeSelectorRegion(string(selectorSource), selectorKey, derivedSelectorIDs(versionSource, selectorKey))
	if selectorChanged {
		if _, writeErr := s.files.Write(ctx, in.Scenario, selectorPath, []byte(selectorText)); writeErr != nil {
			return LinkResult{}, fmt.Errorf("write adopter selector registry: %w", writeErr)
		}
		updatedFiles = append(updatedFiles, selectorPath)
	}
	if changed, err := composeSelectorRegistry(ctx, s.files, in.Scenario); err != nil {
		return LinkResult{}, err
	} else if changed {
		updatedFiles = append(updatedFiles, "ui/src/consts/selectors.ts")
	}
	if changed, err := mountLibraryStringsProvider(ctx, s.files, in.Scenario); err != nil {
		return LinkResult{}, err
	} else if changed {
		updatedFiles = append(updatedFiles, "ui/src/main.tsx")
	}

	var adoption Adoption
	rows, err := s.repo.List(ctx, ListQuery{ComponentID: component.ID, Scenario: in.Scenario, Limit: 50})
	if err != nil {
		return LinkResult{}, err
	}
	for _, row := range rows {
		if row.ComponentID != component.ID {
			continue
		}
		oldFiles := append([]AdoptionFile(nil), row.Files...)
		if rewriter, ok := s.files.(ScenarioImportRewriter); ok && row.AdoptedPath != "" {
			importSites, rewriteErr := rewriter.RewriteImportSites(ctx, in.Scenario, row.AdoptedPath, linkedPackageName+importSubpath[1:])
			if rewriteErr != nil {
				return LinkResult{}, fmt.Errorf("rewrite imports for %s: %w", row.AdoptedPath, rewriteErr)
			}
			updatedFiles = append(updatedFiles, importSites...)
		}
		adoption, err = s.repo.UpdateLinked(ctx, row.ID, importSubpath, version, versionInfo.ContentSHA256)
		if err != nil {
			return LinkResult{}, err
		}
		if remover, ok := s.files.(ScenarioFileRemover); ok && len(oldFiles) > 0 {
			shared := map[string]struct{}{}
			for _, other := range rows {
				if other.ID == row.ID {
					continue
				}
				for _, file := range other.Files {
					shared[file.AdoptedPath] = struct{}{}
				}
			}
			for _, file := range oldFiles {
				if _, exists := shared[file.AdoptedPath]; exists {
					continue
				}
				if removeErr := remover.Remove(ctx, in.Scenario, file.AdoptedPath); removeErr != nil {
					return LinkResult{}, fmt.Errorf("remove copied file %q after linking: %w", file.AdoptedPath, removeErr)
				}
			}
		}
	}
	if adoption.ID != "" {
		return LinkResult{Adoption: adoption, PackagePath: linkedPackagePath, ImportSubpath: importSubpath, UpdatedFiles: updatedFiles}, nil
	}

	adoption, err = s.repo.Create(ctx, CreateInput{
		ComponentID:    component.ID,
		LibraryID:      component.LibraryID,
		Scenario:       in.Scenario,
		AdoptedPath:    importSubpath,
		AdoptedVersion: version,
		SourceSHA256:   versionInfo.ContentSHA256,
		Mode:           AdoptionModeLinked,
	})
	if err != nil {
		return LinkResult{}, err
	}
	return LinkResult{Adoption: adoption, PackagePath: linkedPackagePath, ImportSubpath: importSubpath, UpdatedFiles: updatedFiles}, nil
}

func (s *service) Eject(ctx context.Context, in EjectInput) (ApplyResult, error) {
	if strings.TrimSpace(in.Reason) == "" {
		return ApplyResult{}, ErrInvalidAdoption{Field: "reason", Reason: "required for ejected adoption"}
	}
	in.ApplyInput.ForkReason = strings.TrimSpace(in.Reason)
	prior, err := s.repo.List(ctx, ListQuery{ComponentID: in.ApplyInput.ComponentID, Scenario: in.ApplyInput.Scenario, Limit: 100})
	if err != nil {
		return ApplyResult{}, err
	}
	for _, row := range prior {
		if row.AdoptedPath != in.ApplyInput.AdoptedPath || (row.Mode != AdoptionModeCopied && row.Mode != AdoptionModeEjected) {
			continue
		}
		updated, updateErr := s.repo.UpdateMode(ctx, row.ID, AdoptionModeEjected, in.Reason)
		if updateErr != nil {
			return ApplyResult{}, updateErr
		}
		return ApplyResult{Adoption: updated, WrittenPath: in.ApplyInput.AdoptedPath}, nil
	}
	result, err := s.Apply(ctx, in.ApplyInput)
	if err != nil {
		return ApplyResult{}, err
	}
	updated, err := s.repo.UpdateMode(ctx, result.Adoption.ID, AdoptionModeEjected, in.Reason)
	if err != nil {
		return ApplyResult{}, err
	}
	result.Adoption = updated
	// Apply is the historical copy path and creates a row. Collapse any older
	// row for the same pinned adoption so eject is a mode transition, not a
	// second competing adoption record. This also repairs rows produced by the
	// pre-transition implementation.
	for _, row := range prior {
		if row.ID == updated.ID || row.ComponentID != updated.ComponentID || row.Scenario != updated.Scenario || row.AdoptedPath != updated.AdoptedPath {
			continue
		}
		if err := s.repo.Delete(ctx, row.ID); err != nil {
			return ApplyResult{}, fmt.Errorf("remove superseded adoption %s: %w", row.ID, err)
		}
	}
	return result, nil
}

func canonicalImportSubpath(component components.Component, version string) string {
	return "./" + component.Slug + "/" + version
}

var (
	stringsDeclarationPattern = regexp.MustCompile(`(?m)["']([^"']+)["']\s*:\s*["']([^"']*)["']\s*,?`)
	selectorLiteral           = regexp.MustCompile("data-testid\\s*=\\s*[\\\"'`]([^\\\"'`]+)[\\\"'`]")
)

func derivedLocaleEntries(source string) map[string]string {
	entries := map[string]string{}
	for _, match := range stringsDeclarationPattern.FindAllStringSubmatch(source, -1) {
		if len(match) > 2 {
			entries[match[1]] = match[2]
		}
	}
	return entries
}

func mergeLocaleCatalog(ctx context.Context, files ScenarioFileWriter, scenario string, entries map[string]string) error {
	if len(entries) == 0 {
		return nil
	}
	path := "ui/src/i18n/locales/en.json"
	exists, err := files.Exists(ctx, scenario, path)
	if err != nil {
		return fmt.Errorf("check adopter English catalog: %w", err)
	}
	var raw []byte
	if exists {
		raw, err = files.Read(ctx, scenario, path)
		if err != nil {
			return fmt.Errorf("read adopter English catalog: %w", err)
		}
	} else {
		raw = []byte("{}")
	}
	var catalog map[string]any
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return fmt.Errorf("decode adopter English catalog: %w", err)
	}
	changed := false
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts := strings.Split(key, ".")
		cursor := catalog
		for _, part := range parts[:len(parts)-1] {
			next, ok := cursor[part].(map[string]any)
			if !ok {
				next = map[string]any{}
				cursor[part] = next
			}
			cursor = next
		}
		leaf := parts[len(parts)-1]
		if _, exists := cursor[leaf]; !exists {
			cursor[leaf] = entries[key]
			changed = true
		}
	}
	if !changed {
		return nil
	}
	updated, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("encode adopter English catalog: %w", err)
	}
	updated = append(updated, '\n')
	if _, err := files.Write(ctx, scenario, path, updated); err != nil {
		return fmt.Errorf("write adopter English catalog: %w", err)
	}
	return nil
}

func derivedSelectorIDs(source, root string) []string {
	seen := map[string]struct{}{root: {}}
	for _, match := range selectorLiteral.FindAllStringSubmatch(source, -1) {
		if len(match) > 1 {
			value := strings.TrimSpace(match[1])
			if value != "" {
				seen[canonicalSelectorID(root, value)] = struct{}{}
			}
		}
	}
	ids := []string{root}
	for id := range seen {
		if id != root {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids[1:])
	return ids
}

// canonicalSelectorID gives every emitted selector one namespace: the catalog
// id, followed by a semantic dotted suffix. Existing kebab-case test ids are
// accepted as input so the linker can migrate adopters without changing BAS
// flow semantics by hand.
func canonicalSelectorID(root, value string) string {
	if value == "" || value == root {
		return root
	}
	if strings.HasPrefix(value, root+".") {
		suffix := strings.TrimPrefix(value, root+".")
		if strings.HasPrefix(suffix, "shell") && len(suffix) > len("shell") && suffix[len("shell")] >= 'A' && suffix[len("shell")] <= 'Z' {
			suffix = strings.ToLower(suffix[len("shell"):len("shell")+1]) + suffix[len("shell")+1:]
		}
		return root + "." + suffix
	}
	last := root
	if dot := strings.LastIndexByte(root, '.'); dot >= 0 {
		last = root[dot+1:]
	}
	value = strings.ReplaceAll(value, "_", "-")
	if value == last || value == last+"-shell" {
		return root
	}
	if strings.HasPrefix(value, last+"-") {
		value = strings.TrimPrefix(value, last+"-")
	}
	value = strings.TrimPrefix(value, "shell-")
	if value == "" {
		return root
	}
	return root + "." + value
}

func mergeSelectorRegion(source, key string, ids []string) (string, bool) {
	start := strings.Index(source, selectorBridgeStart)
	end := strings.Index(source, "// vrooli:library-selectors end")
	entryName := strconv.Quote(key)
	if start < 0 || end < start {
		body := "// vrooli:library-selectors start\nexport const librarySelectors = {\n"
		body += selectorEntry(entryName, key, ids)
		body += "} as const;\n// vrooli:library-selectors end\n"
		return source + "\n" + body, true
	}
	regionEnd := end + len("// vrooli:library-selectors end")
	region := source[start:regionEnd]
	needle := "} as const;"
	close := strings.LastIndex(region, needle)
	if close < 0 {
		return source, false
	}
	entry := selectorEntry(entryName, key, ids)
	entryStart := strings.Index(region, entryName+":")
	if entryStart < 0 && regexp.MustCompile(`^[A-Za-z_$][\w$]*$`).MatchString(key) {
		entryStart = strings.Index(region, key+":")
	}
	if entryStart >= 0 {
		entryEnd := selectorObjectEnd(region, entryStart)
		if entryEnd < 0 {
			return source, false
		}
		region = region[:entryStart] + entry + region[entryEnd:]
	} else {
		region = region[:close] + entry + region[close:]
	}
	return source[:start] + region + source[regionEnd:], true
}

func selectorObjectEnd(source string, start int) int {
	open := strings.IndexByte(source[start:], '{')
	if open < 0 {
		return -1
	}
	open += start
	depth := 0
	quote := byte(0)
	for index := open; index < len(source); index++ {
		char := source[index]
		if quote != 0 {
			if char == '\\' {
				index++
			} else if char == quote {
				quote = 0
			}
			continue
		}
		if char == '"' || char == '\'' || char == '`' {
			quote = char
			continue
		}
		switch char {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				if index+1 < len(source) && source[index+1] == ',' {
					return index + 2
				}
				return index + 1
			}
		}
	}
	return -1
}

func selectorFieldName(root, id string) string {
	if id == root {
		return "root"
	}
	suffix := strings.TrimPrefix(id, root+".")
	parts := strings.FieldsFunc(suffix, func(r rune) bool {
		return r == '.' || r == '-' || r == '_' || r == ' '
	})
	if len(parts) == 0 {
		return "root"
	}
	field := strings.ToLower(parts[0][:1]) + parts[0][1:]
	for _, part := range parts[1:] {
		if part == "" {
			continue
		}
		field += strings.ToUpper(part[:1]) + part[1:]
	}
	if field == "" || field[0] < 'a' || field[0] > 'z' {
		return "selector"
	}
	return field
}

func selectorEntry(name, root string, ids []string) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "  %s: {\n", name)
	used := map[string]int{}
	for _, id := range ids {
		field := selectorFieldName(root, id)
		used[field]++
		if used[field] > 1 {
			field = fmt.Sprintf("%s%d", field, used[field])
		}
		fmt.Fprintf(&builder, "    %s: %s,\n", strconv.Quote(field), strconv.Quote(id))
	}
	builder.WriteString("  },\n")
	return builder.String()
}

func composeSelectorRegistry(ctx context.Context, files ScenarioFileWriter, scenario string) (bool, error) {
	path := "ui/src/consts/selectors.ts"
	exists, err := files.Exists(ctx, scenario, path)
	if err != nil {
		return false, fmt.Errorf("check adopter selector composition: %w", err)
	}
	if !exists {
		body := "import { librarySelectors } from \"./selectors.library\";\n\n"
		body += "export { librarySelectors };\n"
		body += "export const selectors = { library: librarySelectors } as const;\n"
		body += "export const selectorsManifest = { selectors: librarySelectors } as const;\n"
		if _, err := files.Write(ctx, scenario, path, []byte(body)); err != nil {
			return false, fmt.Errorf("write adopter selector composition: %w", err)
		}
		return true, nil
	}
	raw, err := files.Read(ctx, scenario, path)
	if err != nil {
		return false, fmt.Errorf("read adopter selector composition: %w", err)
	}
	source := string(raw)
	updated := source
	if !librarySelectorsImportPresent(updated) {
		updated = "import { librarySelectors } from \"./selectors.library\";\n" + updated
	}
	if !strings.Contains(updated, "export { librarySelectors }") {
		updated = strings.Replace(updated, "\n", "\nexport { librarySelectors };\n", 1)
	}
	updated = strings.Replace(updated, "createSelectorRegistry(literalSelectors", "createSelectorRegistry({ library: librarySelectors, ...literalSelectors },", 1)
	if updated == source {
		return false, nil
	}
	if _, err := files.Write(ctx, scenario, path, []byte(updated)); err != nil {
		return false, fmt.Errorf("write adopter selector composition: %w", err)
	}
	return true, nil
}

func librarySelectorsImportPresent(source string) bool {
	return regexp.MustCompile(`(?m)from\s+["']\./selectors\.library(?:\.[cm]?[jt]sx?)?["']`).MatchString(source)
}

// mountLibraryStringsProvider makes the adopter's existing i18n translator
// available to package-backed library components. The generated wrapper is
// deliberately idempotent and lives at the application root, so a linked
// component does not need a private locale bridge or a copied source tree.
func mountLibraryStringsProvider(ctx context.Context, files ScenarioFileWriter, scenario string) (bool, error) {
	path := "ui/src/main.tsx"
	exists, err := files.Exists(ctx, scenario, path)
	if err != nil {
		return false, fmt.Errorf("check adopter application root: %w", err)
	}
	if !exists {
		return false, nil
	}
	raw, err := files.Read(ctx, scenario, path)
	if err != nil {
		return false, fmt.Errorf("read adopter application root: %w", err)
	}
	source := string(raw)
	if strings.Contains(source, stringsProviderStart) {
		return false, nil
	}
	providerImport := `import { LibraryStringsProvider } from "@vrooli/react-component-library/useLocale/1.0.1";`
	if !strings.Contains(source, "LibraryStringsProvider") {
		source = providerImport + "\n" + source
	}
	if !strings.Contains(source, `from "./i18n"`) && !strings.Contains(source, `from './i18n'`) {
		source = `import { i18n } from "./i18n";` + "\n" + source
	}
	open := "\n    " + stringsProviderStart + "\n    <LibraryStringsProvider translate={(key, fallback) => i18n.t(key, { defaultValue: fallback })}>"
	close := "\n    </LibraryStringsProvider>\n    // vrooli:library-strings-provider end\n"
	if renderOpen, renderClose, ok := findReactRender(source); ok {
		source = source[:renderOpen+1] + open + source[renderOpen+1:]
		renderClose += len(open)
		source = source[:renderClose] + close + source[renderClose:]
	} else if wrapperOpen, wrapperClose, ok := findRootWrapper(source); ok {
		source = source[:wrapperOpen+1] + open + source[wrapperOpen+1:]
		wrapperClose += len(open)
		source = source[:wrapperClose] + close + source[wrapperClose:]
	} else {
		return false, fmt.Errorf("adopter application root does not contain a React render call or rootWrapper")
	}
	if _, err := files.Write(ctx, scenario, path, []byte(source)); err != nil {
		return false, fmt.Errorf("write adopter library strings provider: %w", err)
	}
	return true, nil
}

func findReactRender(source string) (int, int, bool) {
	for index := 0; index < len(source); index++ {
		index = skipJavaScriptTrivia(source, index)
		if index >= len(source) {
			break
		}
		if strings.HasPrefix(source[index:], ".render(") {
			open := index + len(".render")
			if close, ok := matchingParenthesis(source, open); ok {
				return open, close, true
			}
		}
	}
	return 0, 0, false
}

func findRootWrapper(source string) (int, int, bool) {
	start := strings.Index(source, "rootWrapper:")
	if start < 0 {
		return 0, 0, false
	}
	arrow := strings.Index(source[start:], "=> (")
	if arrow < 0 {
		return 0, 0, false
	}
	open := start + arrow + len("=> ")
	close, ok := matchingParenthesis(source, open)
	return open, close, ok
}

func matchingParenthesis(source string, open int) (int, bool) {
	if open >= len(source) || source[open] != '(' {
		return 0, false
	}
	depth := 0
	for index := open; index < len(source); index++ {
		index = skipJavaScriptTrivia(source, index)
		if index >= len(source) {
			break
		}
		switch source[index] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return index, true
			}
		}
	}
	return 0, false
}

func skipJavaScriptTrivia(source string, index int) int {
	for index < len(source) {
		switch source[index] {
		case '\'', '"', '`':
			quote := source[index]
			index++
			for index < len(source) {
				if source[index] == '\\' {
					index += 2
					continue
				}
				if source[index] == quote {
					index++
					break
				}
				index++
			}
		case '/':
			if index+1 >= len(source) || source[index+1] != '/' && source[index+1] != '*' {
				return index
			}
			if source[index+1] == '/' {
				index += 2
				for index < len(source) && source[index] != '\n' {
					index++
				}
				continue
			}
			index += 2
			for index+1 < len(source) && !(source[index] == '*' && source[index+1] == '/') {
				index++
			}
			if index+1 < len(source) {
				index += 2
			}
		default:
			return index
		}
	}
	return index
}
