package adoptions_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
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
	versions      map[string]components.ComponentVersion
}

func (f *fakeLibrary) Get(_ context.Context, id string) (components.Component, error) {
	c, ok := f.byID[id]
	if !ok {
		return components.Component{}, components.ErrComponentNotFound{IDOrLibraryID: id}
	}
	return c, nil
}

func (f *fakeLibrary) GetByLibraryID(_ context.Context, libraryID string) (components.Component, error) {
	for _, component := range f.byID {
		if component.LibraryID == libraryID {
			return component, nil
		}
	}
	return components.Component{}, components.ErrComponentNotFound{IDOrLibraryID: libraryID}
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
	if v, ok := f.versions[componentID+"@"+version]; ok {
		return v, nil
	}
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

func (f *fakeLibrary) ListVersions(ctx context.Context, componentID string, _ int) ([]components.ComponentVersion, error) {
	var out []components.ComponentVersion
	for key, v := range f.versions {
		if strings.HasPrefix(key, componentID+"@") {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		if _, ok := f.body[componentID]; ok {
			v, err := f.GetVersion(ctx, componentID, f.byID[componentID].Version)
			if err == nil {
				out = append(out, v)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

// fakeFiles is a minimal ScenarioFileReader keyed by "<scenario>::<path>".
type fakeFiles struct {
	bytes      map[string][]byte
	sites      map[string][]string
	provenance []adoptions.ProvenanceFile
	untagged   []adoptions.CandidateFile
}

func (f *fakeFiles) ScanUntagged(context.Context) ([]adoptions.CandidateFile, error) {
	return append([]adoptions.CandidateFile(nil), f.untagged...), nil
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

func (f *fakeFiles) ScanProvenance(context.Context) ([]adoptions.ProvenanceFile, error) {
	return append([]adoptions.ProvenanceFile(nil), f.provenance...), nil
}

type recordingReporter struct{ calls int }

func (r *recordingReporter) Report(context.Context, adoptions.DriftEvent) (adoptions.DriftReport, error) {
	r.calls++
	return adoptions.DriftReport{Ref: "bug/should-not-exist"}, nil
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

func TestService_ApplyVendorsEveryFileInAUnit(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{byID: map[string]components.Component{"cmp": {ID: "cmp", LibraryID: "rcl:Drawer", LatestVersion: "1.0.0"}}, versions: map[string]components.ComponentVersion{
		"cmp@1.0.0": {ComponentID: "cmp", LibraryID: "rcl:Drawer", Version: "1.0.0", Content: "import { trap } from './focus';", ContentSHA256: sha("entry"), Files: []components.ComponentVersionFile{
			{Path: "Drawer.tsx", Content: "import { trap } from './focus';", ContentSHA256: sha("entry"), IsEntry: true},
			{Path: "focus.ts", Content: "export const trap = () => null;", ContentSHA256: sha("focus")},
		}},
	}}
	files := &fakeFiles{bytes: map[string][]byte{}}
	result, err := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Now())).Apply(context.Background(), adoptions.ApplyInput{ComponentID: "cmp", Scenario: "target", AdoptedPath: "ui/src/components/Drawer.tsx"})
	require.NoError(t, err)
	require.Len(t, result.Adoption.Files, 2)
	require.Contains(t, string(files.bytes["target::ui/src/components/Drawer.tsx"]), "@vrooliComponentSource rcl:Drawer")
	require.Contains(t, string(files.bytes["target::ui/src/components/focus.ts"]), "@vrooliComponentSource rcl:Drawer")
}

func TestService_ApplyPlacesHookCompanionsInHookSlotAndRewritesImports(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{byID: map[string]components.Component{"cmp": {ID: "cmp", LibraryID: "rcl:Drawer", LatestVersion: "1.0.0"}}, versions: map[string]components.ComponentVersion{
		"cmp@1.0.0": {ComponentID: "cmp", LibraryID: "rcl:Drawer", Version: "1.0.0", ContentSHA256: sha("entry"), Files: []components.ComponentVersionFile{
			{Path: "Drawer.tsx", Content: `import { useFocusTrap } from "./useFocusTrap"; export const Drawer = () => useFocusTrap();`, ContentSHA256: sha("entry"), IsEntry: true},
			{Path: "useFocusTrap.ts", Content: "export const useFocusTrap = () => {};", ContentSHA256: sha("hook")},
		}},
	}}
	files := &fakeFiles{bytes: map[string][]byte{}}
	_, err := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Now())).Apply(context.Background(), adoptions.ApplyInput{ComponentID: "cmp", Scenario: "target", AdoptedPath: "ui/src/components/Drawer.tsx"})
	require.NoError(t, err)
	require.Contains(t, string(files.bytes["target::ui/src/components/Drawer.tsx"]), `from "../hooks/useFocusTrap"`)
	require.Contains(t, string(files.bytes["target::ui/src/hooks/useFocusTrap.ts"]), "useFocusTrap")
	require.NotContains(t, files.bytes, "target::ui/src/components/useFocusTrap.ts")
}

func TestService_ApplyMaterializesPinnedHookDependencyOnceWithIndependentProvenance(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{byID: map[string]components.Component{
		"panel": {ID: "panel", LibraryID: "rcl:FocusTrapPanel", DisplayName: "FocusTrapPanel", AssetKind: components.AssetKindComponent, LatestVersion: "1.0.0", Dependencies: []components.AssetDependency{{LibraryID: "rcl:useFocusTrap", Version: "1.0.0"}}},
		"hook":  {ID: "hook", LibraryID: "rcl:useFocusTrap", DisplayName: "useFocusTrap", AssetKind: components.AssetKindHook, LatestVersion: "1.0.0"},
	}, versions: map[string]components.ComponentVersion{
		"panel@1.0.0": {ComponentID: "panel", LibraryID: "rcl:FocusTrapPanel", Version: "1.0.0", SourcePath: "components/FocusTrapPanel/versions/1.0.0/FocusTrapPanel.tsx", ContentSHA256: sha("panel"), Files: []components.ComponentVersionFile{{Path: "FocusTrapPanel.tsx", Content: `import { useFocusTrap } from "../../../hooks/useFocusTrap/versions/1.0.0/useFocusTrap"; export const Panel = () => useFocusTrap;`, ContentSHA256: sha("panel"), IsEntry: true}}},
		"hook@1.0.0":  {ComponentID: "hook", LibraryID: "rcl:useFocusTrap", Version: "1.0.0", SourcePath: "hooks/useFocusTrap/versions/1.0.0/useFocusTrap.ts", ContentSHA256: sha("hook"), Files: []components.ComponentVersionFile{{Path: "useFocusTrap.ts", Content: "export const useFocusTrap = () => {};", ContentSHA256: sha("hook"), IsEntry: true}}},
	}}
	files := &fakeFiles{bytes: map[string][]byte{}}
	result, err := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Now())).Apply(context.Background(), adoptions.ApplyInput{ComponentID: "panel", Scenario: "target", AdoptedPath: "ui/src/components/FocusTrapPanel.tsx"})
	require.NoError(t, err)
	require.Equal(t, "panel", result.Adoption.ComponentID)
	require.Contains(t, string(files.bytes["target::ui/src/components/FocusTrapPanel.tsx"]), `from "../hooks/useFocusTrap"`)
	require.Contains(t, string(files.bytes["target::ui/src/hooks/useFocusTrap.ts"]), "useFocusTrap")
	rows, err := repo.List(context.Background(), adoptions.ListQuery{Limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	for _, row := range rows {
		require.NotEmpty(t, row.ID)
		require.Len(t, row.Files, 1)
	}
}

func TestService_ReapplyRefreshesEveryFileInAUnit(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{byID: map[string]components.Component{"cmp": {ID: "cmp", LibraryID: "rcl:Drawer", LatestVersion: "1.0.0"}}, versions: map[string]components.ComponentVersion{
		"cmp@1.0.0": {ComponentID: "cmp", LibraryID: "rcl:Drawer", Version: "1.0.0", ContentSHA256: sha("entry-1"), Files: []components.ComponentVersionFile{{Path: "Drawer.tsx", Content: "export const Drawer = 1;", ContentSHA256: sha("entry-1"), IsEntry: true}, {Path: "focus.ts", Content: "export const focus = 1;", ContentSHA256: sha("focus-1")}}},
	}}
	files := &fakeFiles{bytes: map[string][]byte{}}
	svc := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Now()))
	created, err := svc.Apply(context.Background(), adoptions.ApplyInput{ComponentID: "cmp", Scenario: "target", AdoptedPath: "ui/src/Drawer.tsx"})
	require.NoError(t, err)
	lib.versions["cmp@1.0.0"] = components.ComponentVersion{ComponentID: "cmp", LibraryID: "rcl:Drawer", Version: "1.0.0", ContentSHA256: sha("entry-2"), Files: []components.ComponentVersionFile{{Path: "Drawer.tsx", Content: "export const Drawer = 2;", ContentSHA256: sha("entry-2"), IsEntry: true}, {Path: "focus.ts", Content: "export const focus = 2;", ContentSHA256: sha("focus-2")}}}
	updated, _, err := svc.Reapply(context.Background(), adoptions.ReapplyInput{ID: created.Adoption.ID})
	require.NoError(t, err)
	require.Len(t, updated.Files, 2)
	require.Contains(t, string(files.bytes["target::ui/src/focus.ts"]), "focus = 2")
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

func TestService_ReconcileDryRunAndApplyGroupsProvenanceWithoutReporter(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	repo.Seed(adoptions.Adoption{
		ID:          "known-unit",
		ComponentID: "other",
		LibraryID:   "rcl:Other",
		Scenario:    "other",
		AdoptedPath: "ui/src/Other.tsx",
	})
	// Library bodies match the on-disk copies verbatim, so reconcile records a
	// genuinely-clean baseline. (Production GetVersion always populates Content;
	// drift is decided by comparing the local body to the library body, not by
	// blindly hashing the local file.)
	lib := &fakeLibrary{byID: map[string]components.Component{"cmp": {ID: "cmp", LibraryID: "rcl:Drawer", LatestVersion: "1.0.0"}}, versions: map[string]components.ComponentVersion{
		"cmp@1.0.0": {ComponentID: "cmp", LibraryID: "rcl:Drawer", Version: "1.0.0", Status: components.VersionStatusReleased, Files: []components.ComponentVersionFile{{Path: "Drawer.tsx", Content: "entry", ContentSHA256: sha("entry"), IsEntry: true}, {Path: "useFocusTrap.ts", Content: "hook", ContentSHA256: sha("hook")}}},
	}}
	files := &fakeFiles{bytes: map[string][]byte{"target::ui/src/components/drawer.tsx": []byte("entry"), "target::ui/src/hooks/useFocusTrap.ts": []byte("hook")}, provenance: []adoptions.ProvenanceFile{
		{Scenario: "target", AdoptedPath: "ui/src/components/drawer.tsx", LibraryID: "rcl:Drawer", Version: "1.0.0", AdoptionID: "known-unit", Content: []byte("entry")},
		{Scenario: "target", AdoptedPath: "ui/src/hooks/useFocusTrap.ts", LibraryID: "rcl:Drawer", Version: "1.0.0", AdoptionID: "known-unit", Content: []byte("hook")},
	}}
	svc := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Now()))
	reporter := &recordingReporter{}
	adoptions.SetDriftReporter(svc, reporter, nil)
	dry, err := svc.Reconcile(context.Background(), adoptions.ReconcileInput{})
	require.NoError(t, err)
	require.Equal(t, 2, dry.Scanned)
	require.Equal(t, 1, dry.Created)
	rows, err := svc.List(context.Background(), adoptions.ListQuery{Limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 1)

	applied, err := svc.Reconcile(context.Background(), adoptions.ReconcileInput{Apply: true})
	require.NoError(t, err)
	require.Equal(t, 1, applied.Created)
	require.Zero(t, reporter.calls)
	rows, err = svc.List(context.Background(), adoptions.ListQuery{Limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	var reconciled adoptions.Adoption
	for _, row := range rows {
		if row.Scenario == "target" {
			reconciled = row
		}
	}
	require.NotEqual(t, "known-unit", reconciled.ID)
	require.Len(t, reconciled.Files, 2)
	require.Equal(t, adoptions.LibraryVersionStatusCurrent, reconciled.LibraryVersionStatus)
	require.Equal(t, adoptions.LocalStatusClean, reconciled.LocalStatus)

	again, err := svc.Reconcile(context.Background(), adoptions.ReconcileInput{})
	require.NoError(t, err)
	require.Equal(t, 2, again.AlreadyRecorded)
	require.Zero(t, again.Created)
}
