package hub

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// WarnRelatedOpen is advisory: similarity is not evidence that two choices
// authorize the same action. Only the producer can decide how to reconcile them.
const WarnRelatedOpen = "related_open_decision"

// ErrRelatedDecision is intentionally content-free: the sender must not gain
// another conversation's identity, subject key, or terms from the collision.
var ErrRelatedDecision = errors.New("a potentially related open decision exists in another conversation of this owner; ask the owner to review it and use its subject key, or retry with allow_related after deciding this is a distinct decision")

// checkRelatedConflictTx runs under the same write lock as insertion. It only
// gates new direct decisions from verified agent runs; an explicit override is
// the sender's judgment, not authority to read or modify the other decision.
func checkRelatedConflictTx(ctx context.Context, tx *sql.Tx, spec AskSpec) error {
	if spec.AllowRelated || !spec.VerifiedProducer || !strings.HasPrefix(spec.CreatedBy, "run:") || spec.CreatedSubject == "" || spec.SourceEventType != "" || spec.Kind == "status" || spec.Question == "" {
		return nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT a.question, a.created_by, a.created_conversation FROM asks a
		JOIN notifications n ON n.id = a.notification_id
		WHERE n.requested_by = ? AND a.created_subject = ? AND a.source_event_type = ''
		AND a.state IN ('pending', 'escalated') AND a.kind <> 'status'`, spec.Recipient, spec.CreatedSubject)
	if err != nil {
		return err
	}
	defer rows.Close()
	words := subjectWords(spec.Question)
	for rows.Next() {
		var question, creator, conversation string
		if err := rows.Scan(&question, &creator, &conversation); err != nil {
			return err
		}
		if creator == spec.CreatedBy || (spec.CreatedConversation != "" && conversation == spec.CreatedConversation) {
			continue
		}
		if similarSubject(words, subjectWords(question)) {
			return ErrRelatedDecision
		}
	}
	return rows.Err()
}

func (s *Service) checkRelatedOpen(ctx context.Context, spec AskSpec, except string) []PacketWarning {
	// The full match is disclosed only to its canonical audience. A verified
	// run can receive a content-free collision signal for another direct ask of
	// the same owner; it cannot enumerate the ask or acquire its subject key.
	if !spec.VerifiedProducer || spec.CreatedBy == "" || spec.Question == "" {
		return nil
	}
	audience := AskAudience{CreatedBy: spec.CreatedBy, Conversation: spec.CreatedConversation}
	if strings.HasPrefix(spec.CreatedBy, "owner:") {
		audience = OwnerAudience()
	}
	rows, err := s.db.QueryContext(ctx, `SELECT a.id, a.question, a.subject_key, a.created_by, a.created_conversation, a.created_subject, a.source_event_type FROM asks a
		JOIN notifications n ON n.id = a.notification_id
		WHERE n.requested_by = ? AND (? = 1 OR a.source = ? OR (a.created_subject <> '' AND a.created_subject = ? AND a.source_event_type = ''))
		AND a.state IN ('pending', 'escalated') AND a.id <> ?
		ORDER BY a.created_at DESC, a.id DESC LIMIT 200`, spec.Recipient, audience.Owner, spec.Source, spec.CreatedSubject, except)
	if err != nil {
		s.log.Printf("packet check: related open asks: %v", err)
		return nil
	}
	defer rows.Close()
	words := subjectWords(spec.Question)
	var matches []string
	hiddenMatch := false
	for rows.Next() {
		var id, question, key string
		var candidate AskRecord
		if err := rows.Scan(&id, &question, &key, &candidate.CreatedBy, &candidate.CreatedConversation, &candidate.CreatedSubject, &candidate.SourceEventType); err != nil {
			return nil
		}
		if !similarSubject(words, subjectWords(question)) {
			continue
		}
		if audience.Sees(candidate) {
			label := id
			if key != "" {
				label += " (subject " + key + ")"
			}
			matches = append(matches, label)
			if len(matches) == 3 {
				break
			}
		} else if strings.HasPrefix(spec.CreatedBy, "run:") && spec.CreatedSubject != "" && candidate.CreatedSubject == spec.CreatedSubject && candidate.SourceEventType == "" {
			hiddenMatch = true
		}
	}
	if rows.Err() != nil {
		return nil
	}
	if len(matches) > 0 {
		return []PacketWarning{advice(WarnRelatedOpen, "question", fmt.Sprintf(
			"possibly related open decisions: %s. Review their terms and scope before deciding whether to revise one using its subject key; similarity does not merge or replace decisions.", strings.Join(matches, ", ")))}
	}
	if hiddenMatch {
		return []PacketWarning{advice(WarnRelatedOpen, "question", "A potentially related open decision exists in another conversation of this owner. Ask the owner to review it and provide its subject key, or affirm a distinct decision with allow_related. Similarity does not grant access or merge decisions.")}
	}
	return nil
}
