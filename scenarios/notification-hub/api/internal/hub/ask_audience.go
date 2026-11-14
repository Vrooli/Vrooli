package hub

import (
	"context"
	"errors"
	"strings"
)

// AskAudience says which asks a verified reader may see. The zero value sees
// nothing, so a reader whose identity was not established can never be handed
// an ask by forgetting to fill a field in.
//
// Three kinds of reader exist:
//
//   - the owner (the recipient): every ask addressed to them;
//   - a verified agent run: the asks of its conversation, which are the asks it
//     sent and the asks of the runs that came before or after it in the same
//     line of work (a recovered run, a child run, a re-attached session); and
//     any ask whose subject key it names exactly, when it is a run of the same
//     owner (see HoldsKey);
//   - a verified service: the event-sourced asks whose Source is its own name,
//     because an event-sourced ask has no verified sender but the scenario that
//     raised it is the only one that may read it back.
type AskAudience struct {
	// Owner sees every ask of the recipient.
	Owner bool
	// CreatedBy sees the asks recorded as sent by this identity.
	CreatedBy string
	// Conversation sees the asks sent from this verified conversation. A run
	// whose token carries no conversation leaves it empty and sees only its own.
	Conversation string
	// Subject is the reader's verified owner account. With the exact subject key
	// of an ask sent by the same owner it is a capability (HoldsKey); it never
	// lets the reader list or find asks.
	Subject string
	// Source sees the event-sourced asks of this source scenario.
	Source string
}

// OwnerAudience is the audience of the verified owner.
func OwnerAudience() AskAudience { return AskAudience{Owner: true} }

// Sees reports whether the audience may read ask by its id or in a list.
func (a AskAudience) Sees(ask AskRecord) bool {
	switch {
	case a.Owner:
		return true
	case a.CreatedBy != "" && ask.CreatedBy == a.CreatedBy:
		return true
	case a.Conversation != "" && ask.CreatedConversation == a.Conversation:
		return true
	case a.Source != "" && ask.SourceEventType != "" && ask.Source == a.Source:
		return true
	}
	return false
}

// HoldsKey reports whether a reader who names ask's exact subject key may use
// the ask: a verified run of the same owner account as the run that sent it.
// Naming the key is the proof, the way it is for revising the ask, so an agent
// that lost its run can re-derive the key and carry on; it is how an unrelated
// run of the same owner reaches an ask without being told its id. An ask sent
// with no verified owner, and every event-sourced ask, has no such capability.
func (a AskAudience) HoldsKey(ask AskRecord) bool {
	return a.Subject != "" && ask.CreatedSubject == a.Subject && ask.SourceEventType == ""
}

// SeesByKey reports whether the audience may use ask after naming its subject key.
func (a AskAudience) SeesByKey(ask AskRecord) bool { return a.Sees(ask) || a.HoldsKey(ask) }

// empty reports an audience that can see nothing at all in a list.
func (a AskAudience) empty() bool {
	return !a.Owner && a.CreatedBy == "" && a.Conversation == "" && a.Source == ""
}

// scopeKey names the audience inside a list cursor, so a cursor made for one
// reader is refused for another. The subject is not part of it: it never
// changes what a list shows.
func (a AskAudience) scopeKey() string {
	if a.Owner {
		return "owner"
	}
	return strings.Join([]string{"created:" + a.CreatedBy, "conversation:" + a.Conversation, "source:" + a.Source}, "|")
}

// GetAskFor reads one ask addressed to recipient for a reader. An ask the
// reader may not see is reported exactly like an ask that does not exist, so a
// reader cannot probe for other callers' ask ids.
func (s *Service) GetAskFor(ctx context.Context, id, recipient string, audience AskAudience) (AskRecord, error) {
	ask, err := s.GetAskForRecipient(ctx, id, recipient)
	if err != nil {
		return AskRecord{}, err
	}
	if !audience.Sees(ask) {
		return AskRecord{}, ErrNotFound
	}
	return ask, nil
}

// GetAskBySubjectKeyFor reads the current ask of a direct subject key
// addressed to recipient, for a reader that names the exact key: the open ask,
// else the latest. A reader that is neither in the ask's audience nor holds the
// key's capability is told the key has no ask, exactly like a key that has none.
func (s *Service) GetAskBySubjectKeyFor(ctx context.Context, subjectKey, recipient string, audience AskAudience) (AskRecord, error) {
	recipient = strings.TrimSpace(recipient)
	subjectKey = strings.TrimSpace(subjectKey)
	if recipient == "" || subjectKey == "" {
		return AskRecord{}, ErrNotFound
	}
	ask, err := s.latestDirectSubjectAsk(ctx, recipient, subjectKey)
	if err != nil {
		return AskRecord{}, err
	}
	if !audience.SeesByKey(ask) {
		return AskRecord{}, ErrNotFound
	}
	return ask, nil
}

// latestDirectSubjectAsk is the open ask of a direct subject key, else its most
// recent one.
func (s *Service) latestDirectSubjectAsk(ctx context.Context, recipient, subjectKey string) (AskRecord, error) {
	ask, err := openSubjectAskTx(ctx, s.db, recipient, "", subjectKey)
	if errors.Is(err, ErrNotFound) {
		return latestSubjectAskTx(ctx, s.db, recipient, "", subjectKey)
	}
	return ask, err
}
