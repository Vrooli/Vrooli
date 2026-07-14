package adoptions_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/adoptions"
	adoptmocks "react-component-library/internal/adoptions/mocks"
	"react-component-library/internal/components"
	"react-component-library/internal/deps"
	"react-component-library/internal/testutil/mocks"
)

// fakeLibrary is a minimal LibraryReader keyed by component_id.
type fakeLibrary struct {
	byID          map[string]components.Component
	body          map[string]string // id -> file content
	versionStatus map[string]components.ComponentVersionStatus
}

func (f *fakeLibrary) Get(_ context.Context, id string) (components.Component, error) {
	c, ok := f.byID[id]
	if !ok {
		return components.Component{}, components.ErrComponentNotFound{IDOrLibraryID: id}
	}
	return c, nil
}

func (f *fakeLibrary) List(_ context.Context, _ components.SearchQuery) ([]components.Component, error) {
	rows := make([]components.Component, 0, len(f.byID))
	for _, component := range f.byID {
		rows = append(rows, component)
	}
	return rows, nil
}

func (f *fakeLibrary) GetContent(_ context.Context, id string) (components.Content, error) {
	body, ok := f.body[id]
	if !ok {
		return components.Content{}, components.ErrComponentNotFound{IDOrLibraryID: id}
	}
	sum := sha256.Sum256([]byte(body))
	return components.Content{Body: body, SHA256: hex.EncodeToString(sum[:])}, nil
}

func (f *fakeLibrary) GetVersion(_ context.Context, componentID, version string) (components.ComponentVersion, error) {
	body, ok := f.body[componentID]
	if !ok {
		return components.ComponentVersion{}, components.ErrComponentNotFound{IDOrLibraryID: componentID + "@" + version}
	}
	status := components.VersionStatusReleased
	if f.versionStatus != nil {
		if override := f.versionStatus[componentID+"@"+version]; override != "" {
			status = override
		}
	}
	return components.ComponentVersion{ComponentID: componentID, LibraryID: f.byID[componentID].LibraryID, Version: version, Status: status, Content: body, ContentSHA256: sha(body)}, nil
}

// fakeFiles is a minimal ScenarioFileReader keyed by "<scenario>::<path>".
type fakeFiles struct {
	bytes map[string][]byte
	sites map[string][]string
}

type validationDeps struct {
	verdict deps.Verdict
	calls   int
}

func (f *validationDeps) ValidateAdoption(context.Context, string, string, string) (deps.Verdict, error) {
	f.calls++
	return f.verdict, nil
}

type validationStyles struct{ calls int }

func (f *validationStyles) ValidateStyleFit(context.Context, string, string, string) (components.StyleFitVerdict, error) {
	f.calls++
	return components.StyleFitVerdict{Kind: components.StyleFitVerdictWarn}, nil
}

func (f *fakeFiles) Read(_ context.Context, scenario, p string) ([]byte, error) {
	b, ok := f.bytes[scenario+"::"+p]
	if !ok {
		return nil, adoptions.ErrAdoptedFileMissing{Scenario: scenario, AdoptedPath: p}
	}
	return b, nil
}

func (f *fakeFiles) Exists(_ context.Context, scenario, p string) (bool, error) {
	_, ok := f.bytes[scenario+"::"+p]
	return ok, nil
}

func (f *fakeFiles) Write(_ context.Context, scenario, p string, b []byte) (string, error) {
	if f.bytes == nil {
		f.bytes = map[string][]byte{}
	}
	f.bytes[scenario+"::"+p] = append([]byte(nil), b...)
	return scenario + "/" + p, nil
}

func (f *fakeFiles) FindImportSites(_ context.Context, scenario, p string) ([]string, error) {
	return append([]string(nil), f.sites[scenario+"::"+p]...), nil
}

