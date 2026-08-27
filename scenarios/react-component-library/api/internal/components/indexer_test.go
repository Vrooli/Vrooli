package components_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
	"react-component-library/internal/components/mocks"
)

func testDigest(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

func TestIndexer_DrawerShellDeclaresReusableHookDependencies(t *testing.T) {
	repo := mocks.NewFakeRepository()
	result, err := components.NewIndexer(repo, ".", os.DirFS("../../../library")).Run(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, result.LibraryIDs)

	drawer, err := repo.GetByLibraryID(context.Background(), "react-component-library:DrawerShell")
	require.NoError(t, err)
	require.Equal(t, []components.AssetDependency{
		{LibraryID: "react-component-library:useFocusTrap", Version: "1.0.0"},
		{LibraryID: "react-component-library:useEscapeKey", Version: "1.0.0"},
	}, drawer.Dependencies)

	for _, libraryID := range []string{"react-component-library:useFocusTrap", "react-component-library:useEscapeKey"} {
		hook, err := repo.GetByLibraryID(context.Background(), libraryID)
		require.NoError(t, err)
		require.Equal(t, components.AssetKindHook, hook.AssetKind)
	}

	tokens, err := repo.GetByLibraryID(context.Background(), "react-component-library:Tokens")
	require.NoError(t, err)
	require.Equal(t, components.AssetKindFoundation, tokens.AssetKind)
}

const buttonTSX = `/**
 * @libraryId   react-component-library:Button
 * @version     1.0.0
 * @tags        ["form", "interactive"]
 * @category    controls
 * @warning     DO NOT REMOVE THIS HEADER
 */
import React from 'react';
export const Button = () => <button>click</button>;
`

const cardTSX = `/**
 * @libraryId react-component-library:Card
 * @version 1.0.0
 */
export const Card = () => null;
`

func TestIndexer_RunWalksAndUpserts(t *testing.T) {
	fs := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifestWithStyles("react-component-library:Button", "Button", `["form","interactive"]`, `[{"styleId":"vrooli-default","affinity":"native","reason":"token-native baseline"},{"styleId":"vrooli-conversion-landing","affinity":"discouraged"}]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
		"components/Card/component.json":              {Data: []byte(manifest("react-component-library:Card", "Card", `["layout","container"]`))},
		"components/Card/versions/1.0.0/Card.tsx":     {Data: []byte(cardTSX)},
		"components/Card/versions/1.0.0/README.md":    {Data: []byte("# nope")},
	}
	repo := mocks.NewFakeRepository()
	idx := components.NewIndexer(repo, ".", fs)

	res, err := idx.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, res.Scanned)
	require.Equal(t, 2, res.Indexed)
	require.Equal(t, 0, res.Skipped)
	require.Empty(t, res.Errors)

	got, err := repo.GetByLibraryID(context.Background(), "react-component-library:Button")
	require.NoError(t, err)
	require.Equal(t, "Button", got.DisplayName)
	require.Equal(t, "ui-primitive", got.Slot)
	require.Equal(t, "controls", got.Category)
	require.Equal(t, "1.0.0", got.Version)
	require.Equal(t, []string{"form", "interactive"}, got.Tags)
	require.Equal(t, "components/Button/versions/1.0.0/Button.tsx", got.SourcePath)
	require.Equal(t, "DO NOT REMOVE THIS HEADER", got.Headers["warning"])
	require.NotContains(t, got.Headers, "libraryId")
	require.NotContains(t, got.Headers, "version")
	require.NotContains(t, got.Headers, "category")
	require.Equal(t, []components.ComponentDesignAffinity{
		{StyleID: "vrooli-default", Affinity: components.DesignAffinityNative, Reason: "token-native baseline"},
		{StyleID: "vrooli-conversion-landing", Affinity: components.DesignAffinityDiscouraged},
	}, got.DesignStyles)

	got2, err := repo.GetByLibraryID(context.Background(), "react-component-library:Card")
	require.NoError(t, err)
	require.Equal(t, []string{"layout", "container"}, got2.Tags)
}

func TestIndexer_IndexManifestIsolatesAuthoringFromUnrelatedComponents(t *testing.T) {
	fs := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
		"components/Broken/component.json":            {Data: []byte(`{"displayName":"Broken","latest":"1.0.0"}`)},
	}
	repo := mocks.NewFakeRepository()
	indexed, err := components.NewIndexer(repo, ".", fs).IndexManifest(context.Background(), "components/Button/component.json")
	require.NoError(t, err)
	require.Equal(t, "react-component-library:Button", indexed.LibraryID)
	_, err = repo.GetByLibraryID(context.Background(), "react-component-library:Broken")
	var notFound components.ErrComponentNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestIndexer_RunIndexesEntryAndCompanionFiles(t *testing.T) {
	fs := fstest.MapFS{
		"components/FocusTrap/component.json": {Data: []byte(`{"libraryId":"react-component-library:FocusTrap","displayName":"Focus Trap","slot":"ui-pattern","entry":"FocusTrap.tsx","latest":"1.0.0","deprecatedVersions":[]}`)},
		"components/FocusTrap/versions/1.0.0/FocusTrap.tsx": {Data: []byte(`/**
 * @libraryId react-component-library:FocusTrap
 * @version 1.0.0
 */
import { cycle } from "./focus";
export const FocusTrap = () => cycle();`)},
		"components/FocusTrap/versions/1.0.0/focus.ts": {Data: []byte(`export const cycle = () => null;`)},
	}
	repo := mocks.NewFakeRepository()
	res, err := components.NewIndexer(repo, ".", fs).Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Indexed)
	component, err := repo.GetByLibraryID(context.Background(), "react-component-library:FocusTrap")
	require.NoError(t, err)
	version, err := repo.GetVersion(context.Background(), component.ID, "1.0.0")
	require.NoError(t, err)
	require.Len(t, version.Files, 2)
	require.Equal(t, "FocusTrap.tsx", version.Files[0].Path)
	require.True(t, version.Files[0].IsEntry)
	require.Equal(t, "focus.ts", version.Files[1].Path)
}

func TestIndexer_PreservesDeclaredEvictedVersionFromLedger(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	_, err := repo.UpsertManifest(ctx, components.IndexManifestInput{
		Manifest: components.ComponentManifest{LibraryID: "react-component-library:ColdButton", Slug: "ColdButton", DisplayName: "Cold Button", LatestVersion: "1.0.0", AssetKind: components.AssetKindComponent},
		Versions: []components.ComponentVersion{
			{Version: "0.9.0", Status: components.VersionStatusReleased, Presence: "materialized", SourcePath: "components/ColdButton/versions/0.9.0/ColdButton.tsx", Content: "old", ContentSHA256: testDigest("old"), Files: []components.ComponentVersionFile{{Path: "ColdButton.tsx", Content: "old", ContentSHA256: testDigest("old"), IsEntry: true}}},
			{Version: "1.0.0", Status: components.VersionStatusReleased, Presence: "materialized", SourcePath: "components/ColdButton/versions/1.0.0/ColdButton.tsx", Content: "new", ContentSHA256: testDigest("new"), Files: []components.ComponentVersionFile{{Path: "ColdButton.tsx", Content: "new", ContentSHA256: testDigest("new"), IsEntry: true}}},
		},
	})
	require.NoError(t, err)
	fs := fstest.MapFS{
		"components/ColdButton/component.json":                {Data: []byte(`{"libraryId":"react-component-library:ColdButton","displayName":"Cold Button","entry":"ColdButton.tsx","latest":"1.0.0","deprecatedVersions":[],"evictedVersions":["0.9.0"]}`)},
		"components/ColdButton/versions/1.0.0/ColdButton.tsx": {Data: []byte("new")},
	}
	result, err := components.NewIndexer(repo, ".", fs).Run(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, result.Indexed)
	component, err := repo.GetByLibraryID(ctx, "react-component-library:ColdButton")
	require.NoError(t, err)
	version, err := repo.GetVersion(ctx, component.ID, "0.9.0")
	require.NoError(t, err)
	require.Equal(t, "evicted", version.Presence)
}

func TestIndexerIgnoresLegacyManifestTokenContractAndUsesDerivedTokens(t *testing.T) {
	fs := fstest.MapFS{
		"components/TokenBound/component.json": {Data: []byte(`{"libraryId":"react-component-library:TokenBound","displayName":"Token Bound","entry":"TokenBound.tsx","latest":"1.0.0","deprecatedVersions":[],"requiredTokens":["--tap-target-min","--space-sm","--tap-target-min"]}`)},
		"components/TokenBound/versions/1.0.0/TokenBound.tsx": {Data: []byte(`/**
 * @libraryId react-component-library:TokenBound
 * @version 1.0.0
 */
export const TokenBound = () => <button style={{ color: "var(--color-foreground)" }} />;`)},
	}
	repo := mocks.NewFakeRepository()
	_, err := components.NewIndexer(repo, ".", fs).Run(context.Background())
	require.NoError(t, err)
	component, err := repo.GetByLibraryID(context.Background(), "react-component-library:TokenBound")
	require.NoError(t, err)
	version, err := repo.GetVersion(context.Background(), component.ID, "1.0.0")
	require.NoError(t, err)
	require.Equal(t, []string{"--color-foreground"}, version.RequiredTokens)
}

func TestIndexerIgnoresInvalidLegacyManifestTokenContract(t *testing.T) {
	fs := fstest.MapFS{
		"components/TokenBound/component.json": {Data: []byte(`{"libraryId":"react-component-library:TokenBound","displayName":"Token Bound","entry":"TokenBound.tsx","latest":"1.0.0","deprecatedVersions":[],"requiredTokens":["color-foreground"]}`)},
		"components/TokenBound/versions/1.0.0/TokenBound.tsx": {Data: []byte(`/**
 * @libraryId react-component-library:TokenBound
 * @version 1.0.0
 */`)},
	}
	_, err := components.NewIndexer(mocks.NewFakeRepository(), ".", fs).IndexManifest(context.Background(), "components/TokenBound/component.json")
	require.NoError(t, err)
}

func TestIndexer_RunIndexesHookAsNonRenderableAsset(t *testing.T) {
	fs := fstest.MapFS{
		"hooks/useFocusTrap/component.json":                     {Data: []byte(`{"libraryId":"react-component-library:useFocusTrap","displayName":"useFocusTrap","assetKind":"hook","latest":"1.0.0","dependencies":[]}`)},
		"hooks/useFocusTrap/versions/1.0.0/useFocusTrap.ts":     {Data: []byte(`export const useFocusTrap = () => undefined;`)},
		"hooks/useEscapeKey/component.json":                     {Data: []byte(`{"libraryId":"react-component-library:useEscapeKey","displayName":"useEscapeKey","assetKind":"hook","latest":"1.0.0","dependencies":[]}`)},
		"hooks/useEscapeKey/versions/1.0.0/useEscapeKey.ts":     {Data: []byte(`export const useEscapeKey = () => undefined;`)},
		"components/DrawerShell/component.json":                 {Data: []byte(`{"libraryId":"react-component-library:DrawerShell","displayName":"DrawerShell","slot":"ui-pattern","latest":"1.0.0","dependencies":[{"libraryId":"react-component-library:useFocusTrap","version":"1.0.0"},{"libraryId":"react-component-library:useEscapeKey","version":"1.0.0"}]}`)},
		"components/DrawerShell/versions/1.0.0/DrawerShell.tsx": {Data: []byte(`export const DrawerShell = () => null;`)},
	}
	repo := mocks.NewFakeRepository()
	res, err := components.NewIndexer(repo, ".", fs).Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 3, res.Indexed)

	hook, err := repo.GetByLibraryID(context.Background(), "react-component-library:useFocusTrap")
	require.NoError(t, err)
	require.Equal(t, components.AssetKindHook, hook.AssetKind)
	require.Empty(t, hook.Slot)
	escapeKey, err := repo.GetByLibraryID(context.Background(), "react-component-library:useEscapeKey")
	require.NoError(t, err)
	require.Equal(t, components.AssetKindHook, escapeKey.AssetKind)
	require.Empty(t, escapeKey.Slot)

	component, err := repo.GetByLibraryID(context.Background(), "react-component-library:DrawerShell")
	require.NoError(t, err)
	require.Equal(t, components.AssetKindComponent, component.AssetKind)
	require.Equal(t, []components.AssetDependency{
		{LibraryID: "react-component-library:useFocusTrap", Version: "1.0.0"},
		{LibraryID: "react-component-library:useEscapeKey", Version: "1.0.0"},
	}, component.Dependencies)
}

func TestIndexer_RunIndexesPrimitiveAsRenderableComponent(t *testing.T) {
	fs := fstest.MapFS{
		"primitives/Presence/component.json": {Data: []byte(`{"libraryId":"react-component-library:Presence","catalogId":"motion.presence","displayName":"Presence","assetKind":"primitive","latest":"1.0.0","dependencies":[]}`)},
		"primitives/Presence/versions/1.0.0/Presence.tsx": {Data: []byte(`/** @vrooliComponentSource react-component-library:Presence */
export const Presence = () => null;`)},
	}
	repo := mocks.NewFakeRepository()
	res, err := components.NewIndexer(repo, ".", fs).Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Scanned)
	require.Equal(t, 1, res.Indexed)

	got, err := repo.GetByLibraryID(context.Background(), "react-component-library:Presence")
	require.NoError(t, err)
	require.Equal(t, components.AssetKindComponent, got.AssetKind)
	require.Equal(t, "primitives/Presence/component.json", got.ManifestPath)
	require.Equal(t, "primitives/Presence/versions/1.0.0/Presence.tsx", got.SourcePath)
}

func TestIndexer_RunIndexesRuntimeServiceFromCanonicalRoot(t *testing.T) {
	fs := fstest.MapFS{
		"services/FormStore/component.json":               {Data: []byte(`{"libraryId":"react-component-library:FormStore","displayName":"Form Store","assetKind":"service","latest":"1.0.0","dependencies":[]}`)},
		"services/FormStore/versions/1.0.0/FormStore.tsx": {Data: []byte(`export const FormStore = () => null;`)},
	}
	repo := mocks.NewFakeRepository()
	res, err := components.NewIndexer(repo, ".", fs).Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Scanned)
	require.Equal(t, 1, res.Indexed)

	service, err := repo.GetByLibraryID(context.Background(), "react-component-library:FormStore")
	require.NoError(t, err)
	require.Equal(t, components.AssetKindComponent, service.AssetKind)
	require.Equal(t, "services/FormStore/component.json", service.ManifestPath)
	require.Equal(t, "services/FormStore/versions/1.0.0/FormStore.tsx", service.SourcePath)
}

func TestCanonicalCatalogRootsMatchTestGenieApplicability(t *testing.T) {
	root := filepath.Clean("../../../")
	configBytes, err := os.ReadFile(filepath.Join(root, "catalog", "config.json"))
	require.NoError(t, err)
	var config struct {
		Gates []struct {
			AppliesTo []string `json:"appliesTo"`
		} `json:"gates"`
	}
	require.NoError(t, json.Unmarshal(configBytes, &config))

	canonical := components.CatalogKindDirectories(root)
	expected := map[string]struct{}{}
	for _, gate := range config.Gates {
		for _, kind := range gate.AppliesTo {
			directory := "components"
			switch strings.TrimSpace(kind) {
			case "foundation":
				directory = "foundations"
			case "primitive":
				directory = "primitives"
			case "runtime-hook":
				directory = "hooks"
			case "runtime-service":
				directory = "services"
			}
			expected[directory] = struct{}{}
		}
	}
	got := map[string]struct{}{}
	for _, directory := range canonical {
		got[directory] = struct{}{}
	}
	require.Equal(t, expected, got, "indexer roots must derive from catalog/config.json")

	var descriptor struct {
		Applicability struct {
			Any []struct {
				PathGlob string `json:"pathGlob"`
			} `json:"any"`
		} `json:"applicability"`
	}
	descriptorBytes, err := os.ReadFile(filepath.Join(root, ".vrooli", "test-genie.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(descriptorBytes, &descriptor))
	applicable := map[string]struct{}{}
	for _, item := range descriptor.Applicability.Any {
		parts := strings.Split(item.PathGlob, "/")
		if len(parts) == 4 && parts[0] == "library" && parts[2] == "*" && parts[3] == "component.json" {
			applicable[parts[1]] = struct{}{}
		}
	}
	require.Equal(t, got, applicable, "test-genie applicability roots must match canonical catalog roots")
}

func TestIndexer_RunRejectsNestedCompanionFixture(t *testing.T) {
	fs := fstest.MapFS{
		"components/FocusTrap/component.json": {Data: []byte(`{"libraryId":"react-component-library:FocusTrap","displayName":"Focus Trap","slot":"ui-pattern","entry":"FocusTrap.tsx","latest":"1.0.0","deprecatedVersions":[]}`)},
		"components/FocusTrap/versions/1.0.0/FocusTrap.tsx": {Data: []byte(`/**
 * @libraryId react-component-library:FocusTrap
 * @version 1.0.0
 */
export const FocusTrap = () => null;`)},
		// A nested source would be silently omitted by the flat adoption and
		// conformance model. The indexer must reject this calibration fixture.
		"components/FocusTrap/versions/1.0.0/hooks/useFocusTrap.ts": {Data: []byte(`export const useFocusTrap = () => undefined;`)},
	}
	res, err := components.NewIndexer(mocks.NewFakeRepository(), ".", fs).Run(context.Background())
	require.NoError(t, err)
	require.Zero(t, res.Indexed)
	require.Len(t, res.Errors, 1)
	var invalid components.ErrInvalidHeader
	require.ErrorAs(t, res.Errors[0], &invalid)
	require.Equal(t, "version", invalid.Field)
	require.Contains(t, invalid.Reason, "subdirectories are not supported")
}

func TestIndexer_RunReportsInvalidSlot(t *testing.T) {
	fs := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(`{"libraryId":"react-component-library:Button","displayName":"Button","description":"","slot":"marketing-hero","tags":[],"latest":"1.0.0","deprecatedVersions":[]}`)},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
	}
	repo := mocks.NewFakeRepository()
	idx := components.NewIndexer(repo, ".", fs)

	res, err := idx.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, res.Indexed)
	require.Len(t, res.Errors, 1)
	var hdr components.ErrInvalidHeader
	require.True(t, errors.As(res.Errors[0], &hdr), "got %T", res.Errors[0])
	require.Equal(t, "slot", hdr.Field)
	require.Contains(t, hdr.Reason, "marketing-hero")
}

func TestIndexer_RunFindsHeaderStatusDisagreementWithoutRejecting(t *testing.T) {
	tsx := `/**
 * @libraryId react-component-library:Button
 * @version 1.0.0
 * @status deprecated
 */
export const Button = () => null;
`
	fs := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(tsx)},
	}
	repo := mocks.NewFakeRepository()
	idx := components.NewIndexer(repo, ".", fs)

	res, err := idx.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Indexed)
	require.Empty(t, res.Errors)
	require.Len(t, res.Findings, 1)
	require.Equal(t, components.IndexFindingHeaderDisagreement, res.Findings[0].Kind)
	require.Equal(t, "status", res.Findings[0].Field)
	require.Equal(t, "released", res.Findings[0].Expected)
	require.Equal(t, "deprecated", res.Findings[0].Actual)
}

func TestIndexer_RunFindsHeaderVersionDisagreementWithoutRejecting(t *testing.T) {
	tsx := `/**
 * @libraryId react-component-library:Button
 * @version 9.9.9
 */
export const Button = () => null;
`
	fs := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(tsx)},
	}
	repo := mocks.NewFakeRepository()
	idx := components.NewIndexer(repo, ".", fs)

	res, err := idx.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Indexed)
	require.Empty(t, res.Errors)
	require.Len(t, res.Findings, 1)
	require.Equal(t, components.IndexFindingHeaderDisagreement, res.Findings[0].Kind)
	require.Equal(t, "version", res.Findings[0].Field)
	require.Equal(t, "1.0.0", res.Findings[0].Expected)
	require.Equal(t, "9.9.9", res.Findings[0].Actual)

	got, err := repo.GetByLibraryID(context.Background(), "react-component-library:Button")
	require.NoError(t, err)
	require.Equal(t, "1.0.0", got.Version)
	require.NotContains(t, got.Headers, "version")
}

func TestIndexer_RunFindsHeaderDepsDisagreementWithoutRejecting(t *testing.T) {
	tsx := `/**
 * @libraryId react-component-library:Button
 * @version 1.0.0
 * @deps react
 */
export const Button = () => null;
`
	fs := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(tsx)},
	}
	repo := mocks.NewFakeRepository()
	idx := components.NewIndexer(repo, ".", fs)

	res, err := idx.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Indexed)
	require.Empty(t, res.Errors)
	require.Len(t, res.Findings, 1)
	require.Equal(t, components.IndexFindingHeaderDisagreement, res.Findings[0].Kind)
	require.Equal(t, "deps", res.Findings[0].Field)
	require.Equal(t, "JSON object or array", res.Findings[0].Expected)
	require.Equal(t, "react", res.Findings[0].Actual)

	got, err := repo.GetByLibraryID(context.Background(), "react-component-library:Button")
	require.NoError(t, err)
	require.NotContains(t, got.Headers, "deps")
}

func TestIndexer_RunFlagsUnknownDesignStyleWithoutRejecting(t *testing.T) {
	fs := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifestWithStyles("react-component-library:Button", "Button", `[]`, `[{"styleId":"missing-style","affinity":"native"}]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
	}
	repo := mocks.NewFakeRepository()
	idx := components.NewIndexer(repo, ".", fs)

	res, err := idx.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Indexed)
	require.Empty(t, res.Errors)
	require.Len(t, res.Findings, 1)
	require.Equal(t, components.IndexFindingStaleDesignStyle, res.Findings[0].Kind)
	require.Equal(t, "designStyles", res.Findings[0].Field)
	require.Equal(t, "missing-style", res.Findings[0].Actual)
	require.Contains(t, res.Findings[0].Detail, "missing-style")
}

