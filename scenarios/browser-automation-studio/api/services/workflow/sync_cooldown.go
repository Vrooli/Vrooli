package workflow

import (
	"time"

	"github.com/google/uuid"
)

func (s *WorkflowService) shouldSyncProject(projectID uuid.UUID) bool {
	if projectID == uuid.Nil {
		return false
	}
	if value, ok := s.projectSyncTimes.Load(projectID); ok {
		if last, ok := value.(time.Time); ok {
			if time.Since(last) < projectSyncCooldown {
				return false
			}
		}
	}
	return true
}

func (s *WorkflowService) recordProjectSync(projectID uuid.UUID) {
	s.projectSyncTimes.Store(projectID, time.Now())
}

