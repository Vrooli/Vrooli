package supervision

import (
	"context"
	"errors"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// EffortEnrollment returns one effort's current enrollment. Run-authority
// checks read it to learn which scenario an effort targets; it grants nothing
// by itself. A nil service reports unavailability instead of panicking, so the
// composition root can pass an optional supervision service through.
func (s *Service) EffortEnrollment(ctx context.Context, ref string) (*pb.EffortEnrollment, error) {
	if s == nil || s.Efforts == nil || s.Efforts.repo == nil {
		return nil, errors.New("effort supervision unavailable")
	}
	enrollment, _, err := s.Efforts.repo.GetEffort(ctx, ref)
	return enrollment, err
}
