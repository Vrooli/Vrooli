package main

import (
	"errors"
	"strings"
	"testing"
)

// TestClassifyEmptySummary pins the four failure categories the service
// distinguishes when the model returns no usable content. The sentinels drive
// both the human-facing banner message and the tts-summarize log line.
func TestClassifyEmptySummary(t *testing.T) {
	cases := []struct {
		name string
		resp TTSSummarizerResponse
		want error
	}{
		{
			name: "budget exhausted inside unclosed think block",
			resp: TTSSummarizerResponse{
				RawContent: "<think>\nOkay, let me think about this",
				DoneReason: "length",
			},
			want: errSummarizeBudgetInThink,
		},
		{
			name: "truncated after think block closed but no summary",
			resp: TTSSummarizerResponse{
				RawContent: "some answer beginning",
				DoneReason: "length",
			},
			want: errSummarizeTruncated,
		},
		{
			name: "stop reason but think block consumed everything",
			resp: TTSSummarizerResponse{
				RawContent: "<think>\nreasoning\n</think>\n",
				DoneReason: "stop",
			},
			want: errSummarizeEmptyAfterStrip,
		},
		{
			name: "model returned truly empty",
			resp: TTSSummarizerResponse{
				RawContent: "",
				DoneReason: "stop",
			},
			want: errSummarizeTrulyEmpty,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyEmptySummary(tc.resp)
			if !errors.Is(got, tc.want) {
				t.Errorf("classifyEmptySummary: got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestSummarizeErrorMessage_Categories ensures every categorized sentinel maps
// to an actionable user-facing string (no "unknown error" fall-through).
func TestSummarizeErrorMessage_Categories(t *testing.T) {
	cases := []struct {
		err          error
		wantContains string
	}{
		{errSummarizeBudgetInThink, "reasoning"},
		{errSummarizeTruncated, "truncated"},
		{errSummarizeEmptyAfterStrip, "only reasoning"},
		{errSummarizeTrulyEmpty, "empty response"},
	}
	for _, tc := range cases {
		msg := summarizeErrorMessage(tc.err)
		if !strings.Contains(strings.ToLower(msg), strings.ToLower(tc.wantContains)) {
			t.Errorf("summarizeErrorMessage(%v) = %q, should contain %q", tc.err, msg, tc.wantContains)
		}
	}
}
