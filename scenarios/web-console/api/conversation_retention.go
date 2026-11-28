package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"web-console/internal/continuity"
)

const conversationRetentionInterval = 15 * time.Minute

// conversationRetentionSweeper owns automatic pruning of semantic event
// history. It accepts the narrow optional capability so in-memory or remote
// repositories are not forced to implement a SQL-specific maintenance API.
type conversationRetentionSweeper struct {
	pruner        conversationEventPruner
	retentionDays func() int
	maxPerSession func() int
	ledger        continuity.Ledger
	maintain      func(context.Context)
	interval      time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	startOnce     sync.Once
	done          chan struct{}
}

func newConversationRetentionSweeper(
	pruner conversationEventPruner,
	retentionDays func() int,
	maxPerSession func() int,
	ledgers ...continuity.Ledger,
) *conversationRetentionSweeper {
	var ledger continuity.Ledger
	if len(ledgers) > 0 {
		ledger = ledgers[0]
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &conversationRetentionSweeper{
		pruner:        pruner,
		retentionDays: retentionDays,
		maxPerSession: maxPerSession,
		ledger:        ledger,
		interval:      conversationRetentionInterval,
		ctx:           ctx,
		cancel:        cancel,
		done:          make(chan struct{}),
	}
}

func (s *conversationRetentionSweeper) Start() {
	if s == nil || s.pruner == nil {
		return
	}
	s.startOnce.Do(func() {
		go func() {
			defer close(s.done)
			ticker := time.NewTicker(s.interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if s.ctx.Err() != nil {
						return
					}
					s.sweep()
					if s.maintain != nil && s.ctx.Err() == nil {
						s.maintain(s.ctx)
					}
				case <-s.ctx.Done():
					return
				}
			}
		}()
	})
}

func (s *conversationRetentionSweeper) Stop() {
	if s == nil {
		return
	}
	s.cancel()
	s.startOnce.Do(func() { close(s.done) })
	<-s.done
}

func (s *conversationRetentionSweeper) sweep() int64 {
	if s == nil || s.pruner == nil {
		return 0
	}
	days := 0
	if s.retentionDays != nil {
		days = s.retentionDays()
	}
	maxPerSession := 0
	if s.maxPerSession != nil {
		maxPerSession = s.maxPerSession()
	}
	if days <= 0 && maxPerSession <= 0 {
		return 0
	}
	if s.ledger == nil {
		log.Printf("conversation-retention: sweep skipped; continuity receipt ledger is unavailable")
		return 0
	}
	cutoff := time.Time{}
	if days > 0 {
		cutoff = time.Now().UTC().AddDate(0, 0, -days)
	}
	ctx := context.Background()
	operationID := fmt.Sprintf("web-console:conversation-retention:%d", time.Now().UTC().UnixNano())
	if _, err := s.ledger.Put(ctx, continuity.Receipt{
		OperationID: operationID,
		SessionID:   "conversation-retention",
		ActorKind:   "system",
		ActorID:     "conversation-retention",
		Command:     "retention",
		FromState:   continuity.StateLive,
		ToState:     continuity.StateLive,
		ReasonCode:  "configured_retention_policy",
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		log.Printf("conversation-retention: sweep skipped; create receipt: %v", err)
		return 0
	}
	removed, err := s.pruner.PruneEvents(ctx, cutoff, maxPerSession)
	if err != nil {
		if _, completeErr := s.ledger.Complete(ctx, operationID, "failed", "retention_prune_failed", time.Now().UTC()); completeErr != nil {
			log.Printf("conversation-retention: complete failed receipt: %v", completeErr)
		}
		log.Printf("conversation-retention: sweep failed: %v", err)
		return 0
	}
	if _, err := s.ledger.Complete(ctx, operationID, "succeeded", "", time.Now().UTC()); err != nil {
		log.Printf("conversation-retention: complete receipt: %v", err)
		return 0
	}
	if removed > 0 {
		log.Printf("conversation-retention: removed %d event(s)", removed)
	}
	return removed
}
