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
	"react-component-library/internal/testutil/mocks"
)

// fakeLibrary is a minimal LibraryReader keyed by component_id.
type fakeLibrary struct {
	byID map[string]components.Component
	body map[string]string // id -> file content
}

func (f *fakeLibrary) Get(_ context.Context, id string) (components.Component, error) {
	c, ok := f.byID[id]
	if !ok {
		return components.Component{}, components.ErrComponentNotFound{IDOrLibraryID: id}
	}
	return c, nil
}

func (f *fakeLibrary) GetContent(_ context.Context, id string) (components.Content, error) {
	body, ok := f.body[id]
	if !ok {
		return components.Content{}, components.ErrComponentNotFound{IDOrLibraryID: id}
	}
	sum := sha256.Sum256([]byte(body))
	return components.Content{Body: body, SHA256: hex.EncodeToString(sum[:])}, nil
}

// fakeFiles is a minimal ScenarioFileReader keyed by "<scenario>::<path>".
type fakeFiles struct {
	bytes map[string][]byte
}

func (f *fakeFiles) Read(_ context.Context, scenario, p string) ([]byte, error) {
	b, ok := f.bytes[scenario+"::"+p]
	if !ok {
		return nil, adoptions.ErrAdoptedFileMissing{Scenario: scenario, AdoptedPath: p}
	}
	return b, nil
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
			"cmp-btn": {ID: "cmp-btn", LibraryID: "rcl:Button", Version: "1.1.0"},
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
		"swarm-manager::a-current.tsx":  []byte(bodyV11),     // matches library
		"swarm-manager::b-behind.tsx":   []byte(bodyV10),     // matches snapshot
		"swarm-manager::c-modified.tsx": []byte(bodyEdited),  // diverged
		// d-unknown.tsx intentionally missing
		// e-component-gone.tsx exists but component no longer in library
		"swarm-manager::e-component-gone.tsx": []byte("anything"),
	}}

	clk := mocks.NewFakeClock(time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC))

	// Seed adoption rows directly (Seed bypasses Create's gates).
	rows := []adoptions.Adoption{
		{ID: "row-current", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
			Scenario: "swarm-manager", AdoptedPath: "a-current.tsx",
			AdoptedVersion: "1.1.0", AdoptedSnapshotSHA256: sha(bodyV11)},
		{ID: "row-behind", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
			Scenario: "swarm-manager", AdoptedPath: "b-behind.tsx",
			AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha(bodyV10)},
		{ID: "row-modified", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
			Scenario: "swarm-manager", AdoptedPath: "c-modified.tsx",
			AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha(bodyV10)},
		{ID: "row-unknown-missing", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
			Scenario: "swarm-manager", AdoptedPath: "d-unknown.tsx",
			AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha(bodyV10)},
		{ID: "row-unknown-gone", ComponentID: "cmp-gone", LibraryID: "rcl:Removed",
			Scenario: "swarm-manager", AdoptedPath: "e-component-gone.tsx",
			AdoptedVersion: "0.1.0", AdoptedSnapshotSHA256: sha("anything")},
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
	require.Equal(t, adoptions.StatusCurrent, byID["row-current"].Status)
	require.Equal(t, adoptions.StatusBehind, byID["row-behind"].Status)
	require.Contains(t, byID["row-behind"].StatusDetail, "1.1.0")
	require.Equal(t, adoptions.StatusModified, byID["row-modified"].Status)
	require.Equal(t, adoptions.StatusUnknown, byID["row-unknown-missing"].Status)
	require.Contains(t, byID["row-unknown-missing"].StatusDetail, "missing")
	require.Equal(t, adoptions.StatusUnknown, byID["row-unknown-gone"].Status)
	require.Contains(t, byID["row-unknown-gone"].StatusDetail, "removed")

	require.Equal(t, adoptions.RefreshSummary{Current: 1, Behind: 1, Modified: 1, Unknown: 2}, summary)

	// Refresh wrote refreshed_at on every row.
	for _, a := range got {
		require.False(t, a.RefreshedAt.IsZero(), "row %s did not get refreshed_at", a.ID)
	}
}

func TestService_Refresh_FilterByComponent(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-a": {ID: "cmp-a", LibraryID: "x:A"},
			"cmp-b": {ID: "cmp-b", LibraryID: "x:B"},
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
