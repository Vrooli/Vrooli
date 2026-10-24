package remediation

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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
	response, err := v.client.GetAsk(ctx, connect.NewRequest(&conversationv1.GetAskRequest{AskId: askID}))
	if err != nil {
		return AskApproval{}, fmt.Errorf("read notification ask %q: %w", askID, err)
	}
	ask := response.Msg.GetAsk()
	if ask == nil || ask.GetId() != askID || ask.GetState() != "answered" || !ApprovedAsk(AskApproval{Answer: ask.GetAnswer()}) {
		answer := ""
		if ask != nil {
			answer = ask.GetAnswer()
		}
		return AskApproval{AskID: askID, Answer: answer}, ErrAskNotApproved
	}
	return AskApproval{AskID: askID, Answer: ask.GetAnswer(), Actor: ask.GetAnsweredBy(), RequesterRef: ask.GetRequesterRef(), SourceEventType: ask.GetSourceEventType()}, nil
}
