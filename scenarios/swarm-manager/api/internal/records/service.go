package records

import (
	"context"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/idgen"
)

// Indexer is the semantic-search write seam. Production wiring upserts the
// record into the swarm-manager Qdrant collection with an entity_type:record
// discriminator; tests can substitute a fake.
//
// seam: records.Indexer
type Indexer interface {
	IndexRecord(ctx context.Context, r Record) error
}

// noopIndexer is used when no aisearch wiring is available (dev-mode
// degradation, not test substitution).
type noopIndexer struct{}

func (noopIndexer) IndexRecord(ctx context.Context, r Record) error { return nil }

// EventLogger emits typed events for records lifecycle transitions. The
// real implementation is eventlog.Emitter; tests can pass a fake.
//
// seam: records.EventLogger
type EventLogger interface {
	EmitRecordCreated(recordID, kind, scenario, backlogRef string, stub bool)
	EmitRecordSuperseded(recordID, supersededID, reason string)
}

type noopLogger struct{}

func (noopLogger) EmitRecordCreated(string, string, string, string, bool) {}
func (noopLogger) EmitRecordSuperseded(string, string, string)            {}

// Service is the records business-logic layer. It owns id generation,
// stub-vs-filled invariants, supersede cycle detection, and indexing
// coordination.
type Service struct {
	store   Store
	indexer Indexer
	events  EventLogger
	now     func() time.Time
}

// SetEventLogger rewires the event-log emitter post-construction. main.go
// uses this from wireEventLoggers after the eventlog.Emitter is built. Passing
// nil reverts to no-op.
func (s *Service) SetEventLogger(events EventLogger) {
	if events == nil {
		events = noopLogger{}
	}
	s.events = events
}

// SetIndexer rewires the semantic-search indexer post-construction. main.go
// uses this to install the aisearch-backed indexer after the aisearch service
// is constructed (which happens after records, because aisearch is registered
// last so its readers see fully-wired stores). Passing nil reverts to no-op.
func (s *Service) SetIndexer(indexer Indexer) {
	if indexer == nil {
		indexer = noopIndexer{}
	}
	s.indexer = indexer
}

