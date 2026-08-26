package main

import (
	"context"
	"log"
	"sync"
	"time"
)

const conversationRetentionInterval = 15 * time.Minute

// conversationRetentionSweeper owns automatic pruning of semantic event
// history. It accepts the narrow optional capability so in-memory or remote
// repositories are not forced to implement a SQL-specific maintenance API.
type conversationRetentionSweeper struct {
	pruner        conversationEventPruner
	retentionDays func() int
	maxPerSession func() int
	interval      time.Duration
	stopCh        chan struct{}
	stopOnce      sync.Once
	wg            sync.WaitGroup
}

func newConversationRetentionSweeper(
	pruner conversationEventPruner,
	retentionDays func() int,
	maxPerSession func() int,
) *conversationRetentionSweeper {
	return &conversationRetentionSweeper{
		pruner:        pruner,
		retentionDays: retentionDays,
		maxPerSession: maxPerSession,
		interval:      conversationRetentionInterval,
		stopCh:        make(chan struct{}),
	}
}

func (s *conversationRetentionSweeper) Start() {
	if s == nil || s.pruner == nil {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.sweep()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *conversationRetentionSweeper) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() { close(s.stopCh) })
	s.wg.Wait()
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
	cutoff := time.Time{}
	if days > 0 {
		cutoff = time.Now().UTC().AddDate(0, 0, -days)
	}
	removed, err := s.pruner.PruneEvents(context.Background(), cutoff, maxPerSession)
	if err != nil {
		log.Printf("conversation-retention: sweep failed: %v", err)
		return 0
	}
	if removed > 0 {
		log.Printf("conversation-retention: removed %d event(s)", removed)
	}
	return removed
}
