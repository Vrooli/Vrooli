// Package skillreads owns bounded skill-read telemetry. It is independent of
// the storage engine used by its repository.
package skillreads

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

const MaxEntries = 20000
const Retention = 30 * 24 * time.Hour

type Repository interface {
	Append(context.Context, Read, time.Time, int, time.Duration) error
	ReadSince(context.Context, time.Duration, time.Time) ([]Read, error)
	RecordReceipt(context.Context, string, string, time.Time, time.Duration) error
	Receipt(context.Context, string, string) (Receipt, bool, error)
}

type Service struct {
	repository Repository
	ctx        context.Context
	now        func() time.Time
	maxEntries int
	retention  time.Duration
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, ctx: context.Background(), now: time.Now, maxEntries: MaxEntries, retention: Retention}
}

// WithContext preserves request-scoped database routing without capturing a pool
// or mutating the shared service.
func (s *Service) WithContext(ctx context.Context) *Service {
	copy := *s
	copy.ctx = ctx
	return &copy
}

func (s *Service) Append(read Read) error {
	now := s.now().UTC()
	if strings.TrimSpace(read.At) == "" {
		read.At = now.Format(time.RFC3339)
	}
	if strings.TrimSpace(read.ID) == "" {
		read.ID = uuid.NewString()
	}
	return s.repository.Append(s.ctx, read, now, s.maxEntries, s.retention)
}

func (s *Service) ReadSince(window time.Duration) ([]Read, error) {
	return s.repository.ReadSince(s.ctx, window, s.now().UTC())
}

// RecordReceipt marks that a verified run read a skill. Callers pass only a
// run ID obtained from server-side identity verification.
func (s *Service) RecordReceipt(runID, skillID string) error {
	runID, skillID = strings.TrimSpace(runID), strings.TrimSpace(skillID)
	if runID == "" || skillID == "" {
		return nil
	}
	return s.repository.RecordReceipt(s.ctx, runID, skillID, s.now().UTC(), s.retention)
}

// Receipt reports the run's read of a skill. found is false when the run has
// no recorded read inside the retention window.
func (s *Service) Receipt(runID, skillID string) (Receipt, bool, error) {
	runID, skillID = strings.TrimSpace(runID), strings.TrimSpace(skillID)
	if runID == "" || skillID == "" {
		return Receipt{}, false, nil
	}
	receipt, found, err := s.repository.Receipt(s.ctx, runID, skillID)
	if err != nil || !found {
		return Receipt{}, false, err
	}
	if s.retention > 0 && receipt.LastReadAt.Before(s.now().UTC().Add(-s.retention)) {
		return Receipt{}, false, nil
	}
	return receipt, true, nil
}
