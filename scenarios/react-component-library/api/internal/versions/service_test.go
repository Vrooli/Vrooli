package versions_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/versions"
	"react-component-library/internal/versions/mocks"
)

type stubAdoptions struct {
	content map[string]string
	err     error
}

func (s *stubAdoptions) ResolveAdoption(_ context.Context, id string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.content[id], nil
}

func TestService_Record_NoOpOnIdenticalContent(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	svc := versions.NewService(repo, nil)

	content := "// @version 1.0.0\nexport default function X(){}"
	recorded, _, err := svc.Record(ctx, versions.RecordInput{ComponentID: "cmp-1", Content: content})
	require.NoError(t, err)
	require.True(t, recorded)

	recorded2, _, err := svc.Record(ctx, versions.RecordInput{ComponentID: "cmp-1", Content: content})
	require.NoError(t, err)
	require.False(t, recorded2, "identical re-save must not insert a new version")

	rows, err := svc.List(ctx, versions.ListQuery{ComponentID: "cmp-1"})
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

func TestService_Record_NewRowOnContentChange(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	svc := versions.NewService(repo, nil)

	_, _, err := svc.Record(ctx, versions.RecordInput{
		ComponentID: "cmp-1", Content: "// @version 1.0.0\nbody",
	})
	require.NoError(t, err)

	recorded, _, err := svc.Record(ctx, versions.RecordInput{
		ComponentID: "cmp-1", Content: "// @version 1.0.0\nbody changed",
	})
	require.NoError(t, err)
	require.True(t, recorded, "content change must insert a new version")
}

func TestService_Record_NewRowOnVersionBump(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	svc := versions.NewService(repo, nil)

	const body = "exported"
	_, _, err := svc.Record(ctx, versions.RecordInput{ComponentID: "cmp-1", Content: "// @version 1.0.0\n" + body})
	require.NoError(t, err)
	recorded, _, err := svc.Record(ctx, versions.RecordInput{ComponentID: "cmp-1", Content: "// @version 1.0.1\n" + body})
	require.NoError(t, err)
	require.True(t, recorded, "@version bump must insert a new row even when sha is identical*")
	// *sha will differ here too because the header line text changed,
	// which is exactly the contract — either axis triggers a record.
}

func TestService_Record_RejectsMissingComponent(t *testing.T) {
	svc := versions.NewService(mocks.NewFakeRepository(), nil)
	_, _, err := svc.Record(context.Background(), versions.RecordInput{Content: "x"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "component_id")
}

func TestService_Record_RejectsBlankComponentAndContent(t *testing.T) {
	svc := versions.NewService(mocks.NewFakeRepository(), nil)
	ctx := context.Background()

	_, _, err := svc.Record(ctx, versions.RecordInput{ComponentID: " \t", Content: "body"})
	var invalid versions.ErrInvalidVersion
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "component_id", invalid.Field)

	_, _, err = svc.Record(ctx, versions.RecordInput{ComponentID: "cmp-1"})
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "content", invalid.Field)
}

func TestService_RecordDefaultsChangelogAndPropagatesRepositoryErrors(t *testing.T) {
	ctx := context.Background()
	repo := &recordingRepository{latestErr: errors.New("read failed")}
	svc := versions.NewService(repo, nil)

	_, _, err := svc.Record(ctx, versions.RecordInput{ComponentID: "cmp-1", Content: "body"})
	require.EqualError(t, err, "read failed")
	repo.latestErr = nil
	repo.insertErr = errors.New("write failed")

	_, _, err = svc.Record(ctx, versions.RecordInput{ComponentID: "cmp-1", Content: "body", ChangelogMD: "  "})
	require.EqualError(t, err, "write failed")
	require.Equal(t, "auto-recorded on save", repo.inserted.ChangelogMD)
}

