package hub

import (
	"context"
	"errors"
	"strings"
	"time"
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
	// DirectSubjectSource names a verified service whose exact direct asks may
	// be recovered only through a durable ask/source/recipient attestation.
	DirectSubjectSource string
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
func (a AskAudience) SeesByKey(ask AskRecord) bool {
	return a.Sees(ask) || a.HoldsKey(ask)
}

// empty reports an audience that can see nothing at all in a list.
func (a AskAudience) empty() bool {
	return !a.Owner && a.CreatedBy == "" && a.Conversation == "" && a.Source == "" && a.DirectSubjectSource == ""
}

// scopeKey names the audience inside a list cursor, so a cursor made for one
// reader is refused for another. The subject is not part of it: it never
// changes what a list shows.
func (a AskAudience) scopeKey() string {
	if a.Owner {
		return "owner"
	}
	return strings.Join([]string{"created:" + a.CreatedBy, "conversation:" + a.Conversation, "source:" + a.Source, "direct-subject-source:" + a.DirectSubjectSource}, "|")
}

// GetAskFor reads one ask addressed to recipient for a reader. An ask the
// reader may not see is reported exactly like an ask that does not exist, so a
// reader cannot probe for other callers' ask ids.
func (s *Service) GetAskFor(ctx context.Context, id, recipient string, audience AskAudience) (AskRecord, error) {
	ask, err := s.GetAskForRecipient(ctx, id, recipient)
	if err != nil {
		return AskRecord{}, err
	}
	if !audience.Sees(ask) && !s.hasSourceAttestation(ctx, ask, audience) {
		return AskRecord{}, ErrNotFound
	}
	s.decorateRead(ctx, &ask, audience)
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
	var ask AskRecord
	var err error
	if audience.Source != "" {
		// A verified service source re-derives event-sourced asks by their exact
		// subject key. Direct asks remain reachable only through their sender or
		// owner capability.
		ask, err = s.latestEventSubjectAsk(ctx, recipient, audience.Source, subjectKey)
	} else {
		ask, err = s.latestDirectSubjectAsk(ctx, recipient, subjectKey)
	}
	if err != nil {
		return AskRecord{}, err
	}
	if !audience.SeesByKey(ask) && !s.hasSourceAttestation(ctx, ask, audience) {
		return AskRecord{}, ErrNotFound
	}
	s.decorateRead(ctx, &ask, audience)
	return ask, nil
}

func (s *Service) hasSourceAttestation(ctx context.Context, ask AskRecord, audience AskAudience) bool {
	if audience.DirectSubjectSource != "portfolio-steward" || ask.SourceEventType != "" ||
		ask.Recipient == "" || !validPortfolioSubjectKey(ask.SubjectKey) {
		return false
	}
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM ask_source_attestations WHERE ask_id=? AND source=? AND recipient=?`, ask.ID, audience.DirectSubjectSource, ask.Recipient).Scan(&one)
	return err == nil
}

func validPortfolioSubjectKey(key string) bool {
	parts := strings.Split(key, ":")
	if len(parts) != 3 || parts[0] != "portfolio-steward" || (parts[1] != "roadmap" && parts[1] != "daily") {
		return false
	}
	_, err := time.Parse("2006-01-02", parts[2])
	return err == nil && len(parts[2]) == 10
}

// AttestPortfolioAsk records a verified same-owner authorization for
// portfolio-steward to continue this one pending direct roadmap/daily ask.
func (s *Service) AttestPortfolioAsk(ctx context.Context, askID, owner string) (AskRecord, error) {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return AskRecord{}, ErrNotFound
	}
	return s.attestPortfolioAsk(ctx, askID, OwnerAudience(), owner)
}

// AttestPortfolioAskFromRun records the steward's exact association only when
// the verified run already belongs to the stored ask's creator or conversation
// audience. Subject-key capability alone is deliberately insufficient.
func (s *Service) AttestPortfolioAskFromRun(ctx context.Context, askID string, audience AskAudience) (AskRecord, error) {
	if audience.Owner || !strings.HasPrefix(strings.TrimSpace(audience.CreatedBy), "run:") {
		return AskRecord{}, ErrNotFound
	}
	return s.attestPortfolioAsk(ctx, askID, audience, "")
}

func (s *Service) attestPortfolioAsk(ctx context.Context, askID string, audience AskAudience, expectedRecipient string) (AskRecord, error) {
	tx, err := s.beginWriteTx(ctx)
	if err != nil {
		return AskRecord{}, err
	}
	defer tx.Rollback()

	ask, err := scanAsk(tx.QueryRowContext(ctx, askSelect+` WHERE a.id = ?`, strings.TrimSpace(askID)))
	if err != nil || ask.Recipient == "" ||
		(ask.Source != ask.Recipient && ask.Source != "operator" && ask.Source != "portfolio-steward") ||
		ask.SourceEventType != "" || ask.State != AskPending || !validPortfolioSubjectKey(ask.SubjectKey) {
		return AskRecord{}, ErrNotFound
	}
	if audience.Owner {
		if expectedRecipient == "" || ask.Recipient != expectedRecipient {
			return AskRecord{}, ErrNotFound
		}
	} else if !audience.Sees(ask) || (audience.Subject != "" && audience.Subject != ask.Recipient) {
		return AskRecord{}, ErrNotFound
	}

	// The write lock was acquired before reading the ask, so an answer cannot
	// race between the pending check and this exact ask/source/recipient proof.
	now := storedTime(s.clock.Now())
	result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO ask_source_attestations(ask_id,source,recipient,attested_at)
		SELECT a.id,'portfolio-steward',?,? FROM asks a
		JOIN notifications n ON n.id=a.notification_id
		WHERE a.id=? AND n.requested_by=? AND a.source_event_type=''
		AND a.state='pending' AND a.subject_key=? AND a.source IN (?, 'operator', 'portfolio-steward')`,
		ask.Recipient, now, ask.ID, ask.Recipient, ask.SubjectKey, ask.Recipient)
	if err != nil {
		return AskRecord{}, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return AskRecord{}, err
	}
	if changed != 1 {
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM ask_source_attestations WHERE ask_id=? AND source='portfolio-steward' AND recipient=?`, ask.ID, ask.Recipient).Scan(&exists); err != nil {
			return AskRecord{}, ErrNotFound
		}
	}
	if err := tx.Commit(); err != nil {
		return AskRecord{}, err
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

// latestEventSubjectAsk is the open ask for a verified source and subject,
// regardless of event type, else the latest one. A source identity can only
// reach event-sourced asks from that source; it cannot use this path to read a
// direct ask or another scenario's ask.
func (s *Service) latestEventSubjectAsk(ctx context.Context, recipient, source, subjectKey string) (AskRecord, error) {
	const where = ` WHERE a.notification_id IN (SELECT id FROM notifications WHERE requested_by = ?) AND a.source = ? AND a.source_event_type <> '' AND a.subject_key = ?`
	ask, err := scanAsk(s.db.QueryRowContext(ctx, askSelect+where+` AND a.state IN ('pending','escalated') ORDER BY a.created_at DESC, a.id DESC LIMIT 1`, recipient, source, subjectKey))
	if errors.Is(err, ErrNotFound) {
		return scanAsk(s.db.QueryRowContext(ctx, askSelect+where+` ORDER BY a.created_at DESC, a.id DESC LIMIT 1`, recipient, source, subjectKey))
	}
	return ask, err
}
