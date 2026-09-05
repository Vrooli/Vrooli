package adoptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/scheduletest"
	"react-component-library/internal/adoptions"
	adoptmocks "react-component-library/internal/adoptions/mocks"
	"react-component-library/internal/components"
	"react-component-library/internal/deps"
)

// materializingLibrary keeps the service tests focused on the lifecycle
// boundary: the catalog reader remains the same fake, while the extra
// capability proves adoption cannot be recorded before an evicted version is
// restored.
type materializingLibrary struct {
	*fakeLibrary
	err   error
	calls int
}

func (f *materializingLibrary) EnsureMaterialized(_ context.Context, _, version, _ string) (components.MaterializeResult, error) {
	f.calls++
	if f.err != nil {
		return components.MaterializeResult{}, f.err
	}
	return components.MaterializeResult{Version: version, FilesWritten: 1}, nil
}

type presenceRecorder struct {
	components []string
}

func (r *presenceRecorder) ReconcilePresence(_ context.Context, componentID string, apply bool) error {
	if apply {
		r.components = append(r.components, componentID)
	}
	return nil
}

func lifecycleLinkFiles() *fakeFiles {
	return &fakeFiles{bytes: map[string][]byte{
		"target::ui/package.json":                    []byte(`{"name":"target-ui","dependencies":{}}`),
		"target::ui/src/i18n/index.ts":               []byte(`export const i18n = { t: (key: string, options?: { defaultValue?: string }) => options?.defaultValue ?? key };`),
		"target::ui/src/consts/selectors.library.ts": []byte(`export const librarySelectors = {} as const;`),
		"target::ui/src/consts/selectors.ts":         []byte(`const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);`),
		"target::ui/src/i18n/locales/en.json":        []byte(`{}`),
		"target::ui/src/main.tsx":                    []byte("import ReactDOM from \"react-dom/client\";\nReactDOM.createRoot(document.body).render(<div />);\n"),
	}}
}

func lifecycleLibrary() *materializingLibrary {
	return &materializingLibrary{fakeLibrary: &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-button": {
				ID: "cmp-button", LibraryID: "react-component-library:Button", Slug: "Button",
				LatestVersion: "1.2.0", Version: "1.2.0", AssetKind: components.AssetKindComponent,
			},
		},
		body: map[string]string{"cmp-button": "export function Button() { return null }"},
	}}
}

func TestAdoptionLinkMaterializesEvictedVersion(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := lifecycleLibrary()
	svc := adoptions.NewService(repo, lib, lifecycleLinkFiles(), scheduletest.New(time.Unix(0, 0)))

	result, err := svc.Link(context.Background(), adoptions.LinkInput{ComponentID: "cmp-button", Scenario: "target", Version: "1.2.0"})
	require.NoError(t, err)
	require.Equal(t, 1, lib.calls)
	require.NotEmpty(t, result.Adoption.ID)
	require.Equal(t, int64(1), repo.CreateCalls.Load())
}

func TestAdoptionLinkFailsWhenMaterializationFails(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := lifecycleLibrary()
	lib.err = errors.New("ledger mirror is corrupt")
	svc := adoptions.NewService(repo, lib, lifecycleLinkFiles(), scheduletest.New(time.Unix(0, 0)))

	_, err := svc.Link(context.Background(), adoptions.LinkInput{ComponentID: "cmp-button", Scenario: "target", Version: "1.2.0"})
	require.ErrorContains(t, err, "materialize cmp-button@1.2.0 before adoption")
	require.Zero(t, repo.CreateCalls.Load(), "an adoption must not be recorded after materialization fails")
}

func TestAdoptionEjectMaterializesEvictedVersion(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := lifecycleLibrary()
	files := lifecycleLinkFiles()
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))

	result, err := svc.Eject(context.Background(), adoptions.EjectInput{
		ApplyInput: adoptions.ApplyInput{
			ComponentID:        "cmp-button",
			Scenario:           "target",
			AdoptedPath:        "ui/src/Button.tsx",
			Version:            "1.2.0",
			OverrideValidation: true,
		},
		Reason: "local integration boundary",
	})
	require.NoError(t, err)
	require.Equal(t, adoptions.AdoptionModeEjected, result.Adoption.Mode)
	require.Equal(t, 1, lib.calls)
}

func TestAdoptionDeleteMakesVersionEvictable(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	repo.Seed(adoptions.Adoption{
		ID: "adoption-1", ComponentID: "cmp-button", LibraryID: "rcl:Button", Scenario: "target",
		AdoptedPath: "ui/src/Button.tsx", AdoptedVersion: "1.2.0",
	})
	reconciler := &presenceRecorder{}
	svc := adoptions.NewService(repo, lifecycleLibrary(), &fakeFiles{bytes: map[string][]byte{}}, scheduletest.New(time.Unix(0, 0)))
	adoptions.SetPresenceReconciler(svc, reconciler)

	_, err := svc.DeleteWithOptions(context.Background(), "adoption-1", true)
	require.NoError(t, err)
	require.Equal(t, []string{"cmp-button"}, reconciler.components)
	require.Equal(t, int64(1), repo.DeleteCalls.Load())
}

func TestAdoptionReconvergeReconcilesMovedVersion(t *testing.T) {
	ctx := context.Background()
	repo := adoptmocks.NewFakeRepository()
	base := reconvergeLibrary("BODY-V10", "BODY-V11")
	lib := &materializingLibrary{fakeLibrary: base}
	files := &fakeFiles{bytes: map[string][]byte{tmButtonKey(): []byte("BODY-V10")}}
	reconciler := &presenceRecorder{}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Now()))
	adoptions.SetValidationGates(svc, &validationDeps{verdict: deps.Verdict{Kind: deps.VerdictWarn}}, &validationStyles{})
	adoptions.SetPresenceReconciler(svc, reconciler)
	repo.Seed(adoptions.Adoption{
		ID: "row-reconverge", ComponentID: "cmp-btn", LibraryID: "rcl:Button", Scenario: "template-manager",
		AdoptedPath: tmButtonPath, AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha("BODY-V10"),
		Files: []adoptions.AdoptionFile{{LibraryPath: "Button.tsx", AdoptedPath: tmButtonPath, AdoptedSnapshotSHA256: sha("BODY-V10")}},
	})

	result, err := svc.Reconverge(ctx, adoptions.ReconvergeInput{Apply: true})
	require.NoError(t, err)
	require.Equal(t, 1, result.Reapplied)
	require.Equal(t, []string{"cmp-btn"}, reconciler.components)
	require.GreaterOrEqual(t, lib.calls, 1)
}
