// Responsibility: stamp user-role message events with who delivered them.
package orchestration

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/identity"

	"agent-manager/internal/domain"
)

// messageSender maps the verified caller of a request to a message sender: a
// human principal is a person, an agent principal is an agent. A service, an
// unverified or absent principal (background wakes, recovery) is neither, so no
// sender is recorded and readers fall back to the run's lineage.
func messageSender(ctx context.Context) (sender, id string) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return "", ""
	}
	switch principal.Kind {
	case identity.ActorHuman:
		return domain.MessageSenderPerson, principal.Subject
	case identity.ActorAgent:
		return domain.MessageSenderAgent, principal.Subject
	}
	return "", ""
}

// newUserMessage builds the event for a user-role message accepted from the
// caller in ctx, with attachments when there are any. Every place Agent Manager
// accepts a user message builds it here, so the sender is never forgotten.
func newUserMessage(ctx context.Context, runID uuid.UUID, content string, attachments []domain.MessageAttachmentInfo) *domain.RunEvent {
	var event *domain.RunEvent
	if len(attachments) > 0 {
		event = domain.NewMessageEventWithAttachments(runID, "user", content, attachments)
	} else {
		event = domain.NewMessageEvent(runID, "user", content)
	}
	if data, ok := event.Data.(*domain.MessageEventData); ok {
		data.Sender, data.SenderID = messageSender(ctx)
	}
	return event
}