// TestIndexer_RunFlagsMissingDesignAffinityWithoutRejecting is the calibration
// for the promote-time affinity conformance gate: a released component that
// declares no design-style affinities must surface a soft finding (never a hard
// error) so the catalog gap is visible on every reindex.
func TestIndexer_RunFlagsMissingDesignAffinityWithoutRejecting(t *testing.T) {
	fs := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifestWithStyles("react-component-library:Button", "Button", `["form"]`, `[]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
	}
	repo := mocks.NewFakeRepository()
	idx := components.NewIndexer(repo, ".", fs)

	res, err := idx.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Indexed)
	require.Empty(t, res.Errors)
	require.Len(t, res.Findings, 1)
	require.Equal(t, components.IndexFindingMissingDesignAffinity, res.Findings[0].Kind)
	require.Equal(t, "designStyles", res.Findings[0].Field)
	require.Equal(t, "none", res.Findings[0].Actual)
	require.Contains(t, res.Findings[0].Detail, "react-component-library:Button")

	// A declared affinity clears the finding — the gate is not a false positive.
	fs["components/Button/component.json"] = &fstest.MapFile{Data: []byte(manifestWithStyles("react-component-library:Button", "Button", `["form"]`, `[{"styleId":"vrooli-default","affinity":"native"}]`))}
	res, err = components.NewIndexer(mocks.NewFakeRepository(), ".", fs).Run(context.Background())
	require.NoError(t, err)
	require.Empty(t, res.Findings)
}

func TestIndexer_RunReportsInvalidDesignAffinity(t *testing.T) {
	fs := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifestWithStyles("react-component-library:Button", "Button", `[]`, `[{"styleId":"vrooli-default","affinity":"bespoke"}]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
	}
	repo := mocks.NewFakeRepository()
	idx := components.NewIndexer(repo, ".", fs)

	res, err := idx.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, res.Indexed)
	require.Len(t, res.Errors, 1)
	var hdr components.ErrInvalidHeader
	require.True(t, errors.As(res.Errors[0], &hdr), "got %T", res.Errors[0])
	require.Equal(t, "designStyles", hdr.Field)
	require.Contains(t, hdr.Reason, "invalid affinity")
	require.Contains(t, hdr.Reason, "bespoke")
}

