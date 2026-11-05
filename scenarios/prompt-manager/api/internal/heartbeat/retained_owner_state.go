package heartbeat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"prompt-manager/internal/store"
)

// RetainedOwnerStateFact is a factual snapshot of a queue obligation that
// remained unresolved after the existing startup reconciliation pass. It is
// intentionally separate from dispatch-attempt history and carries no action
// or authority grant.
type RetainedOwnerStateFact struct {
	TeamID            string
	AgentID           string
	OwnerObligationID string
	State             string
	StartedAt         string
	ObservedAt        string
	TaskID            string
	RunID             string
	ReviewURL         string
}

type retainedOwnerStatePublisher interface {
	TeamReviewURLFor(ctx context.Context, teamID string) string
	PublishRetainedOwnerState(ctx context.Context, fact RetainedOwnerStateFact) error
}

func isRetainedOwnerReviewState(state string) bool {
	switch state {
	case ObligationDispatchUncertain, ObligationOwnerUnreachable, ObligationNeedsReview, ObligationWatcherUnavailable, ObligationPreparationInterrupted:
		return true
	default:
		return false
	}
}

func heartbeatOwnerObligationID(teamID, agentID, idempotencyKey, runID, startedAt string) string {
	seed := strings.TrimSpace(idempotencyKey)
	if seed == "" {
		seed = strings.TrimSpace(runID)
	}
	if seed == "" {
		seed = strings.TrimSpace(startedAt)
	}
	if seed == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(strings.TrimSpace(teamID) + "\x00" + strings.TrimSpace(agentID) + "\x00" + seed))
	return "pm-heartbeat-owner-" + hex.EncodeToString(digest[:])
}

func cloneRetainedOwnerStateSnapshots(input map[string]retainedOwnerStateSnapshot) map[string]retainedOwnerStateSnapshot {
	if len(input) == 0 {
		return nil
	}
	copyOf := make(map[string]retainedOwnerStateSnapshot, len(input))
	for state, snapshot := range input {
		copyOf[state] = snapshot
	}
	return copyOf
}

// TeamReviewURLFor resolves the existing trusted PM team view. It is called
// only when a retained-state snapshot is first persisted; later publication
// retries reuse the frozen URL.
func (e *Executor) TeamReviewURLFor(ctx context.Context, teamID string) string {
	if e == nil {
		return ""
	}
	e.mu.Lock()
	resolver := e.TeamReviewURL
	e.mu.Unlock()
	if resolver == nil {
		return ""
	}
	return strings.TrimSpace(resolver(ctx, teamID))
}

// PublishRetainedOwnerState publishes one owner-state lifecycle fact through
// the same durable event publisher as heartbeat attempts. ObservedAt is the
// persisted time PM first observed this retained state, distinct from the
// original StartedAt (which may be old or unavailable).
func (e *Executor) PublishRetainedOwnerState(ctx context.Context, fact RetainedOwnerStateFact) error {
	if e == nil || e.teamStore == nil {
		return fmt.Errorf("heartbeat team store unavailable")
	}
	if strings.TrimSpace(fact.OwnerObligationID) == "" || strings.TrimSpace(fact.State) == "" || strings.TrimSpace(fact.ObservedAt) == "" {
		return fmt.Errorf("retained heartbeat owner-state fact is missing stable identity, state, or observation time")
	}
	entry := &store.HeartbeatAttempt{
		ID:                 fact.OwnerObligationID,
		TeamID:             fact.TeamID,
		AgentID:            fact.AgentID,
		TaskID:             fact.TaskID,
		RunID:              fact.RunID,
		Status:             fact.State,
		Phase:              "retained_owner_state:" + fact.State,
		StartedAt:          strings.TrimSpace(fact.StartedAt),
		OwnerObligationID:  fact.OwnerObligationID,
		RetainedOwnerState: fact.State,
		ObservedAt:         fact.ObservedAt,
		ReviewURL:          strings.TrimSpace(fact.ReviewURL),
	}
	return e.teamStore.AppendHeartbeatAttempt(ctx, fact.TeamID, entry)
}

