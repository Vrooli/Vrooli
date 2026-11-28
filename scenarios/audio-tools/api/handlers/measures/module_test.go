package measures

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	measuretypes "audio-tools/internal/measures"
	"audio-tools/internal/testutil/mocks"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
	sharedmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

type countSources struct {
	byok, corpus, speakers, wakewords int64
	experiments, runs                 int64
	windows                           [][2]time.Time
}

type (
	speakerCountSource          struct{ source *countSources }
	wakewordCountSource         struct{ source *countSources }
	duplexAccountingCountSource struct{ source *countSources }
)

func TestModuleMountsWithMissingOptionalCounters(t *testing.T) {
	module := Module(Deps{Logger: &mocks.FakeLogger{}, Clock: scheduletest.New(time.Now().UTC())})
	router := mux.NewRouter()
	module.Mount(router)

	req := httptest.NewRequest(http.MethodPost, "/vrooli.audio_tools.v1.measures.MeasuresService/CountCorpusClips", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
	require.Contains(t, resp.Body.String(), "corpus counter not configured")
}

func (s *countSources) observe(from, to time.Time) {
	s.windows = append(s.windows, [2]time.Time{from, to})
}

func (s *countSources) CountCreatedBetween(_ context.Context, from, to time.Time) (int64, error) {
	s.observe(from, to)
	return s.byok, nil
}

func (s *countSources) CountClipsCreatedInWindow(_ context.Context, from, to time.Time) (int64, error) {
	s.observe(from, to)
	return s.corpus, nil
}

func (s *countSources) CountCreatedInRange(_ context.Context, from, to time.Time) (measuretypes.ExperimentCreatedCounts, error) {
	s.observe(from, to)
	return measuretypes.ExperimentCreatedCounts{Experiments: s.experiments, Runs: s.runs}, nil
}

func (s speakerCountSource) CountCreatedBetween(_ context.Context, from, to time.Time) (int64, error) {
	s.source.observe(from, to)
	return s.source.speakers, nil
}

func (s wakewordCountSource) CountCreatedBetween(_ context.Context, from, to time.Time) (int64, error) {
	s.source.observe(from, to)
	return s.source.wakewords, nil
}

func (s duplexAccountingCountSource) CountCreatedBetween(_ context.Context, from, to time.Time) (int64, error) {
	s.source.observe(from, to)
	return 9, nil
}

func TestModuleServesAggregateCountsThroughConnectAndRegistry(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	clk := scheduletest.New(now)
	sources := &countSources{byok: 3, corpus: 4, runs: 5, experiments: 6, speakers: 7, wakewords: 8}
	d := Deps{BYOK: sources, Corpus: sources, Experiments: sources, Speakers: speakerCountSource{sources}, Wakewords: wakewordCountSource{sources}, DuplexAccounting: duplexAccountingCountSource{sources}, Logger: &mocks.FakeLogger{}, Clock: clk}
	module := Module(d)
	router := mux.NewRouter()
	module.Mount(router)

	body := `{"window":{"token":"TIME_WINDOW_TOKEN_LAST_7D"}}`
	req := httptest.NewRequest(http.MethodPost, "/vrooli.audio_tools.v1.measures.MeasuresService/CountBYOKCredentials", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var connectResult struct {
		Count string `json:"count"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &connectResult))
	require.Equal(t, "3", connectResult.Count)
	require.Equal(t, now.Add(-7*24*time.Hour), sources.windows[0][0])
	require.Equal(t, now, sources.windows[0][1])

	req = httptest.NewRequest(http.MethodPost, "/measures/execute", strings.NewReader(`{"measure":"corpus_clips.corpus-clips-created","params":{"window":"last_30d"}}`))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	var measureResult struct {
		Value      string `json:"value"`
		Provenance struct {
			ExecutedQuery string    `json:"executed_query"`
			ComputedAt    time.Time `json:"computed_at"`
		} `json:"provenance"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &measureResult))
	require.Equal(t, "4", measureResult.Value)
	require.Contains(t, measureResult.Provenance.ExecutedQuery, "[")
	require.NotZero(t, measureResult.Provenance.ComputedAt)
	require.Equal(t, now.Add(-30*24*time.Hour), sources.windows[1][0])
	require.Equal(t, now, sources.windows[1][1])
	require.NotContains(t, resp.Body.String(), "secret")

	req = httptest.NewRequest(http.MethodGet, "/measures/declarations", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	require.Contains(t, resp.Body.String(), "corpus_clips.corpus-clips-created")
	require.NotContains(t, resp.Body.String(), "secret")

	// Preserve the pre-namespace procedure URL while callers roll to the
	// canonical vrooli.audio_tools.v1 service path.
	req = httptest.NewRequest(http.MethodPost, "/audio_tools.v1.measures.MeasuresService/CountDuplexAccountingOperations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &connectResult))
	require.Equal(t, "9", connectResult.Count)
	require.Equal(t, now.Add(-7*24*time.Hour), sources.windows[2][0])
	require.Equal(t, now, sources.windows[2][1])
	require.NotContains(t, resp.Body.String(), "owner")
	require.NotContains(t, resp.Body.String(), "provider")

	req = httptest.NewRequest(http.MethodPost, "/measures/execute", strings.NewReader(`{"measure":"duplex_accounting_operations.duplex-accounting-operations-created","params":{"window":"last_30d"}}`))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &measureResult))
	require.Equal(t, "9", measureResult.Value)
	require.Equal(t, now.Add(-30*24*time.Hour), sources.windows[3][0])
	require.Equal(t, now, sources.windows[3][1])
}

func TestMeasureWindowRejectsUnspecifiedToken(t *testing.T) {
	_, err := countInWindow(context.Background(), &sharedmeasuresv1.TimeWindow{
		Window: &sharedmeasuresv1.TimeWindow_Token{Token: sharedmeasuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_UNSPECIFIED},
	}, time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC), func(context.Context, time.Time, time.Time) (int64, error) {
		t.Fatal("invalid window must not reach the source")
		return 0, nil
	})
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
