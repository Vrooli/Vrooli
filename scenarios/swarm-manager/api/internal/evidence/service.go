package evidence

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	store    *Store
	resolver RunOwnerResolver
}

func NewService(store *Store, resolver RunOwnerResolver) *Service {
	return &Service{store: store, resolver: resolver}
}

// SetOwnerResolver completes service wiring after both owner indexes exist.
func (s *Service) SetOwnerResolver(resolver RunOwnerResolver) {
	if s != nil {
		s.resolver = resolver
	}
}

func (s *Service) Ingest(ctx context.Context, observation Observation) (IngestResult, error) {
	if s == nil || s.store == nil {
		return IngestResult{}, fmt.Errorf("evidence service is not configured")
	}
	status, owner, resolveErr := s.resolver.Resolve(ctx, observation.RunID)
	if resolveErr != nil && status != OwnershipUnavailable {
		return IngestResult{}, resolveErr
	}
	id, duplicate, err := s.store.UpsertObservation(ctx, observation, status)
	if err != nil {
		return IngestResult{}, err
	}
	result := IngestResult{ObservationID: id, Owner: owner, OwnershipStatus: status, Duplicate: duplicate}
	if status != OwnershipResolved {
		return result, nil
	}
	if err := s.store.Link(ctx, id, *owner); err != nil {
		return IngestResult{}, err
	}
	return result, nil
}

// IngestForOwner imports a source observation whose owner is already
// authoritative at the migration boundary (for example a historical Session
// artifact stored under that Session). It never guesses a run owner.
func (s *Service) IngestForOwner(ctx context.Context, owner Owner, observation Observation) (IngestResult, error) {
	if s == nil || s.store == nil {
		return IngestResult{}, fmt.Errorf("evidence service is not configured")
	}
	results, err := s.store.IngestForOwnerBatch(ctx, owner, []Observation{observation})
	if err != nil {
		return IngestResult{}, err
	}
	return results[0], nil
}

// IngestForOwnerBatch records an explicit-owner batch as one atomic ledger
// operation. It is used by compatibility projections so the canonical ledger
// never exposes a partially attached group of Session artifacts.
func (s *Service) IngestForOwnerBatch(ctx context.Context, owner Owner, observations []Observation) ([]IngestResult, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("evidence service is not configured")
	}
	return s.store.IngestForOwnerBatch(ctx, owner, observations)
}

// RecordOperatorVerified appends an explicit operator repair. It is never a
// promotion of an existing observation: the actor and reason become a new,
// immutable source observation that can independently satisfy a mode gate.
func (s *Service) RecordOperatorVerified(ctx context.Context, owner Owner, sourceEventID, runID string, subject Subject, action, actor, reason string, metadata map[string]string) (IngestResult, error) {
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(reason) == "" {
		return IngestResult{}, fmt.Errorf("operator evidence requires actor and reason")
	}
	if metadata == nil {
		metadata = map[string]string{}
	} else {
		copyMetadata := make(map[string]string, len(metadata)+2)
		for key, value := range metadata {
			copyMetadata[key] = value
		}
		metadata = copyMetadata
	}
	metadata["operator_actor"] = strings.TrimSpace(actor)
	metadata["operator_reason"] = strings.TrimSpace(reason)
	return s.IngestForOwner(ctx, owner, Observation{
		SourceSystem: "swarm-manager.operator", SourceEventID: strings.TrimSpace(sourceEventID), RunID: strings.TrimSpace(runID),
		Subject: subject, Action: strings.TrimSpace(action), Confidence: ConfidenceOperator, Verification: VerificationVerified,
		Metadata: metadata, ObservedAt: time.Now().UTC(),
	})
}

func (s *Service) ListByOwner(ctx context.Context, owner Owner) ([]Record, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("evidence service is not configured")
	}
	return s.store.ListByOwner(ctx, owner)
}

func (s *Service) ListByOwnerID(ctx context.Context, kind OwnerKind, id string) ([]Record, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("evidence service is not configured")
	}
	return s.store.ListByOwnerID(ctx, kind, id)
}

func (s *Service) ListByRun(ctx context.Context, runID string) ([]Record, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("evidence service is not configured")
	}
	return s.store.ListByRun(ctx, runID)
}

func (s *Service) ListByEntity(ctx context.Context, subject Subject) ([]Record, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("evidence service is not configured")
	}
	return s.store.ListByEntity(ctx, subject)
}

func (s *Service) SaveMigrationAudit(ctx context.Context, audit MigrationAudit) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("evidence service is not configured")
	}
	return s.store.SaveMigrationAudit(ctx, audit)
}
