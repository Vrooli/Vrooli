package agents

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// [REQ:SWBD-P0-007] [REQ:SWBD-P0-008]
func TestResolveRejectsAmbiguous(t *testing.T) {
	_, err := Resolve([]Binding{{AgentID: "a", ChannelID: "x", ThreadKey: "t", Address: "u"}, {AgentID: "b", ChannelID: "x", ThreadKey: "t", Address: "u"}}, "x", "t", "u")
	var amb AmbiguousBinding
	require.ErrorAs(t, err, &amb)
}

func TestResolveExact(t *testing.T) {
	b, err := Resolve([]Binding{{AgentID: "a", ChannelID: "x", ThreadKey: "t", Address: "u"}}, "x", "t", "u")
	require.NoError(t, err)
	require.Equal(t, "a", b.AgentID)
}
