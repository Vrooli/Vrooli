package threads

import (
	"testing"

	"github.com/stretchr/testify/require"
	"switchboard/internal/channels"
)

// [REQ:SWBD-P1-001] [REQ:SWBD-P1-002] [REQ:SWBD-P1-003] [REQ:SWBD-P1-004]
func TestGroupDisciplineAndLoopBreaker(t *testing.T) {
	require.False(t, ShouldRespond(channels.Envelope{AuthorKind: channels.AuthorAgent}, false, true))
	require.False(t, ShouldRespond(channels.Envelope{AuthorKind: channels.AuthorHuman}, true, false))
	require.True(t, ShouldRespond(channels.Envelope{AuthorKind: channels.AuthorHuman}, true, true))
}
