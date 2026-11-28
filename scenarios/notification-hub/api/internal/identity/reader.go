package identity

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/hubsender"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"

	"notification-hub/internal/hub"
)

// An ask carries the questions, answers and notes the operator exchanged with
// one requester, and the sender's host paths, so reading one is not open to
// whoever can reach the API. A reader is the owner, the verified run that sent
// the ask, or the verified service that raised it as an event; nobody else. The
// service principal names, audience and scope are api-core's askreader contract,
// shared with every scenario that reads its asks back.

// Creator is the identity recorded on an ask a producer sends: "run:<run id>"
// for a verified agent run, "owner:<subject>" for the owner, and "" for a
// caller with no identity (and so for nobody who can read it back).
func (p Producer) Creator() string {
	if p.Anonymous() {
		return ""
	}
	return p.Key
}

// Audience is the audience a verified producer reads back what it sent: the
// owner reads everything, and a run reads its conversation and, naming the exact
// subject key, the asks of the same owner. A producer with no identity reads
// nothing.
func (p Producer) Audience() hub.AskAudience {
	switch {
	case p.Anonymous():
		return hub.AskAudience{}
	case strings.HasPrefix(p.Key, "owner:"):
		return hub.OwnerAudience()
	}
	return hub.AskAudience{
		CreatedBy: p.Key, Conversation: p.Conversation, Subject: p.Subject,
		DirectSubjectSource: p.Source,
	}
}

// ReaderFor resolves which asks the caller of GetAsk, ListAsks or Wait may see.
//
// A request that presents an agent identity is that run and never the owner,
// even where a loopback caller would otherwise be the owner: a run reads the
// asks of its verified conversation (its own, and those of the runs recovered
// from it or spawned by it), and, naming an exact subject key, the ask of that
// key sent by a run of the same owner. Otherwise the owner's verified login, cookie or bearer makes an
// owner reader; otherwise a bearer that is a verified ask-reader service token
// reads the asks that service raised as events; a bearer that is a verified
// sender service token reads the asks that service sent directly. A caller with no identity, or whose
// credential does not verify, gets owneridentity.ErrUnauthenticated.
func ReaderFor(ctx context.Context, headers http.Header, owner Verifier, services ServiceVerifiers) (hub.AskAudience, error) {
	if ownerCtx, ok := ownerFromContext(ctx); ok && ownerCtx.refused {
		return hub.AskAudience{}, owneridentity.ErrUnauthenticated
	}
	agent := provenance.FromContext(ctx)
	if strings.TrimSpace(headers.Get(cliutil.HeaderAgentIdentityToken)) != "" || agent.IsAgent() {
		if !agent.IsVerifiedAgent() || strings.TrimSpace(agent.RunID) == "" {
			return hub.AskAudience{}, owneridentity.ErrUnauthenticated
		}
		return hub.AskAudience{
			CreatedBy:    "run:" + strings.TrimSpace(agent.RunID),
			Conversation: strings.TrimSpace(agent.ConversationID),
			Subject:      strings.TrimSpace(agent.Subject),
		}, nil
	}
	if _, err := Subject(ctx, headers, owner); err == nil {
		return hub.OwnerAudience(), nil
	}
	if source, err := askreader.VerifiedSource(ctx, services.AskReaders, headers); err == nil {
		return hub.AskAudience{Source: source}, nil
	} else if strings.TrimSpace(headers.Get("Authorization")) != "" {
		// Keep the public refusal generic, but retain the role verifier's
		// noncredential diagnostic so configured JWKS and claim mismatches can
		// be separated without logging bearer material.
		slog.Warn("ask-reader service identity refused", "reason", err)
	}
	if source, err := hubsender.VerifiedSource(ctx, services.Senders, headers); err == nil {
		// A sender reads back the asks it sent, as a run does; it never reads the
		// asks another scenario raised.
		return hub.AskAudience{CreatedBy: serviceKeyPrefix + source, DirectSubjectSource: source}, nil
	}
	return hub.AskAudience{}, owneridentity.ErrUnauthenticated
}
