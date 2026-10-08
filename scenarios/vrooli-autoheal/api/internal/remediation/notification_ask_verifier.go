package remediation

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

// NotificationHubAskVerifier reads an already-answered ask. It deliberately
// does not answer asks and it never accepts an approval value from the
// autoheal caller.
type NotificationHubAskVerifier struct {
	client conversationconnect.ConversationsServiceClient
}

func NewNotificationHubAskVerifier(baseURL string) (*NotificationHubAskVerifier, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, ErrAskVerifierUnavailable
	}
	return &NotificationHubAskVerifier{
		client: conversationconnect.NewConversationsServiceClient(http.DefaultClient, baseURL),
	}, nil
}

func (v *NotificationHubAskVerifier) Verify(ctx context.Context, askID string) (AskApproval, error) {
	if v == nil || v.client == nil {
		return AskApproval{}, ErrAskVerifierUnavailable
	}
	askID = strings.TrimSpace(askID)
	if askID == "" {
		return AskApproval{}, fmt.Errorf("%w: ask id is empty", ErrAskNotApproved)
	}
	// An answered ask returns immediately. A pending ask is intentionally
	// expired by the read deadline rather than being waited on by an execution
	// endpoint; execution must only follow a prior operator answer.
	deadline := time.Now().UTC().Format(time.RFC3339Nano)
	response, err := v.client.Wait(ctx, connect.NewRequest(&conversationv1.WaitRequest{AskId: askID, Deadline: deadline}))
	if err != nil {
		return AskApproval{}, fmt.Errorf("read notification ask %q: %w", askID, err)
	}
	if response.Msg.GetState() != "answered" || !ApprovedAsk(AskApproval{Answer: response.Msg.GetAnswer()}) {
		return AskApproval{AskID: askID, Answer: response.Msg.GetAnswer()}, ErrAskNotApproved
	}
	return AskApproval{AskID: askID, Answer: response.Msg.GetAnswer(), Actor: "notification-hub"}, nil
}