func TestIndexer_RunProjectsValidatedStoryContract(t *testing.T) {
	fs := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
		"components/Button/versions/1.0.0/story.json": {Data: []byte(`{
  "schemaVersion": 5,
  "kind":"component",
  "args":{"fields":[{"path":"tone","kind":"enum","options":["primary","secondary"],"default":"primary"}]},
  "environment":{"fixtures":[]},
  "stories":[{"id":"primary","name":"Primary","args":{"tone":"primary"},"expect":[{"kind":"text","value":"Save"}]}]
}`)},
	}
	repo := mocks.NewFakeRepository()
	res, err := components.NewIndexer(repo, ".", fs).Run(context.Background())
	require.NoError(t, err)
	require.Empty(t, res.Findings)
	component, err := repo.GetByLibraryID(context.Background(), "react-component-library:Button")
	require.NoError(t, err)
	stories, err := repo.ListStories(context.Background(), components.StoryQuery{ComponentID: component.ID, Version: "1.0.0"})
	require.NoError(t, err)
	require.Len(t, stories, 1)
	require.Equal(t, components.StoryKindComponent, stories[0].Kind)
	require.JSONEq(t, `{"fields":[{"path":"tone","kind":"enum","options":["primary","secondary"],"default":"primary"}]}`, stories[0].ArgsJSON)
}

