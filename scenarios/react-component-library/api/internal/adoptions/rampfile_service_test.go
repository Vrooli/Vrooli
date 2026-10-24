package adoptions_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"

	"react-component-library/internal/adoptions"
	adoptmocks "react-component-library/internal/adoptions/mocks"
	"react-component-library/internal/components"
	"react-component-library/internal/themes"
)

var testRampTokenRE = regexp.MustCompile(`--[A-Za-z0-9_-]+`)

type rampTokenInventory struct {
	files   *fakeFiles
	runtime []string
}

type importedRampFiles struct {
	*fakeFiles
	imports []components.LibraryPackageSpecifier
}

type tieredRampTokenInventory struct {
	rampTokenInventory
	tokens       map[string]themes.DesignToken
	fallbacks    map[string]string
	referenceErr error
	fallbackErr  error
}

func (r tieredRampTokenInventory) ReferenceTokens(context.Context) (map[string]themes.DesignToken, error) {
	return r.tokens, r.referenceErr
}

func (r tieredRampTokenInventory) TokenFallbackExemptions(context.Context) (map[string]string, error) {
	return r.fallbacks, r.fallbackErr
}

func (f *importedRampFiles) ImportedLibrarySpecifiers(context.Context, string) ([]components.LibraryPackageSpecifier, error) {
	return append([]components.LibraryPackageSpecifier(nil), f.imports...), nil
}

type contractCoverageVerdicts map[string]string

func (r contractCoverageVerdicts) GateVerdict(context.Context, components.Component, string, string, string) (string, error) {
	return r["default"], nil
}

func (r rampTokenInventory) TokenNamespace(context.Context, string) (string, error) {
	return "app", nil
}

func (r rampTokenInventory) DeclaredTokens(_ context.Context, scenario string) ([]string, error) {
	raw := r.files.bytes[scenario+"::ui/src/design-tokens.css"]
	return append(testRampTokenRE.FindAllString(string(raw), -1), r.runtime...), nil
}

func (r rampTokenInventory) RuntimeWrittenTokens(context.Context, string) ([]string, error) {
	return append([]string(nil), r.runtime...), nil
}

func newRampService(ramp string) (adoptions.Service, *fakeFiles) {
	return newRampServiceWithTokens(ramp, []string{"--color-primary"})
}

func newRampServiceWithTokens(ramp string, requiredTokens []string) (adoptions.Service, *fakeFiles) {
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
				RequiredTokens: append([]string(nil), requiredTokens...),
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

func TestAdoptTokenContractAcceptsCompleteScenarioStylesheet(t *testing.T) {
	svc, _ := newRampServiceWithTokens(":root { --color-primary: blue; --tap-target-min: 44px; }", []string{"--color-primary", "--tap-target-min"})
	result, err := svc.Apply(context.Background(), adoptions.ApplyInput{
		ComponentID: "cmp-button", Scenario: "target", AdoptedPath: "ui/src/Button.tsx",
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Adoption.ID)
}

func TestAdoptTokenContractRejectsAndNamesEveryMissingProperty(t *testing.T) {
	svc, files := newRampServiceWithTokens(":root { --color-primary: blue; }", []string{"--color-primary", "--space-sm", "--tap-target-min"})
	_, err := svc.Apply(context.Background(), adoptions.ApplyInput{
		ComponentID: "cmp-button", Scenario: "target", AdoptedPath: "ui/src/Button.tsx",
	})
	var unsatisfied adoptions.ErrAdoptionTokensUnsatisfied
	require.ErrorAs(t, err, &unsatisfied)
	require.Equal(t, []string{"--space-sm", "--tap-target-min"}, unsatisfied.Properties)
	require.Contains(t, unsatisfied.Error(), "--space-sm")
	require.Contains(t, unsatisfied.Error(), "--tap-target-min")
	require.NotContains(t, files.bytes, "target::ui/src/Button.tsx")
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

func TestSyncScenarioTokensUnionsImportedAssetsWithAdoptionLedger(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-button": {ID: "cmp-button", LibraryID: "react-component-library:Button", LatestVersion: "2.1.0"},
		},
		versions: map[string]components.ComponentVersion{
			"cmp-button@1.4.2": {ComponentID: "cmp-button", LibraryID: "react-component-library:Button", Version: "1.4.2", Status: components.VersionStatusReleased, RequiredTokens: []string{"--color-old"}},
			"cmp-button@2.1.0": {ComponentID: "cmp-button", LibraryID: "react-component-library:Button", Version: "2.1.0", Status: components.VersionStatusReleased, RequiredTokens: []string{"--color-imported"}},
		},
	}
	baseFiles := &fakeFiles{bytes: map[string][]byte{}}
	files := &importedRampFiles{fakeFiles: baseFiles, imports: []components.LibraryPackageSpecifier{{Name: "Button", RequestedVersion: "2"}}}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))
	adoptions.SetTokenNamespaceReader(svc, rampTokenInventory{files: baseFiles})

	result, err := svc.SyncScenarioTokens(context.Background(), adoptions.TokenSyncInput{Scenario: "target", DryRun: true})
	require.NoError(t, err)
	require.Equal(t, []string{"--color-imported"}, result.Added)
}

