package summarize

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/clock"
	"audio-tools/internal/testutil/mocks"
)

func mkServiceRuntime(t *testing.T, response map[string]any) *SummarizationService {
	t.Helper()
	s := NewSummarizerWithRunner("resource-ollama-test", func(context.Context, []string, string) ([]byte, error) {
		out, err := json.Marshal(response)
		require.NoError(t, err)
		return out, nil
	})
	cfg := DefaultSummarizeConfig()
	cfg.Level = "moderate"
	cfg.Model = DefaultSummarizeModel
	return NewSummarizationServiceWith(s, func() SummarizeConfig { return cfg }, clock.System{})
}

func TestSummarizationService_HappyPath(t *testing.T) {
	svc := mkServiceRuntime(t, map[string]any{
		"response":    "summary text",
		"done_reason": "stop",
		"eval_count":  7,
	})
	res, err := svc.Summarize(context.Background(), SummarizeRequest{EventID: "e1", Path: "manual", Text: "hello world"})
	require.NoError(t, err)
	require.Equal(t, "summary text", res.Summary)
}

func TestSummarizationService_EmptySummary_Classified(t *testing.T) {
	svc := mkServiceRuntime(t, map[string]any{
		"response":    "",
		"done_reason": "length",
		"eval_count":  3,
	})
	_, err := svc.Summarize(context.Background(), SummarizeRequest{EventID: "e2", Path: "manual", Text: "hello"})
	require.ErrorIs(t, err, ErrSummarizeTruncated)
}

func TestSummarizationService_EmptyText(t *testing.T) {
	svc := mkServiceRuntime(t, map[string]any{"response": ""})
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
	cfg := DefaultSummarizeConfig()
	cfg.TimeoutSeconds = 1
	s := NewSummarizerWithRunner("resource-ollama-test", func(context.Context, []string, string) ([]byte, error) {
		return nil, context.DeadlineExceeded
	})
	svc := NewSummarizationServiceWith(s, func() SummarizeConfig { return cfg }, clk)
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