func TestIndexer_RunValidatesStoryHarnessArtifacts(t *testing.T) {
	const story = `{
  "schemaVersion": 5,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id": "interactive", "name": "Interactive", "composition": {"specimen": {"module": "./story.tsx", "export": "StatefulHarness"}}, "args": {}}]
}`

	tests := []struct {
		name       string
		storyTSX   string
		finding    components.IndexFindingKind
		wantSource bool
	}{
		{name: "missing artifact", finding: components.IndexFindingStoryHarnessMissing},
		{name: "missing export", storyTSX: `export const AnotherHarness = () => null;`, finding: components.IndexFindingStoryHarnessExport},
		{name: "matching export is projected into read-only files", storyTSX: `export const StatefulHarness = () => null;`, wantSource: true},
		{name: "re-exported harness is accepted", storyTSX: `export { StatefulHarness } from "./shared-stories";`, wantSource: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
				"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
				"components/Button/versions/1.0.0/story.json": {Data: []byte(story)},
			}
			if tt.storyTSX != "" {
				fsys["components/Button/versions/1.0.0/story.tsx"] = &fstest.MapFile{Data: []byte(tt.storyTSX)}
			}
			repo := mocks.NewFakeRepository()
			res, err := components.NewIndexer(repo, ".", fsys).Run(context.Background())
			require.NoError(t, err)
			if tt.finding != "" {
				require.Len(t, res.Findings, 1)
				require.Equal(t, tt.finding, res.Findings[0].Kind)
				return
			}
			require.Empty(t, res.Findings)
			component, err := repo.GetByLibraryID(context.Background(), "react-component-library:Button")
			require.NoError(t, err)
			version, err := repo.GetVersion(context.Background(), component.ID, "1.0.0")
			require.NoError(t, err)
			require.Len(t, version.Files, 3)
			require.Equal(t, "Button.tsx", version.Files[0].Path)
			require.Equal(t, "story.json", version.Files[1].Path)
			require.Equal(t, "story.tsx", version.Files[2].Path)
		})
	}
}

