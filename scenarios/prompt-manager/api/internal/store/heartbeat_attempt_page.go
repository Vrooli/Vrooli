package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	heartbeatAttemptEventPageSize = 100
	heartbeatAttemptEventScanCap  = 1000
)

var ErrHeartbeatAttemptCursor = errors.New("invalid heartbeat attempt cursor")

// HeartbeatAttemptPage describes the scanned recent event window. Total is
// exact only when TotalExact is true; otherwise it is a lower bound.
type HeartbeatAttemptPage struct {
	Attempts      []HeartbeatAttempt
	Total         int
	TotalExact    bool
	HasMore       bool
	ScannedEvents int
}

// ListHeartbeatAttemptPage returns the newest logical attempt facts from a
// bounded event window. Limit and offset page the known attempts in that
// window; they do not provide a stable cursor through all event history.
func (s *FileTeamStore) ListHeartbeatAttemptPage(ctx context.Context, teamID, agentID, status, profileKey string, limit, offset int) (*HeartbeatAttemptPage, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.ListHeartbeatAttemptPage(ctx, teamID, agentID, status, profileKey, limit, offset)
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	if s.ledger == nil {
		return s.listHeartbeatAttemptPageFromMemory(ctx, teamID, agentID, status, profileKey, limit, offset)
	}
	if strings.TrimSpace(s.eventsBase) == "" {
		return nil, fmt.Errorf("heartbeat attempt event endpoint unavailable")
	}
	return s.listHeartbeatAttemptPageFromEvents(ctx, teamID, agentID, status, profileKey, limit, offset)
}

func (s *FileTeamStore) listHeartbeatAttemptPageFromMemory(ctx context.Context, teamID, agentID, status, profileKey string, limit, offset int) (*HeartbeatAttemptPage, error) {
	teamIDs := []string{}
	if teamID != "" {
		teamIDs = append(teamIDs, teamID)
	} else {
		dirs, err := ListDirectories(s.teamsDir())
		if err != nil {
			return nil, fmt.Errorf("listing team directories: %w", err)
		}
		teamIDs = dirs
	}

	var entries []HeartbeatAttempt
	s.corpus.mu.Lock()
	for _, id := range teamIDs {
		for _, entry := range s.corpus.attempts[id] {
			if attemptMatches(entry, teamID, agentID, status, profileKey) {
				entries = append(entries, entry)
			}
		}
	}
	s.corpus.mu.Unlock()
	entries = mergeHeartbeatAttempts(entries, s.legacyPreRunAttempts(ctx, teamIDs, teamID, agentID, status, profileKey))
	sortHeartbeatAttempts(entries)
	return heartbeatAttemptSlice(entries, limit, offset, true, 0), nil
}

func (s *FileTeamStore) listHeartbeatAttemptPageFromEvents(ctx context.Context, teamID, agentID, status, profileKey string, limit, offset int) (*HeartbeatAttemptPage, error) {
	facts := make(map[string]heartbeatAttemptFact)
	order := make([]string, 0, heartbeatAttemptEventPageSize)
	before := ""
	scanned := 0
	complete := false

	for scanned < heartbeatAttemptEventScanCap {
		batchSize := heartbeatAttemptEventPageSize
		if remaining := heartbeatAttemptEventScanCap - scanned; batchSize > remaining {
			batchSize = remaining
		}
		events, err := s.queryHeartbeatAttemptEvents(ctx, "", before, batchSize)
		if err != nil {
			return nil, err
		}
		if len(events) == 0 {
			complete = true
			break
		}

		for _, event := range events {
			scanned++
			entry, occurredAt, err := heartbeatAttemptFromEvent(event)
			if err != nil {
				return nil, err
			}
			retainHeartbeatAttemptFact(facts, &order, entry, occurredAt)
		}
		if len(events) < batchSize {
			complete = true
			break
		}
		before = events[len(events)-1].GetEventId()
		if before == "" {
			return nil, fmt.Errorf("event history page omitted event_id cursor")
		}
	}

	entries := make([]HeartbeatAttempt, 0, len(order))
	for _, key := range order {
		entry := facts[key].entry
		if attemptMatches(entry, teamID, agentID, status, profileKey) {
			entries = append(entries, entry)
		}
	}

	var legacy []HeartbeatAttempt
	legacyTeamIDs := []string{teamID}
	if teamID == "" {
		var err error
		legacyTeamIDs, err = ListDirectories(s.teamsDir())
		if err != nil {
			return nil, fmt.Errorf("listing team directories for legacy heartbeat attempts: %w", err)
		}
	}
	legacy = s.legacyPreRunAttempts(ctx, legacyTeamIDs, teamID, agentID, status, profileKey)
	entries = mergeHeartbeatAttempts(entries, legacy)
	sortHeartbeatAttempts(entries)

	// A capped scan reports only the count observed in its newest event window.
	return heartbeatAttemptSlice(entries, limit, offset, complete, scanned), nil
}

