package hub_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"notification-hub/internal/hub"
)

// A tap on a push opens the Hub's own record of the notification, whatever
// context_url the producer sent. The link is shown on the record as a second tap.
func TestPushOpensTheHubRecordNotTheProducersContextURL(t *testing.T) {
	for name, contextURL := range map[string]string{
		"none":                               "",
		"a Hub-relative path":                "/asks/another-ask",
		"an absolute external site":          "https://evil.example/phish",
		"an absolute URL on a sibling site":  "https://web-console.example.test/?session=s1",
		"an absolute URL with another host":  "http://203.0.113.9:8080/x",
		"a path that names a different page": "/settings/reach",
	} {
		t.Run(name, func(t *testing.T) {
			service, _ := testService(t)
			ctx := context.Background()
			sender := &recordingSender{}
			service.SetPushSender(sender)
			registerSubscription(t, service, "alice")
			n, err := service.Send(ctx, hub.SendInput{
				RequestedBy: "alice", Title: "Backup finished", Body: "All volumes verified", SensitivityLabel: "public",
				IdempotencyKey: "push-target-" + name, ContextURL: contextURL,
			})
			require.NoError(t, err)
			require.NoError(t, service.Process(ctx, n.ID))
			require.Eventually(t, func() bool { return pushCount(sender) == 1 }, asyncWait, 10*time.Millisecond)

			sender.mu.Lock()
			defer sender.mu.Unlock()
			require.Equal(t, "delivery/"+n.ID, sender.messages[0].URL, "the tap opens the delivery record")
			require.Empty(t, sender.messages[0].Navigate, "Chimesail does not use the declarative envelope")
		})
	}
}

func TestPushForAnAskOpensTheAskEvenWhenItsNotificationCarriesAContextURL(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	sender := &recordingSender{}
	service.SetPushSender(sender)
	registerSubscription(t, service, "alice")
	service.SetClockForTest(fixedClock{now: listNow})
	ask, _, err := service.AskWithSpec(ctx, decisionSpec(listNow, "push-ask-link"))
	require.NoError(t, err)
	waitDelivered(t, service, ask)
	// An event-sourced ask's notification can carry the producer's link.
	_, err = db.ExecContext(ctx, `UPDATE notifications SET context_url = 'https://evil.example/' WHERE id = ?`, ask.NotificationID)
	require.NoError(t, err)
	second := decisionSpec(listNow, "push-ask-link-2")
	second.Question = "Second question?"
	other, _, err := service.AskWithSpec(ctx, second)
	require.NoError(t, err)
	waitDelivered(t, service, other)

	sender.mu.Lock()
	defer sender.mu.Unlock()
	require.Len(t, sender.messages, 2)
	require.Equal(t, fmt.Sprintf("asks/%s", ask.ID), sender.messages[0].URL)
	require.Equal(t, fmt.Sprintf("asks/%s", other.ID), sender.messages[1].URL)
}