func TestIndexer_RunFindsOrphanStoryHarnessArtifact(t *testing.T) {
	fsys := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
		"components/Button/versions/1.0.0/story.json": {Data: []byte(`{"schemaVersion": 5,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"primary","name":"Primary","args":{}}]}`)},
		"components/Button/versions/1.0.0/story.tsx":  {Data: []byte(`export const UnusedHarness = () => null;`)},
	}
	res, err := components.NewIndexer(mocks.NewFakeRepository(), ".", fsys).Run(context.Background())
	require.NoError(t, err)
	require.Len(t, res.Findings, 1)
	require.Equal(t, components.IndexFindingStoryHarnessOrphan, res.Findings[0].Kind)
}

func TestIndexer_RunReportsMalformedHeaderErrors(t *testing.T) {
	fs := fstest.MapFS{
		"components/Broken/component.json": {Data: []byte(`{"displayName":"Broken","latest":"1.0.0"}`)},
	}
	repo := mocks.NewFakeRepository()
	idx := components.NewIndexer(repo, ".", fs)

	res, err := idx.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, res.Indexed)
	require.Len(t, res.Errors, 1)
	var hdr components.ErrInvalidHeader
	require.True(t, errors.As(res.Errors[0], &hdr), "got %T", res.Errors[0])
	require.Equal(t, "libraryId", hdr.Field)
}

