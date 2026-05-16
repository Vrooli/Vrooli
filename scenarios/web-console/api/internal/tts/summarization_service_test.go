package tts

import (
	"errors"
	"strings"
	"testing"
)

// TestClassifyEmptySummary pins the four failure categories the service
// distinguishes when the model returns no usable content.
func TestClassifyEmptySummary(t *testing.T) {
	cases := []struct {
		name string
		resp SummarizerResponse
		want error
	}{
		{
			name: "budget exhausted inside unclosed think block",
			resp: SummarizerResponse{
				RawContent: "<think>\nOkay, let me think about this",
				DoneReason: "length",
			},
			want: ErrSummarizeBudgetInThink,
		},
		{
			name: "truncated after think block closed but no summary",
			resp: SummarizerResponse{
				RawContent: "some answer beginning",
				DoneReason: "length",
			},
			want: ErrSummarizeTruncated,
		},
		{
			name: "stop reason but think block consumed everything",
			resp: SummarizerResponse{
				RawContent: "<think>\nreasoning\n</think>\n",
				DoneReason: "stop",
			},
			want: ErrSummarizeEmptyAfterStrip,
		},
		{
			name: "model returned truly empty",
			resp: SummarizerResponse{
				RawContent: "",
				DoneReason: "stop",
			},
			want: ErrSummarizeTrulyEmpty,
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

func TestSummarizeErrorMessage_Categories(t *testing.T) {
	cases := []struct {
		err          error
		wantContains string
	}{
		{ErrSummarizeBudgetInThink, "reasoning"},
		{ErrSummarizeTruncated, "truncated"},
		{ErrSummarizeEmptyAfterStrip, "only reasoning"},
		{ErrSummarizeTrulyEmpty, "empty response"},
	}
	for _, tc := range cases {
		msg := SummarizeErrorMessage(tc.err)
		if !strings.Contains(strings.ToLower(msg), strings.ToLower(tc.wantContains)) {
			t.Errorf("SummarizeErrorMessage(%v) = %q, should contain %q", tc.err, msg, tc.wantContains)
		}
	}
}
