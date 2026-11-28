package hub

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrDecisionRevisionConflict = errors.New("pending decision revision changed")

// PendingDecisionReference is the allowlisted owner projection. It deliberately
// excludes question/answer text, answerer, notes, context, requester/correlation
// data, attachments and scenario-specific payloads.
type PendingDecisionReference struct {
	DecisionRef     string `json:"decision_ref"`
	OwnerID         string `json:"owner_id"`
	Source          string `json:"source"`
	SourceEventType string `json:"source_event_type"`
	Revision        string `json:"revision"`
	State           string `json:"state"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	Deadline        string `json:"deadline,omitempty"`
	Urgency         string `json:"urgency,omitempty"`
}

type PendingDecisionReferencePage struct {
	Decisions  []PendingDecisionReference `json:"decisions"`
	NextCursor string                     `json:"next_cursor,omitempty"`
}

// ListPendingDecisionReferences returns only pending, event-sourced decisions
// addressed to recipient. Hub's signed keyset cursor supplies bounded, stable
// pagination; answer-bearing asks are not copied into the projection.
func (s *Service) ListPendingDecisionReferences(ctx context.Context, recipient, cursor string, limit int) (PendingDecisionReferencePage, error) {
	page, err := s.ListAsksPage(ctx, recipient, AskListOptions{
		Limit: limit, Cursor: cursor, StateFilter: AskPending, Order: AskOrderNewest,
		PendingDecisionOnly: true,
	})
	if err != nil {
		return PendingDecisionReferencePage{}, err
	}
	out := PendingDecisionReferencePage{Decisions: make([]PendingDecisionReference, 0, len(page.Asks)), NextCursor: page.NextCursor}
	for _, ask := range page.Asks {
		out.Decisions = append(out.Decisions, pendingDecisionReference(ask, recipient))
	}
	return out, nil
}

// ReadPendingDecisionReference returns one current pending decision reference.
// A resolved, withdrawn, direct, status or cross-recipient ask is indistinguishable
// from an unknown reference. The expected revision prevents reading a silently
// revised decision through an older registry descriptor.
func (s *Service) ReadPendingDecisionReference(ctx context.Context, recipient, decisionRef, expectedRevision string) (PendingDecisionReference, error) {
	recipient = strings.TrimSpace(recipient)
	decisionRef = strings.TrimSpace(decisionRef)
	expectedRevision = strings.TrimSpace(expectedRevision)
	if recipient == "" || decisionRef == "" || expectedRevision == "" {
		return PendingDecisionReference{}, ErrNotFound
	}
	ask, err := s.GetAskForRecipient(ctx, decisionRef, recipient)
	if err != nil || !isPendingDecision(ask) {
		return PendingDecisionReference{}, ErrNotFound
	}
	current := pendingDecisionReference(ask, recipient)
	if current.Revision != expectedRevision {
		return PendingDecisionReference{}, ErrDecisionRevisionConflict
	}
	return current, nil
}

func isPendingDecision(ask AskRecord) bool {
	return ask.ID != "" && ask.State == AskPending && ask.Source != "" &&
		strings.HasSuffix(ask.SourceEventType, ".decision_requested.v1") && ask.Kind != "status"
}

func pendingDecisionReference(ask AskRecord, owner string) PendingDecisionReference {
	return PendingDecisionReference{
		DecisionRef: ask.ID, OwnerID: owner, Source: ask.Source,
		SourceEventType: ask.SourceEventType, Revision: strconv.Itoa(ask.RevisionCount),
		State: ask.State, CreatedAt: ask.CreatedAt, UpdatedAt: ask.UpdatedAt,
		Deadline: ask.Deadline, Urgency: ask.Urgency,
	}
}

// ValidatePendingDecisionReference protects handler mappings from accidentally
// serializing an incomplete or non-pending read-model row.
func ValidatePendingDecisionReference(value PendingDecisionReference) error {
	if value.DecisionRef == "" || value.OwnerID == "" || value.Source == "" ||
		!strings.HasSuffix(value.SourceEventType, ".decision_requested.v1") ||
		value.Revision == "" || value.State != AskPending || value.CreatedAt == "" || value.UpdatedAt == "" {
		return fmt.Errorf("pending decision reference is incomplete")
	}
	return nil
}