func TestIndexer_RunDeletesMissingOnRerun(t *testing.T) {
	first := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
		"components/Card/component.json":              {Data: []byte(manifest("react-component-library:Card", "Card", `[]`))},
		"components/Card/versions/1.0.0/Card.tsx":     {Data: []byte(cardTSX)},
	}
	repo := mocks.NewFakeRepository()

	res, err := components.NewIndexer(repo, ".", first).Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, res.Indexed)

	// Re-run with one file removed.
	second := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
	}
	res, err = components.NewIndexer(repo, ".", second).Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Indexed)
	require.Equal(t, 1, res.Deleted)
	_, err = repo.GetByLibraryID(context.Background(), "react-component-library:Card")
	require.Error(t, err)
}

func TestIndexer_RunEmitsRegistryOrphanFindingAndSweeps(t *testing.T) {
	d, repo, _ := newComponentsRawDB(t)
	// Pre-existing cruft: a version row whose registry parent is gone.
	seedOrphanVersion(t, d, "orphan-1", "cmp-gone", "react-component-library:tab-bar", "0.1.0",
		"components/tab-bar/versions/0.1.0/tab-bar.tsx")

	fs := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `["form"]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
	}
	res, err := components.NewIndexer(repo, ".", fs).Run(context.Background())
	require.NoError(t, err)
	require.Empty(t, res.Errors)
	require.Equal(t, 1, res.Indexed)

	var orphanFindings []components.IndexFinding
	for _, f := range res.Findings {
		if f.Kind == components.IndexFindingRegistryOrphan {
			orphanFindings = append(orphanFindings, f)
		}
	}
	require.Len(t, orphanFindings, 1, "the registry-orphaned version must surface a conformance finding")
	require.Equal(t, "cmp-gone", orphanFindings[0].Actual)
	require.Equal(t, "component_id", orphanFindings[0].Field)

	// The orphan is swept, and the live component indexed this run
	// survives the sweep-before-DeleteMissing ordering.
	require.Zero(t, countRows(t, d,
		`SELECT count(*) FROM component_versions WHERE component_id NOT IN (SELECT id FROM components)`))
	got, err := repo.GetByLibraryID(context.Background(), "react-component-library:Button")
	require.NoError(t, err)
	require.NotEmpty(t, got.ID)

	// A clean re-run emits no orphan finding.
	res2, err := components.NewIndexer(repo, ".", fs).Run(context.Background())
	require.NoError(t, err)
	for _, f := range res2.Findings {
		require.NotEqual(t, components.IndexFindingRegistryOrphan, f.Kind, "steady state must be orphan-free")
	}
}

func TestReindexPreservesEvictedVersionRows(t *testing.T) {
	d, repo, _ := newComponentsRawDB(t)
	ctx := context.Background()
	first := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
		"components/Button/versions/0.9.0/Button.tsx": {Data: []byte(buttonTSX)},
		"components/Button/versions/0.9.0/story.json": {Data: []byte(`{"schemaVersion":5,"kind":"component","title":"Old Button","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"default","name":"Default","args":{}}]}`)},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
		"components/Button/versions/1.0.0/story.json": {Data: []byte(`{"schemaVersion":5,"kind":"component","title":"Current Button","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"default","name":"Default","args":{}}]}`)},
	}
	_, err := components.NewIndexer(repo, ".", first).Run(ctx)
	require.NoError(t, err)
	component, err := repo.GetByLibraryID(ctx, "react-component-library:Button")
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, `UPDATE component_versions SET presence='evicted' WHERE component_id=? AND version='0.9.0'`, component.ID)
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, `DELETE FROM component_stories WHERE component_id=? AND version='0.9.0'`, component.ID)
	require.NoError(t, err)

	second := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(`{"libraryId":"react-component-library:Button","displayName":"Button","slot":"ui-primitive","latest":"1.0.0","deprecatedVersions":[],"evictedVersions":["0.9.0"],"designStyles":[{"styleId":"vrooli-default","affinity":"native"}]}`)},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
	}
	_, err = components.NewIndexer(repo, ".", second).Run(ctx)
	require.NoError(t, err)
	storiesBefore := countRows(t, d, `SELECT COUNT(*) FROM component_stories WHERE component_id=?`, component.ID)
	require.Equal(t, 1, storiesBefore, "materialized story is rebuilt from disk while evicted story remains durable")
	var storyTitle string
	require.NoError(t, d.QueryRowContext(ctx, `SELECT title FROM component_stories WHERE component_id=? AND version='0.9.0'`, component.ID).Scan(&storyTitle))
	require.Equal(t, "Old Button", storyTitle)
	versionsBefore := countRows(t, d, `SELECT COUNT(*) FROM component_versions WHERE component_id=?`, component.ID)
	filesBefore := countRows(t, d, `SELECT COUNT(*) FROM component_version_files WHERE version_id IN (SELECT id FROM component_versions WHERE component_id=?)`, component.ID)
	_, err = components.NewIndexer(repo, ".", second).Run(ctx)
	require.NoError(t, err)
	require.Equal(t, versionsBefore, countRows(t, d, `SELECT COUNT(*) FROM component_versions WHERE component_id=?`, component.ID))
	require.Equal(t, filesBefore, countRows(t, d, `SELECT COUNT(*) FROM component_version_files WHERE version_id IN (SELECT id FROM component_versions WHERE component_id=?)`, component.ID))
	require.Equal(t, storiesBefore, countRows(t, d, `SELECT COUNT(*) FROM component_stories WHERE component_id=?`, component.ID))
	var presence string
	require.NoError(t, d.QueryRowContext(ctx, `SELECT presence FROM component_versions WHERE component_id=? AND version='0.9.0'`, component.ID).Scan(&presence))
	require.Equal(t, "evicted", presence)
}

