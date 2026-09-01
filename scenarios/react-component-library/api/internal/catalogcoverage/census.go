package catalogcoverage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ShapeRow struct {
	Shape []string `json:"shape"`
	Count int      `json:"count"`
}

type DuplicationRow struct {
	Asset    string `json:"asset"`
	Field    string `json:"field"`
	Catalog  any    `json:"catalog"`
	Manifest any    `json:"manifest"`
}

func ShapeCensus(libraryRoot string) ([]ShapeRow, error) {
	counts := map[string]int{}
	if err := walkLiveVersions(libraryRoot, func(version string) error {
		entries, err := os.ReadDir(version)
		if err != nil {
			return err
		}
		assetName := filepath.Base(filepath.Dir(filepath.Dir(version)))
		files := canonicalVersionShape(assetName, entries)
		// Support modules are published companions rather than standalone
		// component entrypoints. Their canonical contract is the versioned
		// directory itself; normalize their internal helper filenames so the
		// census counts the asset kind without inventing a shape per helper set.
		if strings.Contains(filepath.ToSlash(version), "/library/support/") {
			files = []string{"<Asset>.tsx", "<Asset>.css?", "<Asset>.strings.ts?", "story.tsx", "story.json", "dependencies.json"}
		}
		counts[strings.Join(files, "\x00")]++
		return nil
	}); err != nil {
		return nil, err
	}
	rows := make([]ShapeRow, 0, len(counts))
	for shape, count := range counts {
		files := []string{}
		if shape != "" {
			files = strings.Split(shape, "\x00")
		}
		rows = append(rows, ShapeRow{Shape: files, Count: count})
	}
	sort.Slice(rows, func(i, j int) bool { return strings.Join(rows[i].Shape, "\x00") < strings.Join(rows[j].Shape, "\x00") })
	return rows, nil
}

func canonicalVersionShape(assetName string, entries []os.DirEntry) []string {
	allowed := map[string]bool{
		assetName + ".ts": true, assetName + ".tsx": true,
		assetName + ".css": true, assetName + ".strings.ts": true,
		"story.tsx": true, "story.json": true, "dependencies.json": true,
	}
	// Optional authored companions are part of the declared shape even when a
	// particular asset does not need one. This keeps the census about contract
	// violations, rather than producing a row for every valid CSS/strings
	// combination.
	shape := []string{"<Asset>.tsx", "<Asset>.css?", "<Asset>.strings.ts?", "story.tsx", "story.json", "dependencies.json"}
	entryFound, storyFound, contractFound, lockFound := false, false, false, false
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			shape = append(shape, "<forbidden:"+entry.Name()+">")
			continue
		}
		name := entry.Name()
		if !allowed[name] {
			shape = append(shape, "<forbidden:"+name+">")
			continue
		}
		switch name {
		case assetName + ".ts", assetName + ".tsx":
			entryFound = true
		case assetName + ".css":
			// Optional canonical companion; already represented in the shape.
		case assetName + ".strings.ts":
			// Optional canonical companion; already represented in the shape.
		case "story.tsx":
			storyFound = true
		case "story.json":
			contractFound = true
		case "dependencies.json":
			lockFound = true
		}
	}
	if !entryFound {
		shape = append(shape, "<missing:entry>")
	}
	if !storyFound {
		shape = append(shape, "<missing:story.tsx>")
	}
	if !contractFound {
		shape = append(shape, "<missing:story.json>")
	}
	if !lockFound {
		shape = append(shape, "<missing:dependencies.json>")
	}
	sort.Strings(shape)
	return shape
}

func DuplicationCensus(root string) ([]DuplicationRow, error) {
	catalogRoot := filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets")
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	rows := []DuplicationRow{}
	fields := []string{"description", "displayName", "slot"}
	domains, err := os.ReadDir(catalogRoot)
	if err != nil {
		return nil, err
	}
	for _, domain := range domains {
		if !domain.IsDir() {
			continue
		}
		assets, err := os.ReadDir(filepath.Join(catalogRoot, domain.Name()))
		if err != nil {
			return nil, err
		}
		for _, assetFile := range assets {
			if assetFile.IsDir() || !strings.HasSuffix(assetFile.Name(), ".json") {
				continue
			}
			catalog := map[string]any{}
			data, err := os.ReadFile(filepath.Join(catalogRoot, domain.Name(), assetFile.Name()))
			if err != nil {
				return nil, err
			}
			if json.Unmarshal(data, &catalog) != nil {
				continue
			}
			asset, _ := catalog["asset"].(map[string]any)
			if asset == nil {
				continue
			}
			name, _ := asset["name"].(string)
			if name == "" {
				continue
			}
			manifest, found := findManifest(libraryRoot, name)
			if !found {
				continue
			}
			for _, field := range fields {
				left, lok := asset[field]
				right, rok := manifest[field]
				if lok && rok && fmt.Sprint(left) != fmt.Sprint(right) {
					rows = append(rows, DuplicationRow{Asset: name, Field: field, Catalog: left, Manifest: right})
				}
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Asset == rows[j].Asset {
			return rows[i].Field < rows[j].Field
		}
		return rows[i].Asset < rows[j].Asset
	})
	return rows, nil
}

func walkLiveVersions(libraryRoot string, fn func(string) error) error {
	return filepath.WalkDir(libraryRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".retired" || entry.Name() == "dist" || entry.Name() == "node_modules") {
			if path != libraryRoot {
				return filepath.SkipDir
			}
		}
		if entry.IsDir() && entry.Name() == "versions" {
			return nil
		}
		if entry.IsDir() && filepath.Base(filepath.Dir(path)) == "versions" {
			return fn(path)
		}
		return nil
	})
}

func findManifest(libraryRoot, name string) (map[string]any, bool) {
	var result map[string]any
	_ = filepath.WalkDir(libraryRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil || result != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".retired" || entry.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if !entry.IsDir() && entry.Name() == "component.json" && filepath.Base(filepath.Dir(path)) == name {
			data, readErr := os.ReadFile(path)
			if readErr == nil {
				_ = json.Unmarshal(data, &result)
			}
		}
		return nil
	})
	return result, result != nil
}
