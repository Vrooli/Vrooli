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

	// templates/** is outside the RCL write boundary, so a vendored copy can be
	// legitimately behind the catalog for a cycle. Rather than silently skip the
	// invariant, we allow ONLY divergences a reviewer has explicitly recorded in
	// the reviewed-divergence allowlist. Every other behind/deprecated copy still
	// fails, and a stale allowlist entry (one whose divergence has been resolved)
	// also fails so the trail cannot rot into a permanent suppression list.
	allow, err := adoptions.LoadReviewedDivergences(
		filepath.Join("testdata", "reviewed-template-divergences.json"),
	)
	require.NoError(t, err)

	unreviewed, stale := adoptions.PartitionReviewedFindings(findings, allow)
	require.Empty(t, unreviewed,
		"react-vite template vendored components must match catalog latest or be recorded in "+
			"testdata/reviewed-template-divergences.json (see %s)", allow.Report)
	require.Empty(t, stale,
		"reviewed-divergence allowlist has entries that no longer match any real divergence — "+
			"remove them from testdata/reviewed-template-divergences.json")
}

// TestReviewedDivergenceAllowlist_DoesNotMaskUnlistedStaleCopies is the
// calibration for the allowlist gate: it proves the allowlist excuses ONLY the
// exact divergence it names and does not blanket-suppress other stale copies.
func TestReviewedDivergenceAllowlist_DoesNotMaskUnlistedStaleCopies(t *testing.T) {
	allow := adoptions.ReviewedDivergenceAllowlist{
		Divergences: []adoptions.ReviewedDivergence{{
			Path:            "ui/src/components/ui/data-table.tsx",
			LibraryID:       "react-component-library:DataTable",
			VendoredVersion: "1.1.0",
			CatalogVersion:  "1.1.2",
			Status:          "behind",
			Reason:          "reviewed",
		}},
	}

	listed := adoptions.FileScanFinding{
		Path:           "ui/src/components/ui/data-table.tsx",
		LibraryID:      "react-component-library:DataTable",
		AdoptedVersion: "1.1.0",
		LatestVersion:  "1.1.2",
		Status:         adoptions.LibraryVersionStatusBehind,
	}
	// A different behind copy that nobody reviewed.
	unlisted := adoptions.FileScanFinding{
		Path:           "ui/src/components/ui/button.tsx",
		LibraryID:      "react-component-library:Button",
		AdoptedVersion: "1.0.0",
		LatestVersion:  "1.2.0",
		Status:         adoptions.LibraryVersionStatusBehind,
	}
	// The SAME file the allowlist names, but drifted to a newer behind version
	// than the reviewer recorded — the stale review must NOT excuse it.
	drifted := adoptions.FileScanFinding{
		Path:           "ui/src/components/ui/data-table.tsx",
		LibraryID:      "react-component-library:DataTable",
		AdoptedVersion: "1.1.1",
		LatestVersion:  "1.1.2",
		Status:         adoptions.LibraryVersionStatusBehind,
	}

	unreviewed, stale := adoptions.PartitionReviewedFindings(
		[]adoptions.FileScanFinding{listed, unlisted, drifted}, allow,
	)

	unreviewedPaths := map[string]adoptions.FileScanFinding{}
	for _, f := range unreviewed {
		unreviewedPaths[f.Path+"@"+f.AdoptedVersion] = f
	}
	require.NotContains(t, unreviewedPaths, listed.Path+"@"+listed.AdoptedVersion,
		"the exact reviewed divergence must be excused")
	require.Contains(t, unreviewedPaths, unlisted.Path+"@"+unlisted.AdoptedVersion,
		"an unlisted behind copy must still fail the gate")
	require.Contains(t, unreviewedPaths, drifted.Path+"@"+drifted.AdoptedVersion,
		"a copy that drifted past the reviewed version must still fail the gate")
	require.Empty(t, stale, "the reviewed entry matched a real finding, so it is not stale")
}

// TestReviewedDivergenceAllowlist_FlagsStaleEntries proves that once a reviewed
// divergence is resolved (no matching finding remains), the allowlist entry is
// reported as stale so it cannot linger as a silent permanent suppression.
func TestReviewedDivergenceAllowlist_FlagsStaleEntries(t *testing.T) {
	allow := adoptions.ReviewedDivergenceAllowlist{
		Divergences: []adoptions.ReviewedDivergence{{
			Path:            "ui/src/components/ui/data-table.tsx",
			LibraryID:       "react-component-library:DataTable",
			VendoredVersion: "1.1.0",
			CatalogVersion:  "1.1.2",
			Status:          "behind",
			Reason:          "reviewed",
		}},
	}

	unreviewed, stale := adoptions.PartitionReviewedFindings(nil, allow)
	require.Empty(t, unreviewed)
	require.Len(t, stale, 1, "an allowlist entry matching no finding must be reported stale")
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
		// Register every cut version the library actually carries (from the
		// on-disk versions/ directory) so a vendored copy pinned to a real but
		// older version is classified as `behind`, not `unknown`. Without this
		// the catalog only knows `latest` and every older-but-valid vendored
		// version reads as unknown, which understates the divergence.
		versionsDir := filepath.Join(root, entry.Name(), "versions")
		if versionEntries, verr := os.ReadDir(versionsDir); verr == nil {
			for _, versionEntry := range versionEntries {
				if !versionEntry.IsDir() {
					continue
				}
				version := strings.TrimSpace(versionEntry.Name())
				catalog.versionsByComponentID[id] = append(catalog.versionsByComponentID[id], version)
				// Mirror the real indexer, which defaults on-disk versions to
				// "released" unless a header/component.json marks them draft or
				// deprecated (handled below, overriding this).
				catalog.versionStatusByKey[id+"@"+version] = components.VersionStatusReleased
			}
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