// NewService constructs a Service. Pass nil for indexer/events to no-op.
func NewService(store Store, indexer Indexer, events EventLogger) *Service {
	if indexer == nil {
		indexer = noopIndexer{}
	}
	if events == nil {
		events = noopLogger{}
	}
	return &Service{
		store:   store,
		indexer: indexer,
		events:  events,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// CreateInput is the request payload for full record creation.
type CreateInput struct {
	Kind         RecordKind
	Scenario     string
	BacklogRef   string
	Supersedes   string
	Trigger      string
	Approach     string
	RuledOut     []string
	Commit       string
	FilesChanged []string
	Outcome      Outcome
	CreatedBy    string
}

// Create writes a full (non-stub) record. Required: Kind, Scenario, Outcome,
// and at least one of Trigger / Approach / RuledOut.
func (s *Service) Create(ctx context.Context, in CreateInput) (Record, error) {
	if _, err := ParseKind(string(in.Kind)); err != nil {
		return Record{}, err
	}
	if in.Scenario = strings.TrimSpace(in.Scenario); in.Scenario == "" {
		return Record{}, fmt.Errorf("scenario is required")
	}
	if _, err := ParseOutcome(string(in.Outcome)); err != nil {
		return Record{}, err
	}
	r := Record{
		ID:           "rec-" + idgen.Generate(),
		Kind:         in.Kind,
		Scenario:     in.Scenario,
		BacklogRef:   strings.TrimSpace(in.BacklogRef),
		Supersedes:   strings.TrimSpace(in.Supersedes),
		Trigger:      strings.TrimSpace(in.Trigger),
		Approach:     strings.TrimSpace(in.Approach),
		RuledOut:     trimAll(in.RuledOut),
		Commit:       strings.TrimSpace(in.Commit),
		FilesChanged: trimAll(in.FilesChanged),
		Outcome:      in.Outcome,
		Stub:         false,
		CreatedAt:    s.now(),
		CreatedBy:    strings.TrimSpace(in.CreatedBy),
	}
	r.NarrativeAt = r.CreatedAt
	if !r.hasNarrative() {
		return Record{}, fmt.Errorf("record must include at least one of trigger, approach, or ruled_out")
	}
	if r.Supersedes != "" {
		if err := s.assertNoSupersedeCycle(r.ID, r.Supersedes); err != nil {
			return Record{}, err
		}
		if _, err := s.store.SetSupersededBy(r.Supersedes, r.ID); err != nil {
			return Record{}, fmt.Errorf("link supersedes: %w", err)
		}
		s.events.EmitRecordSuperseded(r.ID, r.Supersedes, "")
	}
	if err := s.store.Create(r); err != nil {
		return Record{}, err
	}
	_ = s.indexer.IndexRecord(ctx, r)
	s.events.EmitRecordCreated(r.ID, string(r.Kind), r.Scenario, r.BacklogRef, false)
	return r, nil
}

// CreateStubInput is the minimal payload to auto-create a stub on backlog
// terminal transitions.
type CreateStubInput struct {
	Kind       RecordKind
	Scenario   string
	BacklogRef string
	Outcome    Outcome
	CreatedBy  string
}

// CreateStub writes a thin record with empty narrative fields. Callers
// (typically the backlog terminal-status hook) surface the returned ID
// and prompt the agent to fill via UpdateNarrative.
func (s *Service) CreateStub(ctx context.Context, in CreateStubInput) (Record, error) {
	if _, err := ParseKind(string(in.Kind)); err != nil {
		return Record{}, err
	}
	if in.Scenario = strings.TrimSpace(in.Scenario); in.Scenario == "" {
		return Record{}, fmt.Errorf("scenario is required")
	}
	if _, err := ParseOutcome(string(in.Outcome)); err != nil {
		return Record{}, err
	}
	r := Record{
		ID:         "rec-" + idgen.Generate(),
		Kind:       in.Kind,
		Scenario:   in.Scenario,
		BacklogRef: strings.TrimSpace(in.BacklogRef),
		Outcome:    in.Outcome,
		Stub:       true,
		CreatedAt:  s.now(),
		CreatedBy:  strings.TrimSpace(in.CreatedBy),
	}
	if err := s.store.Create(r); err != nil {
		return Record{}, err
	}
	s.events.EmitRecordCreated(r.ID, string(r.Kind), r.Scenario, r.BacklogRef, true)
	// Stubs are not indexed: empty embedding text would pollute search.
	return r, nil
}

// UpdateNarrative fills a stub record exactly once. Once filled, further
// changes require Supersede.
func (s *Service) UpdateNarrative(ctx context.Context, id string, n Narrative) (Record, error) {
	if n.Outcome != "" {
		if _, err := ParseOutcome(string(n.Outcome)); err != nil {
			return Record{}, err
		}
	}
	r, err := s.store.UpdateNarrative(id, n, s.now())
	if err != nil {
		return Record{}, err
	}
	_ = s.indexer.IndexRecord(ctx, r)
	return r, nil
}

// Get fetches a record by id.
func (s *Service) Get(id string) (Record, error) { return s.store.Get(id) }

// List returns records matching filter.
func (s *Service) List(filter ListFilter) ([]Record, error) { return s.store.List(filter) }

// Supersede marks `id` as superseded by `successorID`. The successor must
// already exist. Used when the user creates a successor out-of-band
// (e.g. via the supersede CLI subcommand pointing at an existing record).
func (s *Service) Supersede(ctx context.Context, id, successorID, reason string) (Record, error) {
	if id == successorID {
		return Record{}, ErrSupersedeCycle
	}
	if _, err := s.store.Get(successorID); err != nil {
		return Record{}, fmt.Errorf("successor %s: %w", successorID, err)
	}
	if err := s.assertNoSupersedeCycle(successorID, id); err != nil {
		return Record{}, err
	}
	r, err := s.store.SetSupersededBy(id, successorID)
	if err != nil {
		return Record{}, err
	}
	s.events.EmitRecordSuperseded(successorID, id, reason)
	return r, nil
}

// assertNoSupersedeCycle walks the existing supersede chain from `start`
// (a record id) and rejects if `forbidden` appears anywhere in the chain.
// This catches A->B->A and longer cycles before a write commits.
func (s *Service) assertNoSupersedeCycle(forbidden, start string) error {
	seen := map[string]struct{}{}
	cur := start
	for i := 0; i < 64 && cur != ""; i++ {
		if cur == forbidden {
			return ErrSupersedeCycle
		}
		if _, ok := seen[cur]; ok {
			return ErrSupersedeCycle
		}
		seen[cur] = struct{}{}
		r, err := s.store.Get(cur)
		if err != nil {
			return nil // chain ends at a missing record; not a cycle
		}
		cur = r.Supersedes
	}
	return nil
}