func sha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestService_Refresh_StatusMatrix(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()

	// Library: Button at v1.1.0 with current body "BODY-V11".
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-btn":  {ID: "cmp-btn", LibraryID: "rcl:Button", Version: "1.1.0", LatestVersion: "1.1.0"},
			"cmp-gone": {ID: "cmp-gone"}, // exists in repo
		},
		body: map[string]string{
			"cmp-btn":  "BODY-V11",
			"cmp-gone": "irrelevant",
		},
	}
	// Drop cmp-gone from byID to simulate it being removed.
	delete(lib.byID, "cmp-gone")

	// Files on disk in target scenarios.
	bodyV10 := "BODY-V10"
	bodyV11 := "BODY-V11"
	bodyEdited := "BODY-LOCAL-EDIT"
	files := &fakeFiles{bytes: map[string][]byte{
		"swarm-manager::a-current.tsx":  []byte(bodyV11),    // matches library
		"swarm-manager::b-behind.tsx":   []byte(bodyV10),    // matches snapshot
		"swarm-manager::c-modified.tsx": []byte(bodyEdited), // diverged
		// d-unknown.tsx intentionally missing
		// e-component-gone.tsx exists but component no longer in library
		"swarm-manager::e-component-gone.tsx": []byte("anything"),
	}}

	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC))

	// Seed adoption rows directly (Seed bypasses Create's gates).
	rows := []adoptions.Adoption{
		{
			ID: "row-current", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
			Scenario: "swarm-manager", AdoptedPath: "a-current.tsx",
			AdoptedVersion: "1.1.0", AdoptedSnapshotSHA256: sha(bodyV11),
		},
		{
			ID: "row-behind", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
			Scenario: "swarm-manager", AdoptedPath: "b-behind.tsx",
			AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha(bodyV10),
		},
		{
			ID: "row-modified", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
			Scenario: "swarm-manager", AdoptedPath: "c-modified.tsx",
			AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha(bodyV10),
		},
		{
			ID: "row-unknown-missing", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
			Scenario: "swarm-manager", AdoptedPath: "d-unknown.tsx",
			AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha(bodyV10),
		},
		{
			ID: "row-unknown-gone", ComponentID: "cmp-gone", LibraryID: "rcl:Removed",
			Scenario: "swarm-manager", AdoptedPath: "e-component-gone.tsx",
			AdoptedVersion: "0.1.0", AdoptedSnapshotSHA256: sha("anything"),
		},
	}
	for _, r := range rows {
		r.CreatedAt = clk.Now()
		repo.Seed(r)
	}

	svc := adoptions.NewService(repo, lib, files, clk)
	got, summary, err := svc.Refresh(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, got, 5)

	byID := map[string]adoptions.Adoption{}
	for _, a := range got {
		byID[a.ID] = a
	}
	require.Equal(t, adoptions.LibraryVersionStatusCurrent, byID["row-current"].LibraryVersionStatus)
	require.Equal(t, adoptions.LocalStatusClean, byID["row-current"].LocalStatus)
	require.Equal(t, adoptions.LibraryVersionStatusBehind, byID["row-behind"].LibraryVersionStatus)
	require.Equal(t, adoptions.LocalStatusClean, byID["row-behind"].LocalStatus)
	require.Contains(t, byID["row-behind"].StatusDetail, "1.1.0")
	require.Equal(t, adoptions.LocalStatusModified, byID["row-modified"].LocalStatus)
	require.Equal(t, adoptions.LocalStatusMissing, byID["row-unknown-missing"].LocalStatus)
	require.Contains(t, byID["row-unknown-missing"].StatusDetail, "missing")
	require.Equal(t, adoptions.LibraryVersionStatusMissing, byID["row-unknown-gone"].LibraryVersionStatus)
	require.Contains(t, byID["row-unknown-gone"].StatusDetail, "removed")

	require.Equal(t, 1, summary.LibraryCurrent)
	require.Equal(t, 3, summary.LibraryBehind)
	require.Equal(t, 1, summary.LibraryMissing)
	require.Equal(t, 2, summary.LocalClean)
	require.Equal(t, 1, summary.LocalModified)
	require.Equal(t, 1, summary.LocalMissing)

	// Refresh wrote refreshed_at on every row.
	for _, a := range got {
		require.False(t, a.RefreshedAt.IsZero(), "row %s did not get refreshed_at", a.ID)
	}
}

