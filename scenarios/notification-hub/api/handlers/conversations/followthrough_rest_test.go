package conversations_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// The owner reports what the answer started; the read projection on the ask is
// the only thing the UI needs.
func TestFollowthroughIsReportedByTheOwnerAndProjectedOnTheAsk(t *testing.T) {
	f := newReaderFixture(t)
	askID := f.ask(t, "owner", "ft-1")
	_, err := f.client.Answer(context.Background(), reading("owner", &v1.AnswerRequest{AskId: askID, Answer: "yes"}))
	require.NoError(t, err)

	status, body := postFollowthrough(t, f, "alice", askID, `{"state":"running","owner":"scenario:agent-manager","run_ref":"run-1","bound_seconds":3600,"summary":"Rewriting both floors."}`)
	require.Equal(t, http.StatusOK, status, body)
	require.Contains(t, body, `"state":"running"`)

	read, err := f.client.GetAsk(context.Background(), reading("owner", &v1.GetAskRequest{AskId: askID}))
	require.NoError(t, err)
	require.NotNil(t, read.Msg.GetAsk().GetFollowthrough())
	require.Equal(t, "running", read.Msg.GetAsk().GetFollowthrough().GetState())
	require.Equal(t, int32(3600), read.Msg.GetAsk().GetFollowthrough().GetBoundSeconds())

	// Another owner cannot report on this ask, and the response does not say
	// whether it exists.
	status, _ = postFollowthrough(t, f, "bob", askID, `{"state":"done"}`)
	require.Equal(t, http.StatusNotFound, status)

	status, body = postFollowthrough(t, f, "alice", askID, `{"state":"sideways"}`)
	require.Equal(t, http.StatusBadRequest, status)
	require.Contains(t, body, "queued")
}

func postFollowthrough(t *testing.T, f *readerFixture, owner, askID, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost,
		f.url+"/api/v1/conversations/asks/"+url.PathEscape(askID)+"/followthrough", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if owner != "" {
		req.Header.Set("X-Vrooli-Identity-Subject", owner)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

// The proto projection is the contract the UI reads: a detail carries the
// grade, the report, the suggestions and the follow-through; a list row carries
// the grade alone.
func TestDecisionDetailCarriesReadinessAndRelated(t *testing.T) {
	f := newReaderFixture(t)
	first := f.ask(t, "owner", "proj-1")
	second := f.ask(t, "owner", "proj-2")

	read, err := f.client.GetAsk(context.Background(), reading("owner", &v1.GetAskRequest{AskId: first}))
	require.NoError(t, err)
	ask := read.Msg.GetAsk()
	require.NotEmpty(t, ask.GetQualityGrade())
	require.NotEmpty(t, ask.GetQualityReport())
	require.NotEmpty(t, ask.GetRelated(), "the fixture's two asks share a question")
	require.Equal(t, second, ask.GetRelated()[0].GetAskId())
	require.Equal(t, "lexical", ask.GetRelated()[0].GetBasis())

	page, err := f.client.ListAsks(context.Background(), reading("owner", &v1.ListAsksRequest{OpenOnly: true}))
	require.NoError(t, err)
	require.NotEmpty(t, page.Msg.GetAsks())
	for _, row := range page.Msg.GetAsks() {
		require.NotEmpty(t, row.GetQualityGrade(), "a list row carries the grade")
		require.NotEmpty(t, row.GetQualitySummary(), "a list row carries one plain line")
		require.Empty(t, row.GetQualityReport(), "a list row never carries the whole report")
	}
}
