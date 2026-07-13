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
	InitiativeID string
	Supersedes   string
	Trigger      string
	Approach     string
	RuledOut     []string
	Evidence     string
	Commit       string
	FilesChanged []string
	Outcome      Outcome
	CreatedBy    string
}

// CaptureInput keeps the intake edge permissive. It is converted to canonical
// enums only when that conversion is unambiguous; otherwise it becomes a
// private draft with exact diagnostics.
type CaptureInput struct {
	Kind, Scenario, Trigger, Approach, Evidence, Outcome, CreatedBy, IdempotencyKey string
	RuledOut                                                                        []string
}

func (s *Service) Capture(ctx context.Context, in CaptureInput) (CaptureResult, error) {
	if existing, found, err := s.store.FindByCaptureKey(in.IdempotencyKey); err != nil {
		return CaptureResult{}, err
	} else if found {
		return captureResultFor(existing), nil
	}
	r, complete, result := s.assessCapture(in, "")
	if !complete {
		r.ID = "rec-" + idgen.Generate()
		r.Draft = true
		r.CreatedAt = s.now()
		if err := s.store.Create(r); err != nil {
			return CaptureResult{}, err
		}
		result.Record = r
		result.NextAction = repairCommand(r.ID)
		return result, nil
	}
	if err := s.store.Create(r); err != nil {
		return CaptureResult{}, err
	}
	_ = s.indexer.IndexRecord(ctx, r)
	s.events.EmitRecordCreated(r.ID, string(r.Kind), r.Scenario, r.BacklogRef, false)
	result.Record = r
	result.Disposition = "published"
	return result, nil
}

// RepairCapture publishes the existing draft ID exactly once when the merged
// values are valid. An incomplete repair remains the same private draft.
func (s *Service) RepairCapture(ctx context.Context, id string, in CaptureInput) (CaptureResult, error) {
	old, err := s.store.Get(id)
	if err != nil {
		return CaptureResult{}, err
	}
	if !old.Draft {
		return CaptureResult{}, ErrStubLocked
	}
	if old.Capture != nil {
		in.Kind = firstNonBlank(in.Kind, old.Capture.Raw["kind"])
		in.Scenario = firstNonBlank(in.Scenario, old.Capture.Raw["scenario"])
		in.Trigger = firstNonBlank(in.Trigger, old.Capture.Raw["trigger"])
		in.Approach = firstNonBlank(in.Approach, old.Capture.Raw["approach"])
		in.Evidence = firstNonBlank(in.Evidence, old.Capture.Raw["evidence"])
		in.Outcome = firstNonBlank(in.Outcome, old.Capture.Raw["outcome"])
	}
	in.IdempotencyKey = firstNonBlank(in.IdempotencyKey, old.CaptureKey)
	r, complete, result := s.assessCapture(in, id)
	if !complete {
		r.ID = id
		r.Draft = true
		r.CreatedAt = old.CreatedAt
		if err := s.replaceDraft(id, r); err != nil {
			return CaptureResult{}, err
		}
		result.Record = r
		result.NextAction = repairCommand(id)
		return result, nil
	}
	if _, err := s.store.UpdateDraft(id, r); err != nil {
		return CaptureResult{}, err
	}
	_ = s.indexer.IndexRecord(ctx, r)
	s.events.EmitRecordCreated(r.ID, string(r.Kind), r.Scenario, r.BacklogRef, false)
	result.Record = r
	result.Disposition = "published"
	return result, nil
}

func (s *Service) replaceDraft(id string, r Record) error {
	// A draft-to-draft repair is represented by replacing its JSON payload in
	// place through a private temporary published move is deliberately avoided.
	_, err := s.store.UpdateDraft(id, r)
	return err
}

func (s *Service) assessCapture(in CaptureInput, id string) (Record, bool, CaptureResult) {
	raw := map[string]string{"kind": strings.TrimSpace(in.Kind), "scenario": strings.TrimSpace(in.Scenario), "trigger": strings.TrimSpace(in.Trigger), "approach": strings.TrimSpace(in.Approach), "evidence": strings.TrimSpace(in.Evidence), "outcome": strings.TrimSpace(in.Outcome), "idempotency_key": strings.TrimSpace(in.IdempotencyKey)}
	accepted, needs, invalid := map[string]string{}, []string{}, []InvalidField{}
	kind, e := ParseKind(in.Kind)
	if raw["kind"] == "" {
		needs = append(needs, "kind")
	} else if e != nil {
		invalid = append(invalid, InvalidField{Field: "kind", Value: in.Kind, Message: e.Error()})
	} else {
		accepted["kind"] = string(kind)
	}
	outcome, e := ParseOutcome(in.Outcome)
	if raw["outcome"] == "" {
		needs = append(needs, "outcome")
	} else if e != nil {
		invalid = append(invalid, InvalidField{Field: "outcome", Value: in.Outcome, Message: e.Error()})
	} else {
		accepted["outcome"] = string(outcome)
	}
	if raw["scenario"] == "" {
		needs = append(needs, "scenario")
	} else {
		accepted["scenario"] = raw["scenario"]
	}
	if raw["trigger"] == "" && raw["approach"] == "" && len(trimAll(in.RuledOut)) == 0 {
		needs = append(needs, "trigger_or_approach_or_ruled_out")
	}
	r := Record{ID: id, Kind: kind, Scenario: raw["scenario"], Trigger: raw["trigger"], Approach: raw["approach"], Evidence: raw["evidence"], Outcome: outcome, RuledOut: trimAll(in.RuledOut), CreatedBy: strings.TrimSpace(in.CreatedBy), CaptureKey: raw["idempotency_key"], NarrativeAt: s.now()}
	result := CaptureResult{Disposition: "draft", Accepted: accepted, Needs: needs, Invalid: invalid, Warnings: []string{"Draft saved privately; it is not searchable or published."}}
	if len(needs) > 0 || len(invalid) > 0 {
		r.Capture = &CaptureMetadata{Raw: raw, Accepted: accepted, Needs: needs, Invalid: invalid}
		return r, false, result
	}
	r.ID = firstNonBlank(id, "rec-"+idgen.Generate())
	r.CreatedAt = s.now()
	return r, true, result
}

func captureResultFor(r Record) CaptureResult {
	result := CaptureResult{Record: r}
	if r.Draft {
		result.Disposition = "draft"
		if r.Capture != nil {
			result.Accepted = r.Capture.Accepted
			result.Needs = r.Capture.Needs
			result.Invalid = r.Capture.Invalid
			result.Warnings = r.Capture.Warnings
		}
		result.NextAction = repairCommand(r.ID)
		return result
	}
	result.Disposition = "published"
	return result
}

func repairCommand(id string) []string {
	return []string{"swarm-manager", "records", "edit", "--repair", "--id", id, "--trigger", "<trigger>", "--approach", "<approach>", "--outcome", "<outcome>"}
}
func firstNonBlank(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return b
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
		InitiativeID: strings.TrimSpace(in.InitiativeID),
		Supersedes:   strings.TrimSpace(in.Supersedes),
		Trigger:      strings.TrimSpace(in.Trigger),
		Approach:     strings.TrimSpace(in.Approach),
		RuledOut:     trimAll(in.RuledOut),
		Evidence:     strings.TrimSpace(in.Evidence),
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
