package checks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"ui-health/internal/uiinterop"
)

type governedComponent struct {
	Name      string
	LibraryID string
	Version   string
	Source    string
}

var componentPrimitiveMap = map[string][]string{
	"Button":    {"button"},
	"DataTable": {"table"},
	"Input":     {"input", "textarea"},
	"Select":    {"select"},
}

func governedComponents(ctx uiinterop.CheckContext, files []uiinterop.SourceFile) map[string]governedComponent {
	components := map[string]governedComponent{}
	for _, f := range files {
		source := strings.TrimSpace(provenanceField(f.Content, "@vrooliComponentSource"))
		if source == "" {
			continue
		}
		name := strings.TrimPrefix(source, "react-component-library:")
		if _, ok := componentPrimitiveMap[name]; !ok {
			continue
		}
		components[name] = governedComponent{
			Name:      name,
			LibraryID: source,
			Version:   strings.TrimSpace(provenanceField(f.Content, "@vrooliComponentVersion")),
			Source:    "adopted locally",
		}
	}

	if declaresComponentLibraryIntent(ctx) {
		for name := range componentPrimitiveMap {
			if _, ok := components[name]; ok {
				continue
			}
			components[name] = governedComponent{
				Name:      name,
				LibraryID: "react-component-library:" + name,
				Source:    "declared component-library intent",
			}
		}
	}

	repoRoot := findRepoRoot(ctx.ScenarioRoot)
	if repoRoot == "" {
		return components
	}
	catalogDir := filepath.Join(repoRoot, "scenarios", "react-component-library", "library", "components")
	entries, err := os.ReadDir(catalogDir)
	if err != nil {
		return components
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, ok := componentPrimitiveMap[name]; !ok {
			continue
		}
		meta := readCatalogComponent(filepath.Join(catalogDir, name, "component.json"), name)
		if existing, ok := components[name]; ok && existing.Version != "" {
			continue
		}
		components[name] = meta
	}
	return components
}

func declaresComponentLibraryIntent(ctx uiinterop.CheckContext) bool {
	return packageDeclaresComponentLibrary(ctx.ScenarioRoot) ||
		serviceDeclaresReactViteDesignAdapter(ctx.ScenarioRoot) ||
		uiManifestDeclaresReactViteTemplate(ctx.ScenarioRoot)
}

func packageDeclaresComponentLibrary(scenarioRoot string) bool {
	data, err := os.ReadFile(filepath.Join(scenarioRoot, "ui", "package.json"))
	if err != nil {
		return false
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return false
	}
	for dep := range pkg.Dependencies {
		if isComponentLibraryDependency(dep) {
			return true
		}
	}
	for dep := range pkg.DevDependencies {
		if isComponentLibraryDependency(dep) {
			return true
		}
	}
	return false
}

func isComponentLibraryDependency(dep string) bool {
	dep = strings.TrimSpace(dep)
	return dep == "react-component-library" ||
		dep == "@vrooli/react-component-library" ||
		dep == "@vrooli/component-library"
}

func serviceDeclaresReactViteDesignAdapter(scenarioRoot string) bool {
	data, err := os.ReadFile(filepath.Join(scenarioRoot, ".vrooli", "service.json"))
	if err != nil {
		return false
	}
	var service struct {
		Generation struct {
			Design struct {
				Adapter string `json:"adapter"`
			} `json:"design"`
		} `json:"generation"`
	}
	if json.Unmarshal(data, &service) != nil {
		return false
	}
	return strings.TrimSpace(service.Generation.Design.Adapter) == "react-vite-tailwind"
}

func uiManifestDeclaresReactViteTemplate(scenarioRoot string) bool {
	data, err := os.ReadFile(filepath.Join(scenarioRoot, "ui", "manifest.json"))
	if err != nil {
		return false
	}
	var manifest struct {
		Contract struct {
			Template string `json:"template"`
		} `json:"contract"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return false
	}
	return strings.TrimSpace(manifest.Contract.Template) == "react-vite"
}

func readCatalogComponent(path, fallbackName string) governedComponent {
	meta := governedComponent{Name: fallbackName, LibraryID: "react-component-library:" + fallbackName, Source: "react-component-library catalog"}
	data, err := os.ReadFile(path)
	if err != nil {
		return meta
	}
	var doc struct {
		LibraryID string `json:"libraryId"`
		Latest    string `json:"latest"`
	}
	if json.Unmarshal(data, &doc) == nil {
		if doc.LibraryID != "" {
			meta.LibraryID = doc.LibraryID
		}
		meta.Version = doc.Latest
	}
	return meta
}

func provenanceField(source, key string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(key) + `\s+([^\n\r*]+)`)
	m := re.FindStringSubmatch(source)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func findRepoRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			return ""
		}
		dir = next
	}
}

func primitiveComponentIndex(components map[string]governedComponent) map[string]governedComponent {
	out := map[string]governedComponent{}
	for name, component := range components {
		for _, primitive := range componentPrimitiveMap[name] {
			out[primitive] = component
		}
	}
	return out
}

func sortedComponentNames(components map[string]governedComponent) []string {
	names := make([]string, 0, len(components))
	for name := range components {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
