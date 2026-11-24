package versions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
	internalversions "react-component-library/internal/versions"
)

func TestVersionToProtoCopiesMetadataAndOptionalReleaseTime(t *testing.T) {
	recorded := time.Date(2025, 3, 4, 5, 6, 7, 8, time.FixedZone("offset", 3600))
	released := recorded.Add(time.Hour)
	required := []string{"token"}
	patterns := []string{"^pattern$"}
	input := internalversions.Version{
		ID: "id", ComponentID: "component", LibraryID: "library", Version: "1.2.3",
		Status: "released", SourcePath: "src/Button.tsx", ContentSHA256: "sha", ChangelogMD: "change",
		RecordedAt: recorded, CreatedAt: recorded, ReleasedAt: released,
		RequiredTokens: required, RequiredTokenPatterns: patterns, Presence: "evicted",
	}
	got := versionToProto(input, false)
	require.Equal(t, []string{"id", "component", "library", "1.2.3", "released", "src/Button.tsx", "sha", "change", "evicted"}, []string{
		got.Id, got.ComponentId, got.LibraryId, got.Version, got.Status, got.SourcePath, got.ContentSha256, got.ChangelogMd, got.Presence,
	})
	require.Equal(t, []string{"token"}, got.RequiredTokens)
	require.Equal(t, []string{"^pattern$"}, got.RequiredTokenPatterns)
	require.Equal(t, recorded.UTC(), got.CreatedAt.AsTime())
	require.Equal(t, released.UTC(), got.ReleasedAt.AsTime())
	required[0] = "mutated"
	patterns[0] = "mutated"
	require.Equal(t, []string{"token"}, got.RequiredTokens, "proto slices must not alias repository values")
	require.Equal(t, []string{"^pattern$"}, got.RequiredTokenPatterns)

	withoutRelease := versionToProto(internalversions.Version{}, false)
	require.Nil(t, withoutRelease.ReleasedAt, "unreleased versions must preserve absent timestamp")
}

func TestDiffToProtoMapsCellsAndOperationValues(t *testing.T) {
	got := diffToProto(internalversions.DiffResult{
		Rows: []internalversions.DiffRow{
			{Left: internalversions.DiffCell{LineNumber: 4, Text: "same", Op: internalversions.DiffOpEqual}, Right: internalversions.DiffCell{LineNumber: 4, Text: "same", Op: internalversions.DiffOpEqual}},
			{Left: internalversions.DiffCell{LineNumber: 5, Text: "old", Op: internalversions.DiffOpRemove}, Right: internalversions.DiffCell{Op: internalversions.DiffOpEmpty}},
			{Left: internalversions.DiffCell{Op: internalversions.DiffOpEmpty}, Right: internalversions.DiffCell{LineNumber: 6, Text: "new", Op: internalversions.DiffOpAdd}},
			{Left: internalversions.DiffCell{Op: "unknown"}, Right: internalversions.DiffCell{}},
		}, Additions: 1, Removals: 1, FromLabel: "1.0.0", ToLabel: "1.0.1",
	})
	require.Equal(t, int32(1), got.Additions)
	require.Equal(t, int32(1), got.Removals)
	require.Equal(t, "1.0.0", got.FromLabel)
	require.Equal(t, "1.0.1", got.ToLabel)
	require.Len(t, got.Rows, 4)
	require.Equal(t, versionsv1.DiffOp_DIFF_OP_EQUAL, got.Rows[0].Left.Op)
	require.Equal(t, "same", got.Rows[0].Right.Text)
	require.Equal(t, int32(5), got.Rows[1].Left.LineNumber)
	require.Equal(t, versionsv1.DiffOp_DIFF_OP_REMOVE, got.Rows[1].Left.Op)
	require.Equal(t, versionsv1.DiffOp_DIFF_OP_EMPTY, got.Rows[1].Right.Op)
	require.Equal(t, versionsv1.DiffOp_DIFF_OP_EMPTY, got.Rows[2].Left.Op)
	require.Equal(t, versionsv1.DiffOp_DIFF_OP_ADD, got.Rows[2].Right.Op)
	require.Equal(t, versionsv1.DiffOp_DIFF_OP_UNSPECIFIED, got.Rows[3].Left.Op)
}
