package summarize

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/scheduletest"
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
	return NewSummarizationServiceWith(s, func() SummarizeConfig { return cfg }, schedule.System())
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
	clk := scheduletest.New(time.Unix(1000, 0))
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

func TestWaitForSummarizeFuture_RespectsCancellationAndCompletion(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := waitForSummarizeFuture(canceled, &summarizeFuture{done: make(chan struct{})})
	require.ErrorIs(t, err, context.Canceled)

	future := &summarizeFuture{done: make(chan struct{}), result: SummarizeResult{Summary: "ready"}}
	close(future.done)
	result, err := waitForSummarizeFuture(context.Background(), future)
	require.NoError(t, err)
	require.Equal(t, "ready", result.Summary)
}

func TestSummarizeErrorMessage_CategorizesEveryPublicFailure(t *testing.T) {
	require.Empty(t, SummarizeErrorMessage(nil))
	require.Contains(t, SummarizeErrorMessage(ErrSummarizeBudgetInThink), "token budget")
	require.Contains(t, SummarizeErrorMessage(ErrSummarizeEmptyAfterStrip), "reasoning")
	require.Contains(t, SummarizeErrorMessage(ErrSummarizeTrulyEmpty), "empty")
	require.Contains(t, SummarizeErrorMessage(ErrSummarizeCoolingDown), "cooling")
	require.Contains(t, SummarizeErrorMessage(context.DeadlineExceeded), "timed out")
	require.Contains(t, SummarizeErrorMessage(errors.New("network down")), "network down")
}

func TestNewSummarizer_DefaultsAndMissingGateway(t *testing.T) {
	s := NewSummarizer("ignored", nil)
	require.Equal(t, defaultOllamaGatewayBin, s.Bin)
	configured := NewSummarizerWithRunner("", nil)
	require.Equal(t, defaultOllamaGatewayBin, configured.Bin)
	_, err := runGatewayCLI(context.Background(), "definitely-not-an-audio-tools-binary", nil, "")
	require.Error(t, err)
}
