package backlog

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Source identifies who or what triggered a backlog mutation. The Service
// uses Source to drive policy decisions (workshop auto-trigger fires only
// for human-driven HTTP creates) and to attribute the durable event-log
// record to the originating surface.
type Source string

const (
	// SourceHumanHTTP — single-item POST /api/v1/backlog from the UI/CLI.
	// Triggers auto-workshop, records actor as "user".
	SourceHumanHTTP Source = "human_http"

	// SourceBatch — bulk POST /api/v1/backlog/batch. Cycle checking is
	// performed at batch level so per-item cycle checks are redundant.
	// Triggers auto-workshop per item, records actor as "user".
	SourceBatch Source = "batch"

	// SourceProposal — proposals.Applier add_item from a feedback or
	// review round. Skips workshop (the agent already picked the item)
	// and records the originating round as the actor for attribution.
	SourceProposal Source = "proposal"
)

// CreationContext is the per-call attribution + policy parameter passed
// to Service.Create. Source is required; the other fields are populated
// by sources that carry meaningful provenance.
type CreationContext struct {
	Source Source

	// DecidedBy is a free-form actor label (user identity, agent name).
	// Currently advisory — the eventlog ActorID is derived from
	// FeedbackRoundID / ReviewRoundID when present.
	DecidedBy string

	// FeedbackRoundID is "<initiative>/round-<NNN>" populated by
	// proposals.Applier when applying a feedback-round mutation.
	FeedbackRoundID string

	// ReviewRoundID is "<initiative>/review-<NNN>" populated by the
	// initiative review flow's auto-apply path.
	ReviewRoundID string

	// RoundNumber and RoundSlug mirror the round metadata so consumers
	// don't have to re-parse the round ID to filter.
	RoundNumber int
	RoundSlug   string

	// Entrypoint identifies the originating skill or HTTP path so
	// downstream telemetry can group by code surface
	// ("initiative.feedback", "initiative.review", "http.create").
	Entrypoint string

	// SkipCycleCheck lets the batch path opt out of per-item cycle
	// validation after running its bulk check. Other sources should
	// leave this false.
	SkipCycleCheck bool

	// SkipDuplicateCheck lets callers that pre-validated absence (the
	// batch path stat-checks every item up front) skip the redundant
	// LoadItem inside Service.Create.
	SkipDuplicateCheck bool

	// SkipInitiativeAttach lets the batch path persist items first
	// then bulk-attach to initiatives via AddItems (one initiative.json
	// write per initiative, instead of N writes from per-item
	// RememberItem). Other sources should leave this false.
	SkipInitiativeAttach bool

	// SkipWorkshopTrigger lets the batch path defer workshop spawns to
	// after all items in the batch are persisted, so cascade-trigger
	// reasoning sees the full set. Other sources should leave this
	// false; the workshop runs synchronously by default.
	SkipWorkshopTrigger bool

	// SkipGraphInvalidation lets the proposals.Applier batch graph
	// invalidation across an entire mutation set rather than firing
	// one re-projection per add_item.
	SkipGraphInvalidation bool
}

// CreationStore is the persistence surface Service.Create uses. Satisfied
// by *FileStore.
type CreationStore interface {
	ItemDir(kind BacklogKind, name string) string
	LoadItem(kind BacklogKind, name string) (BacklogItem, error)
	SaveItem(item BacklogItem) error
	ValidateDependencies(dependsOn []string) error
}

// CreationEventEmitter is the eventlog surface Service.Create uses. The
// dual signature (default and From-Source variant) keeps existing callers
// unchanged while letting Service emit attributed events. Satisfied by
// *eventlog.Emitter.
type CreationEventEmitter interface {
	EmitBacklogCreatedFromSource(entityID, kind, status string, priority int, initiative, effort, actorType, actorID string)
}

// WorkshopTrigger fires the auto-workshop policy for a freshly-created
// item. Pluggable so the Service stays free of agent-manager / settings
// dependencies; main.go wires Handler.maybeAutoWorkshop in.
type WorkshopTrigger interface {
	MaybeStartWorkshop(item BacklogItem)
}

// WorkshopTriggerFunc is a func adapter for WorkshopTrigger so callers
// can pass a closure without defining a type.
type WorkshopTriggerFunc func(item BacklogItem)

func (f WorkshopTriggerFunc) MaybeStartWorkshop(item BacklogItem) { f(item) }

// GraphInvalidator schedules a graph re-projection after a successful
// creation so the per-initiative graph.json reflects the new node.
// Optional — a nil invalidator means the caller is responsible for
// invalidating (the proposals.Applier batches invalidation across an
// entire mutation set).
type GraphInvalidator interface {
	ScheduleAll()
}

// CycleChecker validates that adding `item` would not introduce a
// dependency cycle. Optional — a nil checker disables per-item cycle
// validation, which is what the batch path wants after running its
// bulk check.
type CycleChecker interface {
	CheckCycles(item BacklogItem) error
}

// CycleCheckerFunc is a func adapter for CycleChecker.
type CycleCheckerFunc func(item BacklogItem) error

func (f CycleCheckerFunc) CheckCycles(item BacklogItem) error { return f(item) }