func TestSyncScenarioTokensManagesKnownNonContractTokensOnly(t *testing.T) {
	svc, files := newRampServiceWithTokens(":root {\n/* rcl:tokens:begin */\n  --layer-modal: 400;\n/* rcl:tokens:end */\n}\n", []string{"--color-primary", "--layer-modal", "--legacy-untiered"})
	adoptions.SetTokenNamespaceReader(svc, tieredRampTokenInventory{
		rampTokenInventory: rampTokenInventory{files: files},
		tokens: map[string]themes.DesignToken{
			"--color-primary":   {Name: "--color-primary", Value: "blue", Tier: themes.TokenTierExpression},
			"--layer-modal":     {Name: "--layer-modal", Value: "400", Tier: themes.TokenTierContract},
			"--legacy-untiered": {Name: "--legacy-untiered", Value: "legacy"},
		},
	})

	result, err := svc.SyncScenarioTokens(context.Background(), adoptions.TokenSyncInput{Scenario: "target"})
	require.NoError(t, err)
	require.Equal(t, []string{"--color-primary", "--legacy-untiered"}, result.Added)
	require.NotContains(t, string(files.bytes["target::ui/src/design-tokens.css"]), "--layer-modal")
}

func TestFSScenarioFileReaderReadsCanonicalTokenTiers(t *testing.T) {
	root := t.TempDir()
	scenariosRoot := filepath.Join(root, "scenarios")
	tokenPath := filepath.Join(root, "templates", "design", "_base", "tokens.css")
	catalogPath := filepath.Join(scenariosRoot, "react-component-library", "catalog", "config.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(tokenPath), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(catalogPath), 0o755))
	require.NoError(t, os.WriteFile(tokenPath, []byte(":root {\n  /* @tier Expression */\n  --color-primary: blue;\n  /* @tier Contract */\n  --layer-modal: 400;\n}\n"), 0o600))
	require.NoError(t, os.WriteFile(catalogPath, []byte(`{"x-token-fallback-exemptions":[{"token":"--rcl-safe-bottom","reason":"host safe-area inset"}]}`), 0o600))

	reader := adoptions.NewFSScenarioFileReader(scenariosRoot)
	tokens, err := reader.ReferenceTokens(context.Background())
	require.NoError(t, err)
	require.Equal(t, themes.TokenTierExpression, tokens["--color-primary"].Tier)
	require.Equal(t, themes.TokenTierContract, tokens["--layer-modal"].Tier)
	exemptions, err := reader.TokenFallbackExemptions(context.Background())
	require.NoError(t, err)
	require.Equal(t, "host safe-area inset", exemptions["--rcl-safe-bottom"])
}

func TestSyncScenarioTokensResolvesReindexedComponentByStableLibraryID(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	repo.Seed(adoptions.Adoption{
		ID: "adoption-1", ComponentID: "stale-component-uuid", LibraryID: "rcl:Button", Scenario: "target", AdoptedPath: "ui/src/Button.tsx", AdoptedVersion: "1.0.0",
	})
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"current-component-uuid": {ID: "current-component-uuid", LibraryID: "rcl:Button", LatestVersion: "1.1.0"},
		},
		versions: map[string]components.ComponentVersion{
			"current-component-uuid@1.1.0": {
				ComponentID: "current-component-uuid", LibraryID: "rcl:Button", Version: "1.1.0",
				Status: components.VersionStatusReleased, SourcePath: "Button.tsx",
				RequiredTokens: []string{"--color-current"},
			},
		},
	}
	files := &fakeFiles{bytes: map[string][]byte{}}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))
	adoptions.SetTokenNamespaceReader(svc, rampTokenInventory{files: files})

	result, err := svc.SyncScenarioTokens(context.Background(), adoptions.TokenSyncInput{Scenario: "target"})
	require.NoError(t, err)
	require.Equal(t, []string{"--color-current"}, result.Added)
}