// ListHeartbeatAttemptsByRun returns the lifecycle facts for one exact owner
// run. It uses Events' indexed agent_run_id filter rather than relying on a
// newest-attempt window, which is a diagnostic listing and may not contain old
// active-owner history.
func (s *FileTeamStore) ListHeartbeatAttemptsByRun(ctx context.Context, runID string) ([]HeartbeatAttempt, error) {
	if scoped := s.forContext(ctx); scoped != s {
		return scoped.ListHeartbeatAttemptsByRun(ctx, runID)
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, fmt.Errorf("run ID is required")
	}
	if s.ledger == nil {
		teamIDs, err := ListDirectories(s.teamsDir())
		if err != nil {
			return nil, fmt.Errorf("listing team directories for heartbeat attempt owner: %w", err)
		}
		var entries []HeartbeatAttempt
		s.corpus.mu.Lock()
		for _, teamID := range teamIDs {
			for _, entry := range s.corpus.attempts[teamID] {
				if entry.RunID == runID {
					entries = append(entries, entry)
				}
			}
		}
		s.corpus.mu.Unlock()
		return entries, nil
	}
	if strings.TrimSpace(s.eventsBase) == "" {
		return nil, fmt.Errorf("heartbeat attempt event endpoint unavailable")
	}

	const pageSize = 100
	entries := make([]HeartbeatAttempt, 0, 4)
	before := ""
	scanned := 0
	for scanned < heartbeatAttemptEventScanCap {
		batchSize := pageSize
		if remaining := heartbeatAttemptEventScanCap - scanned; batchSize > remaining {
			batchSize = remaining
		}
		events, err := s.queryHeartbeatAttemptEvents(ctx, runID, before, batchSize)
		if err != nil {
			return nil, err
		}
		if len(events) == 0 {
			return entries, nil
		}
		for _, event := range events {
			scanned++
			if event.GetCorrelation() == nil || event.GetCorrelation().GetAgentRunId() != runID {
				return nil, fmt.Errorf("heartbeat attempt query returned an event outside run %q", runID)
			}
			entry, _, err := heartbeatAttemptFromEvent(event)
			if err != nil {
				return nil, err
			}
			if entry.RunID != runID {
				return nil, fmt.Errorf("heartbeat attempt event %q has payload run_id %q for exact run %q", event.GetEventId(), entry.RunID, runID)
			}
			entries = append(entries, entry)
			before = event.GetEventId()
		}
		if len(events) < batchSize {
			return entries, nil
		}
		if before == "" {
			return nil, fmt.Errorf("event history page omitted event_id cursor")
		}
	}
	return nil, fmt.Errorf("heartbeat attempt owner history for run %q exceeded the %d-event safety cap", runID, heartbeatAttemptEventScanCap)
}

func (s *FileTeamStore) queryHeartbeatAttemptEvents(ctx context.Context, runID, before string, limit int) ([]*domain.EventEnvelope, error) {
	u, err := url.Parse(s.eventsBase + "/api/v1/events")
	if err != nil {
		return nil, fmt.Errorf("parse vrooli-events endpoint: %w", err)
	}
	query := u.Query()
	query.Set("type", heartbeatAttemptEventType)
	query.Set("source", "prompt-manager")
	query.Set("order", "desc")
	query.Set("limit", fmt.Sprint(limit))
	if runID != "" {
		query.Set("agent_run_id", runID)
	}
	if before != "" {
		query.Set("before_event_id", before)
	}
	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create heartbeat attempt query: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query heartbeat attempt events: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest && before != "" {
			return nil, fmt.Errorf("%w: event cursor is not retained", ErrHeartbeatAttemptCursor)
		}
		return nil, fmt.Errorf("query heartbeat attempt events: %s", resp.Status)
	}
	var raw []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode heartbeat attempt events: %w", err)
	}
	events := make([]*domain.EventEnvelope, 0, len(raw))
	for _, item := range raw {
		env := &domain.EventEnvelope{}
		if err := protojson.Unmarshal(item, env); err != nil {
			return nil, fmt.Errorf("decode heartbeat attempt event envelope: %w", err)
		}
		if env.GetData() == nil {
			return nil, fmt.Errorf("heartbeat attempt event %q has no typed payload", env.GetEventId())
		}
		events = append(events, env)
	}
	return events, nil
}

type heartbeatAttemptFact struct {
	entry      HeartbeatAttempt
	occurredAt time.Time
}

func retainHeartbeatAttemptFact(facts map[string]heartbeatAttemptFact, order *[]string, candidate HeartbeatAttempt, occurredAt time.Time) {
	key := candidate.TeamID + "\x00" + candidate.AgentID + "\x00" + candidate.ID
	current, exists := facts[key]
	if !exists {
		*order = append(*order, key)
	}
	if !exists || heartbeatAttemptFactLater(occurredAt, candidate, current.occurredAt, current.entry) {
		facts[key] = heartbeatAttemptFact{entry: candidate, occurredAt: occurredAt}
	}
}

