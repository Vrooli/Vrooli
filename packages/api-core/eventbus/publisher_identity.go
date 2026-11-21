package eventbus

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/vrooli/api-core/eventpublisher"
	"github.com/vrooli/api-core/serviceprincipal"
	"github.com/vrooli/api-core/servicetoken"
)

// PublisherCredentials reads the credential of source's event-publisher principal
// (eventpublisher.CredentialIdentity(source), eventpublisher.CredentialField) from
// the credential authority. A scenario registers one with UsePublisherCredentials.
type PublisherCredentials func(ctx context.Context, source string) (string, error)

// publisherTokenWait bounds how long a publish waits for a token, so a hung
// authenticator delays an event by seconds at most.
const publisherTokenWait = 3 * time.Second

// newPublisherTokens builds the token client of one source; tests replace it.
var newPublisherTokens = func(source string, credential servicetoken.CredentialSource) *servicetoken.Tokens {
	return eventpublisher.NewTokens(source, credential)
}

// publisherRetry spaces new attempts to get a token for a source after a failure,
// so an authenticator that is down costs one exchange a minute, not one per event.
const publisherRetry = time.Minute

var publishers = struct {
	sync.Mutex
	credentials PublisherCredentials
	bySource    map[string]*publisherIdentity
	now         func() time.Time
}{bySource: map[string]*publisherIdentity{}, now: time.Now}

type publisherIdentity struct {
	tokens   *servicetoken.Tokens
	failedAt time.Time
	reported bool
}

// UsePublisherCredentials makes every Client in this process present the
// "<source>-event-publisher" identity of the source it publishes as, so
// vrooli-events can tell that the source is the publisher's own (see package
// eventpublisher). A source whose credential the operator has not issued
// publishes exactly as it always has, unauthenticated; issuing it is
// serviceprincipal.IssueCommand. A nil provider turns this off.
func UsePublisherCredentials(credentials PublisherCredentials) {
	publishers.Lock()
	defer publishers.Unlock()
	publishers.credentials = credentials
	publishers.bySource = map[string]*publisherIdentity{}
}

// PublisherAuthorization is the Authorization header value that proves source to
// vrooli-events, or "" when there is none to give. Client.PublishDomainEvent sets
// it; a scenario that posts an envelope to vrooli-events itself sets it the same
// way. It never fails a publish: an
// event with no identity is accepted as before and counted as unauthenticated by
// the bus, so adopting this cannot lose an event.
func PublisherAuthorization(ctx context.Context, source string) string {
	publishers.Lock()
	credentials := publishers.credentials
	if credentials == nil || source == "" {
		publishers.Unlock()
		return ""
	}
	identity := publishers.bySource[source]
	if identity == nil {
		identity = &publisherIdentity{tokens: newPublisherTokens(source, func(ctx context.Context) (string, error) {
			return credentials(ctx, source)
		})}
		publishers.bySource[source] = identity
	}
	now := publishers.now()
	if !identity.failedAt.IsZero() && now.Sub(identity.failedAt) < publisherRetry {
		publishers.Unlock()
		return ""
	}
	publishers.Unlock()

	waitCtx, cancel := context.WithTimeout(ctx, publisherTokenWait)
	defer cancel()
	authorization, err := identity.tokens.Authorization(waitCtx)
	if err == nil {
		return authorization
	}
	publishers.Lock()
	identity.failedAt = publishers.now()
	first := !identity.reported
	identity.reported = true
	publishers.Unlock()
	// Not issued is the expected state until the operator applies; say it once.
	if first && errors.Is(err, servicetoken.ErrNotIssued) {
		log.Printf("eventbus: %s publishes without an identity (no event-publisher credential): the operator closes this with `%s`", source, serviceprincipal.IssueCommand)
	} else if first {
		log.Printf("eventbus: %s publishes without an identity this minute: %v", source, err)
	}
	return ""
}
