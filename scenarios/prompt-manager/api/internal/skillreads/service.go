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
