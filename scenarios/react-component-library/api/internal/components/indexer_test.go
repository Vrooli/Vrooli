package components_test

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
	"react-component-library/internal/components/mocks"
)

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

func TestIndexer_RunIndexesVersionExamples(t *testing.T) {
	fs := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
		"components/Button/versions/1.0.0/examples.json": {Data: []byte(`{
			"examples": [
				{
					"name": "primary",
					"displayName": "Primary",
					"props": {"children":{"$text":"Save changes"}},
					"expect": [{"kind":"text","value":"Save changes"}]
				}
			]
		}`)},
	}
	repo := mocks.NewFakeRepository()
	idx := components.NewIndexer(repo, ".", fs)

	res, err := idx.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Indexed)
	require.Empty(t, res.Errors)
	require.Empty(t, res.Findings)

	got, err := repo.GetByLibraryID(context.Background(), "react-component-library:Button")
	require.NoError(t, err)
	examples, err := repo.ListExamples(context.Background(), components.ExampleQuery{ComponentID: got.ID, Version: "1.0.0"})
	require.NoError(t, err)
	require.Len(t, examples, 1)
	require.Equal(t, "primary", examples[0].Name)
	require.JSONEq(t, `{"children":{"$text":"Save changes"}}`, examples[0].PropsJSON)
	require.JSONEq(t, `[{"kind":"text","value":"Save changes"}]`, examples[0].ExpectJSON)
}

func TestIndexer_RunReportsInvalidExamplesWithoutRejectingComponent(t *testing.T) {
	fs := fstest.MapFS{
		"components/Button/component.json":            {Data: []byte(manifest("react-component-library:Button", "Button", `[]`))},
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte(buttonTSX)},
		"components/Button/versions/1.0.0/examples.json": {Data: []byte(`{
			"examples": [
				{"name": "bad", "props": []}
			]
		}`)},
	}
	repo := mocks.NewFakeRepository()
	idx := components.NewIndexer(repo, ".", fs)

	res, err := idx.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Indexed)
	require.Empty(t, res.Errors)
	require.Len(t, res.Findings, 1)
	require.Equal(t, components.IndexFindingInvalidExample, res.Findings[0].Kind)
	require.Equal(t, "examples[0].props", res.Findings[0].Field)
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

func manifest(libraryID, displayName, tags string) string {
	return manifestWithStyles(libraryID, displayName, tags, `[]`)
}

func manifestWithStyles(libraryID, displayName, tags, designStyles string) string {
	return `{"libraryId":"` + libraryID + `","displayName":"` + displayName + `","description":"","slot":"ui-primitive","tags":` + tags + `,"designStyles":` + designStyles + `,"latest":"1.0.0","deprecatedVersions":[]}`
}