func TestService_Refresh_FilterByComponent(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-a": {ID: "cmp-a", LibraryID: "x:A", Version: "1.0.0"},
			"cmp-b": {ID: "cmp-b", LibraryID: "x:B", Version: "1.0.0"},
		},
		body: map[string]string{"cmp-a": "AA", "cmp-b": "BB"},
	}
	files := &fakeFiles{bytes: map[string][]byte{
		"s::a.tsx": []byte("AA"),
		"s::b.tsx": []byte("BB"),
	}}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC))

	repo.Seed(adoptions.Adoption{ID: "ra", ComponentID: "cmp-a", Scenario: "s", AdoptedPath: "a.tsx", CreatedAt: clk.Now()})
	repo.Seed(adoptions.Adoption{ID: "rb", ComponentID: "cmp-b", Scenario: "s", AdoptedPath: "b.tsx", CreatedAt: clk.Now()})

	svc := adoptions.NewService(repo, lib, files, clk)
	got, _, err := svc.Refresh(context.Background(), "cmp-a")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "ra", got[0].ID)
}

func TestService_Refresh_UsesSemverOrdering(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp": {ID: "cmp", LibraryID: "rcl:Button", LatestVersion: "1.2.0"},
		},
		body: map[string]string{"cmp": "BODY"},
	}
	files := &fakeFiles{bytes: map[string][]byte{
		"s::newer.tsx": []byte("BODY"),
		"s::older.tsx": []byte("BODY"),
	}}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC))
	repo.Seed(adoptions.Adoption{ID: "newer", ComponentID: "cmp", LibraryID: "rcl:Button", Scenario: "s", AdoptedPath: "newer.tsx", AdoptedVersion: "1.10.0", AdoptedSnapshotSHA256: sha("BODY"), CreatedAt: clk.Now()})
	repo.Seed(adoptions.Adoption{ID: "older", ComponentID: "cmp", LibraryID: "rcl:Button", Scenario: "s", AdoptedPath: "older.tsx", AdoptedVersion: "1.1.9", AdoptedSnapshotSHA256: sha("BODY"), CreatedAt: clk.Now()})

	svc := adoptions.NewService(repo, lib, files, clk)
	got, summary, err := svc.Refresh(context.Background(), "")
	require.NoError(t, err)
	byID := map[string]adoptions.Adoption{}
	for _, row := range got {
		byID[row.ID] = row
	}
	require.Equal(t, adoptions.LibraryVersionStatusCurrent, byID["newer"].LibraryVersionStatus)
	require.Equal(t, adoptions.LibraryVersionStatusBehind, byID["older"].LibraryVersionStatus)
	require.Equal(t, 1, summary.LibraryCurrent)
	require.Equal(t, 1, summary.LibraryBehind)
}

func TestService_Refresh_ReportsDeprecatedVersion(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp": {ID: "cmp", LibraryID: "rcl:DataTable", LatestVersion: "2.0.0"},
		},
		body: map[string]string{"cmp": "BODY"},
		versionStatus: map[string]components.ComponentVersionStatus{
			"cmp@1.0.0": components.VersionStatusDeprecated,
		},
	}
	files := &fakeFiles{bytes: map[string][]byte{"s::table.tsx": []byte("BODY")}}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC))
	repo.Seed(adoptions.Adoption{ID: "deprecated", ComponentID: "cmp", LibraryID: "rcl:DataTable", Scenario: "s", AdoptedPath: "table.tsx", AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha("BODY"), CreatedAt: clk.Now()})

	svc := adoptions.NewService(repo, lib, files, clk)
	got, summary, err := svc.Refresh(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, adoptions.LibraryVersionStatusDeprecated, got[0].LibraryVersionStatus)
	require.Contains(t, got[0].StatusDetail, "deprecated")
	require.Equal(t, 1, summary.LibraryDeprecated)
}

func TestService_Refresh_DoesNotTreatDraftVersionAsCurrent(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp": {ID: "cmp", LibraryID: "rcl:Button", LatestVersion: "1.0.0", DraftVersion: "1.1.0-beta.1"},
		},
		body: map[string]string{"cmp": "BODY"},
		versionStatus: map[string]components.ComponentVersionStatus{
			"cmp@1.1.0-beta.1": components.VersionStatusDraft,
		},
	}
	files := &fakeFiles{bytes: map[string][]byte{"s::button.tsx": []byte("BODY")}}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC))
	repo.Seed(adoptions.Adoption{ID: "draft", ComponentID: "cmp", LibraryID: "rcl:Button", Scenario: "s", AdoptedPath: "button.tsx", AdoptedVersion: "1.1.0-beta.1", AdoptedSnapshotSHA256: sha("BODY"), CreatedAt: clk.Now()})

	svc := adoptions.NewService(repo, lib, files, clk)
	got, summary, err := svc.Refresh(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, adoptions.LibraryVersionStatusUnknown, got[0].LibraryVersionStatus)
	require.Contains(t, got[0].StatusDetail, "draft")
	require.Equal(t, 1, summary.LibraryUnknown)
}

