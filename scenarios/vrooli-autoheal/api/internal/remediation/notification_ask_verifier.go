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

// AskReaderTokens supplies the bearer token autoheal reads asks with
// (*askreader.Tokens in production).
type AskReaderTokens interface {
	Token(ctx context.Context) (string, error)
}

// NotificationHubAskVerifier reads an already-answered ask. It deliberately
// does not answer asks and it never accepts an approval value from the
// autoheal caller. It reads as autoheal's own verified service identity: the
// Hub shows an ask only to its owner, its sender, or the service that raised it.
type NotificationHubAskVerifier struct {
	client conversationconnect.ConversationsServiceClient
	tokens AskReaderTokens
}

// NewNotificationHubAskVerifier reads asks at baseURL with the bearer that
// tokens supplies. Without a token source it refuses to verify anything.
func NewNotificationHubAskVerifier(baseURL string, tokens AskReaderTokens) (*NotificationHubAskVerifier, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || tokens == nil {
		return nil, ErrAskVerifierUnavailable
	}
	return &NotificationHubAskVerifier{
		client: conversationconnect.NewConversationsServiceClient(http.DefaultClient, baseURL),
		tokens: tokens,
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
	token, err := v.tokens.Token(ctx)
	if err != nil {
		return AskApproval{}, fmt.Errorf("%w: %w", ErrAskVerifierUnavailable, err)
	}
	request := connect.NewRequest(&conversationv1.GetAskRequest{AskId: askID})
	request.Header().Set("Authorization", "Bearer "+token)
	response, err := v.client.GetAsk(ctx, request)
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
