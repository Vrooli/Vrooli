package portability

import (
	"context"
	"strings"
	"time"
)

// Service is the domain's read surface. It is read-only by construction: the
// instrument reports what the manifests declare and never writes a manifest,
// starts a scenario, or reconciles a platform gap. This scenario has no
// controller letter, so an actuation path here would be a contract violation
// rather than a missing feature.
type Service struct {
	root string
	now  func() time.Time
}

// NewService binds the domain to one repository root. The root is validated
// lazily, at read time, so a scenario started outside a repository still
// serves an explicit error instead of failing to boot — but it never serves an
// empty grid.
func NewService(root string, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{root: strings.TrimSpace(root), now: now}
}

// Root returns the configured repository root, resolved or not.
func (s *Service) Root() string { return s.root }

// Reader validates the root and returns the typed manifest reader.
func (s *Service) Reader() (*Reader, error) { return NewReader(s.root) }

// Grid computes the whole capability grid.
func (s *Service) Grid(ctx context.Context) (Grid, error) {
	if err := ctx.Err(); err != nil {
		return Grid{}, err
	}
	reader, err := s.Reader()
	if err != nil {
		return Grid{}, err
	}
	return reader.Grid(s.now())
}

// Fleet computes the fleet view over the same grid.
func (s *Service) Fleet(ctx context.Context) (FleetReadout, error) {
	if err := ctx.Err(); err != nil {
		return FleetReadout{}, err
	}
	reader, err := s.Reader()
	if err != nil {
		return FleetReadout{}, err
	}
	return reader.Fleet(s.now())
}
