package preview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
)

type evictedPreviewCatalog struct {
	*fakeComponentsService
	assets     map[string]components.Component
	versions   map[string]components.ComponentVersion
	libraryIDs map[string]string
	root       string
	calls      []string
}

func (f *evictedPreviewCatalog) Get(ctx context.Context, id string) (components.Component, error) {
	asset, ok := f.assets[id]
	if !ok {
		return components.Component{}, fmt.Errorf("unknown fixture component %q", id)
	}
	return asset, nil
}

func (f *evictedPreviewCatalog) GetByLibraryID(_ context.Context, libraryID string) (components.Component, error) {
	id, ok := f.libraryIDs[libraryID]
	if !ok {
		return components.Component{}, fmt.Errorf("unknown fixture library %q", libraryID)
	}
	return f.assets[id], nil
}

func (f *evictedPreviewCatalog) GetVersion(_ context.Context, componentID, version string) (components.ComponentVersion, error) {
	v, ok := f.versions[componentID+"@"+version]
	if !ok {
		return components.ComponentVersion{}, fmt.Errorf("unknown fixture version %s@%s", componentID, version)
	}
	return v, nil
}

func (f *evictedPreviewCatalog) GetVersionContent(_ context.Context, componentID, version string) (components.Content, error) {
	v, err := f.GetVersion(context.Background(), componentID, version)
	if err != nil {
		return components.Content{}, err
	}
	return components.Content{Body: v.Content, SourcePath: v.SourcePath, SHA256: v.ContentSHA256}, nil
}

func (f *evictedPreviewCatalog) EnsureMaterialized(_ context.Context, componentID, version, into string) (components.MaterializeResult, error) {
	v, err := f.GetVersion(context.Background(), componentID, version)
	if err != nil {
		return components.MaterializeResult{}, err
	}
	key := componentID + "@" + version
	f.calls = append(f.calls, key)
	if into != "" {
		return components.MaterializeResult{}, fmt.Errorf("fixture expected materialization into authored library root")
	}
	libraryRoot := filepath.Join(f.root, "scenarios", "react-component-library", "library")
	destination := filepath.Join(libraryRoot, filepath.FromSlash(filepath.Dir(v.SourcePath)))
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return components.MaterializeResult{}, err
	}
	for _, file := range v.Files {
		if err := os.WriteFile(filepath.Join(destination, file.Path), []byte(file.Content), 0o600); err != nil {
			return components.MaterializeResult{}, err
		}
	}
	v.Presence = "materialized"
	f.versions[key] = v
	return components.MaterializeResult{Directory: filepath.ToSlash(destination), FilesWritten: len(v.Files)}, nil
}

