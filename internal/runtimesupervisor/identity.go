package runtimesupervisor

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// This metadata is written by the running owner when it claims its session.
// An inspecting CLI's build/configuration is never evidence of that owner.
type supervisorOwnerIdentity struct {
	BuildIdentity      string `json:"build_identity"`
	ProcessIncarnation string `json:"process_incarnation,omitempty"`
}

func supervisorSessionIdentity(session scenarioruntime.SupervisorSession) (supervisorOwnerIdentity, error) {
	var identity supervisorOwnerIdentity
	if strings.TrimSpace(session.MetadataJSON) == "" {
		return identity, nil
	}
	if err := json.Unmarshal([]byte(session.MetadataJSON), &identity); err != nil {
		return identity, fmt.Errorf("supervisor owner metadata is malformed")
	}
	return identity, nil
}

func supervisorSessionBuildIdentity(session scenarioruntime.SupervisorSession) string {
	identity, err := supervisorSessionIdentity(session)
	if err != nil || strings.TrimSpace(identity.BuildIdentity) == "" {
		return scenarioruntime.HealthStatusUnknown
	}
	return identity.BuildIdentity
}

func (s *Service) supervisorOwnerMetadata(pid int) string {
	identity := supervisorOwnerIdentity{BuildIdentity: s.buildIdentity()}
	if incarnation, err := process.Incarnation(pid); err == nil {
		identity.ProcessIncarnation = incarnation
	}
	data, _ := json.Marshal(identity)
	return string(data)
}