func TestService_Create_RejectsMissingComponent(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{byID: map[string]components.Component{}, body: map[string]string{}}
	files := &fakeFiles{bytes: map[string][]byte{}}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC))

	svc := adoptions.NewService(repo, lib, files, clk)
	_, err := svc.Create(context.Background(), adoptions.CreateInput{
		ComponentID: "nope", Scenario: "s", AdoptedPath: "p.tsx",
	})
	var inv adoptions.ErrInvalidAdoption
	require.True(t, errors.As(err, &inv))
	require.Equal(t, "component_id", inv.Field)
}

func TestService_Create_HashesSnapshotAndEchoesLibraryID(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp": {ID: "cmp", LibraryID: "rcl:Button"},
		},
		body: map[string]string{"cmp": "irrelevant for create"},
	}
	files := &fakeFiles{bytes: map[string][]byte{
		"s::p.tsx": []byte("ADOPTED-AT-CREATE"),
	}}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC))

	svc := adoptions.NewService(repo, lib, files, clk)
	a, err := svc.Create(context.Background(), adoptions.CreateInput{
		ComponentID: "cmp", Scenario: "s", AdoptedPath: "p.tsx", AdoptedVersion: "1.0.0",
	})
	require.NoError(t, err)
	require.Equal(t, "rcl:Button", a.LibraryID)
	require.Equal(t, sha("ADOPTED-AT-CREATE"), a.AdoptedSnapshotSHA256)
}

func TestService_Apply_UsesSameIDInRecordAndProvenance(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp": {ID: "cmp", LibraryID: "rcl:Button", LatestVersion: "1.0.0"},
		},
		body: map[string]string{
			"cmp": "// @vrooliComponent libraryId=rcl:Button version=1.0.0\nexport function Button() { return <button />; }\n",
		},
	}
	files := &fakeFiles{bytes: map[string][]byte{}}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC))

	svc := adoptions.NewService(repo, lib, files, clk)
	result, err := svc.Apply(context.Background(), adoptions.ApplyInput{
		ComponentID: "cmp", Scenario: "target", AdoptedPath: "ui/src/Button.tsx",
	})
	require.NoError(t, err)
	require.Equal(t, "target/ui/src/Button.tsx", result.WrittenPath)
	require.NotEmpty(t, result.Adoption.ID)

	adopted := string(files.bytes["target::ui/src/Button.tsx"])
	require.Contains(t, adopted, "@vrooliComponentAdoption")
	require.Contains(t, adopted, "@vrooliComponentAdoption "+result.Adoption.ID)
	require.NotContains(t, adopted, "@vrooliComponent libraryId")
	require.Equal(t, sha(adopted), result.Adoption.AdoptedSnapshotSHA256)
}

func TestService_ApplyReplaceExisting_RequiresConfirmationAndReportsImportSites(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{"cmp": {ID: "cmp", LibraryID: "rcl:Button", LatestVersion: "1.0.0"}},
		body: map[string]string{"cmp": "export function Button() { return <button />; }\n"},
	}
	files := &fakeFiles{bytes: map[string][]byte{
		"target::ui/src/Button.tsx": []byte("export function OldButton() { return <button />; }\n"),
	}, sites: map[string][]string{
		"target::ui/src/Button.tsx": {"ui/src/Toolbar.tsx", "ui/src/Workspace.tsx"},
	}}
	svc := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC)))

	_, err := svc.Apply(context.Background(), adoptions.ApplyInput{ComponentID: "cmp", Scenario: "target", AdoptedPath: "ui/src/Button.tsx", ReplaceExisting: true})
	var invalid adoptions.ErrInvalidAdoption
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "confirm_overwrite", invalid.Field)

	result, err := svc.Apply(context.Background(), adoptions.ApplyInput{ComponentID: "cmp", Scenario: "target", AdoptedPath: "ui/src/Button.tsx", ReplaceExisting: true, ConfirmOverwrite: true})
	require.NoError(t, err)
	require.Equal(t, []string{"ui/src/Toolbar.tsx", "ui/src/Workspace.tsx"}, result.ImportSites)
	require.Contains(t, string(files.bytes["target::ui/src/Button.tsx"]), "@vrooliComponentSource rcl:Button")
}