func TestPreviewRejectsColdClosureWithoutWritingAndExplicitMaterializationRestoresPreview(t *testing.T) {
	root := t.TempDir()
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	rootEntry := "components/PromptComposer/versions/1.0.4-draft.1/PromptComposer.tsx"
	rootDir := filepath.Join(libraryRoot, filepath.Dir(filepath.FromSlash(rootEntry)))
	require.NoError(t, os.MkdirAll(rootDir, 0o755))
	rootBody := "import { Direct } from '@vrooli/react-component-library/Direct/1'; export const Root = () => <Direct />;"
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "PromptComposer.tsx"), []byte(rootBody), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "dependencies.json"), []byte(`{"dependencies":[{"libraryId":"react-component-library:Direct","major":1,"observed":"1.2.0"}]}`), 0o600))

	transitiveLock := `{"dependencies":[]}`
	directLock := `{"dependencies":[{"libraryId":"react-component-library:Transitive","major":2,"observed":"2.0.0"}]}`
	rootDeps := []components.AssetDependency{{LibraryID: "react-component-library:Direct", Version: "1.2.0"}}
	directDeps := []components.AssetDependency{{LibraryID: "react-component-library:Transitive", Version: "2.0.0"}}
	assets := map[string]components.Component{
		// The component projection is empty while the selected latest-version
		// lock contains the dependency. Preview must resolve the immutable lock.
		"root":       {ID: "root", LibraryID: "react-component-library:PromptComposer", SourcePath: rootEntry, LatestVersion: "1.0.4-draft.1"},
		"direct":     {ID: "direct", LibraryID: "react-component-library:Direct", SourcePath: "components/Direct/versions/1.2.0/Direct.tsx", LatestVersion: "1.2.0"},
		"transitive": {ID: "transitive", LibraryID: "react-component-library:Transitive", SourcePath: "components/Transitive/versions/2.0.0/Transitive.tsx", LatestVersion: "2.0.0"},
	}
	versions := map[string]components.ComponentVersion{
		"root@1.0.4-draft.1": {ComponentID: "root", LibraryID: assets["root"].LibraryID, Version: "1.0.4-draft.1", Status: components.VersionStatusDraft, Presence: "materialized", SourcePath: rootEntry, Content: rootBody, DependencyLockPresent: true, Dependencies: rootDeps},
		"direct@1.2.0":       {ComponentID: "direct", LibraryID: assets["direct"].LibraryID, Version: "1.2.0", Status: components.VersionStatusReleased, Presence: "evicted", SourcePath: assets["direct"].SourcePath, DependencyLockPresent: true, Dependencies: directDeps, Files: []components.ComponentVersionFile{{Path: "Direct.tsx", Content: "import { Transitive } from '@vrooli/react-component-library/Transitive/2'; export const Direct = () => <Transitive />;"}, {Path: "dependencies.json", Content: directLock}}},
		"transitive@2.0.0":   {ComponentID: "transitive", LibraryID: assets["transitive"].LibraryID, Version: "2.0.0", Status: components.VersionStatusReleased, Presence: "evicted", SourcePath: assets["transitive"].SourcePath, DependencyLockPresent: true, Files: []components.ComponentVersionFile{{Path: "Transitive.tsx", Content: "export const Transitive = () => <span>cold exact pin</span>;"}, {Path: "dependencies.json", Content: transitiveLock}}},
	}
	catalog := &evictedPreviewCatalog{
		fakeComponentsService: &fakeComponentsService{
			getContentFn: func(_ context.Context, id string) (components.Content, error) {
				return components.Content{Body: rootBody, SourcePath: rootEntry}, nil
			},
			getVersionContentFn: func(_ context.Context, id, version string) (components.Content, error) {
				v := versions[id+"@"+version]
				return components.Content{Body: v.Content, SourcePath: v.SourcePath, SHA256: v.ContentSHA256}, nil
			},
		},
		assets: assets, versions: versions,
		libraryIDs: map[string]string{assets["direct"].LibraryID: "direct", assets["transitive"].LibraryID: "transitive"},
		root:       root,
	}

	service := NewServiceWithDepsAtRoot(catalog, NewEsbuilderWithRoot(libraryRoot), nil, root)
	before := previewSourceDigests(t, libraryRoot)
	for attempt := 0; attempt < 2; attempt++ { // cold call followed by a warm repeat
		_, err := service.GetBundle(context.Background(), "root")
		var evicted ErrEvictedDependency
		require.ErrorAs(t, err, &evicted)
		require.Equal(t, "react-component-library:Transitive", evicted.LibraryID)
		require.Equal(t, "2.0.0", evicted.Version)
		require.Contains(t, err.Error(), `react-component-library versions materialize "react-component-library:Transitive" 2.0.0`)
		require.Empty(t, catalog.calls, "preview must not invoke the materializer")
		require.Equal(t, before, previewSourceDigests(t, libraryRoot), "cold/warm read %d changed source files", attempt+1)
	}

	// The existing explicit write capability remains available to restore the
	// exact dependency-first closure. Once materialized, preview works normally.
	_, err := catalog.EnsureMaterialized(context.Background(), "transitive", "2.0.0", "")
	require.NoError(t, err)
	_, err = catalog.EnsureMaterialized(context.Background(), "direct", "1.2.0", "")
	require.NoError(t, err)
	require.Equal(t, []string{"transitive@2.0.0", "direct@1.2.0"}, catalog.calls)
	bundle, err := service.GetBundle(context.Background(), "root")
	require.NoError(t, err)
	require.Contains(t, bundle.JS, "cold exact pin")
	require.Equal(t, []string{"transitive@2.0.0", "direct@1.2.0"}, catalog.calls, "a materialized preview must not trigger another write")
}

func previewSourceDigests(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(body)
		result[filepath.ToSlash(rel)] = hex.EncodeToString(digest[:])
		return nil
	})
	require.NoError(t, err)
	return result
}
