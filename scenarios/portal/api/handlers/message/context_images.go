package message

import (
	"context"
	"errors"

	internalchat "portal/internal/chat"
	"portal/internal/contextcapture"
)

// contextImageResolver is the delivery-side join between the owned chat
// message reference table and context-capture's expiring image records. It
// always renders from the current owner and never trusts renderer-supplied
// pixels or a document that was valid at attach time only.
type contextImageResolver struct {
	chat    *internalchat.Service
	context contextRenderer
}

type contextRenderer interface {
	Render(context.Context, string, string) (contextcapture.Document, []byte, string, error)
}

func newContextImageResolver(chat *internalchat.Service, service contextRenderer) completionImageResolver {
	if chat == nil || service == nil {
		return nil
	}
	return &contextImageResolver{chat: chat, context: service}
}

// completionImageResolver keeps the adapter's contract local to the message
// handler package while satisfying completion.MessageImageResolver.
type completionImageResolver interface {
	ResolveMessageImages(context.Context, string, string) ([][]byte, error)
}

func (r *contextImageResolver) ResolveMessageImages(ctx context.Context, chatID, messageID string) ([][]byte, error) {
	owner := internalchat.RequestOwner(ctx)
	if owner == "" {
		return nil, errors.New("owned context requires an authenticated account")
	}
	ids, err := r.chat.ListMessageContextDocumentIDs(ctx, chatID, messageID)
	if err != nil {
		return nil, err
	}
	images := make([][]byte, 0, len(ids))
	for _, id := range ids {
		_, pixels, _, err := r.context.Render(ctx, owner, id)
		if err != nil {
			return nil, err
		}
		images = append(images, append([]byte(nil), pixels...))
	}
	return images, nil
}
