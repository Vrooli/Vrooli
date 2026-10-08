package versionledger

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/databasetest"
)

const cleanupSchema = `
CREATE TABLE components (id TEXT PRIMARY KEY, library_id TEXT NOT NULL, latest_version TEXT NOT NULL, draft_version TEXT NOT NULL, manifest_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_versions (id TEXT PRIMARY KEY, component_id TEXT NOT NULL, library_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL, source_path TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT '', released_at TEXT NOT NULL DEFAULT '', presence TEXT NOT NULL DEFAULT 'materialized');
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied', scenario TEXT NOT NULL DEFAULT '', adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT, adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_asset_dependencies (component_id TEXT, library_id TEXT, version TEXT);
`

// writeVersion lays down one version directory with source and, when
// dependencies are supplied, the generated lock beside it.
func writeVersion(t *testing.T, root, kind, asset, version, source, lock string) string {
	t.Helper()
	dir := filepath.Join(root, kind, asset, "versions", version)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, asset+".tsx"), []byte(source), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, kind, asset, "component.json"), []byte(`{"libraryId":"react-component-library:`+asset+`"}`), 0o644))
	if lock != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "dependencies.json"), []byte(lock), 0o644))
	}
	return dir
}

// This is the IconButton incident, reduced to its parts.
//
// VoiceInputButton imports IconButton@2.0.1 through the package subpath, which
// is how library assets actually compose, and it has no index row — the two
// conditions that together defeated every prior protection. IconButton@2.0.1 is
// not `latest`, is not adopted, and carries no row in the retired
// component_asset_dependencies table, so before the lock graph existed it was
// reported "safe to retire".
func TestCleanupProtectsVersionsNamedOnlyByAVersionLock(t *testing.T) {
	database := databasetest.NewSQLite(t)
	root := t.TempDir()

	writeVersion(t, root, "components", "IconButton", "2.0.1", "export const IconButton = () => null;", `{"schemaVersion":1,"libraryId":"react-component-library:IconButton","version":"2.0.1","dependencies":[]}`)
	writeVersion(t, root, "components", "IconButton", "3.1.1", "export const IconButton = () => null;", `{"schemaVersion":1,"libraryId":"react-component-library:IconButton","version":"3.1.1","dependencies":[]}`)
	// The referrer composes through the package specifier, not a relative path.
	writeVersion(t, root, "components", "VoiceInputButton", "4.3.1",
		`import { IconButton } from "@vrooli/react-component-library/IconButton/2.0.1";`,
		`{"schemaVersion":1,"libraryId":"react-component-library:VoiceInputButton","version":"4.3.1","dependencies":[{"libraryId":"react-component-library:IconButton","version":"2.0.1","rank":4}]}`)

	_, err := database.ExecContext(context.Background(), cleanupSchema+`
INSERT INTO components(id, library_id, latest_version, draft_version) VALUES ('c-iconbutton', 'react-component-library:IconButton', '3.1.1', '');
INSERT INTO component_versions(id, component_id, library_id, version, status, source_path) VALUES
  ('v-2-0-1', 'c-iconbutton', 'react-component-library:IconButton', '2.0.1', 'deprecated', 'components/IconButton/versions/2.0.1/IconButton.tsx'),
  ('v-3-1-1', 'c-iconbutton', 'react-component-library:IconButton', '3.1.1', 'released', 'components/IconButton/versions/3.1.1/IconButton.tsx');
`)
	require.NoError(t, err)

	repository := NewRepository(database, root)
	items, _, err := repository.PlanCleanup(context.Background(), CleanupScope{})
	require.NoError(t, err)

	var target *CleanupItem
	for index := range items {
		if items[index].Candidate.Version == "2.0.1" {
			target = &items[index]
		}
	}
	require.NotNil(t, target, "IconButton@2.0.1 must appear in the cleanup plan")
	require.False(t, target.Eligible, "a version named by another version's lock must never be eligible for deletion")
	require.Equal(t, "referenced by source import", target.Reason)
	require.NotEmpty(t, target.References)
	require.Equal(t, "version-lock", target.References[0].Kind)
	require.Equal(t, "react-component-library:VoiceInputButton", target.References[0].OwnerLibraryID)
	require.Equal(t, "4.3.1", target.References[0].OwnerVersion)
}