// Service is the chokepoint for backlog item creation. All callers —
// HTTP single, HTTP batch, and proposals.Applier — go through Create so
// the side-effect set (eventlog emission, attribution, workshop trigger,
// graph invalidation) is consistent regardless of who initiated the
// mutation.
type Service struct {
	store        CreationStore
	assigner     ItemAttacher
	events       CreationEventEmitter
	workshop     WorkshopTrigger
	invalidator  GraphInvalidator
	cycleChecker CycleChecker
}

// ServiceConfig bundles Service dependencies. Store is required; the
// rest are optional and degrade gracefully (nil emitter = no event,
// nil workshop = no auto-spawn, etc.).
type ServiceConfig struct {
	Store        CreationStore
	Assigner     ItemAttacher
	Events       CreationEventEmitter
	Workshop     WorkshopTrigger
	Invalidator  GraphInvalidator
	CycleChecker CycleChecker
}

// NewService constructs a Service. Returns an error if Store is nil
// because creation cannot persist without it.
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Store == nil {
		return nil, errors.New("backlog.NewService: Store is required")
	}
	return &Service{
		store:        cfg.Store,
		assigner:     cfg.Assigner,
		events:       cfg.Events,
		workshop:     cfg.Workshop,
		invalidator:  cfg.Invalidator,
		cycleChecker: cfg.CycleChecker,
	}, nil
}

// Create persists `item` and runs the side-effect set appropriate for
// `cc.Source`. Errors map to:
//   - ErrNotFound         — depends_on references a missing item
//   - apierr-style wrapped errors are NOT returned; callers wrap as needed
//   - the duplicate sentinel is fmt.Errorf with errors.Is(ErrItemExists)
func (s *Service) Create(item BacklogItem, cc CreationContext) error {
	if cc.Source == "" {
		return errors.New("backlog.Service.Create: CreationContext.Source is required")
	}
	if item.Name == "" {
		return errors.New("backlog.Service.Create: item.Name is required")
	}
	if item.Kind == "" {
		return errors.New("backlog.Service.Create: item.Kind is required")
	}
	if _, ok := backlogKindDirs[item.Kind]; !ok {
		return fmt.Errorf("backlog.Service.Create: unknown kind %q", item.Kind)
	}

	if !cc.SkipDuplicateCheck {
		if _, err := s.store.LoadItem(item.Kind, item.Name); err == nil {
			return fmt.Errorf("%w: %s/%s", ErrItemExists, item.Kind, item.Name)
		} else if !errors.Is(err, ErrNotFound) {
			return fmt.Errorf("duplicate check: %w", err)
		}
	}

	if len(item.DependsOn) > 0 {
		if err := s.store.ValidateDependencies(item.DependsOn); err != nil {
			return fmt.Errorf("depends_on: %w", err)
		}
		if !cc.SkipCycleCheck && s.cycleChecker != nil {
			if err := s.cycleChecker.CheckCycles(item); err != nil {
				return err
			}
		}
	}

	itemDir := s.store.ItemDir(item.Kind, item.Name)
	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		return fmt.Errorf("create item dir: %w", err)
	}

	if err := s.store.SaveItem(item); err != nil {
		_ = os.RemoveAll(itemDir)
		return fmt.Errorf("save item: %w", err)
	}

	if attachName := strings.TrimSpace(item.Initiative); attachName != "" && s.assigner != nil && !cc.SkipInitiativeAttach {
		ref := string(item.Kind) + "/" + item.Name
		if err := s.assigner.RememberItem(attachName, ref); err != nil {
			_ = os.RemoveAll(itemDir)
			return fmt.Errorf("attach %s to initiative %s: %w", ref, attachName, err)
		}
	}

	if s.events != nil {
		actorType, actorID := actorForSource(cc)
		s.events.EmitBacklogCreatedFromSource(
			string(item.Kind)+"/"+item.Name,
			string(item.Kind),
			string(item.Status),
			item.Priority,
			item.Initiative,
			item.Effort,
			actorType,
			actorID,
		)
	}

	if (cc.Source == SourceHumanHTTP || cc.Source == SourceBatch) && !cc.SkipWorkshopTrigger {
		if s.workshop != nil {
			s.workshop.MaybeStartWorkshop(item)
		}
	}

	if s.invalidator != nil && !cc.SkipGraphInvalidation {
		s.invalidator.ScheduleAll()
	}

	return nil
}

// actorForSource derives the eventlog (actorType, actorID) tuple for a
// creation. Review rounds dominate feedback rounds when both IDs are
// present (review-applied follow-ups attribute to the review).
func actorForSource(cc CreationContext) (string, string) {
	switch {
	case cc.ReviewRoundID != "":
		return "initiative_review", cc.ReviewRoundID
	case cc.FeedbackRoundID != "":
		return "feedback_round", cc.FeedbackRoundID
	default:
		return "user", cc.DecidedBy
	}
}

// ErrItemExists is returned by Service.Create when an item with the
// requested kind+name already exists on disk. Callers map it to their
// transport's conflict response (HTTP 409 for handlers, an Outcome
// failure for proposals.Applier).
var ErrItemExists = errors.New("backlog item already exists")