func TestSyncScenarioTokensRemovesRuntimeOwnedPropertyFromManagedRamp(t *testing.T) {
	ramp := `:root {
/* rcl:tokens:begin */
  --wc-kb-height: 2.5rem;
/* rcl:tokens:end */
}`
	svc, files := newRampServiceWithTokens(ramp, []string{"--wc-kb-height"})
	adoptions.SetTokenNamespaceReader(svc, rampTokenInventory{files: files, runtime: []string{"--wc-kb-height"}})

	result, err := svc.SyncScenarioTokens(context.Background(), adoptions.TokenSyncInput{Scenario: "target"})
	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Empty(t, result.Added)
	require.NotContains(t, string(files.bytes["target::ui/src/design-tokens.css"]), "--wc-kb-height:")
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

func TestPreflightUsesCanonicalTokenClassificationAndExactFallbackExemptions(t *testing.T) {
	required := []string{"--color-primary", "--border-focus", "--rcl-safe-bottom", "--rcl-panel-size", "--unknown-contract"}
	svc, files := newRampServiceWithTokens("", required)
	adoptions.SetTokenNamespaceReader(svc, tieredRampTokenInventory{
		rampTokenInventory: rampTokenInventory{files: files},
		tokens: map[string]themes.DesignToken{
			"--color-primary": {Name: "--color-primary", Value: "blue", Tier: themes.TokenTierExpression},
			"--border-focus":  {Name: "--border-focus", Value: "2px", Tier: themes.TokenTierContract},
		},
		fallbacks: map[string]string{
			"--rcl-safe-bottom": "browser safe-area inset supplied by the host",
			"--rcl-panel-size":  "resizable panel runtime state",
		},
	})

	preflight, err := svc.Preflight(context.Background(), adoptions.PreflightInput{
		ComponentID: "cmp-button", Scenario: "target", Version: "1.0.0",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"--color-primary", "--unknown-contract"}, preflight.Tokens.Unsatisfied)
}

func TestApplyFailsClosedWithoutWritingWhenCanonicalTokenSourcesCannotBeRead(t *testing.T) {
	for name, inventory := range map[string]tieredRampTokenInventory{
		"canonical vocabulary": {referenceErr: os.ErrPermission},
		"fallback exemptions":  {fallbackErr: os.ErrPermission},
	} {
		t.Run(name, func(t *testing.T) {
			svc, files := newRampServiceWithTokens("", []string{"--color-primary"})
			inventory.rampTokenInventory = rampTokenInventory{files: files}
			adoptions.SetTokenNamespaceReader(svc, inventory)

			_, err := svc.Apply(context.Background(), adoptions.ApplyInput{
				ComponentID: "cmp-button", Scenario: "target", AdoptedPath: "ui/src/Button.tsx",
			})
			require.ErrorIs(t, err, os.ErrPermission)
			require.NotContains(t, files.bytes, "target::ui/src/Button.tsx")
		})
	}
}

func TestPreflightSurfacesContractCoverageVerdicts(t *testing.T) {
	svc, _ := newRampService(":root { --color-primary: blue; }")
	coverage := contractCoverageVerdicts{"default": "fail"}
	adoptions.SetContractCoverageReader(svc, coverage)

	preflight, err := svc.Preflight(context.Background(), adoptions.PreflightInput{
		ComponentID: "cmp-button", Scenario: "target", Version: "1.0.0",
	})
	require.NoError(t, err)
	require.Equal(t, "fail", preflight.I18n)
	require.Equal(t, "fail", preflight.Selectors)
	require.True(t, preflight.Blocking)
}

func TestTokenOperationsPreserveGeneratedAuthority(t *testing.T) {
	const original = "/* Generated by node ui/scripts/design-token-generate.mjs. DO NOT EDIT. */\n:root { --color-primary: blue; }\n"
	svc, files := newRampServiceWithTokens(original, []string{"--color-primary"})
	_, err := svc.SyncScenarioTokens(context.Background(), adoptions.TokenSyncInput{Scenario: "target"})
	require.ErrorContains(t, err, "token ramp is generated")
	_, err = svc.PruneScenarioTokens(context.Background(), adoptions.TokenPruneInput{Scenario: "target", Apply: true})
	require.ErrorContains(t, err, "token ramp is generated")
	require.Equal(t, original, string(files.bytes["target::ui/src/design-tokens.css"]))
}
