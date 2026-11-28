package journal

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type HolderOwnership interface {
	Owns(context.Context, string, string) (bool, error)
}

type HolderHistory struct {
	Events   []Event
	Balances []Balance
}

// HolderHistoryRepository requires an authenticated subject for every read
// and checks ownership before querying events. Foreign and absent holders both
// return the same empty result, preventing this repository from becoming an
// existence oracle even if a future handler forgets its own check.
type HolderHistoryRepository struct {
	events    HolderEventReader
	ownership HolderOwnership
}

func NewHolderHistoryRepository(events HolderEventReader, ownership HolderOwnership) *HolderHistoryRepository {
	return &HolderHistoryRepository{events: events, ownership: ownership}
}

func (r *HolderHistoryRepository) Read(ctx context.Context, holderID, authenticatedSubject string) (HolderHistory, error) {
	holderID = strings.TrimSpace(holderID)
	authenticatedSubject = strings.TrimSpace(authenticatedSubject)
	if holderID == "" || authenticatedSubject == "" {
		return HolderHistory{}, fmt.Errorf("%w: holder id and authenticated subject are required", ErrInvalidJournalEvent)
	}
	if r.events == nil || r.ownership == nil {
		return HolderHistory{}, errorsUnavailable("holder history")
	}
	allowed, err := r.ownership.Owns(ctx, holderID, authenticatedSubject)
	if err != nil {
		return HolderHistory{}, fmt.Errorf("authorize holder history: %w", err)
	}
	if !allowed {
		return emptyHolderHistory(), nil
	}
	events, err := r.events.ReadHolder(ctx, holderID)
	if err != nil {
		return HolderHistory{}, err
	}
	grouped := make(map[string][]Event)
	for _, event := range events {
		grouped[event.TokenTypeID] = append(grouped[event.TokenTypeID], event)
	}
	tokenTypeIDs := make([]string, 0, len(grouped))
	for tokenTypeID := range grouped {
		tokenTypeIDs = append(tokenTypeIDs, tokenTypeID)
	}
	sort.Strings(tokenTypeIDs)
	balances := make([]Balance, 0, len(tokenTypeIDs))
	for _, tokenTypeID := range tokenTypeIDs {
		amount, projectErr := projectEvents(grouped[tokenTypeID])
		if projectErr != nil {
			return HolderHistory{}, projectErr
		}
		balances = append(balances, Balance{HolderID: holderID, TokenTypeID: tokenTypeID, Amount: amount})
	}
	return HolderHistory{Events: events, Balances: balances}, nil
}

func emptyHolderHistory() HolderHistory {
	return HolderHistory{Events: []Event{}, Balances: []Balance{}}
}

func errorsUnavailable(name string) error { return fmt.Errorf("%s unavailable", name) }