// publishRetainedOwnerStateFacts emits one immutable FYI fact for each current
// retained review state after Recover has persisted and attempted its existing
// reconciliation. Queue snapshots freeze the first observation per state so
// repeated process starts recapture exact event terms.
func (c *TeamExecutionContext) publishRetainedOwnerStateFacts(ctx context.Context) {
	reporter, ok := c.executor.(retainedOwnerStatePublisher)
	if !ok || reporter == nil {
		return
	}
	c.mu.Lock()
	agents := make([]string, 0, len(c.running))
	for agentID, entry := range c.running {
		if isRetainedOwnerReviewState(entry.State) {
			agents = append(agents, agentID)
		}
	}
	c.mu.Unlock()

	for _, agentID := range agents {
		c.mu.Lock()
		entry, exists := c.running[agentID]
		if !exists || !isRetainedOwnerReviewState(entry.State) {
			c.mu.Unlock()
			continue
		}
		state := entry.State
		if snapshot, hasSnapshot := entry.OwnerStateSnapshots[state]; hasSnapshot {
			fact := RetainedOwnerStateFact{
				TeamID: c.teamID, AgentID: agentID, OwnerObligationID: snapshot.OwnerObligationID,
				State: snapshot.State, StartedAt: snapshot.StartedAt, ObservedAt: snapshot.ObservedAt,
				TaskID: snapshot.TaskID, RunID: snapshot.RunID, ReviewURL: snapshot.ReviewURL,
			}
			c.mu.Unlock()
			if err := reporter.PublishRetainedOwnerState(ctx, fact); err != nil {
				log.Printf("team_execution: retained owner-state notification capture failed for %s/%s", c.teamID, agentID)
			}
			continue
		}
		c.mu.Unlock()

		startedAt := strings.TrimSpace(entry.OwnerStartedAt)
		ownerID := strings.TrimSpace(entry.OwnerObligationID)
		if ownerID == "" {
			ownerID = heartbeatOwnerObligationID(c.teamID, agentID, entry.IdempotencyKey, entry.RunID, startedAt)
		}
		if ownerID == "" {
			// A legacy row without intent, run identity, or historical start time
			// receives an opaque epoch, not a fabricated historical timestamp.
			ownerID = "pm-heartbeat-owner-legacy-" + uuid.NewString()
		}
		fact := RetainedOwnerStateFact{
			TeamID: c.teamID, AgentID: agentID, OwnerObligationID: ownerID,
			State: state, StartedAt: startedAt,
			ObservedAt: time.Now().UTC().Format(time.RFC3339Nano),
			TaskID:     entry.TaskID, RunID: entry.RunID,
			ReviewURL: reporter.TeamReviewURLFor(ctx, c.teamID),
		}
		snapshot := retainedOwnerStateSnapshot{
			OwnerObligationID: fact.OwnerObligationID, State: fact.State, StartedAt: fact.StartedAt,
			ObservedAt: fact.ObservedAt, TaskID: fact.TaskID, RunID: fact.RunID, ReviewURL: fact.ReviewURL,
		}

		c.mu.Lock()
		current, stillOwned := c.running[agentID]
		if !stillOwned || current.State != state {
			c.mu.Unlock()
			continue
		}
		if current.OwnerStateSnapshots == nil {
			current.OwnerStateSnapshots = make(map[string]retainedOwnerStateSnapshot)
		}
		if existing, found := current.OwnerStateSnapshots[state]; found {
			snapshot = existing
		} else {
			previous := current
			current.OwnerObligationID = ownerID
			current.OwnerStartedAt = startedAt
			current.OwnerStateSnapshots[state] = snapshot
			c.running[agentID] = current
			if err := c.persistLocked(); err != nil {
				c.running[agentID] = previous
				c.mu.Unlock()
				log.Printf("team_execution: retained owner-state snapshot persistence failed for %s/%s", c.teamID, agentID)
				continue
			}
		}
		c.mu.Unlock()
		fact = RetainedOwnerStateFact{
			TeamID: c.teamID, AgentID: agentID, OwnerObligationID: snapshot.OwnerObligationID,
			State: snapshot.State, StartedAt: snapshot.StartedAt, ObservedAt: snapshot.ObservedAt,
			TaskID: snapshot.TaskID, RunID: snapshot.RunID, ReviewURL: snapshot.ReviewURL,
		}
		if err := reporter.PublishRetainedOwnerState(ctx, fact); err != nil {
			log.Printf("team_execution: retained owner-state notification capture failed for %s/%s", c.teamID, agentID)
		}
	}
}
