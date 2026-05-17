package summarize

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/clock"
	"audio-tools/internal/testutil/mocks"
)

func mkServiceRuntime(t *testing.T, handler http.HandlerFunc) *SummarizationService {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	s := NewSummarizer(srv.URL)
	cfg := DefaultSummarizeConfig()
	cfg.Level = "moderate"
	cfg.Model = "qwen3"
	return NewSummarizationServiceWith(s, func() SummarizeConfig { return cfg }, clock.System{})
}

func TestSummarizationService_HappyPath(t *testing.T) {
	svc := mkServiceRuntime(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"message":{"content":"summary text"},"done_reason":"stop","eval_count":7}`))
	})
	res, err := svc.Summarize(context.Background(), SummarizeRequest{EventID: "e1", Path: "manual", Text: "hello world"})
	require.NoError(t, err)
	require.Equal(t, "summary text", res.Summary)
}

func TestSummarizationService_EmptySummary_Classified(t *testing.T) {
	svc := mkServiceRuntime(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"message":{"content":""},"done_reason":"length","eval_count":3}`))
	})
	_, err := svc.Summarize(context.Background(), SummarizeRequest{EventID: "e2", Path: "manual", Text: "hello"})
	require.ErrorIs(t, err, ErrSummarizeTruncated)
}

func TestSummarizationService_EmptyText(t *testing.T) {
	svc := mkServiceRuntime(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"message":{"content":""}}`))
	})
	_, err := svc.Summarize(context.Background(), SummarizeRequest{Text: "   "})
	require.Error(t, err)
}

func TestSummarizationService_NilSummarizer(t *testing.T) {
	svc := NewSummarizationService(nil, func() SummarizeConfig { return DefaultSummarizeConfig() })
	_, err := svc.Summarize(context.Background(), SummarizeRequest{Text: "x"})
	require.Error(t, err)
}

func TestSummarizationService_AutoBackoffPath(t *testing.T) {
	clk := mocks.NewFakeClock(time.Unix(1000, 0))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "timeout", http.StatusGatewayTimeout)
	}))
	t.Cleanup(srv.Close)
	cfg := DefaultSummarizeConfig()
	cfg.TimeoutSeconds = 1
	svc := NewSummarizationServiceWith(NewSummarizer(srv.URL), func() SummarizeConfig { return cfg }, clk)
	_, _ = svc.Summarize(context.Background(), SummarizeRequest{EventID: "e3", Path: "auto", Text: "hi"})
	_, _ = svc.Summarize(context.Background(), SummarizeRequest{EventID: "e3", Path: "auto", Text: "hi"})
}

func TestIsDeadlineExceededError(t *testing.T) {
	require.True(t, isDeadlineExceededError(context.DeadlineExceeded))
	require.False(t, isDeadlineExceededError(errors.New("other")))
}

func TestSummarizationService_NowFallback(t *testing.T) {
	s := &SummarizationService{}
	require.False(t, s.now().IsZero())
}

func TestSummarizeError_Error(t *testing.T) {
	require.Equal(t, "x", summarizeError("x").Error())
}

func TestSetConfigLogger(t *testing.T) {
	prev := SetConfigLogger(nil)
	t.Cleanup(func() { SetConfigLogger(prev) })
}