func heartbeatAttemptFactLater(candidateAt time.Time, candidate HeartbeatAttempt, currentAt time.Time, current HeartbeatAttempt) bool {
	if !candidateAt.Equal(currentAt) {
		return candidateAt.After(currentAt)
	}
	if candidate.EndedAt != current.EndedAt {
		return candidate.EndedAt > current.EndedAt
	}
	return candidate.Phase == "run_terminal" && current.Phase != "run_terminal"
}

func heartbeatAttemptFromEvent(env *domain.EventEnvelope) (HeartbeatAttempt, time.Time, error) {
	payload := &structpb.Struct{}
	if env == nil || env.GetData() == nil {
		return HeartbeatAttempt{}, time.Time{}, fmt.Errorf("heartbeat attempt event has no typed payload")
	}
	if err := anypb.UnmarshalTo(env.GetData(), payload, proto.UnmarshalOptions{}); err != nil {
		return HeartbeatAttempt{}, time.Time{}, fmt.Errorf("decode heartbeat attempt event payload: %w", err)
	}
	body, err := json.Marshal(payload.AsMap())
	if err != nil {
		return HeartbeatAttempt{}, time.Time{}, fmt.Errorf("marshal heartbeat attempt event payload: %w", err)
	}
	var entry HeartbeatAttempt
	if err := json.Unmarshal(body, &entry); err != nil {
		return HeartbeatAttempt{}, time.Time{}, fmt.Errorf("decode heartbeat attempt projection: %w", err)
	}
	if entry.ID == "" {
		entry.ID = env.GetEventId()
	}
	if entry.ID == "" || entry.TeamID == "" || entry.AgentID == "" {
		return HeartbeatAttempt{}, time.Time{}, fmt.Errorf("heartbeat attempt event %q lacks logical identity", env.GetEventId())
	}
	occurredAt := time.Time{}
	if env.GetOccurredAt() != nil {
		occurredAt = env.GetOccurredAt().AsTime()
	}
	return entry, occurredAt, nil
}

func (s *FileTeamStore) legacyPreRunAttempts(ctx context.Context, teamIDs []string, teamID, agentID, status, profileKey string) []HeartbeatAttempt {
	var entries []HeartbeatAttempt
	for _, id := range teamIDs {
		configs, _ := s.ListHeartbeatConfigs(ctx, id)
		for _, config := range configs {
			if config.LastExecution == nil || config.LastExecution.RunID != "" {
				continue
			}
			entry := HeartbeatAttempt{
				ID:     "last-" + config.TeamID + "-" + config.AgentID + "-" + strings.ReplaceAll(config.LastExecution.StartedAt, ":", "-"),
				TeamID: config.TeamID, AgentID: config.AgentID, ProfileKey: config.ProfileKey,
				Status: config.LastExecution.Status, Phase: "pre_run_failure", StartedAt: config.LastExecution.StartedAt,
				EndedAt: config.LastExecution.EndedAt, ErrorCategory: "unknown", Error: config.LastExecution.Error,
				Recovery: "inspect_heartbeat_config",
			}
			if attemptMatches(entry, teamID, agentID, status, profileKey) {
				entries = append(entries, entry)
			}
		}
	}
	return entries
}

func attemptMatches(entry HeartbeatAttempt, teamID, agentID, status, profileKey string) bool {
	return (teamID == "" || entry.TeamID == teamID) && (agentID == "" || entry.AgentID == agentID) &&
		(status == "" || entry.Status == status) && (profileKey == "" || entry.ProfileKey == profileKey)
}

func mergeHeartbeatAttempts(primary, extra []HeartbeatAttempt) []HeartbeatAttempt {
	out := append([]HeartbeatAttempt(nil), primary...)
	keys := make(map[string]struct{}, len(primary))
	for _, entry := range primary {
		keys[entry.TeamID+"\x00"+entry.AgentID+"\x00"+entry.ID] = struct{}{}
	}
	for _, entry := range extra {
		duplicate := false
		for _, existing := range out {
			if existing.TeamID == entry.TeamID && existing.AgentID == entry.AgentID && existing.StartedAt == entry.StartedAt && existing.RunID == "" {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		key := entry.TeamID + "\x00" + entry.AgentID + "\x00" + entry.ID
		if _, exists := keys[key]; !exists {
			keys[key] = struct{}{}
			out = append(out, entry)
		}
	}
	return out
}

func sortHeartbeatAttempts(entries []HeartbeatAttempt) {
	sort.SliceStable(entries, func(i, j int) bool {
		left, right := entries[i].StartedAt, entries[j].StartedAt
		if entries[i].EndedAt != "" {
			left = entries[i].EndedAt
		}
		if entries[j].EndedAt != "" {
			right = entries[j].EndedAt
		}
		return left > right
	})
}

func heartbeatAttemptSlice(entries []HeartbeatAttempt, limit, offset int, complete bool, scanned int) *HeartbeatAttemptPage {
	total := len(entries)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	selected := append([]HeartbeatAttempt(nil), entries[offset:end]...)
	hasMore := end < total || !complete
	return &HeartbeatAttemptPage{Attempts: selected, Total: total, TotalExact: complete, HasMore: hasMore, ScannedEvents: scanned}
}