func TestService_ApplyAndReapply_BlockDependencyVerdictsUnlessExplicitlyOverridden(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp": {ID: "cmp", LibraryID: "rcl:Button", LatestVersion: "1.0.0"},
		},
		body: map[string]string{
			"cmp": "// @vrooliComponent libraryId=rcl:Button version=1.0.0\nexport function Button() { return <button />; }\n",
		},
	}
	files := &fakeFiles{bytes: map[string][]byte{}}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC))
	svc := adoptions.NewService(repo, lib, files, clk)
	dependency := &validationDeps{verdict: deps.Verdict{Kind: deps.VerdictBlock}}
	styles := &validationStyles{}
	adoptions.SetValidationGates(svc, dependency, styles)

	_, err := svc.Apply(context.Background(), adoptions.ApplyInput{
		ComponentID: "cmp", Scenario: "target", AdoptedPath: "ui/src/Button.tsx",
	})
	var blocked adoptions.ErrAdoptionValidationBlocked
	require.ErrorAs(t, err, &blocked)
	require.Empty(t, files.bytes, "blocked apply must not write a target file")
	require.Equal(t, 1, dependency.calls)
	require.Equal(t, 1, styles.calls, "style validation still runs when dependency validation blocks")

	applied, err := svc.Apply(context.Background(), adoptions.ApplyInput{
		ComponentID: "cmp", Scenario: "target", AdoptedPath: "ui/src/Button.tsx", OverrideValidation: true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, applied.Adoption.ID)

	_, _, err = svc.Reapply(context.Background(), adoptions.ReapplyInput{ID: applied.Adoption.ID, Version: "1.0.0"})
	require.ErrorAs(t, err, &blocked)
	require.Equal(t, 3, dependency.calls)
	require.Equal(t, 3, styles.calls, "reapply also executes both validation gates")

	_, _, err = svc.Reapply(context.Background(), adoptions.ReapplyInput{ID: applied.Adoption.ID, Version: "1.0.0", OverrideValidation: true})
	require.NoError(t, err)
	require.Equal(t, 4, dependency.calls)
	require.Equal(t, 4, styles.calls)
}

func TestService_Reapply_PersistsNewVersionAndSnapshot(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	initial := "OLD"
	repo.Seed(adoptions.Adoption{
		ID: "adopt-1", ComponentID: "cmp", LibraryID: "rcl:Button", Scenario: "target", AdoptedPath: "Button.tsx",
		AdoptedVersion: "1.0.0", SourceSHA256: sha(initial), AdoptedSnapshotSHA256: sha(initial),
		LibraryVersionStatus: adoptions.LibraryVersionStatusBehind, LocalStatus: adoptions.LocalStatusClean,
		DriftBacklogRef: "bug/drift",
	})
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp": {ID: "cmp", LibraryID: "rcl:Button", LatestVersion: "1.1.0"},
		},
		body: map[string]string{
			"cmp": "// @vrooliComponent libraryId=rcl:Button version=1.1.0\nNEW\n",
		},
	}
	files := &fakeFiles{bytes: map[string][]byte{"target::Button.tsx": []byte(initial)}}
	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC))

	svc := adoptions.NewService(repo, lib, files, clk)
	a, _, err := svc.Reapply(context.Background(), adoptions.ReapplyInput{ID: "adopt-1"})
	require.NoError(t, err)

	adopted := string(files.bytes["target::Button.tsx"])
	require.Equal(t, "1.1.0", a.AdoptedVersion)
	require.Equal(t, sha(adopted), a.AdoptedSnapshotSHA256)
	require.Equal(t, adoptions.LibraryVersionStatusCurrent, a.LibraryVersionStatus)
	require.Equal(t, adoptions.LocalStatusClean, a.LocalStatus)
	require.Empty(t, a.StatusDetail)
	require.Empty(t, a.DriftBacklogRef)

	got, err := repo.Get(context.Background(), "adopt-1")
	require.NoError(t, err)
	require.Equal(t, "1.1.0", got.AdoptedVersion)
	require.Equal(t, sha(adopted), got.AdoptedSnapshotSHA256)
}
