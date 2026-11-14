package jobs

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/apihttptest"
	_ "modernc.org/sqlite"
)

type flakyProvider struct {
	name string
	err  error
}

func (p flakyProvider) Name() string { return p.name }
func (p flakyProvider) Search(context.Context, string) ([]Opening, error) {
	if p.err != nil {
		return nil, p.err
	}
	return []Opening{{Company: "Example", Title: "Engineer"}}, nil
}

func newService(t *testing.T) *Service {
	t.Helper()
	db, e := sql.Open("sqlite", "file:jobs-test?mode=memory&cache=shared")
	require.NoError(t, e)
	t.Cleanup(func() { db.Close() })
	_, e = db.Exec(schemaSQL)
	require.NoError(t, e)
	return New(db, nil)
}

func TestPipelinePersistsRecordsAndFactLinks(t *testing.T) {
	ctx := context.Background()
	s := newService(t)
	o, e := s.CreateOpening(ctx, Opening{Company: "Acme", Title: "Writer"})
	require.NoError(t, e)
	require.NotEmpty(t, o.ID)
	require.Equal(t, "saved", o.Status)
	f, e := s.CreateFact(ctx, Fact{Label: "Experience", Content: "Led a team", Tags: []string{"leadership"}})
	require.NoError(t, e)
	r, e := s.CreateResume(ctx, Resume{OpeningID: o.ID, Title: "Targeted resume", Content: "Led 12 engineers and increased revenue by 50%.", FactIDs: []string{f.ID}, ReviewFindings: []string{"client verdict must be overwritten"}})
	require.NoError(t, e)
	require.NotEmpty(t, r.ID)
	l, e := s.CreateLetter(ctx, CoverLetter{OpeningID: o.ID, Title: "Letter", Content: "Letter text", FactIDs: []string{f.ID}})
	require.NoError(t, e)
	require.NotEmpty(t, l.ID)
	gotR, e := s.Resumes(ctx)
	require.NoError(t, e)
	require.Equal(t, []string{f.ID}, gotR[0].FactIDs)
	reviewed, e := s.ReviewResume(ctx, r.ID)
	require.NoError(t, e)
	require.NotEqual(t, []string{"client verdict must be overwritten"}, reviewed.ReviewFindings)
	require.Contains(t, strings.Join(reviewed.ReviewFindings, " "), "Unsupported claim")
	gotL, e := s.Letters(ctx)
	require.NoError(t, e)
	require.Equal(t, o.ID, gotL[0].OpeningID)
	updated, e := s.UpdateOpening(ctx, o.ID, map[string]any{"status": "applied", "category": "writing"})
	require.NoError(t, e)
	require.Equal(t, "applied", updated.Status)
}

func TestDiscoveryDegradesPerProviderAndUsesLinkOut(t *testing.T) {
	s := newService(t)
	s.providers = []DiscoveryProvider{flakyProvider{name: "good"}, flakyProvider{name: "down", err: errors.New("offline")}}
	out, e := s.Discover(context.Background(), "product writer")
	require.NoError(t, e)
	require.True(t, out.Partial)
	require.GreaterOrEqual(t, len(out.Results), 5)
	require.Equal(t, "link_out", out.Results[0].Status)
	require.Equal(t, "ok", out.Results[3].Status)
	require.Equal(t, "unavailable", out.Results[4].Status)
	require.Contains(t, out.Results[0].URL, "product+writer")
}

func TestCreateRejectsMissingRequiredFields(t *testing.T) {
	_, e := newService(t).CreateFact(context.Background(), Fact{Content: "x"})
	require.ErrorIs(t, e, ErrInvalid)
}

func TestRemoteOKSearchMapsPublicListingsAndPassesQueryTags(t *testing.T) {
	fake := &apihttptest.FakeDoer{}
	fake.AddResponse(200, []byte(`[
		{"id":"","company":"","position":""},
		{"id":"job-123","company":"Acme","position":"Product Engineer","location":"Worldwide","url":"https://remoteok.com/remote-jobs/job-123","description":"Build product experiences"}
	]`))
	provider := NewRemoteOKProvider(fake)
	openings, err := provider.Search(context.Background(), "Product Engineer remote")
	require.NoError(t, err)
	require.Equal(t, "Remote OK", provider.Name())
	require.Len(t, openings, 1)
	require.Equal(t, "Acme", openings[0].Company)
	require.Equal(t, "Product Engineer", openings[0].Title)
	require.Equal(t, "https://remoteok.com/remote-jobs/job-123", openings[0].URL)
	require.Equal(t, "Remote OK", openings[0].Source)
	require.Equal(t, "found", openings[0].Status)
	require.Len(t, fake.Requests, 1)
	require.Equal(t, "product,engineer,remote", fake.Requests[0].URL.Query().Get("tags"))
	require.Equal(t, "application/json", fake.Requests[0].Header.Get("Accept"))
}

func TestRemoteOKSearchReportsProviderErrors(t *testing.T) {
	fake := &apihttptest.FakeDoer{}
	fake.AddResponse(503, []byte("unavailable"))
	_, err := NewRemoteOKProvider(fake).Search(context.Background(), "developer")
	require.ErrorContains(t, err, "HTTP 503")
}

func TestSearchTagsBoundsAndDeduplicatesTags(t *testing.T) {
	require.Equal(t, "dev,go,remote,design,product,manager,engineer,operations", searchTags("dev dev go remote design product manager engineer operations"))
	require.Empty(t, searchTags("a & b"))
}