// An asset the indexer never recorded cannot defend its dependencies, so the
// destructive pass must refuse rather than treat its silence as consent.
func TestCleanupRefusesWhenTheIndexIsMissingAnAssetOnDisk(t *testing.T) {
	database := databasetest.NewSQLite(t)
	root := t.TempDir()

	writeVersion(t, root, "components", "Button", "1.0.0", "export const Button = () => null;", `{"schemaVersion":1,"libraryId":"react-component-library:Button","version":"1.0.0","dependencies":[]}`)
	writeVersion(t, root, "components", "Button", "1.1.0", "export const Button = () => null;", `{"schemaVersion":1,"libraryId":"react-component-library:Button","version":"1.1.0","dependencies":[]}`)
	// On disk, absent from the index — exactly VoiceInputButton's state.
	writeVersion(t, root, "components", "Unindexed", "1.0.0", "export const Unindexed = () => null;", `{"schemaVersion":1,"libraryId":"react-component-library:Unindexed","version":"1.0.0","dependencies":[]}`)

	_, err := database.ExecContext(context.Background(), cleanupSchema+`
INSERT INTO components(id, library_id, latest_version, draft_version) VALUES ('c-button', 'react-component-library:Button', '1.1.0', '');
INSERT INTO component_versions(id, component_id, library_id, version, status, source_path) VALUES
  ('v-1-0-0', 'c-button', 'react-component-library:Button', '1.0.0', 'deprecated', 'components/Button/versions/1.0.0/Button.tsx'),
  ('v-1-1-0', 'c-button', 'react-component-library:Button', '1.1.0', 'released', 'components/Button/versions/1.1.0/Button.tsx');
`)
	require.NoError(t, err)

	repository := NewRepository(database, root)
	drift, err := repository.IndexDrift(context.Background())
	require.NoError(t, err)
	require.False(t, drift.Empty())
	require.Equal(t, []string{"components/Unindexed"}, drift.UnindexedAssets)

	items, hash, err := repository.PlanCleanup(context.Background(), CleanupScope{})
	require.NoError(t, err)
	require.NotEmpty(t, items, "planning stays available so the drift can be diagnosed")

	_, _, retired, err := repository.CleanupVersions(context.Background(), CleanupScope{}, hash, true)
	require.Error(t, err, "an index that disagrees with the tree must not authorise deletion")
	require.Contains(t, err.Error(), "missing from the index")
	require.Contains(t, err.Error(), "Unindexed")
	require.Zero(t, retired)
}

// Retired trees are the one place a dangling reference is expected. Honouring
// their imports would pin live versions to quarantined content.
func TestLockReferencesIgnoreRetiredTrees(t *testing.T) {
	root := t.TempDir()
	writeVersion(t, root, "components", "Card", "1.0.0", "export const Card = () => null;", `{"schemaVersion":1,"libraryId":"react-component-library:Card","version":"1.0.0","dependencies":[]}`)

	retired := filepath.Join(root, ".retired", "deadbeef")
	require.NoError(t, os.MkdirAll(retired, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(retired, "dependencies.json"),
		[]byte(`{"schemaVersion":1,"libraryId":"react-component-library:Ghost","version":"9.9.9","dependencies":[{"libraryId":"react-component-library:Card","version":"1.0.0"}]}`), 0o644))

	references, err := NewRepository(nil, root).lockReferences(context.Background())
	require.NoError(t, err)
	require.Empty(t, references[sourceReferenceKey("react-component-library:Card", "1.0.0")],
		"a lock inside .retired must not keep a live version alive")
}

