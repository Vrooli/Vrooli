package adoptions_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"

	"react-component-library/internal/adoptions"
	adoptmocks "react-component-library/internal/adoptions/mocks"
	"react-component-library/internal/components"
)

var testRampTokenRE = regexp.MustCompile(`--[A-Za-z0-9_-]+`)

type rampTokenInventory struct{ files *fakeFiles }

func (r rampTokenInventory) TokenNamespace(context.Context, string) (string, error) {
	return "app", nil
}

func (r rampTokenInventory) DeclaredTokens(_ context.Context, scenario string) ([]string, error) {
	raw := r.files.bytes[scenario+"::ui/src/design-tokens.css"]
	return testRampTokenRE.FindAllString(string(raw), -1), nil
}

func newRampService(ramp string) (adoptions.Service, *fakeFiles) {
	repo := adoptmocks.NewFakeRepository()
	repo.Seed(adoptions.Adoption{
		ID: "adoption-1", ComponentID: "cmp-button", Scenario: "target", AdoptedPath: "ui/src/Button.tsx", AdoptedVersion: "1.0.0",
	})
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-button": {ID: "cmp-button", LibraryID: "rcl:Button", LatestVersion: "1.0.0"},
		},
		versions: map[string]components.ComponentVersion{
			"cmp-button@1.0.0": {
				ComponentID: "cmp-button", LibraryID: "rcl:Button", Version: "1.0.0",
				Status: components.VersionStatusReleased, SourcePath: "Button.tsx",
				RequiredTokens: []string{"--color-primary"},
			},
		},
	}
	files := &fakeFiles{bytes: map[string][]byte{}}
	if ramp != "" {
		files.bytes["target::ui/src/design-tokens.css"] = []byte(ramp)
	}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))
	adoptions.SetTokenNamespaceReader(svc, rampTokenInventory{files: files})
	return svc, files
}

func TestSyncScenarioTokensMissingFileIsIdempotent(t *testing.T) {
	svc, files := newRampService("")

	first, err := svc.SyncScenarioTokens(context.Background(), adoptions.TokenSyncInput{Scenario: "target"})
	require.NoError(t, err)
	require.Equal(t, []string{"--color-primary"}, first.Added)
	require.True(t, first.Changed)
	written := append([]byte(nil), files.bytes["target::ui/src/design-tokens.css"]...)

	second, err := svc.SyncScenarioTokens(context.Background(), adoptions.TokenSyncInput{Scenario: "target"})
	require.NoError(t, err)
	require.Empty(t, second.Added)
	require.False(t, second.Changed)
	require.Equal(t, written, files.bytes["target::ui/src/design-tokens.css"])
}

func TestSyncScenarioTokensReportsScenarioOwnedCollision(t *testing.T) {
	ramp := ":root { --color-primary: pink; }\n"
	svc, files := newRampService(ramp)

	result, err := svc.SyncScenarioTokens(context.Background(), adoptions.TokenSyncInput{Scenario: "target"})
	require.NoError(t, err)
	require.Equal(t, []string{"--color-primary"}, result.Collisions)
	require.Empty(t, result.Added)
	require.Contains(t, string(files.bytes["target::ui/src/design-tokens.css"]), "--color-primary: pink")
	require.NotContains(t, string(files.bytes["target::ui/src/design-tokens.css"]), "--color-primary: initial")
}

func TestSyncScenarioTokensUsesLatestReleasedVersion(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	repo.Seed(adoptions.Adoption{
		ID: "adoption-1", ComponentID: "cmp-button", Scenario: "target", AdoptedPath: "ui/src/Button.tsx", AdoptedVersion: "0.9.0",
	})
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-button": {ID: "cmp-button", LibraryID: "rcl:Button", LatestVersion: "1.0.0"},
		},
		versions: map[string]components.ComponentVersion{
			"cmp-button@0.9.0": {
				ComponentID: "cmp-button", LibraryID: "rcl:Button", Version: "0.9.0",
				Status: components.VersionStatusReleased, SourcePath: "Button.tsx",
				RequiredTokens: []string{"--color-old"},
			},
			"cmp-button@1.0.0": {
				ComponentID: "cmp-button", LibraryID: "rcl:Button", Version: "1.0.0",
				Status: components.VersionStatusReleased, SourcePath: "Button.tsx",
				RequiredTokens: []string{"--color-latest"},
			},
		},
	}
	files := &fakeFiles{bytes: map[string][]byte{}}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))
	adoptions.SetTokenNamespaceReader(svc, rampTokenInventory{files: files})

	result, err := svc.SyncScenarioTokens(context.Background(), adoptions.TokenSyncInput{Scenario: "target"})
	require.NoError(t, err)
	require.Equal(t, []string{"--color-latest"}, result.Added)
	require.NotContains(t, result.Added, "--color-old")
	require.Contains(t, string(files.bytes["target::ui/src/design-tokens.css"]), "--color-latest: initial;")
}

func TestPruneScenarioTokensMissingFileIsEmpty(t *testing.T) {
	svc, _ := newRampService("")
	result, err := svc.PruneScenarioTokens(context.Background(), adoptions.TokenPruneInput{Scenario: "target"})
	require.NoError(t, err)
	require.Empty(t, result.Removed)
	require.Empty(t, result.Retained)
	require.False(t, result.Changed)
}

func TestTokenVerdictBlocksApplyAndRemainsVisibleAfterOverride(t *testing.T) {
	svc, files := newRampService("")

	_, err := svc.Apply(context.Background(), adoptions.ApplyInput{
		ComponentID: "cmp-button", Scenario: "target", AdoptedPath: "ui/src/Button.tsx",
	})
	var unsatisfied adoptions.ErrAdoptionTokensUnsatisfied
	require.ErrorAs(t, err, &unsatisfied)
	require.Contains(t, unsatisfied.Error(), "--color-primary")
	require.NotContains(t, files.bytes, "target::ui/src/Button.tsx")

	result, err := svc.Apply(context.Background(), adoptions.ApplyInput{
		ComponentID: "cmp-button", Scenario: "target", AdoptedPath: "ui/src/Button.tsx",
		OverrideValidation: true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Adoption.ID)

	preflight, err := svc.Preflight(context.Background(), adoptions.PreflightInput{
		ComponentID: "cmp-button", Scenario: "target", Version: "1.0.0",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"--color-primary"}, preflight.Tokens.Unsatisfied)
}
