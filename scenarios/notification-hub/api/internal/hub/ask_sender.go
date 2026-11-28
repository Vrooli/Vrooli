package hub

import (
	"context"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The sender is who the operator reads as asking. A requester declares it; the
// Hub names the rest from what it verified. It is display and filter text only
// and never grants access: reading an ask back is decided by CreatedBy,
// CreatedConversation and Source alone.
const (
	maxAskSenderRunes = 60
	// SenderAgentRun names an ask sent by a verified run that declared no sender.
	SenderAgentRun = "Agent run"
	// SenderUnnamed names an ask with no declared sender and nothing verified.
	SenderUnnamed = "Unnamed sender"
)

// normalizeSender collapses whitespace in a declared sender and refuses one
// that is not text of 1 to maxAskSenderRunes code points.
func normalizeSender(declared string) (string, error) {
	text := strings.Join(strings.Fields(declared), " ")
	if text == "" {
		return "", nil
	}
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) > maxAskSenderRunes {
		return "", fmt.Errorf("%w: sender must be valid UTF-8 and at most %d characters", ErrInvalidArgument, maxAskSenderRunes)
	}
	for _, r := range text {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("%w: sender must not contain control characters", ErrInvalidArgument)
		}
	}
	return text, nil
}

// repairSender makes a suggested sender fit the sender rules instead of
// refusing it: blanks collapse, control characters and invalid UTF-8 drop, and
// the text is cut to maxAskSenderRunes on a character boundary.
func repairSender(suggested string) string {
	kept := strings.Map(func(r rune) rune {
		if r == utf8.RuneError || unicode.IsControl(r) && !unicode.IsSpace(r) {
			return -1
		}
		return r
	}, suggested)
	text := strings.Join(strings.Fields(kept), " ")
	if runes := []rune(text); len(runes) > maxAskSenderRunes {
		text = strings.TrimSpace(string(runes[:maxAskSenderRunes]))
	}
	return text
}

// SenderSource says where an ask's sender came from, so a dry run can tell the
// requester who the operator will see and why.
type SenderSource string

const (
	// SenderSourceDeclared is the request's own sender.
	SenderSourceDeclared SenderSource = "declared"
	// SenderSourceDefault is the request's default_sender, on a new ask.
	SenderSourceDefault SenderSource = "default"
	// SenderSourceKept is the name an open ask keeps through a revision or replacement.
	SenderSourceKept SenderSource = "kept"
	// SenderSourceService is a verified service's name.
	SenderSourceService SenderSource = "service"
	// SenderSourceAgentRun is "Agent run" for a verified run that named nothing.
	SenderSourceAgentRun SenderSource = "agent_run"
	// SenderSourceUnnamed is "Unnamed sender".
	SenderSourceUnnamed SenderSource = "unnamed"
)

// DeriveSender is the sender an ask is stored with when nothing else names it:
// the one declared, else the verified service's name (an ask raised as an event
// or sent by a service), else the kind of caller.
//
// displayName is the name the Hub has registered for a source scenario (the one
// its Sources screen shows); nil, or an empty answer, falls back to the
// scenario's name set in words.
func DeriveSender(declared, source, sourceEventType, createdBy, recipient string, displayName func(string) string) string {
	name, _ := deriveSenderSource(declared, source, sourceEventType, createdBy, recipient, displayName)
	return name
}

func deriveSenderSource(declared, source, sourceEventType, createdBy, recipient string, displayName func(string) string) (string, SenderSource) {
	if text, err := normalizeSender(declared); err == nil && text != "" {
		return text, SenderSourceDeclared
	}
	if source != recipient && (sourceEventType != "" || strings.HasPrefix(createdBy, serviceCreatorPrefix)) {
		if displayName != nil {
			if name, err := normalizeSender(displayName(source)); err == nil && name != "" {
				return name, SenderSourceService
			}
		}
		if name := humanizeSource(source); name != "" {
			return name, SenderSourceService
		}
	}
	if strings.HasPrefix(createdBy, "run:") {
		return SenderAgentRun, SenderSourceAgentRun
	}
	return SenderUnnamed, SenderSourceUnnamed
}

// serviceCreatorPrefix marks an ask a verified service sent directly.
const serviceCreatorPrefix = "service:"

// humanizeSource turns a scenario name into words: research-desk becomes
// "Research Desk".
func humanizeSource(source string) string {
	words := strings.FieldsFunc(source, func(r rune) bool { return r == '-' || r == '_' || r == '.' || r == ' ' })
	for i, word := range words {
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

// deriveSender is DeriveSender with the Hub's registered source names.
func (s *Service) deriveSender(ctx context.Context, declared, source, sourceEventType, createdBy, recipient string) string {
	return DeriveSender(declared, source, sourceEventType, createdBy, recipient, func(name string) string { return s.resolveDisplayName(ctx, name) })
}

// senderOf is the one rule for the name an ask is shown under, used by every
// create, revise and dry run: the request's declared sender; else the name an
// open ask keeps (kept is empty for a new ask); else the request's default
// sender; else the Hub's own naming from what it verified.
//
// It reads nothing from the database: it runs inside the write transaction,
// where the Hub's single connection is taken, so the registered source name
// comes in on the spec (prepareAsk).
func senderOf(spec AskSpec, kept string) (string, SenderSource) {
	switch {
	case spec.Sender != "":
		return spec.Sender, SenderSourceDeclared
	case kept != "":
		return kept, SenderSourceKept
	case spec.DefaultSender != "":
		return spec.DefaultSender, SenderSourceDefault
	}
	return deriveSenderSource("", spec.Source, spec.SourceEventType, spec.CreatedBy, spec.Recipient, func(string) string { return spec.sourceName })
}

// backfillAskSenders names the asks stored before senders existed. It is
// idempotent: it reads only asks with no stored sender. Boot runs it before the
// source names are registered and again once they are (BackfillAskSenders).
func (s *Service) BackfillAskSenders(ctx context.Context) { s.backfillAskSenders(ctx) }

func (s *Service) backfillAskSenders(ctx context.Context) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT a.source, a.source_event_type, a.created_by, COALESCE((SELECT n.requested_by FROM notifications n WHERE n.id = a.notification_id), '') FROM asks a WHERE a.sender = ''`)
	if err != nil {
		s.log.Printf("backfill ask senders: %v", err)
		return
	}
	type shape struct{ source, eventType, createdBy, recipient string }
	var shapes []shape
	for rows.Next() {
		var next shape
		if err := rows.Scan(&next.source, &next.eventType, &next.createdBy, &next.recipient); err != nil {
			s.log.Printf("backfill ask senders: %v", err)
			_ = rows.Close()
			return
		}
		shapes = append(shapes, next)
	}
	_ = rows.Close()
	for _, next := range shapes {
		sender := s.deriveSender(ctx, "", next.source, next.eventType, next.createdBy, next.recipient)
		if _, err := s.db.ExecContext(ctx, `UPDATE asks SET sender = ? WHERE sender = '' AND source = ? AND source_event_type = ? AND created_by = ?`, sender, next.source, next.eventType, next.createdBy); err != nil {
			s.log.Printf("backfill ask senders: %v", err)
		}
	}
}