// A lock that cannot be parsed is a refusal, not an absent edge: retention is a
// deletion boundary, so an unreadable record must never reduce protection.
func TestUnreadableLockFailsClosed(t *testing.T) {
	root := t.TempDir()
	writeVersion(t, root, "components", "Card", "1.0.0", "export const Card = () => null;", `{ this is not json`)

	_, err := NewRepository(nil, root).lockReferences(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode version lock")
}

// The workbench app pins library versions through the package export map, not
// through relative paths. Following only relative specifiers left those pins
// invisible, and cold-tier eviction removed Input@1.1.2 while ui/src still
// imported it — breaking the app build.
func TestWorkbenchPackageSpecifierPinsAreProtected(t *testing.T) {
	database := databasetest.NewSQLite(t)
	scenarioRoot := t.TempDir()
	root := filepath.Join(scenarioRoot, "library")
	require.NoError(t, os.MkdirAll(root, 0o755))

	writeVersion(t, root, "components", "Input", "1.1.2", "export const Input = () => null;", `{"schemaVersion":1,"libraryId":"react-component-library:Input","version":"1.1.2","dependencies":[]}`)
	writeVersion(t, root, "components", "Input", "1.3.0", "export const Input = () => null;", `{"schemaVersion":1,"libraryId":"react-component-library:Input","version":"1.3.0","dependencies":[]}`)

	uiSource := filepath.Join(scenarioRoot, "ui", "src", "components")
	require.NoError(t, os.MkdirAll(uiSource, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(uiSource, "Input.tsx"),
		[]byte(`import { Input } from "@vrooli/react-component-library/Input/1.1.2";`), 0o644))

	_, err := database.ExecContext(context.Background(), cleanupSchema+`
INSERT INTO components(id, library_id, latest_version, draft_version) VALUES ('c-input', 'react-component-library:Input', '1.3.0', '');
INSERT INTO component_versions(id, component_id, library_id, version, status, source_path) VALUES
  ('v-1-1-2', 'c-input', 'react-component-library:Input', '1.1.2', 'released', 'components/Input/versions/1.1.2/Input.tsx'),
  ('v-1-3-0', 'c-input', 'react-component-library:Input', '1.3.0', 'released', 'components/Input/versions/1.3.0/Input.tsx');
`)
	require.NoError(t, err)

	items, _, err := NewRepository(database, root).PlanCleanup(context.Background(), CleanupScope{})
	require.NoError(t, err)
	var target *CleanupItem
	for index := range items {
		if items[index].Candidate.Version == "1.1.2" {
			target = &items[index]
		}
	}
	require.NotNil(t, target)
	require.False(t, target.Eligible, "a version the workbench pins through the export map must not be retired")
	require.NotEmpty(t, target.References)
	require.Equal(t, "workbench-source-import", target.References[0].Kind)
}

func TestExperienceStoryReferencesProtectVersion(t *testing.T) {
	database := databasetest.NewSQLite(t)
	scenarioRoot := t.TempDir()
	root := filepath.Join(scenarioRoot, "library")
	writeVersion(t, root, "components", "Button", "1.0.0", "export const Button = () => null;", `{"schemaVersion":1,"libraryId":"react-component-library:Button","version":"1.0.0","dependencies":[]}`)
	writeVersion(t, root, "components", "Button", "1.1.0", "export const Button = () => null;", `{"schemaVersion":1,"libraryId":"react-component-library:Button","version":"1.1.0","dependencies":[]}`)

	experienceRoot := filepath.Join(scenarioRoot, "experience", "components")
	require.NoError(t, os.MkdirAll(experienceRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(experienceRoot, "button.json"), []byte(`{"storyRef":"../../library/components/Button/versions/1.0.0/story.json"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "components", "Button", "versions", "1.0.0", "story.json"), []byte(`{"title":"Button"}`), 0o644))

	_, err := database.ExecContext(context.Background(), cleanupSchema+`
INSERT INTO components(id, library_id, latest_version, draft_version) VALUES ('c-button', 'react-component-library:Button', '1.1.0', '');
INSERT INTO component_versions(id, component_id, library_id, version, status, source_path) VALUES
  ('v-1-0-0', 'c-button', 'react-component-library:Button', '1.0.0', 'deprecated', 'components/Button/versions/1.0.0/Button.tsx'),
  ('v-1-1-0', 'c-button', 'react-component-library:Button', '1.1.0', 'released', 'components/Button/versions/1.1.0/Button.tsx');
`)
	require.NoError(t, err)

	items, _, err := NewRepository(database, root).PlanCleanup(context.Background(), CleanupScope{})
	require.NoError(t, err)
	var target *CleanupItem
	for index := range items {
		if items[index].Candidate.Version == "1.0.0" {
			target = &items[index]
		}
	}
	require.NotNil(t, target)
	require.False(t, target.Eligible, "a canonical experience story reference must protect its version")
	require.Len(t, target.References, 1)
	require.Equal(t, "experience-story-ref", target.References[0].Kind)
	require.Equal(t, filepath.Join(experienceRoot, "button.json"), target.References[0].OwnerPath)
	require.Equal(t, "../../library/components/Button/versions/1.0.0/story.json", target.References[0].ImportSpecifier)
}

// A bare specifier follows whatever the manifest calls latest, so it says
// nothing about any particular version and must not pin one.
func TestBarePackageSpecifierPinsNothing(t *testing.T) {
	_, _, ok := exactPackageSpecifier("@vrooli/react-component-library/Input")
	require.False(t, ok)

	libraryID, version, ok := exactPackageSpecifier("@vrooli/react-component-library/Input/1.1.2")
	require.True(t, ok)
	require.Equal(t, "react-component-library:Input", libraryID)
	require.Equal(t, "1.1.2", version)

	// A subpath below the version still pins that version.
	_, version, ok = exactPackageSpecifier("@vrooli/react-component-library/Input/1.1.2/styles.css")
	require.True(t, ok)
	require.Equal(t, "1.1.2", version)

	_, _, ok = exactPackageSpecifier("react")
	require.False(t, ok)
	_, _, ok = exactPackageSpecifier("@vrooli/react-component-library/Input/next")
	require.False(t, ok)
}
