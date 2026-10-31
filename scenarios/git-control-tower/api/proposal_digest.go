package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"git-control-tower/internal/proposals"

	"github.com/vrooli/api-core/eventbus"
)

// The waiting-proposal digest tells the operator, at most once per UTC day,
// which commit proposals wait for review. It is a Vrooli Events domain event
// that Notification Hub turns into a phone notification when its
// notification pattern includes it. Off unless GCT_PROPOSAL_DIGEST_ENABLED
// is true.
const (
	proposalDigestEnv        = "GCT_PROPOSAL_DIGEST_ENABLED"
	proposalDigestMinAgeEnv  = "GCT_PROPOSAL_DIGEST_MIN_AGE"
	proposalDigestEventType  = "git_control_tower.proposals.waiting.v1"
	proposalDigestCheckID    = "gct-proposals-waiting"
	proposalDigestSource     = "git-control-tower"
	proposalDigestInterval   = time.Hour
	proposalDigestMinAge     = 12 * time.Hour
	proposalDigestMaxEntries = 12
)

type domainEventPublisher interface {
	PublishDomainEvent(ctx context.Context, event eventbus.DomainEvent) error
}

func proposalDigestEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(proposalDigestEnv))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func proposalDigestMinAgeFromEnv() time.Duration {
	if raw := strings.TrimSpace(os.Getenv(proposalDigestMinAgeEnv)); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed >= 0 {
			return parsed
		}
		log.Printf("warn: ignoring %s=%q; using %s", proposalDigestMinAgeEnv, raw, proposalDigestMinAge)
	}
	return proposalDigestMinAge
}

func (s *Server) startProposalDigest() {
	if !proposalDigestEnabled() || s.proposalService == nil || s.repos == nil {
		return
	}
	minAge := proposalDigestMinAgeFromEnv()
	publisher := eventbus.NewDiscoveredClient(context.Background())
	go func() {
		send := func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if _, err := s.sendProposalDigest(ctx, publisher, minAge, time.Now().UTC()); err != nil {
				log.Printf("warn: proposal digest: %v", err)
			}
		}
		send()
		ticker := time.NewTicker(proposalDigestInterval)
		defer ticker.Stop()
		for range ticker.C {
			send()
		}
	}()
}

// sendProposalDigest publishes one digest for the UTC day of now when
// proposals have waited at least minAge and no digest went out that day. It
// reports whether it published.
func (s *Server) sendProposalDigest(ctx context.Context, publisher domainEventPublisher, minAge time.Duration, now time.Time) (bool, error) {
	day := now.UTC().Format("2006-01-02")
	sent, err := s.proposalService.DigestSent(ctx, day)
	if err != nil || sent {
		return false, err
	}
	repos, _, err := s.repos.List(ctx)
	if err != nil {
		return false, fmt.Errorf("list repositories: %w", err)
	}
	ids := make([]string, 0, len(repos))
	for i := range repos {
		ids = append(ids, repositoryIDFor(&repos[i]))
	}
	waiting, err := s.proposalService.Waiting(ctx, ids, now.Add(-minAge))
	if err != nil || len(waiting) == 0 {
		return false, err
	}
	// The envelope time is the start of the day, so a repeated publish of
	// the same facts has the same event ID and Vrooli Events drops it.
	occurred, _ := time.Parse("2006-01-02", day)
	if err := publisher.PublishDomainEvent(ctx, eventbus.DomainEvent{
		Source: proposalDigestSource, EventType: proposalDigestEventType, Payload: proposalDigestFacts(waiting, now), Occurred: occurred,
	}); err != nil {
		return false, fmt.Errorf("publish: %w", err)
	}
	return true, s.proposalService.RecordDigest(ctx, day, len(waiting))
}

// proposalDigestFacts are the event facts. Notification Hub owns the wording;
// message is the fact it shows as the notification body.
func proposalDigestFacts(waiting []proposals.WaitingItem, now time.Time) map[string]any {
	byEffort := map[string][]string{}
	var efforts []string
	ids := make([]any, 0, len(waiting))
	for _, item := range waiting {
		effort := strings.TrimPrefix(item.EffortRef, "effort:")
		if effort == "" {
			effort = "no effort"
		}
		if _, ok := byEffort[effort]; !ok {
			efforts = append(efforts, effort)
		}
		label := item.Epoch
		if label == "" {
			label = item.ID
		}
		byEffort[effort] = append(byEffort[effort], label)
		if len(ids) < proposalDigestMaxEntries {
			ids = append(ids, item.ID)
		}
	}
	sort.Strings(efforts)
	parts := make([]string, 0, len(efforts))
	for _, effort := range efforts {
		parts = append(parts, effort+": "+strings.Join(byEffort[effort], ", "))
	}
	oldestHours := int(now.Sub(waiting[0].CreatedAt).Hours())
	noun := "commit proposals wait"
	if len(waiting) == 1 {
		noun = "commit proposal waits"
	}
	message := fmt.Sprintf("%d %s for review in Git Control Tower (oldest %dh). %s.", len(waiting), noun, oldestHours, strings.Join(parts, "; "))
	return map[string]any{
		"check_id":     proposalDigestCheckID,
		"severity":     "informational",
		"message":      message,
		"count":        float64(len(waiting)),
		"oldest_hours": float64(oldestHours),
		"proposal_ids": ids,
	}
}
