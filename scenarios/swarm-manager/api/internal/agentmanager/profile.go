package agentmanager

import (
	"fmt"
	"strings"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

// GetProfileID returns the current profile ID.
func (s *AgentService) GetProfileID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.profileID
}

func (s *AgentService) profileRefFor(profileKey string) (*apipb.ProfileRef, error) {
	key := strings.TrimSpace(profileKey)
	if key == "" {
		key = s.profileKey
	}
	if key == "" {
		return nil, nil
	}

	s.mu.RLock()
	if len(s.profileIDs) > 0 {
		if _, ok := s.profileIDs[key]; !ok {
			s.mu.RUnlock()
			return nil, fmt.Errorf("agent-manager profile %q was not returned by scenario profile reconciliation", key)
		}
	}
	s.mu.RUnlock()

	// Profile defaults are reconciled from .vrooli/agent-profiles at startup.
	// Run creation only references the stable profile key so dispatch cannot
	// overwrite DB edits with code-declared defaults.
	return &apipb.ProfileRef{
		ProfileKey: key,
	}, nil
}
