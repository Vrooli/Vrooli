package conversations_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

func bareAsk(key string) *v1.AskRequest {
	return &v1.AskRequest{
		Question: "Retry the backup now?", IdempotencyKey: key,
		Options: []*v1.AskOption{{Key: "yes", Label: "Yes"}, {Key: "no", Label: "No"}},
	}
}

func TestAsk_NeedsNeitherADeadlineNorASensitivityLabel(t *testing.T) {
	ctx := context.Background()
	client := newFormClient(t)
	before := time.Now()
	asked, err := client.Ask(ctx, as("alice", bareAsk("bare-1")))
	require.NoError(t, err)
	shown, err := client.GetAsk(ctx, as("alice", &v1.GetAskRequest{AskId: asked.Msg.GetAskId()}))
	require.NoError(t, err)

	deadline, err := time.Parse(time.RFC3339Nano, shown.Msg.GetAsk().GetDeadline())
	require.NoError(t, err)
	require.WithinDuration(t, before.Add(24*time.Hour), deadline, time.Minute, "the Hub's default is 24 hours")
	require.Equal(t, "private", asked.Msg.GetNotification().GetSensitivityLabel())
}

// Sending the same request again must return the same ask. A deadline computed
// from the clock on each call differs in every request, so it cannot be part of
// the identity of a request that names none.
func TestAsk_SendingTheSameBareRequestAgainReturnsTheSameAsk(t *testing.T) {
	ctx := context.Background()
	client := newFormClient(t)
	first, err := client.Ask(ctx, as("alice", bareAsk("bare-2")))
	require.NoError(t, err)
	second, err := client.Ask(ctx, as("alice", bareAsk("bare-2")))
	require.NoError(t, err, "a retry with no deadline must not conflict over a deadline that moved")
	require.Equal(t, first.Msg.GetAskId(), second.Msg.GetAskId())
}