func TestReindexDeletesMaterializedVersionMissingFromDisk(t *testing.T) {
	d, repo, _ := newComponentsRawDB(t)
	ctx := context.Background()
	first := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
		"components/Button/versions/0.9.0/Button.tsx": {Data: []byte(buttonTSX)},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
	}
	_, err := components.NewIndexer(repo, ".", first).Run(ctx)
	require.NoError(t, err)
	component, err := repo.GetByLibraryID(ctx, "react-component-library:Button")
	require.NoError(t, err)
	second := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
	}
	result, err := components.NewIndexer(repo, ".", second).Run(ctx)
	require.NoError(t, err)
	require.NoError(t, err)
	require.Equal(t, 1, countRows(t, d, `SELECT COUNT(*) FROM component_versions WHERE component_id=?`, component.ID))
	require.Equal(t, 1, result.Indexed)
}

func TestSweepOrphansSkipsEvictedVersions(t *testing.T) {
	d, repo, _ := newComponentsRawDB(t)
	ctx := context.Background()
	component, err := repo.UpsertManifest(ctx, components.IndexManifestInput{
		Manifest: components.ComponentManifest{LibraryID: "react-component-library:Button", Slug: "Button", DisplayName: "Button", LatestVersion: "1.0.0"},
		Versions: []components.ComponentVersion{{Version: "1.0.0", Status: components.VersionStatusReleased, SourcePath: "components/Button/versions/1.0.0/Button.tsx", Content: string(buttonTSX), ContentSHA256: testDigest(buttonTSX)}},
	})
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, `UPDATE component_versions SET presence='evicted' WHERE component_id=?`, component.ID)
	require.NoError(t, err)
	orphans, err := repo.SweepOrphans(ctx)
	require.NoError(t, err)
	require.Empty(t, orphans)
	require.Equal(t, 1, countRows(t, d, `SELECT COUNT(*) FROM component_versions WHERE component_id=?`, component.ID))
}

// manifest builds a catalog-complete fixture (one declared affinity) so it does
// not trip the missing-design-affinity conformance finding. Tests that need an
// affinity-less manifest call manifestWithStyles with an empty designStyles set.
func manifest(libraryID, displayName, tags string) string {
	return manifestWithStyles(libraryID, displayName, tags, `[{"styleId":"vrooli-default","affinity":"native"}]`)
}

func manifestWithStyles(libraryID, displayName, tags, designStyles string) string {
	return `{"libraryId":"` + libraryID + `","displayName":"` + displayName + `","description":"","slot":"ui-primitive","tags":` + tags + `,"designStyles":` + designStyles + `,"latest":"1.0.0","deprecatedVersions":[]}`
}