func TestService_GetAndListForwardQueries(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	repo.Seed(versions.Version{ComponentID: "cmp-1", Version: "1.0.0", Content: "body"})
	svc := versions.NewService(repo, nil)

	got, err := svc.Get(ctx, "cmp-1", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, "body", got.Content)

	rows, err := svc.List(ctx, versions.ListQuery{ComponentID: "cmp-1", Limit: -1})
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

func TestService_DiffValidationAndResolveFailures(t *testing.T) {
	ctx := context.Background()
	svc := versions.NewService(mocks.NewFakeRepository(), nil)
	_, err := svc.Diff(ctx, versions.DiffInput{From: "v1", To: "v2"})
	var invalid versions.ErrInvalidVersion
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "component_id", invalid.Field)

	_, err = svc.Diff(ctx, versions.DiffInput{ComponentID: "cmp-1", From: "v1"})
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "from/to", invalid.Field)

	_, err = svc.Diff(ctx, versions.DiffInput{ComponentID: "cmp-1", From: "adoption:x", To: "v2"})
	require.ErrorContains(t, err, "no AdoptionResolver wired")

	resolver := &stubAdoptions{err: errors.New("adoption unavailable")}
	svc = versions.NewService(mocks.NewFakeRepository(), resolver)
	_, err = svc.Diff(ctx, versions.DiffInput{ComponentID: "cmp-1", From: "adoption:x", To: "v2"})
	require.EqualError(t, err, "adoption unavailable")
}

type recordingRepository struct {
	latestErr error
	insertErr error
	inserted  versions.Version
}

func (r *recordingRepository) Latest(context.Context, string) (versions.Version, error) {
	return versions.Version{}, r.latestErr
}

func (r *recordingRepository) Insert(_ context.Context, v versions.Version) (versions.Version, error) {
	r.inserted = v
	return versions.Version{}, r.insertErr
}

func (*recordingRepository) List(context.Context, versions.ListQuery) ([]versions.Version, error) {
	return nil, nil
}

func (*recordingRepository) Get(context.Context, string, string) (versions.Version, error) {
	return versions.Version{}, nil
}

func TestService_Diff_VersionToVersion(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	repo.Seed(versions.Version{
		ComponentID: "cmp-1", Version: "v1", Content: "alpha\nbeta\ngamma",
		ContentSHA256: "h1",
	})
	repo.Seed(versions.Version{
		ComponentID: "cmp-1", Version: "v2", Content: "alpha\nBETA\ngamma\ndelta",
		ContentSHA256: "h2",
	})
	svc := versions.NewService(repo, nil)

	got, err := svc.Diff(ctx, versions.DiffInput{ComponentID: "cmp-1", From: "v1", To: "v2"})
	require.NoError(t, err)
	// Expected aligned shape (left | right):
	//   alpha  | alpha
	//   beta   | (empty)        ← remove
	//   (empty)| BETA           ← add
	//   gamma  | gamma
	//   (empty)| delta          ← add
	require.Equal(t, 2, got.Additions)
	require.Equal(t, 1, got.Removals)
	require.Equal(t, "v1", got.FromLabel)
	require.Equal(t, "v2", got.ToLabel)
}

func TestService_Diff_RejectsMissingSides(t *testing.T) {
	svc := versions.NewService(mocks.NewFakeRepository(), nil)
	_, err := svc.Diff(context.Background(), versions.DiffInput{ComponentID: "cmp-1", From: "", To: "v2"})
	require.Error(t, err)
}

func TestService_Diff_AdoptionSide(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	repo.Seed(versions.Version{
		ComponentID: "cmp-1", Version: "v1", Content: "library line\nshared",
		ContentSHA256: "h",
	})
	resolver := &stubAdoptions{content: map[string]string{
		"adp-1": "adopted line\nshared",
	}}
	svc := versions.NewService(repo, resolver)
	got, err := svc.Diff(ctx, versions.DiffInput{
		ComponentID: "cmp-1", From: "v1", To: "adoption:adp-1",
	})
	require.NoError(t, err)
	require.Equal(t, 1, got.Additions, "adopted line should count as an addition")
	require.Equal(t, 1, got.Removals, "library line should count as a removal")
	require.Equal(t, "adoption:adp-1", got.ToLabel)
}

func TestParseVersionHeader(t *testing.T) {
	cases := map[string]string{
		"// @version 1.2.3":           "1.2.3",
		" *  @version  2.0.0-alpha":   "2.0.0-alpha",
		"no header here":              "",
		"prose @version 9.9.9 inline": "9.9.9", // permissive: matches anywhere on a line
		"/**\n * @version 0.1.0\n */": "0.1.0",
	}
	for input, want := range cases {
		got := versions.ParseVersionHeader(strings.ReplaceAll(input, "\n", "\n"))
		require.Equal(t, want, got, "input=%q", input)
	}
}
