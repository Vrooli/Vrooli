package openrouter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSSEEmitsTokensUsageAndFinishReason(t *testing.T) {
	raw := strings.NewReader("event: message\n" +
		"data: {\"id\":\"gen-1\",\"model\":\"model-a\",\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n\n" +
		"data: {\"id\":\"gen-1\",\"model\":\"model-a\",\"choices\":[{\"delta\":{\"content\":\"lo\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5}}\n\n" +
		"data: [DONE]\n\n")

	var events []StreamEvent
	err := parseSSE(raw, func(ev StreamEvent) error {
		events = append(events, ev)
		return nil
	})

	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "Hel", events[0].Token)
	require.Equal(t, "lo", events[1].Token)
	require.Equal(t, "stop", events[1].FinishReason)
	require.EqualValues(t, 3, events[1].Usage.PromptTokens)
	require.EqualValues(t, 2, events[1].Usage.CompletionTokens)
}
