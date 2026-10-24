package retirement

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"react-component-library/internal/components"
)

type retirementRepo struct {
	components.Repository
	component components.Component
	deleteErr error
	deleted   bool
}

func (r *retirementRepo) GetByLibraryID(context.Context, string) (components.Component, error) {
	if r.deleted {
		return components.Component{}, components.ErrComponentNotFound{IDOrLibraryID: r.component.LibraryID}
	}
	return r.component, nil
}
func (r *retirementRepo) ListVersions(context.Context, string, int) ([]components.ComponentVersion, error) {
	return nil, nil
}
func (r *retirementRepo) ListStories(context.Context, components.StoryQuery) ([]components.ComponentStory, error) {
	return nil, nil
}
func (r *retirementRepo) DeleteRetiredComponent(_ context.Context, id, libraryID string) error {
	if id != r.component.ID || libraryID != r.component.LibraryID {
		return errors.New("wrong deletion identity")
	}
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.deleted = true
	return nil
}

func retirementFixture(t *testing.T) (string, *retirementRepo, string, string) {
	t.Helper()
	root := t.TempDir()
	base := filepath.Join(root, "scenarios/react-component-library")
	manifest := filepath.Join(base, "library/components/Old/component.json")
	catalog := filepath.Join(base, "catalog/assets/navigation/old.json")
	for path, data := range map[string]string{
		manifest: `{"libraryId":"library:Old","catalogId":"navigation.old"}`,
		filepath.Join(filepath.Dir(manifest), "versions/1.0.0/Old.tsx"): "export const Old = () => null;",
		catalog: `{"kind":"catalog-asset","asset":{"id":"navigation.old","kind":"component","domain":"navigation","target":{"maturity":"deprecated"}}}`,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root, &retirementRepo{component: components.Component{ID: "registry-old", LibraryID: "library:Old", CatalogID: "navigation.old", Slug: "Old", ManifestPath: "components/Old/component.json"}}, manifest, catalog
}

func TestRetireCoordinatesArchiveAndRollback(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "registry-failure"}[failure], func(t *testing.T) {
			root, repo, manifest, catalog := retirementFixture(t)
			if failure {
				repo.deleteErr = errors.New("injected registry failure")
			}
			out, err := Retire(t.Context(), root, repo, repo.component.LibraryID)
			if failure {
				if err == nil || out.Retired || repo.deleted {
					t.Fatalf("claimed retirement: %+v %v", out, err)
				}
				for _, path := range []string{manifest, catalog} {
					if _, err := os.Stat(path); err != nil {
						t.Fatalf("rollback lost %s: %v", path, err)
					}
				}
			} else {
				if err != nil || !out.Retired || !repo.deleted {
					t.Fatalf("retirement failed: %+v %v", out, err)
				}
				for _, path := range []string{manifest, catalog} {
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Fatalf("active asset remains %s: %v", path, err)
					}
				}
				data, err := os.ReadFile(filepath.Join(out.Archive.SourceArchivePath, "versions/1.0.0/Old.tsx"))
				if err != nil || string(data) != "export const Old = () => null;" {
					t.Fatalf("release bytes lost: %q %v", data, err)
				}
				if _, err := os.Stat(out.CatalogArchivePath); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := os.Stat(out.Archive.SnapshotPath); err != nil {
				t.Fatalf("snapshot lost: %v", err)
			}
			if failure {
				repo.deleteErr = nil
				retry, err := Retire(t.Context(), root, repo, repo.component.LibraryID)
				if err != nil || !retry.Retired || retry.Archive.SnapshotPath != out.Archive.SnapshotPath {
					t.Fatalf("retry after rollback failed: %+v %v", retry, err)
				}
			}
		})
	}
}

func TestRetireRejectsWrongManifestAndConsumersBeforeMutation(t *testing.T) {
	for _, mode := range []string{"identity", "escape", "consumer"} {
		t.Run(mode, func(t *testing.T) {
			root, repo, manifest, catalog := retirementFixture(t)
			switch mode {
			case "identity":
				repo.component.CatalogID = "different"
			case "escape":
				repo.component.ManifestPath = "../Old/component.json"
			case "consumer":
				path := filepath.Join(root, "scenarios/consumer/public/app.js")
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(`import Old from '@vrooli/react-component-library/Old/1';`), 0644); err != nil {
					t.Fatal(err)
				}
			}
			out, err := Retire(t.Context(), root, repo, repo.component.LibraryID)
			if err == nil || out.Retired || repo.deleted || out.Archive.SnapshotPath != "" {
				t.Fatalf("mutated rejected request: %+v %v", out, err)
			}
			for _, path := range []string{manifest, catalog} {
				if _, err := os.Stat(path); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestRetirementRecoveryAtMutationBoundaries(t *testing.T) {
	for _, stage := range []string{"prepared", "source-moved", "catalog-moved", "registry-committed"} {
		t.Run(stage, func(t *testing.T) {
			root, repo, manifest, catalog := retirementFixture(t)
			ctx, release, err := components.AcquireLibraryRetirement(t.Context(), libraryRoot(root))
			if err != nil {
				t.Fatal(err)
			}
			if err := persistJSON(pendingPath(root), pendingRetirement{Component: repo.component, CatalogRelative: "navigation/old.json"}); err != nil {
				t.Fatal(err)
			}
			if stage != "prepared" {
				if _, err := components.ArchiveRetirementHistory(ctx, repo, components.NewFSContentStore(libraryRoot(root)), repo.component); err != nil {
					t.Fatal(err)
				}
			}
			if stage == "catalog-moved" || stage == "registry-committed" {
				if err := moveExclusive(catalog, archivePaths(root, repo.component).SourceArchivePath+".catalog.json"); err != nil {
					t.Fatal(err)
				}
			}
			if stage == "registry-committed" {
				repo.deleted = true
			}
			release()
			if _, unlock, err := components.AcquireLibraryMutation(t.Context(), libraryRoot(root)); err == nil {
				unlock()
				t.Fatal("pending retirement allowed ordinary mutation")
			}
			ctx, release, err = components.AcquireLibraryRetirement(t.Context(), libraryRoot(root))
			if err != nil {
				t.Fatal(err)
			}
			out, found, err := recoverPending(ctx, root, repo)
			release()
			if err != nil || !found {
				t.Fatalf("recovery failed: %+v %v", out, err)
			}
			if out.Retired != (stage == "registry-committed") {
				t.Fatalf("wrong recovery outcome: %+v", out)
			}
			if _, err := os.Stat(pendingPath(root)); !os.IsNotExist(err) {
				t.Fatalf("pending journal remains: %v", err)
			}
			if stage == "registry-committed" {
				if _, err := os.Stat(out.Archive.SourceArchivePath + ".receipt.json"); err != nil {
					t.Fatal(err)
				}
			} else {
				for _, path := range []string{manifest, catalog} {
					if _, err := os.Stat(path); err != nil {
						t.Fatal(err)
					}
				}
			}
			_, release, err = components.AcquireLibraryMutation(t.Context(), libraryRoot(root))
			if err != nil {
				t.Fatal(err)
			}
			release()
		})
	}
}
