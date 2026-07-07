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
	"BottomNav": {"nav"},
	"Button":    {"button"},
	"DataTable": {"table"},
	"Input":     {"input", "textarea"},
	"Select":    {"select"},
}

func governedComponents(ctx uiinterop.CheckContext, files []uiSourceFile) map[string]governedComponent {
	components := map[string]governedComponent{}
	for _, f := range files {
		source := strings.TrimSpace(provenanceField(f.content, "@vrooliComponentSource"))
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
			Version:   strings.TrimSpace(provenanceField(f.content, "@vrooliComponentVersion")),
			Source:    "adopted locally",
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
