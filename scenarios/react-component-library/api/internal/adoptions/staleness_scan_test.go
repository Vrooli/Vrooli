package adoptions_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/adoptions"
	"react-component-library/internal/components"
)

func TestScanVendoredFiles_ReportsRecordlessBehindAndDeprecatedCopies(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "ui/src/components/ui/data-table.tsx", `/**
 * @vrooliComponentSource react-component-library:DataTable
 * @vrooliComponentVersion 1.0.0
 */
export function DataTable() { return null }
`)
	writeFile(t, root, "ui/src/components/ui/button.tsx", `/**
 * @vrooliComponentSource react-component-library:Button
 * @vrooliComponentVersion 0.9.0
 */
export function Button() { return null }
`)
	writeFile(t, root, "node_modules/ignored.tsx", `/**
 * @vrooliComponentSource react-component-library:Button
 * @vrooliComponentVersion 0.1.0
 */
`)

	catalog := &fakeLibrary{
		byID: map[string]components.Component{
			"data-table": {ID: "data-table", LibraryID: "react-component-library:DataTable", LatestVersion: "1.10.0"},
			"button":     {ID: "button", LibraryID: "react-component-library:Button", LatestVersion: "1.0.0"},
		},
		body: map[string]string{"data-table": "data", "button": "button"},
		versionStatus: map[string]components.ComponentVersionStatus{
			"button@0.9.0": components.VersionStatusDeprecated,
		},
	}

	findings, err := adoptions.ScanVendoredFiles(context.Background(), root, catalog)
	require.NoError(t, err)
	require.Len(t, findings, 2)

	byPath := map[string]adoptions.FileScanFinding{}
	for _, finding := range findings {
		byPath[finding.Path] = finding
	}
	require.Equal(t, adoptions.LibraryVersionStatusBehind, byPath["ui/src/components/ui/data-table.tsx"].Status)
	require.Equal(t, "1.10.0", byPath["ui/src/components/ui/data-table.tsx"].LatestVersion)
	require.Equal(t, adoptions.LibraryVersionStatusDeprecated, byPath["ui/src/components/ui/button.tsx"].Status)
	require.Contains(t, byPath["ui/src/components/ui/button.tsx"].Detail, "deprecated")
}

func TestScanVendoredFiles_DoesNotReportSemverNewerCopyAsBehind(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "ui/src/components/ui/button.tsx", `/**
 * @vrooliComponentSource react-component-library:Button
 * @vrooliComponentVersion 1.10.0
 */
export function Button() { return null }
`)
	catalog := &fakeLibrary{
		byID: map[string]components.Component{
			"button": {ID: "button", LibraryID: "react-component-library:Button", LatestVersion: "1.2.0"},
		},
		body: map[string]string{"button": "button"},
	}

	findings, err := adoptions.ScanVendoredFiles(context.Background(), root, catalog)
	require.NoError(t, err)
	require.Empty(t, findings)
}

func TestReactViteTemplateVendoredComponentsMatchCatalogLatest(t *testing.T) {
	repoRoot := findRepoRoot(t)
	catalog, err := loadRealCatalog(filepath.Join(repoRoot, "scenarios", "react-component-library", "library", "components"))
	require.NoError(t, err)

	findings, err := adoptions.ScanVendoredFiles(
		context.Background(),
		filepath.Join(repoRoot, "templates", "scenarios", "react-vite"),
		catalog,
	)
	require.NoError(t, err)
	require.Empty(t, findings, "react-vite template vendored components must match react-component-library catalog latest")
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

type realCatalog struct {
	componentsByID        map[string]components.Component
	versionStatusByKey    map[string]components.ComponentVersionStatus
	versionsByComponentID map[string][]string
}

func loadRealCatalog(root string) (*realCatalog, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	catalog := &realCatalog{
		componentsByID:        map[string]components.Component{},
		versionStatusByKey:    map[string]components.ComponentVersionStatus{},
		versionsByComponentID: map[string][]string{},
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name(), "component.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var doc struct {
			LibraryID          string   `json:"libraryId"`
			Latest             string   `json:"latest"`
			Draft              string   `json:"draft"`
			DeprecatedVersions []string `json:"deprecatedVersions"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		id := strings.ToLower(strings.TrimSpace(entry.Name()))
		catalog.componentsByID[id] = components.Component{
			ID:            id,
			LibraryID:     strings.TrimSpace(doc.LibraryID),
			LatestVersion: strings.TrimSpace(doc.Latest),
		}
		if doc.Latest != "" {
			catalog.versionsByComponentID[id] = append(catalog.versionsByComponentID[id], strings.TrimSpace(doc.Latest))
		}
		if doc.Draft != "" {
			draft := strings.TrimSpace(doc.Draft)
			catalog.versionsByComponentID[id] = append(catalog.versionsByComponentID[id], draft)
			catalog.versionStatusByKey[id+"@"+draft] = components.VersionStatusDraft
		}
		for _, version := range doc.DeprecatedVersions {
			deprecated := strings.TrimSpace(version)
			if deprecated == "" {
				continue
			}
			catalog.versionsByComponentID[id] = append(catalog.versionsByComponentID[id], deprecated)
			catalog.versionStatusByKey[id+"@"+deprecated] = components.VersionStatusDeprecated
		}
	}
	return catalog, nil
}

func (r *realCatalog) List(_ context.Context, _ components.SearchQuery) ([]components.Component, error) {
	out := make([]components.Component, 0, len(r.componentsByID))
	for _, component := range r.componentsByID {
		out = append(out, component)
	}
	return out, nil
}

func (r *realCatalog) GetVersion(_ context.Context, componentID, version string) (components.ComponentVersion, error) {
	for _, known := range r.versionsByComponentID[componentID] {
		if known == version {
			return components.ComponentVersion{
				ComponentID: componentID,
				Version:     version,
				Status:      r.versionStatusByKey[componentID+"@"+version],
			}, nil
		}
	}
	return components.ComponentVersion{}, fmt.Errorf("version %s@%s not found", componentID, version)
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			t.Fatal("repo root not found")
		}
		dir = next
	}
}
