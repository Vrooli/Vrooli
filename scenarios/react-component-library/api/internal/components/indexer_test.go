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
 * @displayName Button
 * @description Primary call-to-action button.
 * @version     1.0.0
 * @tags        ["form", "interactive"]
 * @warning     DO NOT REMOVE THIS HEADER
 */
import React from 'react';
export const Button = () => <button>click</button>;
`

const cardTSX = `/**
 * @libraryId react-component-library:Card
 * @displayName Card
 * @tags layout, container
 */
export const Card = () => null;
`

const noHeaderTSX = `import React from 'react';
export const X = () => null;
`

const malformedTSX = `/**
 * @libraryId
 * @displayName Missing
 */
export const X = () => null;
`

func TestIndexer_RunWalksAndUpserts(t *testing.T) {
	fs := fstest.MapFS{
		"src/Button.tsx":     {Data: []byte(buttonTSX)},
		"src/sub/Card.tsx":   {Data: []byte(cardTSX)},
		"src/Untagged.tsx":   {Data: []byte(noHeaderTSX)},
		"src/Button.test.ts": {Data: []byte("// not tsx")},
		"README.md":          {Data: []byte("# nope")},
	}
	repo := mocks.NewFakeRepository()
	idx := components.NewIndexer(repo, ".", fs)

	res, err := idx.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 3, res.Scanned)
	require.Equal(t, 2, res.Indexed)
	require.Equal(t, 1, res.Skipped)
	require.Empty(t, res.Errors)

	got, err := repo.GetByLibraryID(context.Background(), "react-component-library:Button")
	require.NoError(t, err)
	require.Equal(t, "Button", got.DisplayName)
	require.Equal(t, "1.0.0", got.Version)
	require.Equal(t, []string{"form", "interactive"}, got.Tags)
	require.Equal(t, "src/Button.tsx", got.SourcePath)

	got2, err := repo.GetByLibraryID(context.Background(), "react-component-library:Card")
	require.NoError(t, err)
	require.Equal(t, []string{"layout", "container"}, got2.Tags)
}

func TestIndexer_RunReportsMalformedHeaderErrors(t *testing.T) {
	fs := fstest.MapFS{
		"src/Broken.tsx": {Data: []byte(malformedTSX)},
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
		"a.tsx": {Data: []byte(buttonTSX)},
		"b.tsx": {Data: []byte(cardTSX)},
	}
	repo := mocks.NewFakeRepository()

	res, err := components.NewIndexer(repo, ".", first).Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, res.Indexed)

	// Re-run with one file removed.
	second := fstest.MapFS{
		"a.tsx": {Data: []byte(buttonTSX)},
	}
	res, err = components.NewIndexer(repo, ".", second).Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, res.Indexed)
	require.Equal(t, 1, res.Deleted)
	_, err = repo.GetByLibraryID(context.Background(), "react-component-library:Card")
	require.Error(t, err)
}
